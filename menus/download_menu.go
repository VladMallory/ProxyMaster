package menus

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// EditDownloadApp показывает меню выбора устройства
func EditDownloadApp(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	log.Printf("EDIT_DOWNLOAD_APP: Показ меню выбора устройства для ChatID=%d, MessageID=%d", chatID, messageID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🍎 iOS", "device_ios"),
			tgbotapi.NewInlineKeyboardButtonData("🤖 Android", "device_android"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
		),
	)

	text := "📱 Скачать приложение\n\n" +
		"Какой у вас устройство?"

	log.Printf("EDIT_DOWNLOAD_APP: Текст для выбора устройства ChatID=%d: %s", chatID, text)
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("EDIT_DOWNLOAD_APP: Ошибка редактирования сообщения для ChatID=%d, MessageID=%d: %v", chatID, messageID, err)
	}
}

// EditIOSLinks показывает ссылки для iOS
func EditIOSLinks(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	log.Printf("EDIT_IOS_LINKS: Показ ссылок для iOS для ChatID=%d, MessageID=%d", chatID, messageID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🇷🇺 App Store (Россия)", "https://apps.apple.com/ru/app/happ-proxy-utility-plus/id6746188973"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🌍 App Store (Другие регионы)", "https://apps.apple.com/us/app/happ-proxy-utility/id6504287215"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "download_app"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
		),
	)

	text := "🍎 iOS\n\n" +
		"Выберите подходящую ссылку для вашего региона:\n\n" +
		"🇷🇺 **App Store (Россия)**\n" +
		"Для пользователей из России\n\n" +
		"🌍 **App Store (Другие регионы)**\n" +
		"Для пользователей из других стран"

	log.Printf("EDIT_IOS_LINKS: Текст для iOS ссылок ChatID=%d: %s", chatID, text)
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ParseMode = "Markdown"
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("EDIT_IOS_LINKS: Ошибка редактирования сообщения для ChatID=%d, MessageID=%d: %v", chatID, messageID, err)
	}
}

// EditAndroidLinks показывает ссылки для Android
func EditAndroidLinks(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	log.Printf("EDIT_ANDROID_LINKS: Показ ссылок для Android для ChatID=%d, MessageID=%d", chatID, messageID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🤖 Google Play", "https://play.google.com/store/apps/details?id=com.happproxy"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "download_app"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
		),
	)

	text := "🤖 Android\n\n" +
		"Скачайте приложение из Google Play Store:\n\n"

	log.Printf("EDIT_ANDROID_LINKS: Текст для Android ссылок ChatID=%d: %s", chatID, text)
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ParseMode = "Markdown"
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("EDIT_ANDROID_LINKS: Ошибка редактирования сообщения для ChatID=%d, MessageID=%d: %v", chatID, messageID, err)
	}
}
