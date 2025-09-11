package menus

import (
	"fmt"
	"log"
	"time"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// EditVPN обрабатывает VPN меню
func EditVPN(bot *tgbotapi.BotAPI, chatID int64, messageID int, user *common.User) {
	log.Printf("EDIT_VPN: Начало обработки VPN для TelegramID=%d, MessageID=%d, HasActiveConfig=%v", user.TelegramID, messageID, user.HasActiveConfig)

	if common.IsConfigActive(user) {
		log.Printf("EDIT_VPN: Конфиг активен для TelegramID=%d, ExpiryTime=%s", user.TelegramID, time.UnixMilli(user.ExpiryTime).Format("02.01.2006 15:04"))

		subscriptionURL := common.CONFIG_BASE_URL + user.SubID
		redirectURL := common.GetRedirectURL() + subscriptionURL

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL(fmt.Sprintf("📱 Подключить (%s)", common.GetAppName()), redirectURL)),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Продлить", "extend"),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
			),
		)

		expiryDate := time.UnixMilli(user.ExpiryTime).Format("02.01.2006 15:04")

		// Получаем информацию о лимитах трафика
		trafficInfo := common.GetTrafficConfigDescription()

		text := fmt.Sprintf("🔐 Ваш конфиг активен!\n\n"+
			"📅 Активен до: %s\n"+
			"📊 Лимиты трафика: %s\n"+
			"🔗 Ссылка на подписку:\n`%s`\n\n"+
			"💡 Нажмите 'Подключить (%s)' для автоматического импорта\n\n"+
			"📱 Приложения для самостоятельного импорта:\n"+
			"• Android: v2rayng, Hiddify, v2box\n"+
			"• iOS: v2raytun, v2Box, Streisand, Hiddify\n"+
			"• Windows, Linux: Nekoray, Hiddify \n"+
			"• macOS: v2raytun, v2Box, Streisand, Hiddify\n"+
			"• Роутеры: xkeen (Keenetic), OpenWrt\n"+
			"• ТВ: v2raytun, Happ\n\n"+
			"Если у вас возникли вопросы, вы можете обратиться за помощью к нашей поддержке.",
			expiryDate, trafficInfo, subscriptionURL, common.GetAppName())

		log.Printf("EDIT_VPN: Текст для активного конфига для TelegramID=%d: %s", user.TelegramID, text)
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
		editMsg.ParseMode = "Markdown"
		editMsg.ReplyMarkup = &keyboard
		if _, err := bot.Send(editMsg); err != nil {
			log.Printf("EDIT_VPN: Ошибка редактирования сообщения для активного конфига TelegramID=%d, MessageID=%d: %v", user.TelegramID, messageID, err)
		}
	} else {
		log.Printf("EDIT_VPN: Конфиг неактивен для TelegramID=%d, переход к выбору периода", user.TelegramID)
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("1 день (10₽)", "days:1"),
				tgbotapi.NewInlineKeyboardButtonData("3 дня (30₽)", "days:3"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("7 дней (70₽)", "days:7"),
				tgbotapi.NewInlineKeyboardButtonData("30 дней (300₽)", "days:30"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
			),
		)

		text := fmt.Sprintf("🔐 Создание нового VPN конфига\n\n"+
			"💰 Ваш баланс: %.2f₽\n\n"+
			"Выберите период для создания конфига:", user.Balance)

		log.Printf("EDIT_VPN: Текст для неактивного конфига для TelegramID=%d: %s", user.TelegramID, text)
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
		editMsg.ReplyMarkup = &keyboard
		if _, err := bot.Send(editMsg); err != nil {
			log.Printf("EDIT_VPN: Ошибка редактирования сообщения для неактивного конфига TelegramID=%d, MessageID=%d: %v", user.TelegramID, messageID, err)
		}
	}
}

// EditExtend обрабатывает меню продления
func EditExtend(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	log.Printf("EDIT_EXTEND: Начало обработки продления для ChatID=%d, MessageID=%d", chatID, messageID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1 день (10₽)", "days:1"),
			tgbotapi.NewInlineKeyboardButtonData("3 дня (30₽)", "days:3"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("7 дней (70₽)", "days:7"),
			tgbotapi.NewInlineKeyboardButtonData("30 дней (300₽)", "days:30"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
		),
	)

	text := "🔄 Продление конфига\n\n" +
		"Выберите период для продления:"

	log.Printf("EDIT_EXTEND: Текст для продления ChatID=%d: %s", chatID, text)
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("EDIT_EXTEND: Ошибка редактирования сообщения для ChatID=%d, MessageID=%d: %v", chatID, messageID, err)
	}
}

// EditPayment обрабатывает меню оплаты
func EditPayment(bot *tgbotapi.BotAPI, chatID int64, messageID int, days int) {
	log.Printf("EDIT_PAYMENT: Начало обработки оплаты для ChatID=%d, MessageID=%d, days=%d", chatID, messageID, days)

	cost := float64(days * common.PRICE_PER_DAY)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Оплатить", fmt.Sprintf("pay:%d", days)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "vpn"),
		),
	)

	// Рассчитываем лимит трафика для указанного количества дней
	trafficLimit := common.CalculateTrafficLimit(days)
	trafficInfo := common.FormatTrafficLimit(trafficLimit)

	text := fmt.Sprintf("💳 Подтверждение оплаты\n\n"+
		"📅 Период: %d %s\n"+
		"💰 Стоимость: %.0f₽\n"+
		"📊 Лимит трафика: %s\n\n"+
		"Подтвердите оплату:", days, common.GetDaysWord(days), cost, trafficInfo)

	log.Printf("EDIT_PAYMENT: Текст для оплаты ChatID=%d: %s", chatID, text)
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("EDIT_PAYMENT: Ошибка редактирования сообщения для ChatID=%d, MessageID=%d: %v", chatID, messageID, err)
	}
}
