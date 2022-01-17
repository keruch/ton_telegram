package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/keruch/the_open_art_ton_bot/internal/domain"
	repo "github.com/keruch/the_open_art_ton_bot/internal/repository"
	log "github.com/keruch/the_open_art_ton_bot/pkg/logger"
)

type repository interface {
	AddUser(ctx context.Context, ID int64, username string, invitedID int64) error
	GetFieldForID(ctx context.Context, ID int64, field string) (interface{}, error)
	UpdatePoints(ctx context.Context, ID int64, points int) error
	UpdateSubscription(ctx context.Context, subscription string, ID int64, value bool) error
	GetInvitedByID(ctx context.Context, ID int64) (int64, error)
	GetUsername(ctx context.Context, ID int64) (string, error)
}

type TgBot struct {
	*tgbotapi.BotAPI
	repo   repository
	logger *log.Logger
}

type (
	ChannelName        string
	ChannelDBFiled     string
	SubscriptionAction bool
)

const (
	startCommand  = "start"
	pointsCommand = "points"

	StartMessage             = "[The Open Art](https://t.me/theopenart) совместно с [Investment kingyru](https://t.me/investkingyru) проводит розыгрыш уникальной [NFT](https://ton.org.in/EQCMtTKLYj2588dWgBYvqx4H439pDGYf9jaJLPM-jP3rVRV6) выпущенной эксклюзивно для интервью.\nПринять участие очень просто - выполняйте задания и получайте баллы, которые увеличивают шансы выиграть [NFT](https://ton.org.in/EQCMtTKLYj2588dWgBYvqx4H439pDGYf9jaJLPM-jP3rVRV6).\n\n💎 Больше информации о в официальном сообществе [The Open Art](https://t.me/theopenart), в официальном сообществе [Investment kingyru](https://t.me/investkingyru) и на официальном маркет-сайте ton.org.in .\n\n" + SubscribeToJoinMessage
	SubscribeToJoinMessage   = "Для участия подпишитесь на каналы [The Open Art](https://t.me/theopenart) и [Investment kingyru](https://t.me/investkingyru). Важно быть подписанным до окончания конкурса"
	SubscribedToAllMessage   = "✨ Поздравляем! Вы участвуете в розыгрыше эксклюзивного [NFT](https://ton.org.in/EQCMtTKLYj2588dWgBYvqx4H439pDGYf9jaJLPM-jP3rVRV6).\n[NFT](https://ton.org.in/EQCMtTKLYj2588dWgBYvqx4H439pDGYf9jaJLPM-jP3rVRV6) будет распределены 22 января 2022 года в 16:00. Шанс выиграть [NFT](https://ton.org.in/EQCMtTKLYj2588dWgBYvqx4H439pDGYf9jaJLPM-jP3rVRV6) напрямую зависит от баллов: чем их больше, тем выше вероятность получить [NFT](https://ton.org.in/EQCMtTKLYj2588dWgBYvqx4H439pDGYf9jaJLPM-jP3rVRV6). Вы можете приглашать друзей: за каждоую его подписку вы получите по 50 баллов (максимум 100), но следите, чтобы они не отписывались, а то баллы за них учтены не будут!"
	AlreadyRegisteredMessage = "Вы уже зарегистрированы на участие в конкурсе!"
	SubscribedMessage        = "Вы подписались на канал @%s."
	UnsubscribedMessage      = "Вы отписались от канала @%s и больше не участвуете в конкурсе. Подпишитесь, чтобы опять принять участие."
	MissingCommandMessage    = "Куда-то ты не туда полез дружок..."

	FriendSubscribedFormatString      = "Ваш друг @%s подписался на канал @%s."
	FriendUnsubscribedFormatString    = "Ваш друг @%s отписался от канала @%s и больше не участвует в конкурсе. Пришлось забрать ваши 100 баллов :("
	FriendSubscribedToAllFormatString = "Ваш друг @%s подписался на все каналы из условий и теперь участвует в конкурсе. А вы полуили 100 баллов!"

	TheOpenArtChannelTag ChannelName = "@theopenart"
	TheOpenArtChannel    ChannelName = "theopenart"

	TheOpenArtDBField ChannelDBFiled = "openart"
	AdditionalDBField ChannelDBFiled = "additional"

	subscribeAction   SubscriptionAction = true
	unsubscribeAction SubscriptionAction = false
)

var (
	ChannelToDBMapping = map[ChannelName]ChannelDBFiled{
		TheOpenArtChannel:    TheOpenArtDBField,
		"nvbet":              AdditionalDBField,
		TheOpenArtChannelTag: TheOpenArtDBField,
		"@nvbet":             AdditionalDBField,
	}
	ToSubscribe = []ChannelName{TheOpenArtChannelTag, "@nvbet"}
)

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
	msg.ParseMode = tgbotapi.ModeMarkdown

	switch update.Message.Command() {
	case startCommand:
		tg.logger.Tracef("User @%s typed /start", userName)
		msg.Text = StartMessage
		ID, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if ID == userID {
			ID = 0
		}
		if err == nil && ID != 0 {
			invitedUser, err := tg.repo.GetUsername(ctx, ID)
			if err != nil {
				tg.logger.WithField("Method", "GetFieldForID").Error(err)
			}
			msg.Text = fmt.Sprintf("Вы были приглашены другом @%v!\n\n%s", invitedUser, StartMessage)
		}
		if err = tg.repo.AddUser(ctx, userID, userName, ID); err != nil {
			if errors.Is(err, repo.ErrorAlreadyRegistered) {
				tg.logger.WithField("Method", "AddUser").Info(err)
				msg.Text = AlreadyRegisteredMessage
				msg.ReplyMarkup = createInlineKeyboardMarkupWithID(userID)
				if _, err := tg.Send(msg); err != nil {
					tg.logger.WithField("Method", "Send").Error(err)
					return
				}
				return
			}
			tg.logger.WithField("Method", "AddUser").Error(err)
			return
		}

		if _, err := tg.Send(msg); err != nil {
			tg.logger.WithField("Method", "Send").Error(err)
			return
		}

		subToAll := true
		for _, sub := range ToSubscribe {
			ok, err := tg.isSubscribed(userID, sub)
			if err != nil {
				tg.logger.WithField("Method", "isSubscribed").Error(err)
				return
			}

			if ok {
				if err := tg.updateSubscription(ctx, userID, userName, sub, subscribeAction, 50); err != nil {
					tg.logger.WithField("Method", "updateSubscription").Error(err)
					return
				}
			}

			subToAll = subToAll && ok
		}

		if subToAll {
			msg.ReplyMarkup = createInlineKeyboardMarkupWithID(userID)
			msg.Text = SubscribedToAllMessage
			if _, err := tg.Send(msg); err != nil {
				tg.logger.WithField("Method", "Send").Error(err)
				return
			}
		}
	default:
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
	msg.ParseMode = tgbotapi.ModeMarkdown

	switch update.CallbackQuery.Data {
	case pointsCommand:
		tg.logger.WithField("User", userName).Trace("Got points command callback")
		points, err := tg.repo.GetFieldForID(ctx, userID, domain.PointsField)
		if err != nil {
			tg.logger.WithField("Method", "GetFieldForID").Error(err)
		}
		msg.ReplyMarkup = createInlineKeyboardMarkupWithID(userID)
		msg.Text = fmt.Sprintf("У вас %v баллов", points.(int32))
	}

	if _, err := tg.Send(msg); err != nil {
		tg.logger.WithField("Method", "Send").Error(err)
	}
}

func (tg *TgBot) processChatMember(ctx context.Context, update tgbotapi.Update) {
	var (
		userID      = update.ChatMember.From.ID
		userName    = update.ChatMember.From.UserName
		channelName = ChannelName(update.ChatMember.Chat.UserName)
	)

	if update.ChatMember.NewChatMember.Status == "left" {
		if err := tg.updateSubscription(ctx, userID, userName, channelName, unsubscribeAction, -50); err != nil {
			tg.logger.WithField("Method", "updateSubscription").WithField("Action", "Subscribe").Error(err)
			return
		}
	} else if update.ChatMember.NewChatMember.Status == "member" {
		if err := tg.updateSubscription(ctx, userID, userName, channelName, subscribeAction, 50); err != nil {
			tg.logger.WithField("Method", "updateSubscription").WithField("Action", "Unsubscribe").Error(err)
			return
		}
		ok, err := tg.isSubscribed(userID, ToSubscribe...)
		if err != nil {
			tg.logger.WithField("Method", "isSubscribed").Error(err)
			return
		}
		if ok {
			msg := tgbotapi.NewMessage(userID, "Something went wrong!")
			msg.ParseMode = tgbotapi.ModeMarkdown
			msg.Text = SubscribedToAllMessage

			if _, err = tg.Send(msg); err != nil {
				tg.logger.WithField("Method", "isSubscribed").Error(err)
				return
			}

			invitedByID, err := tg.repo.GetInvitedByID(ctx, userID)
			if err != nil {
				tg.logger.WithField("Method", "isSubscribed").Error(err)
				return
			}

			if invitedByID != 0 {
				msg = tgbotapi.NewMessage(invitedByID, "Something went wrong!")
				msg.ParseMode = tgbotapi.ModeMarkdown
				msg.Text = fmt.Sprintf(FriendSubscribedToAllFormatString, userName)

				if _, err = tg.Send(msg); err != nil {
					tg.logger.WithField("Method", "isSubscribed").Error(err)
					return
				}
			}
		}
	}
}

func (tg *TgBot) updateSubscription(ctx context.Context, userID int64, username string, channelName ChannelName, action SubscriptionAction, points int) error {
	channel := ChannelToDBMapping[channelName]
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
	}

	return nil
}

func createInlineKeyboardMarkupWithID(ID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonSwitch("Пригласить друга", fmt.Sprintf("Ваша персональная ссылка для приглашения: \n\nhttps://t.me/testtheopenartbot?start=%d", ID)),
			tgbotapi.NewInlineKeyboardButtonData("Получить баллы", pointsCommand),
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
