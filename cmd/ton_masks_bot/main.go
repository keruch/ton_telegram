package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/keruch/the_open_art_ton_bot/config"
	"github.com/keruch/the_open_art_ton_bot/internal/repository"
	"github.com/keruch/the_open_art_ton_bot/internal/telegram"
	log "github.com/keruch/the_open_art_ton_bot/pkg/logger"
)

func main() {
	logger := log.NewLogger()
	err := config.SetupConfig()
	if err != nil {
		logger.Panic(err)
	}

	repo, err := repository.NewPostgresSQLPool(config.GetDatabaseURL(), logger)
	if err != nil {
		logger.Panicf("Setup repository failed: %s", err)
	}
	logger.Info("Setup repository")

	tg, err := telegram.NewTgBot(config.GetTelegramBotToken(), repo, logger)
	if err != nil {
		logger.Panic(err)
	}

	go tg.Serve(context.Background())

	r := mux.NewRouter()
	//r.Methods(http.MethodPost).PathPrefix("/msg").HandlerFunc(tg.NotifyUsersRequst)

	srv := &http.Server{
		Handler:      r,
		Addr:         ":8090",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Panicf("ListenAndServe: %s", err)
	}
}

//func NewTelegramBot(token string, logger *log.Logger) (*TelegramBot, error) {
//	bot, err := tgbot.NewBotAPI(token)
//	if err != nil {
//		return nil, err
//	}
//
//	return &TelegramBot{
//		bot:    bot,
//		users:  make(map[string]int64),
//		logger: logger,
//	}, nil
//}
//
//func (tg *TelegramBot) ServeWH(ctx context.Context) {
//	wg := tgbot.DeleteWebhookConfig{}
//	_, _ = tg.bot.Request(wg)
//}
//
//func (tg *TelegramBot) Serve(ctx context.Context) {
//	u := tgbot.NewUpdate(0)
//	u.Timeout = 60
//
//	updates := tg.bot.GetUpdatesChan(u)
//
//	for {
//		select {
//		case <-ctx.Done():
//			tg.logger.Println("Telegram bot: serve done")
//			return
//		case update := <-updates:
//			if update.Message != nil {
//				msg := tgbot.NewMessage(update.Message.Chat.ID, "New User!")
//				msg.ParseMode = tgbot.ModeMarkdown
//				msg.ReplyMarkup = ReKeyboard
//
//				switch update.Message.Text {
//				case "/start":
//					tg.addUser(update.Message.From.UserName, update.Message.Chat.ID)
//					msg.Text = "[The Open Art](https://t.me/theopenart) проводит розыгрыш 1000 монет [TON](https://t.me/theopenart).\nПринять участие очень просто - выполняйте задания и получайте баллы, которые увеличивают шансы получить [TON](https://t.me/theopenart).\n\n💎 Больше информации о в официальном сообществе [The Open Art](https://t.me/theopenart)."
//					msg.ReplyMarkup = InKeyboard
//					//tg.addUser(update.Message.From.UserName, update.Message.Chat.ID)
//					//msg.Text = "✨ Поздравляем! Теперь вы участвуете в розыгрыше.\n\n1000 TON будут распределены 10 февраля 2022 года в 16:00. Шанс выиграть TON напрямую зависит от баллов: чем их больше, тем выше вероятность получить TON."
//
//				case "About":
//					msg.Text = "[The Open Art](https://t.me/theopenart) проводит розыгрыш 1000 монет TON. Принять участие очень просто - выполняйте задания и получайте баллы, которые увеличивают шансы получить TON.\n\n💎 Больше информации о в официальном сообществе The Open Art. (https://t.me/theopenart)"
//				case "Invite":
//					msg.Text = "Ваша ссылка: [code](https://t.me/theopenartbot?start=r111)"
//				}
//
//				if _, err := tg.bot.Send(msg); err != nil {
//					panic(err)
//				}
//			}
//		}
//	}
//}
//
//func (tg *TelegramBot) NotifyUsers(message string) {
//	tg.mu.RLock()
//	for user, ID := range tg.users {
//		msg := tgbot.NewMessage(ID, message)
//
//		_, err := tg.bot.Send(msg)
//		if err != nil {
//			tg.logger.Println("Send msg to %s user failed: %s", user, err)
//		}
//	}
//	tg.mu.RUnlock()
//}
//
//func (tg *TelegramBot) addUser(username string, chatID int64) {
//	tg.mu.Lock()
//	tg.logger.Println("Added new user %s", username)
//	tg.users[username] = chatID
//	tg.mu.Unlock()
//}
//
//func (tg *TelegramBot) removeUser(username string) {
//	tg.mu.Lock()
//	tg.logger.Println("Removed user %s", username)
//	delete(tg.users, username)
//	tg.mu.Unlock()
//}
//
//func (tg *TelegramBot) NotifyUsersRequst(writer http.ResponseWriter, request *http.Request) {
//	body, err := io.ReadAll(request.Body)
//	if err != nil {
//		tg.logger.Printf("error: %s\n", err)
//	}
//
//	tg.NotifyUsers(string(body))
//}
