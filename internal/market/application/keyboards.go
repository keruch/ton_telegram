package application

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv"
)

func (m *Market) createDefaultKeyboardMarkup() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(m.cfg.Buttons.Verse),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(m.cfg.Buttons.Rating),
			tgbotapi.NewKeyboardButton(m.cfg.Buttons.Info),
		),
	)
}

func createInlineKeyboardForPlanets(ids []int) tgbotapi.InlineKeyboardMarkup {
	for len(ids)%4 != 0 {
		ids = append(ids, 0)
	}
	countRows := len(ids) / 4
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, countRows)
	for i := 0; i < countRows; i++ {
		buttons := make([]tgbotapi.InlineKeyboardButton, 0)
		for j := 0; j < 4; j++ {
			var button tgbotapi.InlineKeyboardButton
			id := strconv.Itoa(ids[4*i+j])
			if id == "0" {
				continue
			}
			button = tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(ids[4*i+j]), strconv.Itoa(ids[4*i+j]))
			buttons = append(buttons, button)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(buttons...))
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func createInlineKeyboardMarkupWithID() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1", "2"),
			tgbotapi.NewInlineKeyboardButtonData("1", "2"),
			tgbotapi.NewInlineKeyboardButtonData("1", "2"),
		),
	)
}
