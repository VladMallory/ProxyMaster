package menus

import (
	"fmt"
	"log"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SendMultiSubscriptionMenu отправляет меню выбора серверов для мультиподписки
func SendMultiSubscriptionMenu(bot *tgbotapi.BotAPI, chatID int64, user *common.User) {
	log.Printf("SEND_MULTI_SUBSCRIPTION_MENU: Отправка меню мультиподписки для TelegramID=%d", user.TelegramID)

	// Получаем доступные серверы
	servers, err := common.GetAvailableServers()
	if err != nil {
		log.Printf("SEND_MULTI_SUBSCRIPTION_MENU: Ошибка получения серверов: %v", err)
		SendError(bot, chatID, "Ошибка загрузки серверов. Попробуйте позже.")
		return
	}

	if len(servers) == 0 {
		log.Printf("SEND_MULTI_SUBSCRIPTION_MENU: Нет доступных серверов")
		SendError(bot, chatID, "В данный момент нет доступных серверов.")
		return
	}

	// Создаем состояние выбора серверов
	state := &common.ServerSelectionState{
		UserID:     user.TelegramID,
		Selected:   []string{},
		MaxServers: common.MULTI_SUBSCRIPTION_MAX_SERVERS,
		Step:       "select",
	}

	err = common.SaveServerSelectionState(state)
	if err != nil {
		log.Printf("SEND_MULTI_SUBSCRIPTION_MENU: Ошибка сохранения состояния: %v", err)
		SendError(bot, chatID, "Ошибка инициализации выбора серверов.")
		return
	}

	// Создаем клавиатуру с серверами
	keyboard := createServerSelectionKeyboard(servers, []string{})

	text := "🌍 <b>Выберите серверы для мультиподписки</b>\n\n"
	text += "Вы можете выбрать до " + fmt.Sprintf("%d", common.MULTI_SUBSCRIPTION_MAX_SERVERS) + " серверов.\n"
	text += "Нажмите на серверы, которые хотите включить в подписку.\n\n"
	text += "📋 <b>Доступные серверы:</b>\n"

	// Добавляем информацию о серверах
	for i, server := range servers {
		if i >= 10 { // Ограничиваем количество отображаемых серверов
			text += fmt.Sprintf("... и еще %d серверов\n", len(servers)-10)
			break
		}
		text += fmt.Sprintf("%s %s\n", server.Flag, server.Name)
	}

	text += "\n💡 <b>Совет:</b> Выберите серверы из разных стран для лучшей производительности!"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("SEND_MULTI_SUBSCRIPTION_MENU: Ошибка отправки сообщения: %v", err)
	}
}

// createServerSelectionKeyboard создает клавиатуру для выбора серверов
func createServerSelectionKeyboard(servers []common.Server, selected []string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Создаем кнопки для серверов (по 2 в ряд)
	for i := 0; i < len(servers); i += 2 {
		var row []tgbotapi.InlineKeyboardButton

		// Первая кнопка в ряду
		server1 := servers[i]
		buttonText1 := server1.Flag + " " + server1.Name
		if isServerSelected(server1.ID, selected) {
			buttonText1 = "✅ " + buttonText1
		}
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(buttonText1, "multi_select_server_"+server1.ID))

		// Вторая кнопка в ряду (если есть)
		if i+1 < len(servers) {
			server2 := servers[i+1]
			buttonText2 := server2.Flag + " " + server2.Name
			if isServerSelected(server2.ID, selected) {
				buttonText2 = "✅ " + buttonText2
			}
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(buttonText2, "multi_select_server_"+server2.ID))
		}

		rows = append(rows, row)
	}

	// Добавляем кнопки управления
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", "multi_refresh_servers"),
	})

	// Кнопки подтверждения и отмены
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить выбор", "multi_confirm_selection"),
		tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "multi_cancel_selection"),
	})

	// Кнопка возврата в главное меню
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "back_to_main"),
	})

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
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

// UpdateMultiSubscriptionMenu обновляет меню выбора серверов
func UpdateMultiSubscriptionMenu(bot *tgbotapi.BotAPI, chatID int64, messageID int, user *common.User, selectedServers []string) {
	log.Printf("UPDATE_MULTI_SUBSCRIPTION_MENU: Обновление меню для TelegramID=%d, выбранных серверов: %d", user.TelegramID, len(selectedServers))

	// Получаем доступные серверы
	servers, err := common.GetAvailableServers()
	if err != nil {
		log.Printf("UPDATE_MULTI_SUBSCRIPTION_MENU: Ошибка получения серверов: %v", err)
		return
	}

	// Создаем клавиатуру с обновленным состоянием
	keyboard := createServerSelectionKeyboard(servers, selectedServers)

	text := "🌍 <b>Выберите серверы для мультиподписки</b>\n\n"
	text += "Вы можете выбрать до " + fmt.Sprintf("%d", common.MULTI_SUBSCRIPTION_MAX_SERVERS) + " серверов.\n"
	text += "Нажмите на серверы, которые хотите включить в подписку.\n\n"

	// Показываем выбранные серверы
	if len(selectedServers) > 0 {
		text += "✅ <b>Выбранные серверы:</b>\n"
		for _, serverID := range selectedServers {
			for _, server := range servers {
				if server.ID == serverID {
					text += fmt.Sprintf("• %s %s\n", server.Flag, server.Name)
					break
				}
			}
		}
		text += "\n"
	}

	text += "📋 <b>Доступные серверы:</b>\n"

	// Добавляем информацию о серверах
	for i, server := range servers {
		if i >= 10 { // Ограничиваем количество отображаемых серверов
			text += fmt.Sprintf("... и еще %d серверов\n", len(servers)-10)
			break
		}
		text += fmt.Sprintf("%s %s\n", server.Flag, server.Name)
	}

	text += "\n💡 <b>Совет:</b> Выберите серверы из разных стран для лучшей производительности!"

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ParseMode = "HTML"
	editMsg.ReplyMarkup = &keyboard

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("UPDATE_MULTI_SUBSCRIPTION_MENU: Ошибка обновления сообщения: %v", err)
	}
}

// SendMultiSubscriptionConfirmation отправляет подтверждение выбора серверов
func SendMultiSubscriptionConfirmation(bot *tgbotapi.BotAPI, chatID int64, user *common.User, selectedServers []string) {
	log.Printf("SEND_MULTI_SUBSCRIPTION_CONFIRMATION: Подтверждение для TelegramID=%d, серверов: %d", user.TelegramID, len(selectedServers))

	// Получаем информацию о выбранных серверах
	servers, err := common.GetServersByIDs(selectedServers)
	if err != nil {
		log.Printf("SEND_MULTI_SUBSCRIPTION_CONFIRMATION: Ошибка получения серверов: %v", err)
		SendError(bot, chatID, "Ошибка загрузки информации о серверах.")
		return
	}

	// Создаем клавиатуру подтверждения
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Создать мультиподписку", "multi_create_subscription"),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить выбор", "multi_edit_selection"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "multi_cancel_selection"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "back_to_main"),
		),
	)

	text := "🎯 <b>Подтверждение выбора серверов</b>\n\n"
	text += "Вы выбрали следующие серверы для мультиподписки:\n\n"

	for i, server := range servers {
		text += fmt.Sprintf("%d. %s %s (%s)\n", i+1, server.Flag, server.Name, server.Country)
	}

	text += "\n💰 <b>Стоимость:</b> " + fmt.Sprintf("%.2f₽ в день", float64(common.PRICE_PER_DAY))
	text += "\n⏰ <b>Срок действия:</b> Без ограничений (при наличии баланса)"

	text += "\n\n🚀 <b>Преимущества мультиподписки:</b>"
	text += "\n• Автоматическое переключение между серверами"
	text += "\n• Лучшая производительность и стабильность"
	text += "\n• Возможность обхода блокировок"

	text += "\n\n❓ <b>Подтвердите создание мультиподписки</b>"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("SEND_MULTI_SUBSCRIPTION_CONFIRMATION: Ошибка отправки сообщения: %v", err)
	}
}

// SendMultiSubscriptionSuccess отправляет сообщение об успешном создании мультиподписки
func SendMultiSubscriptionSuccess(bot *tgbotapi.BotAPI, chatID int64, user *common.User, subscription *common.MultiSubscription) {
	log.Printf("SEND_MULTI_SUBSCRIPTION_SUCCESS: Успешное создание мультиподписки для TelegramID=%d", user.TelegramID)

	// Создаем клавиатуру с ссылкой на мультиподписку
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("📱 Подключить мультиподписку", common.GetRedirectURL()+subscription.SubscriptionURL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔐 Мои подписки", "my_subscriptions"),
			tgbotapi.NewInlineKeyboardButtonData("💰 Пополнить", "topup"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "back_to_main"),
		),
	)

	text := "🎉 <b>Мультиподписка успешно создана!</b>\n\n"
	text += "✅ Ваша мультиподписка готова к использованию\n\n"

	text += "🌍 <b>Включенные серверы:</b>\n"
	for i, server := range subscription.Servers {
		text += fmt.Sprintf("%d. %s %s (%s)\n", i+1, server.Flag, server.Name, server.Country)
	}

	text += "\n📱 <b>Как подключиться:</b>\n"
	text += "1. Нажмите кнопку \"Подключить мультиподписку\"\n"
	text += "2. Выберите ваше приложение (Happ или v2raytun)\n"
	text += "3. Подписка автоматически импортируется\n\n"

	text += "💡 <b>Совет:</b> Приложение автоматически выберет лучший сервер для вашего местоположения"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("SEND_MULTI_SUBSCRIPTION_SUCCESS: Ошибка отправки сообщения: %v", err)
	}
}

// SendError отправляет сообщение об ошибке
func SendError(bot *tgbotapi.BotAPI, chatID int64, message string) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Попробовать снова", "multi_subscription"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "back_to_main"),
		),
	)

	text := "❌ <b>Ошибка</b>\n\n" + message

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("SEND_ERROR: Ошибка отправки сообщения об ошибке: %v", err)
	}
}
