package common

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ИНТЕГРАЦИЯ С ЮКАССОЙ ЧЕРЕЗ TELEGRAM BOT API
//
// Этот модуль реализует интеграцию с ЮКассой через Telegram Bot API.
// Это НЕ прямая интеграция с API ЮКассы, а использование платежной системы Telegram.
//
// Как это работает:
// 1. Получаем YUKASSA_PROVIDER_TOKEN от @BotFather в разделе Payments
// 2. Создаем инвойс через tgbotapi.InvoiceConfig
// 3. Telegram показывает пользователю форму оплаты ЮКассы
// 4. При успешной оплате Telegram отправляет SuccessfulPayment
// 5. Обрабатываем платеж и зачисляем средства на баланс
//
// Режим работы (тест/прод) определяется типом токена от BotFather:
// - TEST токены: для тестирования без реальных платежей
// - LIVE токены: для реальных платежей

// TelegramPaymentAPI структура для работы с платежами через Telegram Bot API
type TelegramPaymentAPI struct {
	bot *tgbotapi.BotAPI
}

// NewTelegramPaymentAPI создает новый экземпляр API платежей Telegram
func NewTelegramPaymentAPI(bot *tgbotapi.BotAPI) *TelegramPaymentAPI {
	return &TelegramPaymentAPI{
		bot: bot,
	}
}

// CreateInvoice создает инвойс для оплаты через Telegram
func (t *TelegramPaymentAPI) CreateInvoice(chatID int64, userID int64, amount int, description string) error {
	log.Printf("TELEGRAM_PAYMENTS: Создание инвойса для пользователя %d на сумму %d", userID, amount)

	// Подготавливаем цену в копейках (amount в рублях * 100)
	prices := []tgbotapi.LabeledPrice{
		{
			Label:  description,
			Amount: amount * 100, // Telegram API требует сумму в копейках
		},
	}

	// Создаем инвойс
	invoice := tgbotapi.InvoiceConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID: chatID,
		},
		Title:                     "Пополнение баланса",
		Description:               description,
		Payload:                   fmt.Sprintf("topup_%d_%d", userID, amount), // Payload для идентификации платежа
		ProviderToken:             YUKASSA_PROVIDER_TOKEN,
		Currency:                  "RUB",
		Prices:                    prices,
		StartParameter:            fmt.Sprintf("topup_%d", amount),
		PhotoURL:                  "", // Можно добавить логотип
		PhotoSize:                 0,
		PhotoWidth:                0,
		PhotoHeight:               0,
		NeedName:                  false,
		NeedPhoneNumber:           false,
		NeedEmail:                 false,
		NeedShippingAddress:       false,
		SendPhoneNumberToProvider: false,
		SendEmailToProvider:       false,
		IsFlexible:                false,
		SuggestedTipAmounts:       []int{}, // Пустой массив для чаевых
	}

	// Отправляем инвойс
	msg, err := t.bot.Send(invoice)
	if err != nil {
		log.Printf("TELEGRAM_PAYMENTS: Ошибка создания инвойса для пользователя %d: %v", userID, err)
		return fmt.Errorf("ошибка создания инвойса: %v", err)
	}

	log.Printf("TELEGRAM_PAYMENTS: Инвойс успешно создан для пользователя %d, MessageID=%d", userID, msg.MessageID)
	return nil
}

// ProcessSuccessfulPayment обрабатывает успешный платеж
func (t *TelegramPaymentAPI) ProcessSuccessfulPayment(payment *tgbotapi.SuccessfulPayment, userID int64) error {
	log.Printf("TELEGRAM_PAYMENTS: Обработка успешного платежа для пользователя %d", userID)
	log.Printf("TELEGRAM_PAYMENTS: Payload: %s, TotalAmount: %d, Currency: %s",
		payment.InvoicePayload, payment.TotalAmount, payment.Currency)

	// Извлекаем сумму из payload (формат: topup_userID_amount)
	var extractedUserID, amount int64
	n, err := fmt.Sscanf(payment.InvoicePayload, "topup_%d_%d", &extractedUserID, &amount)
	if err != nil || n != 2 {
		log.Printf("TELEGRAM_PAYMENTS: Ошибка парсинга payload: %s, error: %v", payment.InvoicePayload, err)
		return fmt.Errorf("ошибка парсинга payload: %v", err)
	}

	// Проверяем, что userID соответствует
	if extractedUserID != userID {
		log.Printf("TELEGRAM_PAYMENTS: Несоответствие userID: ожидался %d, получен %d", extractedUserID, userID)
		return fmt.Errorf("несоответствие userID в платеже")
	}

	// Проверяем сумму (в копейках)
	expectedAmount := amount * 100
	if payment.TotalAmount != int(expectedAmount) {
		log.Printf("TELEGRAM_PAYMENTS: Несоответствие суммы: ожидалось %d копеек, получено %d", expectedAmount, payment.TotalAmount)
		return fmt.Errorf("несоответствие суммы платежа")
	}

	log.Printf("TELEGRAM_PAYMENTS: Пополняем баланс пользователя %d на сумму %.2f", userID, float64(amount))

	// Пополняем баланс пользователя
	err = AddBalance(userID, float64(amount))
	if err != nil {
		log.Printf("TELEGRAM_PAYMENTS: Ошибка пополнения баланса для пользователя %d: %v", userID, err)
		return fmt.Errorf("ошибка пополнения баланса: %v", err)
	}

	// Отправляем уведомление администратору о пополнении баланса
	if ADMIN_NOTIFICATIONS_ENABLED && ADMIN_BALANCE_TOPUP_ENABLED {
		user, err := GetUserByTelegramID(userID)
		if err != nil {
			log.Printf("TELEGRAM_PAYMENTS: Ошибка получения данных пользователя для уведомления: %v", err)
		} else {
			notificationText := fmt.Sprintf(
				"💰 <b>Пополнение баланса</b>\n\n"+
					"👤 Пользователь: %s %s\n"+
					"🆔 Telegram ID: %d\n"+
					"💵 Сумма: %.2f₽\n"+
					"💳 Новый баланс: %.2f₽\n"+
					"🏦 Платежная система: ЮКасса (Telegram)\n"+
					"📅 ID платежа: %s",
				user.FirstName, user.LastName, userID, float64(amount), user.Balance,
				payment.TelegramPaymentChargeID)

			msg := tgbotapi.NewMessage(ADMIN_ID, notificationText)
			msg.ParseMode = "HTML"
			if _, err := GlobalBot.Send(msg); err != nil {
				log.Printf("TELEGRAM_PAYMENTS: Ошибка отправки уведомления администратору: %v", err)
			}
		}
	}

	log.Printf("TELEGRAM_PAYMENTS: Платеж успешно обработан для пользователя %d", userID)
	return nil
}

// SendPaymentConfirmation отправляет подтверждение успешного платежа
func (t *TelegramPaymentAPI) SendPaymentConfirmation(chatID int64, amount float64, newBalance float64) error {
	text := fmt.Sprintf("✅ <b>Платеж успешно выполнен!</b>\n\n"+
		"💰 Пополнено: %.2f₽\n"+
		"💳 Новый баланс: %.2f₽\n"+
		"🏦 Платежная система: ЮКасса\n\n"+
		"Спасибо за пополнение! Теперь вы можете пользоваться нашими услугами.",
		amount, newBalance)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	_, err := t.bot.Send(msg)
	if err != nil {
		log.Printf("TELEGRAM_PAYMENTS: Ошибка отправки подтверждения платежа: %v", err)
		return err
	}

	return nil
}
