package menus

import (
	"fmt"
	"log"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SendBalance отправляет информацию о балансе
func SendBalance(bot *tgbotapi.BotAPI, chatID int64, user *common.User) {
	log.Printf("SEND_BALANCE: Отправка информации о балансе для TelegramID=%d", user.TelegramID)

	// Рассчитываем потраченные деньги как разность между пополнениями и текущим балансом
	spent := user.TotalPaid - user.Balance
	if spent < 0 {
		spent = 0 // На случай, если баланс больше пополнений (не должно происходить)
	}

	text := fmt.Sprintf("💰 Ваш баланс: %.2f₽\n💸 Всего потрачено: %.2f₽",
		user.Balance, spent)

	log.Printf("SEND_BALANCE: Текст баланса для TelegramID=%d: %s", user.TelegramID, text)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = nil
	if _, err := bot.Send(msg); err != nil {
		log.Printf("SEND_BALANCE: Ошибка отправки сообщения для TelegramID=%d: %v", user.TelegramID, err)
	}
}

// EditBalance редактирует информацию о балансе
func EditBalance(bot *tgbotapi.BotAPI, chatID int64, messageID int, user *common.User) {
	log.Printf("EDIT_BALANCE: Редактирование информации о балансе для TelegramID=%d, MessageID=%d", user.TelegramID, messageID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳 Пополнить", "topup"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
		),
	)

	// Рассчитываем потраченные деньги как разность между пополнениями и текущим балансом
	spent := user.TotalPaid - user.Balance
	if spent < 0 {
		spent = 0 // На случай, если баланс больше пополнений (не должно происходить)
	}

	text := fmt.Sprintf("💰 Ваш баланс: %.2f₽\n💸 Всего потрачено: %.2f₽",
		user.Balance, spent)

	log.Printf("EDIT_BALANCE: Текст баланса для TelegramID=%d: %s", user.TelegramID, text)

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("EDIT_BALANCE: Ошибка редактирования сообщения для TelegramID=%d, MessageID=%d: %v", user.TelegramID, messageID, err)
	}
}
