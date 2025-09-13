package menus

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// EditTopup обрабатывает меню пополнения
func EditTopup(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	log.Printf("EDIT_TOPUP: Начало обработки пополнения для ChatID=%d, MessageID=%d", chatID, messageID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 1₽", "topup:1"),
			tgbotapi.NewInlineKeyboardButtonData("💰 300₽", "topup:300"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 500₽", "topup:500"),
			tgbotapi.NewInlineKeyboardButtonData("💰 1000₽", "topup:1000"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 2000₽", "topup:2000"),
			tgbotapi.NewInlineKeyboardButtonData("💰 5000₽", "topup:5000"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
		),
	)

	text := "💳 Выберите сумму для пополнения баланса:\n\n" +
		"⚡️ Пополнение происходит мгновенно\n" +
		"💡 Рекомендуем пополнить сразу на нужную сумму"

	log.Printf("EDIT_TOPUP: Текст для пополнения ChatID=%d: %s", chatID, text)
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("EDIT_TOPUP: Ошибка редактирования сообщения для ChatID=%d, MessageID=%d: %v", chatID, messageID, err)
	}
}
