package handlers

import (
	"log"
	"strconv"
	"strings"

	"bot/common"
	"bot/menus"
	"bot/payments"
	"bot/payments/promo"
	"bot/referralLink"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleCallback обрабатывает callback-запросы
func HandleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	data := callback.Data
	chatID := callback.Message.Chat.ID
	messageID := callback.Message.MessageID
	userID := callback.From.ID

	log.Printf("HANDLE_CALLBACK: Обработка callback, TelegramID=%d, Data='%s', ChatID=%d, MessageID=%d", userID, data, chatID, messageID)

	// Получаем пользователя
	user, err := common.GetUserByTelegramID(userID)
	if err != nil {
		log.Printf("HANDLE_CALLBACK: Ошибка получения пользователя TelegramID=%d: %v", userID, err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка получения данных"))
		return
	}
	if user == nil {
		log.Printf("HANDLE_CALLBACK: Пользователь TelegramID=%d не найден", userID)
		bot.Request(tgbotapi.NewCallback(callback.ID, "Пользователь не найден"))
		return
	}
	log.Printf("HANDLE_CALLBACK: Пользователь найден: TelegramID=%d, HasActiveConfig=%v, ClientID=%s, SubID=%s", user.TelegramID, user.HasActiveConfig, user.ClientID, user.SubID)

	// Проверяем, является ли это callback промокодов
	if promo.GlobalPromoManager != nil && promo.GlobalPromoManager.IsPromoCallback(data) {
		err := promo.GlobalPromoManager.HandleCallback(chatID, userID, data)
		if err != nil {
			log.Printf("HANDLE_CALLBACK: Ошибка обработки callback промокодов: %v", err)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка обработки команды"))
		} else {
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		}
		return
	}

	// Проверяем, является ли это callback реферальной системы
	if referralLink.GlobalReferralManager != nil && referralLink.GlobalReferralManager.IsReferralCallback(data) {
		referralLink.GlobalReferralManager.HandleCallback(chatID, userID, data)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	switch {
	case data == "balance":
		log.Printf("HANDLE_CALLBACK: Вызов editBalance для TelegramID=%d", userID)
		menus.EditBalance(bot, chatID, messageID, user)
	case data == "vpn":
		log.Printf("HANDLE_CALLBACK: Вызов editVPN для TelegramID=%d", userID)
		menus.EditVPN(bot, chatID, messageID, user)
	case data == "topup":
		log.Printf("HANDLE_CALLBACK: Вызов editTopup для TelegramID=%d", userID)
		menus.EditTopup(bot, chatID, messageID)
	case data == "main":
		log.Printf("HANDLE_CALLBACK: Вызов editMainMenu для TelegramID=%d", userID)
		menus.EditMainMenu(bot, chatID, messageID, user)
	case data == "ref":
		log.Printf("HANDLE_CALLBACK: Обработка реферального меню для TelegramID=%d", userID)
		handleRefCallback(bot, chatID, messageID, user)
	case data == "extend":
		log.Printf("HANDLE_CALLBACK: Вызов editExtend для TelegramID=%d", userID)
		if common.TARIFF_MODE_ENABLED {
			menus.EditExtend(bot, chatID, messageID)
		} else {
			// В режиме автосписания перенаправляем на пополнение баланса
			menus.EditTopup(bot, chatID, messageID)
		}
	case strings.HasPrefix(data, "days:"):
		if common.TARIFF_MODE_ENABLED {
			handleDaysCallback(bot, chatID, messageID, data, callback)
		} else {
			// В режиме автосписания перенаправляем на пополнение баланса
			menus.EditTopup(bot, chatID, messageID)
		}
	case strings.HasPrefix(data, "pay:"):
		handlePayCallback(bot, chatID, messageID, user, data, callback)
	case strings.HasPrefix(data, "topup:"):
		handleTopupCallback(bot, chatID, messageID, user, data, callback)
	case strings.HasPrefix(data, "check_payment:"):
		handleCheckPaymentCallback(bot, chatID, messageID, user, data, callback)
	case data == "traffic_config":
		handleTrafficConfigCallback(bot, chatID, userID, callback)
	case data == "check_traffic_now":
		handleCheckTrafficNowCallback(bot, chatID, messageID, userID, callback)
	case data == "activate_trial":
		handleActivateTrialCallback(bot, chatID, user, callback)
	case data == "download_app":
		log.Printf("HANDLE_CALLBACK: Вызов editDownloadApp для TelegramID=%d", userID)
		menus.EditDownloadApp(bot, chatID, messageID)
	case data == "device_ios":
		log.Printf("HANDLE_CALLBACK: Вызов editIOSLinks для TelegramID=%d", userID)
		menus.EditIOSLinks(bot, chatID, messageID)
	case data == "device_android":
		log.Printf("HANDLE_CALLBACK: Вызов editAndroidLinks для TelegramID=%d", userID)
		menus.EditAndroidLinks(bot, chatID, messageID)
	case strings.HasPrefix(data, "set_daily:"):
		handleSetDailyTrafficCallback(bot, chatID, messageID, data, callback)
	case strings.HasPrefix(data, "set_weekly:"):
		handleSetWeeklyTrafficCallback(bot, chatID, messageID, data, callback)
	case strings.HasPrefix(data, "set_monthly:"):
		handleSetMonthlyTrafficCallback(bot, chatID, messageID, data, callback)
	case data == "disable_traffic":
		handleDisableTrafficCallback(bot, chatID, messageID, callback)
	case data == "show_users_list":
		HandleShowUsersList(bot, callback)
	case data == "back_to_stats":
		HandleBackToStats(bot, callback)
	case data == "filter_paying":
		HandleFilterCategory(bot, callback, "paying")
	case data == "filter_trial_available":
		HandleFilterCategory(bot, callback, "trial_available")
	case data == "filter_trial_used":
		HandleFilterCategory(bot, callback, "trial_used")
	case data == "filter_inactive":
		HandleFilterCategory(bot, callback, "inactive")
	default:
		log.Printf("HANDLE_CALLBACK: Неизвестный callback для TelegramID=%d, data='%s'", userID, data)
	}

	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleDaysCallback обрабатывает callback для выбора дней
func handleDaysCallback(bot *tgbotapi.BotAPI, chatID int64, messageID int, data string, callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	days, err := strconv.Atoi(strings.TrimPrefix(data, "days:"))
	if err != nil {
		log.Printf("HANDLE_CALLBACK: Ошибка парсинга days для TelegramID=%d: %v", userID, err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка обработки периода"))
		return
	}
	log.Printf("HANDLE_CALLBACK: Вызов editPayment для TelegramID=%d, days=%d", userID, days)
	menus.EditPayment(bot, chatID, messageID, days)
}

// handlePayCallback обрабатывает callback для оплаты
func handlePayCallback(bot *tgbotapi.BotAPI, chatID int64, messageID int, user *common.User, data string, callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	days, err := strconv.Atoi(strings.TrimPrefix(data, "pay:"))
	if err != nil {
		log.Printf("HANDLE_CALLBACK: Ошибка парсинга pay для TelegramID=%d: %v", userID, err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка обработки платежа"))
		return
	}
	log.Printf("HANDLE_CALLBACK: Вызов processPaymentCallback для TelegramID=%d, days=%d", userID, days)
	ProcessPaymentCallback(bot, chatID, messageID, user, days)
}

// handleTopupCallback обрабатывает callback для пополнения
func handleTopupCallback(bot *tgbotapi.BotAPI, chatID int64, messageID int, user *common.User, data string, callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	amount, err := strconv.Atoi(strings.TrimPrefix(data, "topup:"))
	if err != nil {
		log.Printf("HANDLE_CALLBACK: Ошибка парсинга topup для TelegramID=%d: %v", userID, err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка обработки пополнения"))
		return
	}
	log.Printf("HANDLE_CALLBACK: Вызов processTopup для TelegramID=%d, amount=%d", userID, amount)
	ProcessTopup(bot, chatID, messageID, user, amount)
}

// handleCheckPaymentCallback обрабатывает callback для проверки платежа
func handleCheckPaymentCallback(bot *tgbotapi.BotAPI, chatID int64, messageID int, user *common.User, data string, callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	paymentID := strings.TrimPrefix(data, "check_payment:")
	if paymentID == "" {
		log.Printf("HANDLE_CALLBACK: Пустой ID платежа для TelegramID=%d", userID)
		bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка: отсутствует ID платежа"))
		return
	}

	log.Printf("HANDLE_CALLBACK: Проверка платежа %s для TelegramID=%d", paymentID, userID)

	// Проверяем, что платежная система инициализирована
	if payments.GlobalPaymentManager == nil {
		log.Printf("HANDLE_CALLBACK: Платежная система не инициализирована для проверки платежа %s", paymentID)
		bot.Request(tgbotapi.NewCallback(callback.ID, "Платежная система недоступна"))
		return
	}

	// Создаем обработчик веб-хуков для проверки платежа
	webhookHandlers := payments.NewWebhookHandlers(payments.GlobalPaymentManager)

	// Обрабатываем проверку платежа
	err := webhookHandlers.HandleCheckPayment(paymentID, chatID, messageID)
	if err != nil {
		log.Printf("HANDLE_CALLBACK: Ошибка проверки платежа %s для TelegramID=%d: %v", paymentID, userID, err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка проверки платежа"))
		return
	}

	bot.Request(tgbotapi.NewCallback(callback.ID, "Статус платежа обновлен"))
}

// handleTrafficConfigCallback обрабатывает callback для настроек трафика
func handleTrafficConfigCallback(bot *tgbotapi.BotAPI, chatID int64, userID int64, callback *tgbotapi.CallbackQuery) {
	if userID == common.ADMIN_ID {
		log.Printf("HANDLE_CALLBACK: Вызов showTrafficConfig для админа TelegramID=%d", userID)
		common.ShowTrafficConfig(bot, chatID)
	} else {
		bot.Request(tgbotapi.NewCallback(callback.ID, "🚫 Доступ запрещён"))
	}
}

// handleCheckTrafficNowCallback обрабатывает callback для проверки трафика
func handleCheckTrafficNowCallback(bot *tgbotapi.BotAPI, chatID int64, messageID int, userID int64, callback *tgbotapi.CallbackQuery) {
	if userID == common.ADMIN_ID {
		log.Printf("HANDLE_CALLBACK: Вызов CheckAndDisableTrafficLimit для админа TelegramID=%d", userID)
		common.CheckTrafficNow(bot, chatID, messageID)
	} else {
		bot.Request(tgbotapi.NewCallback(callback.ID, "🚫 Доступ запрещён"))
	}
}

// handleActivateTrialCallback обрабатывает callback для активации пробного периода
func handleActivateTrialCallback(bot *tgbotapi.BotAPI, chatID int64, user *common.User, callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	messageID := callback.Message.MessageID
	log.Printf("HANDLE_CALLBACK: Активация пробного периода для TelegramID=%d", userID)

	if err := common.TrialManager.CreateTrialConfig(bot, user, chatID); err != nil {
		log.Printf("HANDLE_CALLBACK: Ошибка создания пробного конфига для TelegramID=%d: %v", userID, err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Ошибка активации пробного периода"))
	} else {
		bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Пробный период активирован!"))
		// Переводим пользователя на главное меню
		log.Printf("HANDLE_CALLBACK: Переход на главное меню для TelegramID=%d", userID)
		menus.EditMainMenu(bot, chatID, messageID, user)
	}
}

// handleSetDailyTrafficCallback обрабатывает callback для установки дневного лимита трафика
func handleSetDailyTrafficCallback(bot *tgbotapi.BotAPI, chatID int64, messageID int, data string, callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	limitGB, err := strconv.Atoi(strings.TrimPrefix(data, "set_daily:"))
	if err != nil {
		log.Printf("HANDLE_CALLBACK: Ошибка парсинга set_daily для TelegramID=%d: %v", userID, err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка обработки лимита"))
		return
	}
	log.Printf("HANDLE_CALLBACK: Установка дневного лимита %d GB для TelegramID=%d", limitGB, userID)
	common.SetDailyTrafficLimit(bot, chatID, messageID, limitGB)
}

// handleSetWeeklyTrafficCallback обрабатывает callback для установки недельного лимита трафика
func handleSetWeeklyTrafficCallback(bot *tgbotapi.BotAPI, chatID int64, messageID int, data string, callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	limitGB, err := strconv.Atoi(strings.TrimPrefix(data, "set_weekly:"))
	if err != nil {
		log.Printf("HANDLE_CALLBACK: Ошибка парсинга set_weekly для TelegramID=%d: %v", userID, err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка обработки лимита"))
		return
	}
	log.Printf("HANDLE_CALLBACK: Установка недельного лимита %d GB для TelegramID=%d", limitGB, userID)
	common.SetWeeklyTrafficLimit(bot, chatID, messageID, limitGB)
}

// handleSetMonthlyTrafficCallback обрабатывает callback для установки месячного лимита трафика
func handleSetMonthlyTrafficCallback(bot *tgbotapi.BotAPI, chatID int64, messageID int, data string, callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	limitGB, err := strconv.Atoi(strings.TrimPrefix(data, "set_monthly:"))
	if err != nil {
		log.Printf("HANDLE_CALLBACK: Ошибка парсинга set_monthly для TelegramID=%d: %v", userID, err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка обработки лимита"))
		return
	}
	log.Printf("HANDLE_CALLBACK: Установка месячного лимита %d GB для TelegramID=%d", limitGB, userID)
	common.SetMonthlyTrafficLimit(bot, chatID, messageID, limitGB)
}

// handleDisableTrafficCallback обрабатывает callback для отключения лимитов трафика
func handleDisableTrafficCallback(bot *tgbotapi.BotAPI, chatID int64, messageID int, callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	log.Printf("HANDLE_CALLBACK: Отключение лимитов трафика для TelegramID=%d", userID)
	common.DisableTrafficLimit(bot, chatID, messageID)
}

// handleRefCallback обрабатывает callback реферальной системы
func handleRefCallback(bot *tgbotapi.BotAPI, chatID int64, messageID int, user *common.User) {
	log.Printf("HANDLE_CALLBACK: Обработка реферального callback для TelegramID=%d", user.TelegramID)

	// Проверяем, включена ли реферальная система
	if !common.REFERRAL_SYSTEM_ENABLED {
		msg := tgbotapi.NewMessage(chatID, "❌ Реферальная система временно отключена")
		bot.Send(msg)
		return
	}

	// Используем глобальный менеджер рефералов
	if referralLink.GlobalReferralManager != nil {
		referralLink.GlobalReferralManager.SendReferralMenu(chatID, user)
	} else {
		msg := tgbotapi.NewMessage(chatID, "❌ Реферальная система не инициализирована")
		bot.Send(msg)
	}
}
