package menus

import (
	"fmt"
	"log"
	"time"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SendMainMenu отправляет главное меню
func SendMainMenu(bot *tgbotapi.BotAPI, chatID int64, user *common.User) {
	log.Printf("SEND_MAIN_MENU: Отправка главного меню для TelegramID=%d", user.TelegramID)

	var keyboard tgbotapi.InlineKeyboardMarkup

	if common.IsConfigActive(user) {
		// Используем HTML редирект страницу
		subscriptionURL := common.CONFIG_BASE_URL + user.SubID
		redirectURL := common.GetRedirectURL() + subscriptionURL

		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📱 Скачать приложение", "download_app"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("📱 Подключить (Happ)", redirectURL)),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💳 Продлить", "extend"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💰 Пополнить", "topup"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔐 Конфиг", "vpn"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
			),
		)
	} else {
		// Проверяем, может ли пользователь использовать пробный период
		if common.TrialManager.CanUseTrial(user) {
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("📱 Скачать приложение", "download_app"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🎁 Активировать пробный период", "activate_trial"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💳 Продлить", "extend"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💰 Пополнить", "topup"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🔐 Конфиг", "vpn"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
				),
			)
		} else {
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("📱 Скачать приложение", "download_app"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💳 Продлить", "extend"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💰 Пополнить", "topup"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🔐 Конфиг", "vpn"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
				),
			)
		}
	}

	text := fmt.Sprintf("🌟 Добро пожаловать, %s!\n\n", user.FirstName)

	text += fmt.Sprintf("💰 Ваш баланс: %.2f₽\n", user.Balance)

	if common.IsConfigActive(user) {
		expiryDate := time.UnixMilli(user.ExpiryTime).Format("2006-01-02")
		text += fmt.Sprintf("✅ Подписка активна до %s\n\n", expiryDate)
		text += "🚀 Для того чтобы конфиг начал работать, выполните 2 простых шага:\n\n"
		text += "1️⃣ Сначала скачайте приложение нажав кнопку ниже\n"
		text += "2️⃣ После установки нажмите 'Подключить (Happ)' для настройки"
	} else {
		if common.TrialManager.CanUseTrial(user) {
			text += "🎁 У вас есть возможность попробовать наш сервис бесплатно!\n"
			text += fmt.Sprintf("Пробный период: %d дней\n", common.TRIAL_PERIOD_DAYS)
			text += "✨ Нажмите кнопку ниже, чтобы активировать пробный период."
		} else {
			text += "🔐 У вас нет активного конига для подключения\n"
			text += "💡 Выберите подходящий тариф и начните пользоваться безопасным интернетом!"
		}
	}

	log.Printf("SEND_MAIN_MENU: Текст меню для TelegramID=%d: %s", user.TelegramID, text)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = &keyboard
	if _, err := bot.Send(msg); err != nil {
		log.Printf("SEND_MAIN_MENU: Ошибка отправки сообщения для TelegramID=%d: %v", user.TelegramID, err)
	}
}

// EditMainMenu редактирует главное меню
func EditMainMenu(bot *tgbotapi.BotAPI, chatID int64, messageID int, user *common.User) {
	log.Printf("EDIT_MAIN_MENU: Редактирование главного меню для TelegramID=%d, MessageID=%d", user.TelegramID, messageID)

	var keyboard tgbotapi.InlineKeyboardMarkup

	if common.IsConfigActive(user) {
		// Используем HTML редирект страницу
		subscriptionURL := common.CONFIG_BASE_URL + user.SubID
		redirectURL := common.GetRedirectURL() + subscriptionURL

		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📱 Скачать приложение", "download_app"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("📱 Подключить (Happ)", redirectURL)),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💳 Продлить", "extend"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💰 Пополнить", "topup"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔐 Конфиг", "vpn"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
			),
		)
	} else {
		// Проверяем, может ли пользователь использовать пробный период
		if common.TrialManager.CanUseTrial(user) {
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("📱 Скачать приложение", "download_app"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🎁 Активировать пробный период", "activate_trial"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💳 Продлить", "extend"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💰 Пополнить", "topup"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🔐 Конфиг", "vpn"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
				),
			)
		} else {
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("📱 Скачать приложение", "download_app"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💳 Продлить", "extend"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💰 Пополнить", "topup"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🔐 Конфиг", "vpn"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
				),
			)
		}
	}

	text := fmt.Sprintf("🌟 Добро пожаловать, %s!\n\n", user.FirstName)

	text += fmt.Sprintf("💰 Ваш баланс: %.2f₽\n", user.Balance)

	if common.IsConfigActive(user) {
		expiryDate := time.UnixMilli(user.ExpiryTime).Format("2006-01-02")
		text += fmt.Sprintf("✅ Подписка активна до %s\n\n", expiryDate)
		text += "🚀 Для того чтобы конфиг начал работать, выполните 2 простых шага:\n\n"
		text += "1️⃣ Сначала скачайте приложение нажав кнопку ниже\n"
		text += "2️⃣ После установки нажмите 'Подключить (Happ)' для настройки"
	} else {
		if common.TrialManager.CanUseTrial(user) {
			text += "🎁 У вас есть возможность попробовать наш сервис бесплатно!\n"
			text += fmt.Sprintf("Пробный период: %d дней\n", common.TRIAL_PERIOD_DAYS)
			text += "✨ Нажмите кнопку ниже, чтобы активировать пробный период."
		} else {
			text += "🔐 У вас нет активного конфига для подключения\n"
			text += "💡 Выберите подходящий тариф и начните пользоваться безопасным интернетом!"
		}
	}

	log.Printf("EDIT_MAIN_MENU: Текст меню для TelegramID=%d: %s", user.TelegramID, text)

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("EDIT_MAIN_MENU: Ошибка редактирования сообщения для TelegramID=%d, MessageID=%d: %v", user.TelegramID, messageID, err)
	}
}
