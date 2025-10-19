package telegramPayment

import (
	"bot/common"
	paymentCommon "bot/payments/common"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramPaymentProvider реализует платежи через Telegram Bot API
type TelegramPaymentProvider struct {
	bot *tgbotapi.BotAPI
}

// NewTelegramPaymentProvider создает новый провайдер Telegram платежей
func NewTelegramPaymentProvider(bot *tgbotapi.BotAPI) *TelegramPaymentProvider {
	return &TelegramPaymentProvider{
		bot: bot,
	}
}

// IsEnabled проверяет, включены ли платежи через Telegram
func (t *TelegramPaymentProvider) IsEnabled() bool {
	return common.TELEGRAM_PAYMENTS_ENABLED && common.YUKASSA_PROVIDER_TOKEN != ""
}

// GetMethod возвращает метод оплаты
func (t *TelegramPaymentProvider) GetMethod() paymentCommon.PaymentMethod {
	return paymentCommon.PaymentMethodTelegram
}

// CreatePayment создает платеж через Telegram Bot API
func (t *TelegramPaymentProvider) CreatePayment(userID int64, amount float64, description string) (*paymentCommon.PaymentInfo, error) {
	paymentCommon.LogPaymentEvent("INFO", paymentCommon.PaymentMethodTelegram,
		"Создание платежа для пользователя %d на сумму %.2f", userID, amount)

	// Валидация суммы
	if err := paymentCommon.ValidateAmount(amount); err != nil {
		return nil, err
	}

	// Генерируем ID платежа
	paymentID := paymentCommon.GeneratePaymentID()

	// Создаем информацию о платеже
	paymentInfo := &paymentCommon.PaymentInfo{
		ID:          paymentID,
		UserID:      userID,
		Amount:      amount,
		Currency:    "RUB",
		Status:      paymentCommon.PaymentStatusPending,
		Method:      paymentCommon.PaymentMethodTelegram,
		Description: paymentCommon.SanitizeDescription(description),
		CreatedAt:   paymentCommon.GetCurrentTimestamp(),
		UpdatedAt:   paymentCommon.GetCurrentTimestamp(),
		Metadata: paymentCommon.CreatePaymentMetadata(userID, map[string]interface{}{
			"payment_id": paymentID,
			"bot_token":  strings.Split(common.YUKASSA_PROVIDER_TOKEN, ":")[0], // Скрываем токен
		}),
	}

	paymentCommon.LogPaymentEvent("INFO", paymentCommon.PaymentMethodTelegram,
		"Платеж создан: ID=%s, UserID=%d, Amount=%.2f", paymentID, userID, amount)

	return paymentInfo, nil
}

// SendInvoice отправляет инвойс пользователю через Telegram
func (t *TelegramPaymentProvider) SendInvoice(chatID int64, paymentInfo *paymentCommon.PaymentInfo) error {
	paymentCommon.LogPaymentEvent("INFO", paymentCommon.PaymentMethodTelegram,
		"Отправка инвойса для платежа %s в чат %d", paymentInfo.ID, chatID)

	// Подготавливаем цену в копейках
	prices := []tgbotapi.LabeledPrice{
		{
			Label:  paymentInfo.Description,
			Amount: paymentCommon.ConvertRublesToKopecks(paymentInfo.Amount),
		},
	}

	// Создаем payload для идентификации платежа
	payload := fmt.Sprintf("topup_%d_%.0f_%s", paymentInfo.UserID, paymentInfo.Amount, paymentInfo.ID)

	// Создаем инвойс
	invoice := tgbotapi.InvoiceConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID: chatID,
		},
		Title:                     "Пополнение баланса",
		Description:               paymentInfo.Description,
		Payload:                   payload,
		ProviderToken:             common.YUKASSA_PROVIDER_TOKEN,
		Currency:                  paymentInfo.Currency,
		Prices:                    prices,
		StartParameter:            fmt.Sprintf("topup_%.0f", paymentInfo.Amount),
		PhotoURL:                  "", // Можно добавить логотип
		PhotoSize:                 0,
		PhotoWidth:                0,
		PhotoHeight:               0,
		NeedName:                  false, // Не запрашиваем имя
		NeedPhoneNumber:           false, // Не запрашиваем телефон
		NeedEmail:                 false, // Не запрашиваем email (Telegram сам передаст)
		NeedShippingAddress:       false,
		SendPhoneNumberToProvider: false, // Не отправляем телефон провайдеру
		SendEmailToProvider:       false, // Не отправляем email провайдеру
		IsFlexible:                false,
		SuggestedTipAmounts:       []int{},
	}

	// Отправляем инвойс
	msg, err := t.bot.Send(invoice)
	if err != nil {
		paymentCommon.LogPaymentEvent("ERROR", paymentCommon.PaymentMethodTelegram,
			"Ошибка отправки инвойса для платежа %s: %v", paymentInfo.ID, err)
		return fmt.Errorf("ошибка отправки инвойса: %v", err)
	}

	paymentCommon.LogPaymentEvent("INFO", paymentCommon.PaymentMethodTelegram,
		"Инвойс успешно отправлен для платежа %s, MessageID=%d", paymentInfo.ID, msg.MessageID)

	return nil
}

// GetPayment получает информацию о платеже (заглушка для Telegram API)
func (t *TelegramPaymentProvider) GetPayment(paymentID string) (*paymentCommon.PaymentInfo, error) {
	// Telegram Bot API не предоставляет метод для получения информации о платеже по ID
	// Эта функция используется только для совместимости с интерфейсом
	return nil, fmt.Errorf("получение платежа по ID не поддерживается в Telegram Bot API")
}

// ProcessWebhook обрабатывает уведомления от Telegram (SuccessfulPayment)
func (t *TelegramPaymentProvider) ProcessWebhook(data []byte) (*paymentCommon.PaymentInfo, error) {
	// Эта функция не используется для Telegram Bot API
	// Обработка успешных платежей происходит в handlers/successful_payment_handler.go
	return nil, fmt.Errorf("webhook обработка не используется для Telegram Bot API")
}

// ProcessSuccessfulPayment обрабатывает успешный платеж от Telegram
func (t *TelegramPaymentProvider) ProcessSuccessfulPayment(payment *tgbotapi.SuccessfulPayment, userID int64) (*paymentCommon.PaymentInfo, error) {
	paymentCommon.LogPaymentEvent("INFO", paymentCommon.PaymentMethodTelegram,
		"Обработка успешного платежа для пользователя %d", userID)
	paymentCommon.LogPaymentEvent("DEBUG", paymentCommon.PaymentMethodTelegram,
		"Payload: %s, TotalAmount: %d, Currency: %s", payment.InvoicePayload, payment.TotalAmount, payment.Currency)

	// Парсим payload (формат: topup_userID_amount_paymentID)
	parts := strings.Split(payment.InvoicePayload, "_")
	if len(parts) < 3 {
		return nil, fmt.Errorf("неверный формат payload: %s", payment.InvoicePayload)
	}

	// Извлекаем данные из payload
	extractedUserIDStr := parts[1]
	amountStr := parts[2]
	var paymentID string
	if len(parts) >= 4 {
		paymentID = parts[3]
	} else {
		paymentID = paymentCommon.GeneratePaymentID()
	}

	extractedUserID, err := strconv.ParseInt(extractedUserIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга userID из payload: %v", err)
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга суммы из payload: %v", err)
	}

	// Проверяем соответствие userID
	if extractedUserID != userID {
		return nil, fmt.Errorf("несоответствие userID: ожидался %d, получен %d", extractedUserID, userID)
	}

	// Проверяем сумму (в копейках)
	expectedAmountKopecks := paymentCommon.ConvertRublesToKopecks(amount)
	if payment.TotalAmount != expectedAmountKopecks {
		return nil, fmt.Errorf("несоответствие суммы: ожидалось %d копеек, получено %d", expectedAmountKopecks, payment.TotalAmount)
	}

	// Создаем информацию о платеже
	paymentInfo := &paymentCommon.PaymentInfo{
		ID:          paymentID,
		UserID:      userID,
		Amount:      amount,
		Currency:    payment.Currency,
		Status:      paymentCommon.PaymentStatusSucceeded,
		Method:      paymentCommon.PaymentMethodTelegram,
		Description: fmt.Sprintf("Пополнение баланса на %.2f₽", amount),
		CreatedAt:   paymentCommon.GetCurrentTimestamp(),
		UpdatedAt:   paymentCommon.GetCurrentTimestamp(),
		Metadata: paymentCommon.CreatePaymentMetadata(userID, map[string]interface{}{
			"telegram_payment_charge_id": payment.TelegramPaymentChargeID,
			"provider_payment_charge_id": payment.ProviderPaymentChargeID,
			"original_payload":           payment.InvoicePayload,
		}),
	}

	paymentCommon.LogPaymentEvent("INFO", paymentCommon.PaymentMethodTelegram,
		"Платеж успешно обработан: ID=%s, UserID=%d, Amount=%.2f", paymentID, userID, amount)

	// Пополняем баланс пользователя
	err = common.AddBalance(userID, amount)
	if err != nil {
		paymentCommon.LogPaymentEvent("ERROR", paymentCommon.PaymentMethodTelegram,
			"Ошибка пополнения баланса для пользователя %d: %v", userID, err)
		return nil, fmt.Errorf("ошибка пополнения баланса: %v", err)
	}

	// Логируем успешный платеж
	if err := paymentCommon.LogPaymentToFile(paymentID, userID, amount, "succeeded"); err != nil {
		paymentCommon.LogPaymentEvent("ERROR", paymentCommon.PaymentMethodTelegram,
			"Ошибка логирования платежа %s: %v", paymentID, err)
	}

	return paymentInfo, nil
}

// SendPaymentConfirmation отправляет подтверждение успешного платежа
func (t *TelegramPaymentProvider) SendPaymentConfirmation(chatID int64, paymentInfo *paymentCommon.PaymentInfo, newBalance float64) error {
	text := fmt.Sprintf("✅ <b>Платеж успешно выполнен!</b>\n\n"+
		"💰 Пополнено: %s\n"+
		"💳 Новый баланс: %.2f₽\n"+
		"🏦 Платежная система: %s\n"+
		"🆔 ID платежа: %s\n\n"+
		"Спасибо за пополнение! Теперь вы можете пользоваться нашими услугами.",
		paymentCommon.FormatAmount(paymentInfo.Amount), newBalance,
		paymentCommon.GetMethodDescription(paymentInfo.Method), paymentInfo.ID)

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
		paymentCommon.LogPaymentEvent("ERROR", paymentCommon.PaymentMethodTelegram,
			"Ошибка отправки подтверждения платежа: %v", err)
		return err
	}

	return nil
}
