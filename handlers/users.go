package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleUsersCommand обрабатывает команду /users
func HandleUsersCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("HANDLE_USERS_COMMAND: Выполнение команды /users для TelegramID=%d", message.From.ID)

	if message.From.ID != common.ADMIN_ID {
		log.Printf("HANDLE_USERS_COMMAND: Пользователь TelegramID=%d не является админом", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚫 Доступ запрещён")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_USERS_COMMAND: Ошибка отправки сообщения о запрете: %v", err)
		}
		return
	}

	// Получаем статистику пользователей
	stats, err := common.GetUsersStatistics()
	if err != nil {
		log.Printf("HANDLE_USERS_COMMAND: Ошибка получения статистики: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка получения статистики: %v", err))
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_USERS_COMMAND: Ошибка отправки сообщения об ошибке: %v", err)
		}
		return
	}

	// Формируем сообщение со статистикой
	text := fmt.Sprintf("📊 Статистика пользователей:\n\n"+
		"👥 Общее количество: %d\n"+
		"💰 Платящие клиенты: %d (%.1f%%)\n"+
		"🆓 Только пробный период: %d (%.1f%%)\n"+
		"❌ Неактивные: %d (%.1f%%)\n\n"+
		"📈 Детальная информация:\n"+
		"• Активные конфиги: %d\n"+
		"• Доступен пробный: %d\n"+
		"• Потратили пробный, но не платили: %d\n"+
		"• Общий доход: %.2f₽\n"+
		"• Новые за неделю: %d\n"+
		"• Новые за месяц: %d\n"+
		"• Конверсия в платящих: %.1f%%",
		stats.TotalUsers,
		stats.PayingUsers, float64(stats.PayingUsers)/float64(stats.TotalUsers)*100,
		stats.TrialAvailableUsers, float64(stats.TrialAvailableUsers)/float64(stats.TotalUsers)*100,
		stats.InactiveUsers, float64(stats.InactiveUsers)/float64(stats.TotalUsers)*100,
		stats.ActiveConfigs,
		stats.TrialAvailableUsers,
		stats.TrialUsedUsers,
		stats.TotalRevenue,
		stats.NewThisWeek,
		stats.NewThisMonth,
		stats.ConversionRate)

	// Создаем клавиатуру с кнопкой
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Показать клиентов", "show_users_list"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyMarkup = &keyboard
	if _, err := bot.Send(msg); err != nil {
		log.Printf("HANDLE_USERS_COMMAND: Ошибка отправки сообщения: %v", err)
	}
}

// HandleUsersLimitCommand обрабатывает команды /users50, /users100, /users400 и т.д.
func HandleUsersLimitCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("HANDLE_USERS_LIMIT_COMMAND: Выполнение команды %s для TelegramID=%d", message.Command(), message.From.ID)

	if message.From.ID != common.ADMIN_ID {
		log.Printf("HANDLE_USERS_LIMIT_COMMAND: Пользователь TelegramID=%d не является админом", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚫 Доступ запрещён")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_USERS_LIMIT_COMMAND: Ошибка отправки сообщения о запрете: %v", err)
		}
		return
	}

	// Извлекаем лимит из команды
	command := message.Command()
	limitStr := strings.TrimPrefix(command, "users")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		log.Printf("HANDLE_USERS_LIMIT_COMMAND: Ошибка парсинга лимита из команды %s: %v", command, err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Неверный формат команды. Используйте: /users50, /users100, /users400 и т.д.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_USERS_LIMIT_COMMAND: Ошибка отправки сообщения об ошибке: %v", err)
		}
		return
	}

	// Ограничиваем максимальный лимит
	if limit > 5000 {
		limit = 5000
	}

	// Получаем отсортированных пользователей
	users, err := common.GetUsersSorted(limit)
	if err != nil {
		log.Printf("HANDLE_USERS_LIMIT_COMMAND: Ошибка получения пользователей: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка получения пользователей: %v", err))
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_USERS_LIMIT_COMMAND: Ошибка отправки сообщения об ошибке: %v", err)
		}
		return
	}

	// Формируем сообщение со списком пользователей
	text := fmt.Sprintf("📊 Все пользователи (%d из %d):\n\n", len(users), limit)
	text += formatUsersList(users)

	// Создаем клавиатуру
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к статистике", "back_to_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Платили", "filter_paying"),
			tgbotapi.NewInlineKeyboardButtonData("🆓 Могут пробовать", "filter_trial_available"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Пробовали", "filter_trial_used"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Неактивные", "filter_inactive"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyMarkup = &keyboard
	if _, err := bot.Send(msg); err != nil {
		log.Printf("HANDLE_USERS_LIMIT_COMMAND: Ошибка отправки сообщения: %v", err)
	}
}

// HandleShowUsersList обрабатывает callback "show_users_list"
func HandleShowUsersList(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery) {
	log.Printf("HANDLE_SHOW_USERS_LIST: Показ списка пользователей для TelegramID=%d", callbackQuery.From.ID)

	// Получаем отсортированных пользователей (первые 20)
	users, err := common.GetUsersSorted(20)
	if err != nil {
		log.Printf("HANDLE_SHOW_USERS_LIST: Ошибка получения пользователей: %v", err)
		text := fmt.Sprintf("❌ Ошибка получения пользователей: %v", err)
		editMsg := tgbotapi.NewEditMessageText(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, text)
		if _, err := bot.Send(editMsg); err != nil {
			log.Printf("HANDLE_SHOW_USERS_LIST: Ошибка отправки сообщения об ошибке: %v", err)
		}
		return
	}

	// Формируем сообщение со списком пользователей
	text := "📊 Пользователи (отсортированы по категориям):\n\n"
	text += formatUsersList(users)

	// Создаем клавиатуру
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к статистике", "back_to_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Платили", "filter_paying"),
			tgbotapi.NewInlineKeyboardButtonData("🆓 Могут пробовать", "filter_trial_available"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Пробовали", "filter_trial_used"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Неактивные", "filter_inactive"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("HANDLE_SHOW_USERS_LIST: Ошибка отправки сообщения: %v", err)
	}

	// Отвечаем на callback
	callback := tgbotapi.NewCallback(callbackQuery.ID, "")
	if _, err := bot.Request(callback); err != nil {
		log.Printf("HANDLE_SHOW_USERS_LIST: Ошибка ответа на callback: %v", err)
	}
}

// HandleBackToStats обрабатывает callback "back_to_stats"
func HandleBackToStats(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery) {
	log.Printf("HANDLE_BACK_TO_STATS: Возврат к статистике для TelegramID=%d", callbackQuery.From.ID)

	// Получаем статистику пользователей
	stats, err := common.GetUsersStatistics()
	if err != nil {
		log.Printf("HANDLE_BACK_TO_STATS: Ошибка получения статистики: %v", err)
		text := fmt.Sprintf("❌ Ошибка получения статистики: %v", err)
		editMsg := tgbotapi.NewEditMessageText(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, text)
		if _, err := bot.Send(editMsg); err != nil {
			log.Printf("HANDLE_BACK_TO_STATS: Ошибка отправки сообщения об ошибке: %v", err)
		}
		return
	}

	// Формируем сообщение со статистикой
	text := fmt.Sprintf("📊 Статистика пользователей:\n\n"+
		"👥 Общее количество: %d\n"+
		"💰 Платящие клиенты: %d (%.1f%%)\n"+
		"🆓 Только пробный период: %d (%.1f%%)\n"+
		"❌ Неактивные: %d (%.1f%%)\n\n"+
		"📈 Детальная информация:\n"+
		"• Активные конфиги: %d\n"+
		"• Доступен пробный: %d\n"+
		"• Потратили пробный, но не платили: %d\n"+
		"• Общий доход: %.2f₽\n"+
		"• Новые за неделю: %d\n"+
		"• Новые за месяц: %d\n"+
		"• Конверсия в платящих: %.1f%%",
		stats.TotalUsers,
		stats.PayingUsers, float64(stats.PayingUsers)/float64(stats.TotalUsers)*100,
		stats.TrialAvailableUsers, float64(stats.TrialAvailableUsers)/float64(stats.TotalUsers)*100,
		stats.InactiveUsers, float64(stats.InactiveUsers)/float64(stats.TotalUsers)*100,
		stats.ActiveConfigs,
		stats.TrialAvailableUsers,
		stats.TrialUsedUsers,
		stats.TotalRevenue,
		stats.NewThisWeek,
		stats.NewThisMonth,
		stats.ConversionRate)

	// Создаем клавиатуру с кнопкой
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Показать клиентов", "show_users_list"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("HANDLE_BACK_TO_STATS: Ошибка отправки сообщения: %v", err)
	}

	// Отвечаем на callback
	callback := tgbotapi.NewCallback(callbackQuery.ID, "")
	if _, err := bot.Request(callback); err != nil {
		log.Printf("HANDLE_BACK_TO_STATS: Ошибка ответа на callback: %v", err)
	}
}

// HandleFilterCategory обрабатывает фильтрацию по категориям
func HandleFilterCategory(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, category string) {
	log.Printf("HANDLE_FILTER_CATEGORY: Фильтрация по категории '%s' для TelegramID=%d", category, callbackQuery.From.ID)

	// Получаем пользователей по категории
	users, err := common.GetUsersByCategory(category, 50)
	if err != nil {
		log.Printf("HANDLE_FILTER_CATEGORY: Ошибка получения пользователей категории '%s': %v", category, err)
		text := fmt.Sprintf("❌ Ошибка получения пользователей: %v", err)
		editMsg := tgbotapi.NewEditMessageText(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, text)
		if _, err := bot.Send(editMsg); err != nil {
			log.Printf("HANDLE_FILTER_CATEGORY: Ошибка отправки сообщения об ошибке: %v", err)
		}
		return
	}

	// Определяем название категории
	var categoryName string
	switch category {
	case "paying":
		categoryName = "💰 ПЛАТЯЩИЕ КЛИЕНТЫ"
	case "trial_available":
		categoryName = "🆓 ДОСТУПЕН ПРОБНЫЙ ПЕРИОД"
	case "trial_used":
		categoryName = "🔄 ПОТРАТИЛИ ПРОБНЫЙ, НО НЕ ПЛАТИЛИ"
	case "inactive":
		categoryName = "❌ НЕАКТИВНЫЕ"
	default:
		categoryName = "📊 ПОЛЬЗОВАТЕЛИ"
	}

	// Формируем сообщение
	text := fmt.Sprintf("%s (%d):\n\n", categoryName, len(users))
	text += formatUsersList(users)

	// Создаем клавиатуру
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к статистике", "back_to_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Платили", "filter_paying"),
			tgbotapi.NewInlineKeyboardButtonData("🆓 Могут пробовать", "filter_trial_available"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Пробовали", "filter_trial_used"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Неактивные", "filter_inactive"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("HANDLE_FILTER_CATEGORY: Ошибка отправки сообщения: %v", err)
	}

	// Отвечаем на callback
	callback := tgbotapi.NewCallback(callbackQuery.ID, "")
	if _, err := bot.Request(callback); err != nil {
		log.Printf("HANDLE_FILTER_CATEGORY: Ошибка ответа на callback: %v", err)
	}
}

// formatUsersList форматирует список пользователей для отображения
func formatUsersList(users []common.User) string {
	if len(users) == 0 {
		return "Пользователи не найдены."
	}

	var text strings.Builder
	currentCategory := ""
	counter := 1

	for _, user := range users {
		// Определяем категорию пользователя
		var category string
		if user.Balance > 0 || user.TotalPaid > 0 {
			category = "💰 ПЛАТЯЩИЕ КЛИЕНТЫ"
		} else if !user.HasUsedTrial {
			category = "🆓 ДОСТУПЕН ПРОБНЫЙ ПЕРИОД"
		} else if user.HasUsedTrial && !user.HasActiveConfig {
			category = "🔄 ПОТРАТИЛИ ПРОБНЫЙ, НО НЕ ПЛАТИЛИ"
		} else {
			category = "❌ НЕАКТИВНЫЕ"
		}

		// Добавляем заголовок категории, если она изменилась
		if category != currentCategory {
			if currentCategory != "" {
				text.WriteString("\n")
			}
			text.WriteString(fmt.Sprintf("%s:\n", category))
			currentCategory = category
		}

		// Формируем username
		username := ""
		if user.Username != "" {
			username = " @" + user.Username
		}

		// Статусы
		trialStatus := "❌"
		if !user.HasUsedTrial {
			trialStatus = "✅"
		}
		configStatus := "❌"
		if user.HasActiveConfig {
			configStatus = "✅"
		}

		// Дата регистрации
		regDate := user.CreatedAt.Format("02.01.2006")

		// Добавляем пользователя
		text.WriteString(fmt.Sprintf("%d. %s ID: %d%s - Пробный: %s, Конфиг: %s, Баланс: %.2f₽\n",
			counter, user.FirstName, user.TelegramID, username, trialStatus, configStatus, user.Balance))
		text.WriteString(fmt.Sprintf("   📅 Регистрация: %s\n", regDate))

		counter++
	}

	return text.String()
}
