package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/keruch/ton_masks_bot/config"
	repo "github.com/keruch/ton_masks_bot/internal/repository"
	log "github.com/keruch/ton_masks_bot/pkg/logger"
)

type repository interface {
	AddUser(ctx context.Context, ID int64, username string, invitedID int64) error
	GetFieldForID(ctx context.Context, ID int64, field string) (interface{}, error)
	UpdatePoints(ctx context.Context, ID int64, points int) error
	UpdateSubscription(ctx context.Context, subscription string, ID int64, value bool) error
	GetInvitedByID(ctx context.Context, ID int64) (int64, error)
	GetUsername(ctx context.Context, ID int64) (string, error)
	GetPointsByID(ctx context.Context, ID int64) (int, error)
	GetRating(ctx context.Context, limit int) ([]string, error)
}
type TgBot struct {
	*tgbotapi.BotAPI
	repo   repository
	logger *log.Logger
}

type (
	ChannelName        = string
	ChannelDBFiled     = string
	SubscriptionAction bool
)

const (
	startCommand  = "start"
	pointsCommand = "points"
	ratingCommand = "rating"
	infoCommand   = "info"

	AlreadyRegisteredMessage = "Вы уже зарегистрированы на участие в конкурсе!"
	UnsubscribedMessage      = "Вы отписались от канала @%s и больше не участвуете в конкурсе. Подпишитесь, чтобы опять принять участие."
	MissingCommandMessage    = "Куда-то ты не туда полез дружок..."

	FriendUnsubscribedFormatString    = "Ваш друг @%s отписался от канала @%s и больше не участвует в конкурсе. Пришлось забрать ваши 50 баллов :("
	FriendSubscribedToAllFormatString = "Ваш друг @%s подписался на все каналы из условий и теперь участвует в конкурсе. А вы получили 100 баллов!"

	PersonalLinkFormatString = "Ваша персональная ссылка для приглашения: \n\nhttps://t.me/%s?start=%d"

	TheOpenArtChannelTag ChannelName = "@theopenart"
	TheOpenArtChannel    ChannelName = "theopenart"

	TheOpenArtDBField ChannelDBFiled = "openart"
	AdditionalDBField ChannelDBFiled = "additional"

	subscribeAction   SubscriptionAction = true
	unsubscribeAction SubscriptionAction = false
)

func ChannelToDBMapping(name ChannelName) ChannelDBFiled {
	switch name {
	case TheOpenArtChannelTag:
		return TheOpenArtDBField
	case TheOpenArtChannel:
		return TheOpenArtDBField
	default:
		return AdditionalDBField
	}
}

func NewTgBot(token string, repo repository, logger *log.Logger) (*TgBot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &TgBot{
		BotAPI: bot,
		repo:   repo,
		logger: logger,
	}, nil
}

func (tg *TgBot) Serve(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"chat_member", "inline_query", "callback_query", "message"}
	updates := tg.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			tg.logger.Println("Telegram bot: serve done")
			return
		case update := <-updates:
			tg.processUpdate(ctx, update)
		}
	}
}

func (tg *TgBot) processUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.Message != nil {
		tg.processMessage(ctx, update)
	} else if update.CallbackQuery != nil {
		tg.processCallback(ctx, update)
	} else if update.ChatMember != nil {
		tg.processChatMember(ctx, update)
	}
}

func (tg *TgBot) processMessage(ctx context.Context, update tgbotapi.Update) {
	var (
		chatID   = update.Message.Chat.ID
		userID   = update.Message.From.ID
		userName = update.Message.From.UserName
	)

	msg := tgbotapi.NewMessage(chatID, "Something went wrong!")
	msg.ParseMode = tgbotapi.ModeHTML

	switch update.Message.Command() {
	case startCommand:
		msg.Text = config.GetStartMessage() + config.GetSubscribeToJoinMessage()
		ID, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if ID == userID {
			ID = 0
		}
		tg.logger.WithField("Command", startCommand).WithField("User", userName).WithField("User ID", userID).WithField("Invited By", ID).Info()
		if err == nil && ID != 0 {
			invitedUser, err := tg.repo.GetUsername(ctx, ID)
			if err != nil {
				tg.logger.WithField("Command", startCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "GetFieldForID").Error(err)
			}
			msg.Text = fmt.Sprintf("Вы были приглашены другом @%v!\n\n%s", invitedUser, config.GetStartMessage()+config.GetSubscribeToJoinMessage())
		}
		if err = tg.repo.AddUser(ctx, userID, userName, ID); err != nil {
			if errors.Is(err, repo.ErrorAlreadyRegistered) {
				tg.logger.WithField("Command", startCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "AddUser").Info(err)
				msg.Text = AlreadyRegisteredMessage
				msg.ReplyMarkup = createInlineKeyboardMarkupWithID(userID)
				if _, err := tg.Send(msg); err != nil {
					tg.logger.WithField("Command", startCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "Send").Error(err)
					return
				}
				return
			}
			tg.logger.WithField("Command", startCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "AddUser").Error(err)
			return
		}
		tg.logger.WithField("Command", startCommand).WithField("User", userName).WithField("User ID", userID).WithField("Invited By", ID).Info("new user added to db")
		if _, err := tg.Send(msg); err != nil {
			tg.logger.WithField("Command", startCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "Send").Error(err)
			return
		}

		subToAll := true
		for _, sub := range config.GetRequiredChannels() {
			ok, err := tg.isSubscribed(userID, sub)
			if err != nil {
				tg.logger.WithField("Command", startCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "isSubscribed").WithField("Channel", sub).Error(err)
				return
			}

			if ok {
				if err := tg.updateSubscription(ctx, userID, userName, sub, subscribeAction, 50); err != nil {
					tg.logger.WithField("Command", startCommand).WithField("User", userName).WithField("User ID", userID).WithField("Channel", sub).WithField("Method", "updateSubscription").Error(err)
					return
				}
			}

			subToAll = subToAll && ok
		}

		if subToAll {
			msg.ReplyMarkup = createInlineKeyboardMarkupWithID(userID)
			msg.Text = config.GetSubscribedToAllMessage()
			if _, err := tg.Send(msg); err != nil {
				tg.logger.WithField("Command", startCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "Send").WithField("Message", "Subscribed to all message").Error(err)
				return
			}

			if ID != 0 {
				msg = tgbotapi.NewMessage(ID, "Something went wrong!")
				msg.ParseMode = tgbotapi.ModeHTML
				msg.Text = fmt.Sprintf(FriendSubscribedToAllFormatString, userName)

				if _, err = tg.Send(msg); err != nil {
					tg.logger.WithField("Command", startCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "Send").WithField("Message", "Friend subscribed to all message").Error(err)
					return
				}
			}
		}
	default:
		tg.logger.WithField("Command", "missing command").WithField("User", userName).WithField("User ID", userID).Info()
		msg.Text = MissingCommandMessage
		if _, err := tg.Send(msg); err != nil {
			tg.logger.WithField("Method", "Send").Error(err)
			return
		}
	}
}

func (tg *TgBot) processCallback(ctx context.Context, update tgbotapi.Update) {
	var (
		chatID   = update.CallbackQuery.Message.Chat.ID
		userID   = update.CallbackQuery.From.ID
		userName = update.CallbackQuery.From.UserName
	)

	callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackData())
	if _, err := tg.Request(callback); err != nil {
		tg.logger.WithField("Method", "Request").Error(err)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "Something went wrong!")
	msg.ParseMode = tgbotapi.ModeHTML

	switch update.CallbackQuery.Data {
	case pointsCommand:
		tg.logger.WithField("Command", pointsCommand).WithField("User", userName).WithField("User ID", userID).Info()
		points, err := tg.repo.GetPointsByID(ctx, userID)
		if err != nil {
			tg.logger.WithField("Command", pointsCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "GetPointsByID").Error(err)
		}
		msg.ReplyMarkup = createInlineKeyboardMarkupWithID(userID)
		msg.Text = fmt.Sprintf("У вас %v баллов", points)
	case ratingCommand:
		tg.logger.WithField("Command", ratingCommand).WithField("User", userName).WithField("User ID", userID).Info()
		rating, err := tg.repo.GetRating(ctx, 10)
		if err != nil {
			tg.logger.WithField("Command", pointsCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "GetRating").Error(err)
		}
		ratingString := createRating(rating)
		msg.ReplyMarkup = createInlineKeyboardMarkupWithID(userID)
		msg.Text = ratingString
	case infoCommand:
		msg.ReplyMarkup = createInlineKeyboardMarkupWithID(userID)
		msg.Text = config.GetStartMessage() + config.GetSubscribeToJoinMessage()
	}

	if _, err := tg.Send(msg); err != nil {
		tg.logger.WithField("Command", pointsCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "Send").Error(err)
	}
}

func (tg *TgBot) processChatMember(ctx context.Context, update tgbotapi.Update) {
	var (
		userID      = update.ChatMember.From.ID
		userName    = update.ChatMember.From.UserName
		channelName = ChannelName(update.ChatMember.Chat.UserName)
	)

	if update.ChatMember.NewChatMember.Status == "left" {
		tg.logger.WithField("When", "Update status to left").WithField("User", userName).WithField("User ID", userID).WithField("Channel", channelName).WithField("Method", "updateSubscription").WithField("Action", "Unsubscribe").Info("user unsubscribed from channel")
		if err := tg.updateSubscription(ctx, userID, userName, channelName, unsubscribeAction, -50); err != nil {
			tg.logger.WithField("When", "Update status to left").WithField("User", userName).WithField("User ID", userID).WithField("Channel", channelName).WithField("Method", "updateSubscription").WithField("Action", "Subscribe").Error(err)
			return
		}
	} else if update.ChatMember.NewChatMember.Status == "member" {
		tg.logger.WithField("When", "Update status to member").WithField("User", userName).WithField("User ID", userID).WithField("Channel", channelName).WithField("Method", "updateSubscription").WithField("Action", "Unsubscribe").Info("user subscribed to channel")
		if err := tg.updateSubscription(ctx, userID, userName, channelName, subscribeAction, 50); err != nil {
			tg.logger.WithField("When", "Update status to member").WithField("User", userName).WithField("User ID", userID).WithField("Channel", channelName).WithField("Method", "updateSubscription").WithField("Action", "Unsubscribe").Error(err)
			return
		}
		ok, err := tg.isSubscribed(userID, config.GetRequiredChannels()...)
		if err != nil {
			tg.logger.WithField("When", "Update status to member").WithField("User", userName).WithField("User ID", userID).WithField("Channel", channelName).WithField("Method", "isSubscribed").WithField("Channels to sub", config.GetRequiredChannels()).Error(err)
			return
		}
		if ok {
			msg := tgbotapi.NewMessage(userID, "Something went wrong!")
			msg.ParseMode = tgbotapi.ModeHTML
			msg.Text = config.GetSubscribedToAllMessage()

			if _, err = tg.Send(msg); err != nil {
				tg.logger.WithField("When", "Update status to member").WithField("User", userName).WithField("User ID", userID).WithField("Method", "Send").Error(err)
				return
			}

			invitedByID, err := tg.repo.GetInvitedByID(ctx, userID)
			if err != nil {
				tg.logger.WithField("When", "Update status to member").WithField("User", userName).WithField("User ID", userID).WithField("Method", "GetInvitedByID").Error(err)
				return
			}

			if invitedByID != 0 {
				msg = tgbotapi.NewMessage(invitedByID, "Something went wrong!")
				msg.ParseMode = tgbotapi.ModeHTML
				msg.Text = fmt.Sprintf(FriendSubscribedToAllFormatString, userName)

				if _, err = tg.Send(msg); err != nil {
					tg.logger.WithField("When", "Update status to member").WithField("User", userName).WithField("User ID", userID).WithField("Method", "Send").Error(err)
					return
				}
			}
		}
	}
}

func (tg *TgBot) updateSubscription(ctx context.Context, userID int64, username string, channelName ChannelName, action SubscriptionAction, points int) error {
	channel := ChannelToDBMapping(channelName)
	if err := tg.repo.UpdateSubscription(ctx, string(channel), userID, bool(action)); err != nil {
		return err
	}

	invitedByID, err := tg.repo.GetInvitedByID(ctx, userID)
	if err != nil {
		return err
	}
	if invitedByID != 0 {
		err = tg.repo.UpdatePoints(ctx, invitedByID, points)
		if err != nil {
			return err
		}

		if action == unsubscribeAction {
			msg := tgbotapi.NewMessage(invitedByID, "Something went wrong!")
			msg.ParseMode = tgbotapi.ModeHTML
			msg.Text = fmt.Sprintf(FriendUnsubscribedFormatString, username, string(channelName))

			if _, err = tg.Send(msg); err != nil {
				return err
			}
		}
	}

	if action == unsubscribeAction {
		msg := tgbotapi.NewMessage(userID, "Something went wrong!")
		msg.ParseMode = tgbotapi.ModeHTML
		msg.Text = fmt.Sprintf(UnsubscribedMessage, string(channelName))

		if _, err = tg.Send(msg); err != nil {
			return err
		}
	}

	return nil
}

func createInlineKeyboardMarkupWithID(ID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonSwitch("Пригласить друга", fmt.Sprintf(PersonalLinkFormatString, config.GetTelegramBotTag(), ID)),
			tgbotapi.NewInlineKeyboardButtonData("Баллы", pointsCommand),
			tgbotapi.NewInlineKeyboardButtonData("Рейтинг", ratingCommand),
			tgbotapi.NewInlineKeyboardButtonData("Информация", infoCommand),
		),
	)
}

func (tg *TgBot) isSubscribed(userID int64, channels ...ChannelName) (bool, error) {
	result := true
	for _, channel := range channels {
		config := tgbotapi.ChatConfigWithUser{
			SuperGroupUsername: string(channel),
			UserID:             userID,
		}
		member, err := tg.GetChatMember(tgbotapi.GetChatMemberConfig{ChatConfigWithUser: config})
		if err != nil {
			return false, err
		}

		result = (member.Status == "member" || member.Status == "creator" || member.Status == "administrator") && result
	}

	return result, nil
}

const line = "%d: %s"

func createRating(rows []string) string {
	var result string
	for idx, row := range rows {
		result = result + fmt.Sprintf(line, idx+1, row) + "\n"
	}
	return result
}
