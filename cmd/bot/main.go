package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/keruch/ton_masks_bot/cmd/bot/config"
	"github.com/keruch/ton_masks_bot/internal/repository"
	"github.com/keruch/ton_masks_bot/internal/telegram"
	log "github.com/keruch/ton_masks_bot/pkg/logger"
)

func main() {
	logger := log.NewLogger()
	err := config.SetupConfig()
	if err != nil {
		logger.Panic(err)
	}
	botCfg := config.GetBotConfig()

	repo, err := repository.NewPostgresSQLPool(config.GetDatabaseURL(), logger)
	if err != nil {
		logger.Panicf("Setup repository failed: %s", err)
	}
	logger.Info("Setup repository")

	tg, err := telegram.NewTgBot(config.GetTelegramBotToken(), repo, botCfg, logger)
	if err != nil {
		logger.Panic(err)
	}

	go tg.Serve(context.Background())

	r := mux.NewRouter()
	r.Methods(http.MethodGet).PathPrefix("/health").HandlerFunc(getHealth)

	srv := &http.Server{
		Handler:      r,
		Addr:         config.GetServerAddress(),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Panicf("ListenAndServe: %s", err)
	}
}

func getHealth(writer http.ResponseWriter, req *http.Request) {
	writer.WriteHeader(http.StatusOK)
}
