package table

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/keruch/ton_masks_bot/internal/market/domain"
)

func CreateRatingTable(rating []domain.RatingRow) string {
	t := table.NewWriter()
	t.AppendHeader(table.Row{"#", "Username", "NFT count"})
	t.AppendSeparator()
	for id, row := range rating {
		t.AppendRow([]interface{}{id, row.Username, row.Nft})
	}
	return t.Render()
}
