package common

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ShowTrafficConfig показывает настройки трафика
func ShowTrafficConfig(bot *tgbotapi.BotAPI, chatID int64) {
	log.Printf("SHOW_TRAFFIC_CONFIG: Показ настроек трафика для ChatID=%d", chatID)

	var trafficLimitText string
	if TRAFFIC_LIMIT_GB <= 0 {
		trafficLimitText = "❌ Отключен (безлимит)"
	} else {
		trafficLimitText = fmt.Sprintf("✅ %d GB", TRAFFIC_LIMIT_GB)
	}

	var resetText string
	if TRAFFIC_RESET_ENABLED && TRAFFIC_RESET_INTERVAL > 0 {
		resetText = fmt.Sprintf("✅ %d минут", TRAFFIC_RESET_INTERVAL)
	} else {
		resetText = "❌ Отключен"
	}

	text := fmt.Sprintf("📊 Настройки мониторинга трафика\n\n"+
		"🔍 Интервал проверки: %d минут\n"+
		"📈 Лимит трафика: %s\n"+
		"🔄 Интервал сброса: %s\n\n"+
		"💡 Система автоматически отключает конфиги при превышении лимита и включает их обратно при сбросе трафика.",
		TRAFFIC_CHECK_INTERVAL, trafficLimitText, resetText)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Проверить сейчас", "check_traffic_now"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "main"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = &keyboard
	if _, err := bot.Send(msg); err != nil {
		log.Printf("SHOW_TRAFFIC_CONFIG: Ошибка отправки сообщения: %v", err)
	}
}

// CheckTrafficNow выполняет ручную проверку трафика
func CheckTrafficNow(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	log.Printf("CHECK_TRAFFIC_NOW: Ручная проверка трафика для ChatID=%d", chatID)

	// Обновляем сообщение
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "⏳ Проверка трафика...")
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("CHECK_TRAFFIC_NOW: Ошибка отправки сообщения о процессе: %v", err)
	}

	// Выполняем проверку трафика
	if err := CheckAndDisableTrafficLimit(); err != nil {
		log.Printf("CHECK_TRAFFIC_NOW: Ошибка проверки трафика: %v", err)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Повторить", "check_traffic_now"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "traffic_config"),
			),
		)

		text := fmt.Sprintf("❌ Ошибка проверки трафика:\n%v", err)
		editMsg = tgbotapi.NewEditMessageText(chatID, messageID, text)
		editMsg.ReplyMarkup = &keyboard
		if _, err := bot.Send(editMsg); err != nil {
			log.Printf("CHECK_TRAFFIC_NOW: Ошибка отправки сообщения об ошибке: %v", err)
		}
		return
	}

	// Успешная проверка
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Проверить снова", "check_traffic_now"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "traffic_config"),
		),
	)

	text := "✅ Проверка трафика завершена!\n\n" +
		"📊 Все клиенты проверены на превышение лимита трафика.\n" +
		"🔍 Следующая автоматическая проверка через " + fmt.Sprintf("%d", TRAFFIC_CHECK_INTERVAL) + " минут."

	editMsg = tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("CHECK_TRAFFIC_NOW: Ошибка отправки сообщения об успехе: %v", err)
	}
}

// EditTrafficDaily редактирует дневной лимит трафика
func EditTrafficDaily(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	log.Printf("EDIT_TRAFFIC_DAILY: Редактирование дневного лимита для ChatID=%d", chatID)

	config := GetTrafficConfig()

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1 ГБ", "set_daily:1"),
			tgbotapi.NewInlineKeyboardButtonData("2 ГБ", "set_daily:2"),
			tgbotapi.NewInlineKeyboardButtonData("5 ГБ", "set_daily:5"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("10 ГБ", "set_daily:10"),
			tgbotapi.NewInlineKeyboardButtonData("20 ГБ", "set_daily:20"),
			tgbotapi.NewInlineKeyboardButtonData("50 ГБ", "set_daily:50"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "traffic_config"),
		),
	)

	text := fmt.Sprintf("📊 Установка дневного лимита трафика\n\n"+
		"Текущий дневной лимит: %s\n\n"+
		"Выберите новый лимит:",
		FormatTrafficLimit(config.DailyLimitGB))

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("EDIT_TRAFFIC_DAILY: Ошибка отправки сообщения: %v", err)
	}
}

// EditTrafficWeekly редактирует недельный лимит трафика
func EditTrafficWeekly(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	log.Printf("EDIT_TRAFFIC_WEEKLY: Редактирование недельного лимита для ChatID=%d", chatID)

	config := GetTrafficConfig()

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("10 ГБ", "set_weekly:10"),
			tgbotapi.NewInlineKeyboardButtonData("20 ГБ", "set_weekly:20"),
			tgbotapi.NewInlineKeyboardButtonData("50 ГБ", "set_weekly:50"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("100 ГБ", "set_weekly:100"),
			tgbotapi.NewInlineKeyboardButtonData("200 ГБ", "set_weekly:200"),
			tgbotapi.NewInlineKeyboardButtonData("500 ГБ", "set_weekly:500"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "traffic_config"),
		),
	)

	text := fmt.Sprintf("📊 Установка недельного лимита трафика\n\n"+
		"Текущий недельный лимит: %s\n\n"+
		"Выберите новый лимит:",
		FormatTrafficLimit(config.WeeklyLimitGB))

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("EDIT_TRAFFIC_WEEKLY: Ошибка отправки сообщения: %v", err)
	}
}

// EditTrafficMonthly редактирует месячный лимит трафика
func EditTrafficMonthly(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	log.Printf("EDIT_TRAFFIC_MONTHLY: Редактирование месячного лимита для ChatID=%d", chatID)

	config := GetTrafficConfig()

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("50 ГБ", "set_monthly:50"),
			tgbotapi.NewInlineKeyboardButtonData("100 ГБ", "set_monthly:100"),
			tgbotapi.NewInlineKeyboardButtonData("200 ГБ", "set_monthly:200"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("300 ГБ", "set_monthly:300"),
			tgbotapi.NewInlineKeyboardButtonData("500 ГБ", "set_monthly:500"),
			tgbotapi.NewInlineKeyboardButtonData("1 ТБ", "set_monthly:1024"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "traffic_config"),
		),
	)

	text := fmt.Sprintf("📊 Установка месячного лимита трафика\n\n"+
		"Текущий месячный лимит: %s\n\n"+
		"Выберите новый лимит:",
		FormatTrafficLimit(config.MonthlyLimitGB))

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("EDIT_TRAFFIC_MONTHLY: Ошибка отправки сообщения: %v", err)
	}
}

// SetDailyTrafficLimit устанавливает дневной лимит трафика
func SetDailyTrafficLimit(bot *tgbotapi.BotAPI, chatID int64, messageID int, limitGB int) {
	log.Printf("SET_DAILY_TRAFFIC_LIMIT: Установка дневного лимита %d GB для ChatID=%d", limitGB, chatID)

	config := GetTrafficConfig()
	config.DailyLimitGB = limitGB
	config.WeeklyLimitGB = 0
	config.MonthlyLimitGB = 0
	config.LimitGB = 0
	config.Enabled = true
	config.ResetDays = 1

	SetTrafficConfig(config)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к настройкам", "traffic_config"),
		),
	)

	text := fmt.Sprintf("✅ Дневной лимит трафика установлен!\n\n"+
		"📊 Новый лимит: %s в день\n"+
		"🔄 Сброс: каждый день\n\n"+
		"Настройка применена ко всем новым подпискам.",
		FormatTrafficLimit(limitGB))

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("SET_DAILY_TRAFFIC_LIMIT: Ошибка отправки сообщения: %v", err)
	}
}

// SetWeeklyTrafficLimit устанавливает недельный лимит трафика
func SetWeeklyTrafficLimit(bot *tgbotapi.BotAPI, chatID int64, messageID int, limitGB int) {
	log.Printf("SET_WEEKLY_TRAFFIC_LIMIT: Установка недельного лимита %d GB для ChatID=%d", limitGB, chatID)

	config := GetTrafficConfig()
	config.DailyLimitGB = 0
	config.WeeklyLimitGB = limitGB
	config.MonthlyLimitGB = 0
	config.LimitGB = 0
	config.Enabled = true
	config.ResetDays = 7

	SetTrafficConfig(config)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к настройкам", "traffic_config"),
		),
	)

	text := fmt.Sprintf("✅ Недельный лимит трафика установлен!\n\n"+
		"📊 Новый лимит: %s в неделю\n"+
		"🔄 Сброс: каждую неделю\n\n"+
		"Настройка применена ко всем новым подпискам.",
		FormatTrafficLimit(limitGB))

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("SET_WEEKLY_TRAFFIC_LIMIT: Ошибка отправки сообщения: %v", err)
	}
}

// SetMonthlyTrafficLimit устанавливает месячный лимит трафика
func SetMonthlyTrafficLimit(bot *tgbotapi.BotAPI, chatID int64, messageID int, limitGB int) {
	log.Printf("SET_MONTHLY_TRAFFIC_LIMIT: Установка месячного лимита %d GB для ChatID=%d", limitGB, chatID)

	config := GetTrafficConfig()
	config.DailyLimitGB = 0
	config.WeeklyLimitGB = 0
	config.MonthlyLimitGB = limitGB
	config.LimitGB = 0
	config.Enabled = true
	config.ResetDays = 30

	SetTrafficConfig(config)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к настройкам", "traffic_config"),
		),
	)

	text := fmt.Sprintf("✅ Месячный лимит трафика установлен!\n\n"+
		"📊 Новый лимит: %s в месяц\n"+
		"🔄 Сброс: каждый месяц\n\n"+
		"Настройка применена ко всем новым подпискам.",
		FormatTrafficLimit(limitGB))

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("SET_MONTHLY_TRAFFIC_LIMIT: Ошибка отправки сообщения: %v", err)
	}
}

// DisableTrafficLimit отключает лимиты трафика
func DisableTrafficLimit(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	log.Printf("DISABLE_TRAFFIC_LIMIT: Отключение лимитов трафика для ChatID=%d", chatID)

	config := GetTrafficConfig()
	config.Enabled = false
	config.DailyLimitGB = 0
	config.WeeklyLimitGB = 0
	config.MonthlyLimitGB = 0
	config.LimitGB = 0
	config.ResetDays = 0

	SetTrafficConfig(config)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к настройкам", "traffic_config"),
		),
	)

	text := "❌ Лимиты трафика отключены!\n\n" +
		"📊 Новые подписки будут без ограничений трафика.\n\n" +
		"⚠️ Существующие подписки сохранят свои текущие лимиты."

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("DISABLE_TRAFFIC_LIMIT: Ошибка отправки сообщения: %v", err)
	}
}
