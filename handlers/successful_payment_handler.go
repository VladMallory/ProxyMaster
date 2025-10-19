package handlers

import (
	"fmt"
	"log"

	"bot/common"
	"bot/payments"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleSuccessfulPayment обрабатывает успешные платежи через Telegram Bot API
func HandleSuccessfulPayment(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	payment := message.SuccessfulPayment
	userID := message.From.ID
	chatID := message.Chat.ID

	log.Printf("SUCCESSFUL_PAYMENT: Получен успешный платеж от пользователя %d", userID)
	log.Printf("SUCCESSFUL_PAYMENT: Payload: %s, TotalAmount: %d, Currency: %s",
		payment.InvoicePayload, payment.TotalAmount, payment.Currency)

	// Проверяем, что новая платежная система инициализирована
	if payments.GlobalPaymentManager == nil {
		log.Printf("SUCCESSFUL_PAYMENT: Платежная система не инициализирована, используем старый метод")

		// Fallback на старый метод
		paymentAPI := common.NewTelegramPaymentAPI(bot)
		err := paymentAPI.ProcessSuccessfulPayment(payment, userID)
		if err != nil {
			log.Printf("SUCCESSFUL_PAYMENT: Ошибка обработки платежа (старый метод): %v", err)
			sendPaymentErrorMessage(chatID, bot)
			return
		}

		// Отправляем простое подтверждение
		sendSimplePaymentConfirmation(chatID, int64(payment.TotalAmount/100), userID, bot)
		return
	}

	// Используем новую платежную систему
	paymentInfo, err := payments.GlobalPaymentManager.ProcessTelegramSuccessfulPayment(payment, userID)
	if err != nil {
		log.Printf("SUCCESSFUL_PAYMENT: Ошибка обработки платежа для пользователя %d: %v", userID, err)
		sendPaymentErrorMessage(chatID, bot)
		return
	}

	// Получаем обновленные данные пользователя
	user, err := common.GetUserByTelegramID(userID)
	if err != nil {
		log.Printf("SUCCESSFUL_PAYMENT: Ошибка получения данных пользователя: %v", err)
		sendPaymentErrorMessage(chatID, bot)
		return
	}

	// Отправляем подтверждение через новую систему
	err = payments.GlobalPaymentManager.SendTelegramPaymentConfirmation(chatID, paymentInfo, user.Balance)
	if err != nil {
		log.Printf("SUCCESSFUL_PAYMENT: Ошибка отправки подтверждения: %v", err)
		// Отправляем простое подтверждение как fallback
		sendSimplePaymentConfirmation(chatID, int64(paymentInfo.Amount), userID, bot)
		return
	}

	log.Printf("SUCCESSFUL_PAYMENT: Платеж успешно обработан для пользователя %d на сумму %.2f через новую систему",
		userID, paymentInfo.Amount)

	// Логируем успешный платеж пользователя
	username := ""
	firstName := ""
	lastName := ""
	if user != nil {
		if user.Username != "" {
			username = user.Username
		}
		firstName = user.FirstName
		lastName = user.LastName
	}
	common.LogUserPayment(userID, username, firstName, lastName, "Telegram Bot API", paymentInfo.Amount, fmt.Sprintf("Payload=%s, Currency=%s", payment.InvoicePayload, payment.Currency))
}

// sendPaymentErrorMessage отправляет сообщение об ошибке платежа
func sendPaymentErrorMessage(chatID int64, bot *tgbotapi.BotAPI) {
	text := "❌ Произошла ошибка при обработке платежа. Обратитесь в поддержку."
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = &keyboard
	if _, err := bot.Send(msg); err != nil {
		log.Printf("SUCCESSFUL_PAYMENT: Ошибка отправки сообщения об ошибке: %v", err)
	}
}

// sendSimplePaymentConfirmation отправляет простое подтверждение платежа
func sendSimplePaymentConfirmation(chatID int64, amount int64, userID int64, bot *tgbotapi.BotAPI) {
	user, err := common.GetUserByTelegramID(userID)
	if err != nil {
		log.Printf("SUCCESSFUL_PAYMENT: Ошибка получения пользователя для простого подтверждения: %v", err)
		return
	}

	text := fmt.Sprintf("✅ <b>Платеж успешно выполнен!</b>\n\n"+
		"💰 Пополнено: %d₽\n"+
		"💳 Новый баланс: %.2f₽\n"+
		"🏦 Платежная система: Telegram Bot API\n\n"+
		"Спасибо за пополнение!",
		amount, user.Balance)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	_, err = bot.Send(msg)
	if err != nil {
		log.Printf("SUCCESSFUL_PAYMENT: Ошибка отправки простого подтверждения: %v", err)
	}
}
