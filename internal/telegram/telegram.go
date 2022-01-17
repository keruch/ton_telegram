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
	AddUser(ctx context.Context, ID int64, username string, invitedID int64, chatID int64) error
	GetFieldForID(ctx context.Context, ID int64, field string) (interface{}, error)
	UpdatePoints(ctx context.Context, ID int64, points int) error
	UpdateSubscription(ctx context.Context, subscription string, ID int64, value bool) error
}

type TgBot struct {
	*tgbotapi.BotAPI
	repo   repository
	logger *log.Logger
}

type (
	ChannelName        string
	SubscriptionAction bool
)

const (
	startCommand  = "start"
	pointsCommand = "points"

	StartMessage             = "[The Open Art](https://t.me/theopenart) проводит розыгрыш 100 монет [TON](https://t.me/theopenart).\nПринять участие очень просто - выполняйте задания и получайте баллы, которые увеличивают шансы получить [TON](https://t.me/theopenart).\n\n💎 Больше информации о в официальном сообществе [The Open Art](https://t.me/theopenart).\n\n" + SubscribeToJoinMessage
	SubscribeToJoinMessage   = "Для участия подпишитесь на канал [The Open Art](https://t.me/theopenart). Важно быть подписанным до окончания конкурса!"
	SubscribedMessage        = "✨ Поздравляем! Вы участвуете в розыгрыше.\n\n100 TON будут распределены 01 февраля 2022 года в 16:00. Шанс выиграть TON напрямую зависит от баллов: чем их больше, тем выше вероятность получить TON. Вы можете приглашать друзей: за каждого получите по 100 баллов, но следите, чтобы они не отписывались, а то баллы за них учтены не будут!"
	AlreadyRegisteredMessage = "Вы уже зарегистрированы на участие в конкурсе!"
	UnsubscribedMessage      = "Вы отписались от канала [The Open Art](https://t.me/theopenart) и больше не участвуете в конкурсе. Подпишитесь, чтобы опять принять участие."
	MissingCommandMessage    = "Куда-то ты не туда полез, дружок..."

	FriendSubscribedFormatString   = "Ваш друг @%s подписался на канал [The Open Art](https://t.me/theopenart) и теперь участвует в конкурсе. А вы получили 100 баллов!"
	FriendUnsubscribedFormatString = "Ваш друг @%s отписался от канала [The Open Art](https://t.me/theopenart) и больше не участвует в конкурсе. Пришлось забрать ваши 100 баллов :("

	TheOpenArtChannel ChannelName = "@theopenart"

	theOpenArtFDBField ChannelName = "openart"

	subscribeAction   SubscriptionAction = true
	unsubscribeAction SubscriptionAction = false
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
			invitedUser, err := tg.repo.GetFieldForID(ctx, ID, domain.UsernameField)
			if err != nil {
				tg.logger.WithField("Method", "GetFieldForID").Error(err)
			}
			msg.Text = fmt.Sprintf("Вы были приглашены другом @%v!\n\n%s", invitedUser, StartMessage)
		}
		if err = tg.repo.AddUser(ctx, userID, userName, ID, chatID); err != nil {
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

		ok, err := tg.isSubscribed(userID, TheOpenArtChannel)
		if err != nil {
			tg.logger.WithField("Method", "isSubscribed").Error(err)
			return
		}

		if ok {
			if err := tg.updateSubscription(ctx, userID, userName, theOpenArtFDBField, subscribeAction, 100); err != nil {
				tg.logger.WithField("Method", "updateSubscription").Error(err)
				return
			}
		} else {
			msg.Text = msg.Text + "\n\n" + SubscribeToJoinMessage
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
		userID   = update.ChatMember.From.ID
		userName = update.ChatMember.From.UserName
	)

	if update.ChatMember.NewChatMember.Status == "left" {
		if err := tg.updateSubscription(ctx, userID, userName, theOpenArtFDBField, unsubscribeAction, -100); err != nil {
			tg.logger.WithField("Method", "updateSubscription").WithField("Action", "Subscribe").Error(err)
			return
		}
	} else if update.ChatMember.NewChatMember.Status == "member" {
		if err := tg.updateSubscription(ctx, userID, userName, theOpenArtFDBField, subscribeAction, 100); err != nil {
			tg.logger.WithField("Method", "updateSubscription").WithField("Action", "Unsubscribe").Error(err)
			return
		}
	}
}

func (tg *TgBot) getInvitedByUser(ctx context.Context, userID int64) (int64, error) {
	invitedBy, err := tg.repo.GetFieldForID(ctx, userID, domain.InvitedByField)
	if err != nil {
		return 0, err
	}
	invitedByID, ok := invitedBy.(int64)
	if !ok {
		return 0, err
	}

	return invitedByID, nil
}

func (tg *TgBot) updateSubscription(ctx context.Context, userID int64, username string, channel ChannelName, action SubscriptionAction, points int) error {
	if err := tg.repo.UpdateSubscription(ctx, string(channel), userID, bool(action)); err != nil {
		return err
	}

	invitedByID, err := tg.getInvitedByUser(ctx, userID)
	if err != nil {
		return err
	}
	if invitedByID != 0 {
		err = tg.repo.UpdatePoints(ctx, invitedByID, points)
		if err != nil {
			return err
		}

		msg := tgbotapi.NewMessage(invitedByID, "Something went wrong!")
		msg.ParseMode = tgbotapi.ModeMarkdown

		if action == subscribeAction {
			msg.Text = fmt.Sprintf(FriendSubscribedFormatString, username)
		} else {
			msg.Text = fmt.Sprintf(FriendUnsubscribedFormatString, username)
		}

		if _, err = tg.Send(msg); err != nil {
			return err
		}
	}

	msg := tgbotapi.NewMessage(userID, "Something went wrong!")
	msg.ParseMode = tgbotapi.ModeMarkdown

	if action == subscribeAction {
		msg.Text = fmt.Sprintf(SubscribedMessage)
		msg.ReplyMarkup = createInlineKeyboardMarkupWithID(userID)
	} else {
		msg.Text = fmt.Sprintf(UnsubscribedMessage)
	}

	if _, err = tg.Send(msg); err != nil {
		return err
	}

	return nil
}

func createInlineKeyboardMarkupWithID(ID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonSwitch("Пригласить друга", fmt.Sprintf("Ваша персональная ссылка для приглашения: \n\nhttps://t.me/theopenartbot?start=%d", ID)),
			//tgbotapi.NewInlineKeyboardButtonURL("Subscribe", "https://t.me/theopenart"),
			tgbotapi.NewInlineKeyboardButtonData("Получить баллы", pointsCommand),
		),
	)
}

func (tg *TgBot) isSubscribed(userID int64, channel ChannelName) (bool, error) {
	config := tgbotapi.ChatConfigWithUser{
		SuperGroupUsername: string(channel),
		UserID:             userID,
	}
	member, err := tg.GetChatMember(tgbotapi.GetChatMemberConfig{ChatConfigWithUser: config})
	if err != nil {
		return false, err
	}
	return member.Status == "member" || member.Status == "creator" || member.Status == "administrator", nil
}
