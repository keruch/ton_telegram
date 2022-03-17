package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	repo "github.com/keruch/ton_masks_bot/internal/repository"
	"github.com/keruch/ton_masks_bot/internal/telegram/config"
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
	cfg    *config.BotConfig
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

func NewTgBot(token string, repo repository, cfg *config.BotConfig, logger *log.Logger) (*TgBot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &TgBot{
		BotAPI: bot,
		repo:   repo,
		cfg:    cfg,
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
	msg.DisableWebPagePreview = true

	switch update.Message.Command() {
	case startCommand:
		msg.Text = tg.cfg.Messages.Start + tg.cfg.Messages.SubscribeToJoin
		msg.ReplyMarkup = tg.createInlineKeyboardMarkupWithID(userID, tg.cfg.Name)
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
			msg.Text = fmt.Sprintf(tg.cfg.FormatStrings.YouWereInvited, invitedUser, tg.cfg.Messages.Start+tg.cfg.Messages.SubscribeToJoin)
		}
		if err = tg.repo.AddUser(ctx, userID, userName, ID); err != nil {
			if errors.Is(err, repo.ErrorAlreadyRegistered) {
				tg.logger.WithField("Command", startCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "AddUser").Info(err)
				msg.Text = tg.cfg.Messages.AlreadyRegistered
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
		for _, sub := range tg.cfg.RequiredSubscriptions {
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
			msg.ReplyMarkup = tg.createInlineKeyboardMarkupWithID(userID, tg.cfg.Name)
			msg.Text = tg.cfg.Messages.SubscribedToAll
			if _, err := tg.Send(msg); err != nil {
				tg.logger.WithField("Command", startCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "Send").WithField("Message", "Subscribed to all message").Error(err)
				return
			}

			if ID != 0 {
				msg = tgbotapi.NewMessage(ID, "Something went wrong!")
				msg.ParseMode = tgbotapi.ModeHTML
				msg.Text = fmt.Sprintf(tg.cfg.FormatStrings.FriendSubscribedToAll, userName)
				msg.DisableWebPagePreview = true

				if _, err = tg.Send(msg); err != nil {
					tg.logger.WithField("Command", startCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "Send").WithField("Message", "Friend subscribed to all message").Error(err)
					return
				}
			}
		}
	default:
		tg.logger.WithField("Command", "missing command").WithField("User", userName).WithField("User ID", userID).Info()
		msg.Text = tg.cfg.Messages.MissingCommand
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
	msg.DisableWebPagePreview = true

	switch update.CallbackQuery.Data {
	case pointsCommand:
		tg.logger.WithField("Command", pointsCommand).WithField("User", userName).WithField("User ID", userID).Info()
		points, err := tg.repo.GetPointsByID(ctx, userID)
		if err != nil {
			tg.logger.WithField("Command", pointsCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "GetPointsByID").Error(err)
		}
		msg.ReplyMarkup = tg.createInlineKeyboardMarkupWithID(userID, tg.cfg.Name)
		msg.Text = fmt.Sprintf(tg.cfg.FormatStrings.Points, points)
	case ratingCommand:
		tg.logger.WithField("Command", ratingCommand).WithField("User", userName).WithField("User ID", userID).Info()
		rating, err := tg.repo.GetRating(ctx, tg.cfg.RatingLimit)
		if err != nil {
			tg.logger.WithField("Command", pointsCommand).WithField("User", userName).WithField("User ID", userID).WithField("Method", "GetRating").Error(err)
		}
		ratingString := createRating(rating)
		msg.ReplyMarkup = tg.createInlineKeyboardMarkupWithID(userID, tg.cfg.Name)
		msg.Text = ratingString
	case infoCommand:
		msg.ReplyMarkup = tg.createInlineKeyboardMarkupWithID(userID, tg.cfg.Name)
		msg.Text = tg.cfg.Messages.Start + tg.cfg.Messages.SubscribeToJoin
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
		ok, err := tg.isSubscribed(userID, tg.cfg.RequiredSubscriptions...)
		if err != nil {
			tg.logger.WithField("When", "Update status to member").WithField("User", userName).WithField("User ID", userID).WithField("Channel", channelName).WithField("Method", "isSubscribed").WithField("Channels to sub", tg.cfg.RequiredSubscriptions).Error(err)
			return
		}
		if ok {
			msg := tgbotapi.NewMessage(userID, "Something went wrong!")
			msg.ParseMode = tgbotapi.ModeHTML
			msg.Text = tg.cfg.Messages.SubscribedToAll
			msg.DisableWebPagePreview = true
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
				msg.Text = fmt.Sprintf(tg.cfg.FormatStrings.FriendSubscribedToAll, userName)
				msg.DisableWebPagePreview = true

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
			msg.Text = fmt.Sprintf(tg.cfg.FormatStrings.FriendUnsubscribed, username, string(channelName))
			msg.DisableWebPagePreview = true

			if _, err = tg.Send(msg); err != nil {
				return err
			}
		}
	}

	if action == unsubscribeAction {
		msg := tgbotapi.NewMessage(userID, "Something went wrong!")
		msg.ParseMode = tgbotapi.ModeHTML
		msg.Text = fmt.Sprintf(tg.cfg.FormatStrings.Unsubscribed, string(channelName))
		msg.DisableWebPagePreview = true

		if _, err = tg.Send(msg); err != nil {
			return err
		}
	}

	return nil
}

func (tg *TgBot) createInlineKeyboardMarkupWithID(ID int64, tag string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonSwitch(tg.cfg.Buttons.Invite, fmt.Sprintf(tg.cfg.FormatStrings.PersonalLink, tag, ID)),
			tgbotapi.NewInlineKeyboardButtonData(tg.cfg.Buttons.Points, pointsCommand),
			tgbotapi.NewInlineKeyboardButtonData(tg.cfg.Buttons.Rating, ratingCommand),
			tgbotapi.NewInlineKeyboardButtonData(tg.cfg.Buttons.Info, infoCommand),
		),
	)
}

func (tg *TgBot) isSubscribed(userID int64, channels ...ChannelName) (bool, error) {
	result := true
	for _, channel := range channels {
		cfg := tgbotapi.ChatConfigWithUser{
			SuperGroupUsername: string(channel),
			UserID:             userID,
		}
		member, err := tg.GetChatMember(tgbotapi.GetChatMemberConfig{ChatConfigWithUser: cfg})
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
