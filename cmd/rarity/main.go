package main

import (
	"github.com/gorilla/mux"
	"github.com/keruch/ton_telegram/internal/rarity/application"
	"github.com/keruch/ton_telegram/internal/rarity/handlers"
	"github.com/keruch/ton_telegram/internal/rarity/repository"
	"github.com/keruch/ton_telegram/pkg/logger"
	"net/http"
	"time"
)

func main() {
	log := logger.NewLogger()
	log.Info("Setup repository")
	storage, err := repository.NewRarityTable("gen.csv")
	if err != nil {
		log.Errorf("Failed to setup repository: %s", err)
		return
	}

	log.Info("Setup service")
	service := application.NewRarityService(log, storage)

	log.Info("Setup handler")
	handler := handlers.NewRarityHandler(service)
	r := mux.NewRouter()
	handler.Attach(r)

	srv := &http.Server{
		Handler:      r,
		Addr:         ":8200",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("ListenAndServe error: %s", err)
		return
	}
}
