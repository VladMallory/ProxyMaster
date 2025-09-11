package handlers

import (
	"fmt"
	"log"
	"time"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ProcessPaymentCallback обрабатывает callback оплаты
func ProcessPaymentCallback(bot *tgbotapi.BotAPI, chatID int64, messageID int, user *common.User, days int) {
	log.Printf("PROCESS_PAYMENT_CALLBACK: Начало обработки платежа для TelegramID=%d, days=%d", user.TelegramID, days)

	// Обновляем данные пользователя из базы
	updatedUser, err := common.GetUserByTelegramID(user.TelegramID)
	if err != nil {
		log.Printf("PROCESS_PAYMENT_CALLBACK: Ошибка получения пользователя TelegramID=%d: %v", user.TelegramID, err)
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "❌ Ошибка получения данных пользователя")
		if _, err := bot.Send(editMsg); err != nil {
			log.Printf("PROCESS_PAYMENT_CALLBACK: Ошибка отправки сообщения об ошибке для TelegramID=%d: %v", user.TelegramID, err)
		}
		return
	}
	user = updatedUser
	log.Printf("PROCESS_PAYMENT_CALLBACK: Данные пользователя обновлены: TelegramID=%d, Balance=%.2f, HasActiveConfig=%v", user.TelegramID, user.Balance, user.HasActiveConfig)

	cost := float64(days * common.PRICE_PER_DAY)

	// Проверяем баланс
	if user.Balance < cost {
		log.Printf("PROCESS_PAYMENT_CALLBACK: Недостаточно средств для TelegramID=%d, Balance=%.2f, Cost=%.2f", user.TelegramID, user.Balance, cost)
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💳 Пополнить", "topup"),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
			),
		)

		text := fmt.Sprintf("❌ Недостаточно средств!\n\n"+
			"💰 Ваш баланс: %.2f₽\n"+
			"💸 Нужно: %.0f₽\n"+
			"💎 Не хватает: %.2f₽\n\n"+
			"Пополните баланс для продолжения",
			user.Balance, cost, cost-user.Balance)

		log.Printf("PROCESS_PAYMENT_CALLBACK: Текст ошибки недостатка средств для TelegramID=%d: %s", user.TelegramID, text)
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
		editMsg.ReplyMarkup = &keyboard
		if _, err := bot.Send(editMsg); err != nil {
			log.Printf("PROCESS_PAYMENT_CALLBACK: Ошибка отправки сообщения об ошибке для TelegramID=%d: %v", user.TelegramID, err)
		}
		return
	}

	// Показываем процесс
	log.Printf("PROCESS_PAYMENT_CALLBACK: Показ процесса оплаты для TelegramID=%d", user.TelegramID)
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "⏳ Обработка платежа...")
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("PROCESS_PAYMENT_CALLBACK: Ошибка отправки сообщения о процессе для TelegramID=%d: %v", user.TelegramID, err)
	}

	// Обрабатываем платеж
	log.Printf("PROCESS_PAYMENT_CALLBACK: Вызов ProcessPayment для TelegramID=%d, days=%d", user.TelegramID, days)
	configURL, err := common.ProcessPayment(user, days)
	if err != nil {
		log.Printf("PROCESS_PAYMENT_CALLBACK: Ошибка обработки платежа для TelegramID=%d: %v", user.TelegramID, err)
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Повторить", fmt.Sprintf("pay:%d", days)),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
			),
		)

		text := fmt.Sprintf("❌ Ошибка создания конфига: %v", err)
		log.Printf("PROCESS_PAYMENT_CALLBACK: Текст ошибки конфига для TelegramID=%d: %s", user.TelegramID, text)
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
		editMsg.ReplyMarkup = &keyboard
		if _, err := bot.Send(editMsg); err != nil {
			log.Printf("PROCESS_PAYMENT_CALLBACK: Ошибка отправки сообщения об ошибке конфига для TelegramID=%d: %v", user.TelegramID, err)
		}
		return
	}

	// Успешная оплата
	log.Printf("PROCESS_PAYMENT_CALLBACK: Платеж успешен для TelegramID=%d, ConfigURL=%s", user.TelegramID, configURL)

	// Используем HTML редирект страницу
	redirectURL := common.GetRedirectURL() + configURL

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(fmt.Sprintf("📱 Подключить (%s)", common.GetAppName()), redirectURL)),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔐 Конфиг", "vpn"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
		),
	)

	expiryDate := time.UnixMilli(user.ExpiryTime).Format("2006-01-02")

	var actionText string
	if common.IsConfigActive(user) && user.ConfigsCount > 1 {
		actionText = "продлен"
	} else {
		actionText = "создан"
	}

	text := fmt.Sprintf("✅ VPN конфиг успешно %s!\n\n"+
		"📅 Период: %d %s\n"+
		"💰 Списано: %.0f₽\n"+
		"💳 Остаток: %.2f₽\n"+
		"⏰ Активен до: %s\n\n"+
		"🔗 Ссылка на подписку:\n`%s`\n\n"+
		"💡 Нажмите 'Подключить (%s)' для автоматического импорта",
		actionText, days, common.GetDaysWord(days), cost, user.Balance, expiryDate, configURL, common.GetAppName())

	log.Printf("PROCESS_PAYMENT_CALLBACK: Текст успешного платежа для TelegramID=%d: %s", user.TelegramID, text)
	editMsg = tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ParseMode = "Markdown"
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("PROCESS_PAYMENT_CALLBACK: Ошибка отправки сообщения об успехе для TelegramID=%d: %v", user.TelegramID, err)
	}
}

// ProcessTopup обрабатывает пополнение баланса
func ProcessTopup(bot *tgbotapi.BotAPI, chatID int64, messageID int, user *common.User, amount int) {
	log.Printf("PROCESS_TOPUP: Начало обработки пополнения для TelegramID=%d, amount=%d", user.TelegramID, amount)

	// Имитируем пополнение баланса (в реальности здесь была бы интеграция с платежной системой)
	err := common.AddBalance(user.TelegramID, float64(amount))
	if err != nil {
		log.Printf("PROCESS_TOPUP: Ошибка пополнения баланса для TelegramID=%d: %v", user.TelegramID, err)
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "❌ Ошибка пополнения баланса")
		if _, err := bot.Send(editMsg); err != nil {
			log.Printf("PROCESS_TOPUP: Ошибка отправки сообщения об ошибке для TelegramID=%d: %v", user.TelegramID, err)
		}
		return
	}

	// Обновляем данные пользователя
	user.Balance += float64(amount)
	user.TotalPaid += float64(amount)

	log.Printf("PROCESS_TOPUP: Баланс обновлён для TelegramID=%d, Balance=%.2f, TotalPaid=%.2f", user.TelegramID, user.Balance, user.TotalPaid)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔐 Купить VPN", "vpn"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
		),
	)

	text := fmt.Sprintf("✅ Баланс успешно пополнен!\n\n"+
		"💰 Пополнено: %d₽\n"+
		"💳 Ваш баланс: %.2f₽\n\n"+
		"Теперь вы можете купить VPN конфиг!",
		amount, user.Balance)

	log.Printf("PROCESS_TOPUP: Текст успешного пополнения для TelegramID=%d: %s", user.TelegramID, text)
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("PROCESS_TOPUP: Ошибка отправки сообщения об успехе для TelegramID=%d: %v", user.TelegramID, err)
	}
}
