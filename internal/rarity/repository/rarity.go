package repository

import (
	"encoding/csv"
	"github.com/keruch/ton_telegram/internal/rarity/domain"
	"io"
	"os"
	"strconv"
)

type RarityStorage struct {
	rarityTable domain.RarityTable
}

func NewRarityTable(filename string) (*RarityStorage, error) {
	rarityTable, err := loadCsv(filename)
	return &RarityStorage{rarityTable: rarityTable}, err
}

func (r *RarityStorage) GetRarity(id int) (int, bool) {
	rarity, ok := r.rarityTable[id]
	return rarity, ok
}

func loadCsv(filename string) (domain.RarityTable, error) {
	file, err := os.Open(filename)
	if err != nil {
		return domain.RarityTable{}, err
	}

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 3
	reader.Comment = '#'

	rarityTable := make(domain.RarityTable)
	_, err = reader.Read() // read header
	if err != nil {
		return domain.RarityTable{}, err
	}

	for {
		record, e := reader.Read()
		if e != nil {
			if e == io.EOF {
				break
			}
			return domain.RarityTable{}, e
		}
		id, e := strconv.Atoi(record[0])
		if e != nil {
			return domain.RarityTable{}, e
		}
		rarity, e := strconv.Atoi(record[2])
		if e != nil {
			return domain.RarityTable{}, e
		}
		rarityTable[id] = rarity
	}
	return rarityTable, nil
}
