package powerOff

import (
	"fmt"
	"log"
	"strings"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandlePoweroffCommand обрабатывает команду /poweroff
func HandlePoweroffCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if !common.POWEROFF_SYSTEM_ENABLED {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Система безопасного выключения отключена в конфигурации")
		bot.Send(msg)
		return
	}

	if message.From.ID != common.ADMIN_ID {
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚫 Доступ запрещён")
		bot.Send(msg)
		return
	}

	// Парсим аргументы команды
	args := strings.Fields(message.Text)
	if len(args) > 1 {
		switch args[1] {
		case "status":
			handleStatusCommand(bot, message)
			return
		case "cancel":
			handleCancelCommand(bot, message)
			return
		case "force":
			handleForceCommand(bot, message)
			return
		}
	}

	// Обычная команда /poweroff
	handleShutdownRequest(bot, message)
}

// handleShutdownRequest обрабатывает запрос на выключение
func handleShutdownRequest(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("POWEROFF_HANDLER: handleShutdownRequest вызвана, GlobalShutdownManager = %v", GlobalShutdownManager != nil)
	if GlobalShutdownManager == nil {
		log.Printf("POWEROFF_HANDLER: GlobalShutdownManager is nil")
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Система выключения не инициализирована")
		bot.Send(msg)
		return
	}

	// Проверяем, не запущен ли уже процесс выключения
	isInProgress := GlobalShutdownManager.IsShutdownInProgress()
	log.Printf("POWEROFF_HANDLER: IsShutdownInProgress = %v", isInProgress)
	if isInProgress {
		status := GlobalShutdownManager.GetStatus()
		log.Printf("POWEROFF_HANDLER: Процесс уже запущен, состояние: %s", status.State.String())
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("⚠️ Процесс выключения уже запущен администратором %d\n"+
				"Состояние: %s\n"+
				"Активных платежей: %d\n"+
				"Время до принудительного выключения: %d сек",
				status.RequestedBy, status.State.String(), status.ActivePayments, status.TimeRemaining))
		bot.Send(msg)
		return
	}

	// Запрашиваем выключение
	log.Printf("POWEROFF_HANDLER: Запрашиваем выключение для администратора %d", message.From.ID)
	err := GlobalShutdownManager.RequestShutdown(message.From.ID)
	if err != nil {
		log.Printf("POWEROFF_HANDLER: Ошибка запроса выключения: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка: %v", err))
		bot.Send(msg)
		return
	}
	log.Printf("POWEROFF_HANDLER: Выключение успешно запрошено")

	// Отправляем подтверждение
	msg := tgbotapi.NewMessage(message.Chat.ID,
		"⚠️ <b>Запрошено безопасное выключение</b>\n\n"+
			"Проверяем активные платежи...\n\n"+
			"Используйте /poweroff status для проверки статуса\n"+
			"Используйте /poweroff cancel для отмены")
	msg.ParseMode = "HTML"
	bot.Send(msg)

	// Запускаем процесс выключения
	go GlobalShutdownManager.StartShutdownProcess(bot, message.Chat.ID)
}

// handleStatusCommand обрабатывает команду /poweroff status
func handleStatusCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if GlobalShutdownManager == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Система выключения не инициализирована")
		bot.Send(msg)
		return
	}

	status := GlobalShutdownManager.GetStatus()

	var statusText string
	switch status.State {
	case ShutdownStateNormal:
		statusText = "✅ <b>Нормальная работа</b>\n\nСистема готова к работе"
	case ShutdownStatePreparation:
		statusText = fmt.Sprintf("⚠️ <b>Подготовка к выключению</b>\n\n"+
			"Запросил: %d\n"+
			"Время запроса: %s\n"+
			"Активных платежей: %d\n"+
			"Время до принудительного выключения: %d сек",
			status.RequestedBy, status.RequestedAt.Format("15:04:05"),
			status.ActivePayments, status.TimeRemaining)
	case ShutdownStateShuttingDown:
		statusText = "🔌 <b>Процесс выключения</b>\n\nБот выключается..."
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup()
	if status.State != ShutdownStateNormal {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", "poweroff_status"),
			),
		)
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, statusText)
	msg.ParseMode = "HTML"
	if len(keyboard.InlineKeyboard) > 0 {
		msg.ReplyMarkup = &keyboard
	}
	bot.Send(msg)
}

// handleCancelCommand обрабатывает команду /poweroff cancel
func handleCancelCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if GlobalShutdownManager == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Система выключения не инициализирована")
		bot.Send(msg)
		return
	}

	err := GlobalShutdownManager.CancelShutdown(message.From.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка отмены: %v", err))
		bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "✅ Выключение отменено. Система вернулась к нормальной работе")
	bot.Send(msg)
}

// handleForceCommand обрабатывает команду /poweroff force
func handleForceCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if GlobalShutdownManager == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Система выключения не инициализирована")
		bot.Send(msg)
		return
	}

	// Подтверждение принудительного выключения
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, выключить принудительно", "poweroff_force_confirm"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "poweroff_cancel"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID,
		"⚠️ <b>Принудительное выключение</b>\n\n"+
			"Это действие немедленно выключит бота без ожидания завершения платежей.\n"+
			"Активные платежи могут быть потеряны!\n\n"+
			"Вы уверены?")
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard
	bot.Send(msg)
}

// HandlePoweroffCallback обрабатывает callback-запросы от кнопок
func HandlePoweroffCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	data := callback.Data
	chatID := callback.Message.Chat.ID
	messageID := callback.Message.MessageID

	log.Printf("POWEROFF_CALLBACK: Обработка callback '%s' от пользователя %d", data, callback.From.ID)

	switch data {
	case "poweroff_status":
		// Обновляем статус
		handleStatusCommand(bot, &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: chatID},
			From: callback.From,
		})
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))

	case "poweroff_force_confirm":
		// Подтверждение принудительного выключения
		if callback.From.ID != common.ADMIN_ID {
			bot.Request(tgbotapi.NewCallback(callback.ID, "Доступ запрещён"))
			return
		}

		// Выполняем принудительное выключение
		msg := tgbotapi.NewEditMessageText(chatID, messageID, "🔌 Принудительное выключение...")
		bot.Send(msg)

		// Останавливаем сервисы
		if GlobalShutdownManager != nil {
			GlobalShutdownManager.stopServices()
		}

		// Закрываем соединения с БД
		common.DisconnectMongoDB()

		// Завершаем работу
		log.Println("POWEROFF: Принудительное выключение завершено")
		// Здесь можно добавить вызов os.Exit(0)

	case "poweroff_cancel":
		// Отмена
		msg := tgbotapi.NewEditMessageText(chatID, messageID, "❌ Операция отменена")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, "Отменено"))

	default:
		bot.Request(tgbotapi.NewCallback(callback.ID, "Неизвестная команда"))
	}
}

// IsPoweroffCallback проверяет, является ли callback связанным с системой выключения
func IsPoweroffCallback(data string) bool {
	poweroffCallbacks := []string{
		"poweroff_status",
		"poweroff_force_confirm",
		"poweroff_cancel",
	}

	for _, callback := range poweroffCallbacks {
		if data == callback {
			return true
		}
	}
	return false
}
