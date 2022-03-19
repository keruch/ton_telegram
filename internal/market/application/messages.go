package application

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func (m *Market) sendNewMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, "Internal Server Error")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true
	msg.ReplyMarkup = m.createDefaultKeyboardMarkup()

	if text == "" {
		msg.Text = m.cfg.Messages.Error
	} else {
		msg.Text = text
	}
	msg.Text = wrapMessage(msg.Text)

	_, err := m.Send(msg)
	return err
}

func (m *Market) sendLock(chatID int64) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, "Internal Server Error")
	msg.Text = "🔐"
	return m.Send(msg)
}

func (m *Market) sendPlanetsMessage(chatID int64, messageID int, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewEditMessageText(chatID, messageID, "Internal Server Error")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true
	msg.ReplyMarkup = &keyboard

	if text == "" {
		msg.Text = m.cfg.Messages.Error
	} else {
		msg.Text = text
	}
	msg.Text = wrapMessage(msg.Text)

	_, err := m.Send(msg)
	return err
}

func (m *Market) deleteMessage(chatID int64, messageID int) error {
	msg := tgbotapi.NewDeleteMessage(chatID, messageID)
	_, err := m.Send(msg)
	return err
}

func wrapMessage(msg string) string {
	return "<pre>" + msg + "</pre>"
}
