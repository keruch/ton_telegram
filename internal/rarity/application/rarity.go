package application

import (
	"errors"
	"github.com/keruch/ton_telegram/internal/rarity/repository"
	log "github.com/keruch/ton_telegram/pkg/logger"
)

var ErrIdNotFound = errors.New("unknown planet id")

type RarityService struct {
	log     *log.Logger
	storage *repository.RarityStorage
}

func NewRarityService(log *log.Logger, storage *repository.RarityStorage) *RarityService {
	return &RarityService{
		log:     log,
		storage: storage,
	}
}

func (s *RarityService) GetRarity(id int) (int, error) {
	rarity, ok := s.storage.GetRarity(id)
	if ok == false {
		return 0, ErrIdNotFound
	}
	return rarity, nil
}
