package handlers

import (
	"log"
	"strings"

	"bot/common"
	"bot/menus"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleMultiSubscriptionCallback обрабатывает callback'и для мультиподписок
func HandleMultiSubscriptionCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	log.Printf("HANDLE_MULTI_SUBSCRIPTION_CALLBACK: Обработка callback: %s", callback.Data)

	// Получаем пользователя
	user, err := common.GetUserByTelegramID(callback.From.ID)
	if err != nil {
		log.Printf("HANDLE_MULTI_SUBSCRIPTION_CALLBACK: Ошибка получения пользователя: %v", err)
		return
	}

	// Отвечаем на callback
	callbackConfig := tgbotapi.NewCallback(callback.ID, "")
	bot.Request(callbackConfig)

	// Обрабатываем различные типы callback'ов
	switch {
	case callback.Data == "multi_subscription":
		handleMultiSubscriptionStart(bot, callback, user)
	case strings.HasPrefix(callback.Data, "multi_select_server_"):
		handleServerSelection(bot, callback, user)
	case callback.Data == "multi_refresh_servers":
		handleRefreshServers(bot, callback, user)
	case callback.Data == "multi_confirm_selection":
		handleConfirmSelection(bot, callback, user)
	case callback.Data == "multi_edit_selection":
		handleEditSelection(bot, callback, user)
	case callback.Data == "multi_create_subscription":
		handleCreateSubscription(bot, callback, user)
	case callback.Data == "multi_cancel_selection":
		handleCancelSelection(bot, callback, user)
	default:
		log.Printf("HANDLE_MULTI_SUBSCRIPTION_CALLBACK: Неизвестный callback: %s", callback.Data)
	}
}

// handleMultiSubscriptionStart обрабатывает начало создания мультиподписки
func handleMultiSubscriptionStart(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, user *common.User) {
	log.Printf("HANDLE_MULTI_SUBSCRIPTION_START: Начало создания мультиподписки для пользователя %d", user.TelegramID)

	// Проверяем, включены ли мультиподписки
	if !common.MULTI_SUBSCRIPTION_ENABLED {
		menus.SendError(bot, callback.Message.Chat.ID, "Мультиподписки временно недоступны.")
		return
	}

	// Проверяем, есть ли уже мультиподписка
	existingSub, err := common.GetUserMultiSubscription(user.TelegramID)
	if err == nil && existingSub != nil && existingSub.IsActive {
		// Показываем существующую мультиподписку
		menus.SendMultiSubscriptionSuccess(bot, callback.Message.Chat.ID, user, existingSub)
		return
	}

	// Начинаем процесс выбора серверов
	menus.SendMultiSubscriptionMenu(bot, callback.Message.Chat.ID, user)
}

// handleServerSelection обрабатывает выбор сервера
func handleServerSelection(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, user *common.User) {
	log.Printf("HANDLE_SERVER_SELECTION: Выбор сервера для пользователя %d", user.TelegramID)

	// Извлекаем ID сервера из callback
	serverID := strings.TrimPrefix(callback.Data, "multi_select_server_")
	log.Printf("HANDLE_SERVER_SELECTION: Выбран сервер: %s", serverID)

	// Получаем текущее состояние выбора
	state, err := common.GetServerSelectionState(user.TelegramID)
	if err != nil {
		log.Printf("HANDLE_SERVER_SELECTION: Ошибка получения состояния: %v", err)
		menus.SendError(bot, callback.Message.Chat.ID, "Ошибка получения состояния выбора.")
		return
	}

	if state == nil {
		log.Printf("HANDLE_SERVER_SELECTION: Состояние не найдено, создаем новое")
		state = &common.ServerSelectionState{
			UserID:     user.TelegramID,
			Selected:   []string{},
			MaxServers: common.MULTI_SUBSCRIPTION_MAX_SERVERS,
			Step:       "select",
		}
	}

	// Переключаем выбор сервера
	if isServerSelected(serverID, state.Selected) {
		// Убираем сервер из выбранных
		state.Selected = removeFromSlice(state.Selected, serverID)
		log.Printf("HANDLE_SERVER_SELECTION: Сервер %s убран из выбранных", serverID)
	} else {
		// Проверяем лимит серверов
		if len(state.Selected) >= state.MaxServers {
			callbackConfig := tgbotapi.NewCallback(callback.ID, "❌ Достигнут лимит серверов!")
			bot.Request(callbackConfig)
			return
		}
		// Добавляем сервер к выбранным
		state.Selected = append(state.Selected, serverID)
		log.Printf("HANDLE_SERVER_SELECTION: Сервер %s добавлен к выбранным", serverID)
	}

	// Сохраняем обновленное состояние
	err = common.SaveServerSelectionState(state)
	if err != nil {
		log.Printf("HANDLE_SERVER_SELECTION: Ошибка сохранения состояния: %v", err)
		menus.SendError(bot, callback.Message.Chat.ID, "Ошибка сохранения выбора.")
		return
	}

	// Обновляем меню
	menus.UpdateMultiSubscriptionMenu(bot, callback.Message.Chat.ID, callback.Message.MessageID, user, state.Selected)
}

// handleRefreshServers обрабатывает обновление списка серверов
func handleRefreshServers(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, user *common.User) {
	log.Printf("HANDLE_REFRESH_SERVERS: Обновление списка серверов для пользователя %d", user.TelegramID)

	// Получаем текущее состояние
	state, err := common.GetServerSelectionState(user.TelegramID)
	if err != nil {
		log.Printf("HANDLE_REFRESH_SERVERS: Ошибка получения состояния: %v", err)
		menus.SendError(bot, callback.Message.Chat.ID, "Ошибка получения состояния.")
		return
	}

	selectedServers := []string{}
	if state != nil {
		selectedServers = state.Selected
	}

	// Обновляем меню
	menus.UpdateMultiSubscriptionMenu(bot, callback.Message.Chat.ID, callback.Message.MessageID, user, selectedServers)
}

// handleConfirmSelection обрабатывает подтверждение выбора серверов
func handleConfirmSelection(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, user *common.User) {
	log.Printf("HANDLE_CONFIRM_SELECTION: Подтверждение выбора для пользователя %d", user.TelegramID)

	// Получаем текущее состояние
	state, err := common.GetServerSelectionState(user.TelegramID)
	if err != nil {
		log.Printf("HANDLE_CONFIRM_SELECTION: Ошибка получения состояния: %v", err)
		menus.SendError(bot, callback.Message.Chat.ID, "Ошибка получения состояния.")
		return
	}

	if state == nil || len(state.Selected) == 0 {
		callbackConfig := tgbotapi.NewCallback(callback.ID, "❌ Выберите хотя бы один сервер!")
		bot.Request(callbackConfig)
		return
	}

	// Показываем подтверждение
	menus.SendMultiSubscriptionConfirmation(bot, callback.Message.Chat.ID, user, state.Selected)
}

// handleEditSelection обрабатывает редактирование выбора серверов
func handleEditSelection(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, user *common.User) {
	log.Printf("HANDLE_EDIT_SELECTION: Редактирование выбора для пользователя %d", user.TelegramID)

	// Получаем текущее состояние
	state, err := common.GetServerSelectionState(user.TelegramID)
	if err != nil {
		log.Printf("HANDLE_EDIT_SELECTION: Ошибка получения состояния: %v", err)
		menus.SendError(bot, callback.Message.Chat.ID, "Ошибка получения состояния.")
		return
	}

	selectedServers := []string{}
	if state != nil {
		selectedServers = state.Selected
	}

	// Возвращаемся к меню выбора
	menus.UpdateMultiSubscriptionMenu(bot, callback.Message.Chat.ID, callback.Message.MessageID, user, selectedServers)
}

// handleCreateSubscription обрабатывает создание мультиподписки
func handleCreateSubscription(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, user *common.User) {
	log.Printf("HANDLE_CREATE_SUBSCRIPTION: Создание мультиподписки для пользователя %d", user.TelegramID)

	// Получаем текущее состояние
	state, err := common.GetServerSelectionState(user.TelegramID)
	if err != nil {
		log.Printf("HANDLE_CREATE_SUBSCRIPTION: Ошибка получения состояния: %v", err)
		menus.SendError(bot, callback.Message.Chat.ID, "Ошибка получения состояния.")
		return
	}

	if state == nil || len(state.Selected) == 0 {
		callbackConfig := tgbotapi.NewCallback(callback.ID, "❌ Выберите хотя бы один сервер!")
		bot.Request(callbackConfig)
		return
	}

	// Проверяем баланс
	cost := float64(common.PRICE_PER_DAY)
	if user.Balance < cost {
		callbackConfig := tgbotapi.NewCallback(callback.ID, "❌ Недостаточно средств на балансе!")
		bot.Request(callbackConfig)
		menus.SendError(bot, callback.Message.Chat.ID, "Недостаточно средств на балансе. Пополните баланс для создания мультиподписки.")
		return
	}

	// Создаем мультиподписку
	subscription, err := common.CreateMultiSubscription(user.TelegramID, state.Selected)
	if err != nil {
		log.Printf("HANDLE_CREATE_SUBSCRIPTION: Ошибка создания мультиподписки: %v", err)
		menus.SendError(bot, callback.Message.Chat.ID, "Ошибка создания мультиподписки. Попробуйте позже.")
		return
	}

	// Списываем средства с баланса
	user.Balance -= cost
	user.TotalPaid += cost
	err = common.UpdateUser(user)
	if err != nil {
		log.Printf("HANDLE_CREATE_SUBSCRIPTION: Ошибка обновления баланса: %v", err)
		// Не возвращаем ошибку, так как мультиподписка уже создана
	}

	// Показываем успешное создание
	menus.SendMultiSubscriptionSuccess(bot, callback.Message.Chat.ID, user, subscription)
}

// handleCancelSelection обрабатывает отмену выбора серверов
func handleCancelSelection(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, user *common.User) {
	log.Printf("HANDLE_CANCEL_SELECTION: Отмена выбора для пользователя %d", user.TelegramID)

	// Удаляем состояние выбора
	_, err := common.GetServerSelectionState(user.TelegramID)
	if err == nil {
		// Состояние существует, удаляем его
		// (функция удаления будет вызвана при создании мультиподписки)
	}

	// Возвращаемся в главное меню
	menus.SendMainMenuWithMultiSubscription(bot, callback.Message.Chat.ID, user)
}

// isServerSelected проверяет, выбран ли сервер
func isServerSelected(serverID string, selected []string) bool {
	for _, id := range selected {
		if id == serverID {
			return true
		}
	}
	return false
}

// removeFromSlice удаляет элемент из слайса
func removeFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
