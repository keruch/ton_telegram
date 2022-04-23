package application

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/keruch/ton_telegram/internal/market/config"
	"github.com/keruch/ton_telegram/internal/market/domain"
	"github.com/keruch/ton_telegram/internal/market/pkg/table"
	log "github.com/keruch/ton_telegram/pkg/logger"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

type marketRepo interface {
	AddUser(ctx context.Context, ID int64, username string) error
	GetRating(ctx context.Context, limit int) ([]domain.RatingRow, error)
	GetIDs(ctx context.Context, userID int64) ([]int, error)
}

type Market struct {
	*tgbotapi.BotAPI
	repo   marketRepo
	cfg    *config.MarketConfig
	logger *log.Logger

	mu    *sync.RWMutex
	users map[string][]int
}

func NewMarket(token string, repo marketRepo, cfg *config.MarketConfig, logger *log.Logger) (*Market, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Market{
		BotAPI: bot,
		repo:   repo,
		cfg:    cfg,
		logger: logger,

		mu:    new(sync.RWMutex),
		users: make(map[string][]int, 0),
	}, nil
}

func (m *Market) Serve(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"inline_query", "callback_query", "message"}
	updates := m.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			m.logger.Println("Telegram bot: serve done")
			return
		case update := <-updates:
			m.processUpdate(ctx, update)
		}
	}
}

func (m *Market) processUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.Message != nil {
		if err := m.processMessage(ctx, update); err != nil {
			m.logger.WithError(err).Errorf("error while processing message")
		}
	} else if update.CallbackQuery != nil {
		if err := m.processCallback(ctx, update); err != nil {
			m.logger.WithError(err).Errorf("error while processing message")
		}
	}
}

const startCommand = "/start"

func (m *Market) processMessage(ctx context.Context, update tgbotapi.Update) error {
	var (
		chatID   = update.Message.Chat.ID
		userID   = update.Message.From.ID
		userName = update.Message.From.UserName
	)

	switch update.Message.Text {
	case startCommand:
		m.logger.WithField("username", userName).Infof("user typed /start")
		err := m.repo.AddUser(ctx, userID, userName)
		if err != nil {
			m.logger.WithField("username", userName).WithError(err).Error("failed to add user to db")
			return m.sendNewMessage(chatID, "")
		}
		return m.sendNewMessage(chatID, m.cfg.Messages.Start)

	case m.cfg.Buttons.Verse:
		m.logger.WithField("username", userName).Infof("user typed verse")

		ids, ok := m.users[userName]
		if !ok {
			var err error
			ids, err = m.repo.GetIDs(ctx, userID)
			if err != nil {
				m.logger.WithField("username", userName).WithError(err).Error("failed to get nft ids")
				return m.sendNewMessage(chatID, "")
			}
			m.users[userName] = ids
		}

		err := m.sendNewMessage(chatID, "Opening your \U0001FA90Verse...")
		if err != nil {
			return err
		}
		lock, err := m.sendLock(chatID)
		if err != nil {
			return err
		}
		time.Sleep(2 * time.Second)

		if len(ids) == 0 {
			return m.sendPlanetsMessage(chatID, lock.MessageID,
				"Your \U0001FA90Verse is empty now. For details see TON VERSE\U0001FA90.",
				createInlineKeyboardForTONVERSE())
		} else {
			return m.sendPlanetsMessage(chatID, lock.MessageID,
				"Welcome to your \U0001FA90Verse!", createInlineKeyboardForPlanets(ids))
		}

	case m.cfg.Buttons.Rating:
		m.logger.WithField("username", userName).Infof("user typed rating")
		rating, err := m.repo.GetRating(ctx, m.cfg.RatingLimit)
		if err != nil {
			m.logger.WithError(err).Error("failed get rating")
			return m.sendNewMessage(chatID, "")
		}
		return m.sendNewMessage(chatID, table.CreateRatingTable(rating))

	case m.cfg.Buttons.Info:
		m.logger.WithField("username", userName).Infof("user typed info")
		return m.sendNewMessage(chatID, m.cfg.Messages.Info)

	default:
		return m.sendNewMessage(chatID, m.cfg.Messages.MissingCommand)
	}
}

func (m *Market) processCallback(ctx context.Context, update tgbotapi.Update) error {
	var (
		chatID   = update.CallbackQuery.Message.Chat.ID
		userID   = update.CallbackQuery.From.ID
		userName = update.CallbackQuery.From.UserName
		picID    = update.CallbackQuery.Data
	)

	callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackData())
	if _, err := m.Request(callback); err != nil {
		m.logger.WithError(err).Errorf("Failed to request callback")
		return err
	}

	pwd, _ := os.Getwd()
	photoBytes, err := ioutil.ReadFile(fmt.Sprintf("%s/cmd/market/pics/%s.jpeg", pwd, picID))
	if err != nil {
		return err
	}
	photoFileBytes := tgbotapi.FileBytes{
		Name:  "picture",
		Bytes: photoBytes,
	}

	ids, ok := m.users[userName]
	if !ok {
		ids, err = m.repo.GetIDs(ctx, userID)
		if err != nil {
			m.logger.WithField("username", userName).WithError(err).Error("failed to get nft ids")
			return m.sendNewMessage(chatID, "")
		}
		m.users[userName] = ids
	}

	photo := tgbotapi.NewPhoto(chatID, photoFileBytes)
	photo.ReplyMarkup = createInlineKeyboardForPlanets(ids)
	photo.ParseMode = tgbotapi.ModeHTML
	photo.Caption = wrapMessage("Enjoy your planet with ID " + picID)
	_, err = m.Send(photo)

	return err
}
