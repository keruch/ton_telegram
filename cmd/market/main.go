package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/keruch/ton_masks_bot/cmd/market/config"
	"github.com/keruch/ton_masks_bot/internal/market/application"
	"github.com/keruch/ton_masks_bot/internal/market/repository"
	log "github.com/keruch/ton_masks_bot/pkg/logger"
)

func main() {
	logger := log.NewLogger()
	logger.Info("Setup config")
	err := config.SetupConfig()
	if err != nil {
		logger.WithError(err).Errorf("Setup config failed")
	}
	botCfg := config.GetBotConfig()

	logger.Info("Setup repository")
	repo, err := repository.NewPostgresSQLPool(config.GetDatabaseURL(), logger)
	if err != nil {
		logger.WithError(err).Errorf("Setup repository failed")
		return
	}

	logger.Info("Setup telegram")
	tg, err := application.NewMarket(config.GetTelegramBotToken(), repo, &botCfg, logger)
	if err != nil {
		logger.WithError(err).Errorf("Setup telegram failed")
		return
	}

	tgContext, tgCancel := context.WithCancel(context.Background())
	go tg.Serve(tgContext)

	r := mux.NewRouter()
	r.Methods(http.MethodGet).PathPrefix("/health").HandlerFunc(getHealth)

	srv := &http.Server{
		Handler:      r,
		Addr:         config.GetServerAddress(),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go func() {
		logger.Infof("Setup server on port %s", config.GetServerAddress())
		if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Panicf("Setup server failed: ListenAndServe: %s", err)
		}
	}()

	stop := make(chan struct{})
	sigquit := make(chan os.Signal, 1)
	signal.Ignore(syscall.SIGHUP, syscall.SIGPIPE)
	signal.Notify(sigquit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Infof("Captured signal: %s", <-sigquit)

		tgCancel()

		if err := srv.Shutdown(context.Background()); err != nil {
			logger.WithError(err).Errorf("Can't shutdown server")
		} else {
			logger.Infof("Server: serve done")
		}

		stop <- struct{}{}
	}()

	logger.Infof("Setup done")

	<-stop
}

func getHealth(writer http.ResponseWriter, req *http.Request) {
	writer.WriteHeader(http.StatusOK)
}
