package menus

import (
	"fmt"
	"log"

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

		if common.TARIFF_MODE_ENABLED {
			// Режим тарифов - показываем кнопку "Продлить"
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("📱 Скачать приложение", "download_app"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL(fmt.Sprintf("📱 Подключить (%s)", common.GetAppName()), redirectURL),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💳 Продлить", "extend"),
					tgbotapi.NewInlineKeyboardButtonData("💰 Пополнить", "topup"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🔐 Конфиг", "vpn"),
					tgbotapi.NewInlineKeyboardButtonData("🎯 Рефералы", "ref"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
				),
			)
		} else {
			// Режим автосписания - без кнопки "Продлить"
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("📱 Скачать приложение", "download_app"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL(fmt.Sprintf("📱 Подключить (%s)", common.GetAppName()), redirectURL),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💰 Пополнить", "topup"),
					tgbotapi.NewInlineKeyboardButtonData("🔐 Конфиг", "vpn"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🎯 Рефералы", "ref"),
					tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
				),
			)
		}
	} else {
		// Проверяем, может ли пользователь использовать пробный период
		if common.TrialManager.CanUseTrial(user) {
			if common.TARIFF_MODE_ENABLED {
				// Режим тарифов - показываем кнопку "Продлить"
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
				// Режим автосписания - без кнопки "Продлить"
				keyboard = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("📱 Скачать приложение", "download_app"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("🎁 Активировать пробный период", "activate_trial"),
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
		} else {
			if common.TARIFF_MODE_ENABLED {
				// Режим тарифов - показываем кнопку "Продлить"
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
			} else {
				// Режим автосписания - без кнопки "Продлить"
				keyboard = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("💰 Пополнить", "topup"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
					),
				)
			}
		}
	}

	text := ""
	text = fmt.Sprintf("🌟 Добро пожаловать, %s!\n\n", user.FirstName)
	text += "<blockquote>"
	text += fmt.Sprintf("—💰 Ваш баланс: %.2f₽\n", user.Balance)
	text += fmt.Sprintf("—💳 Стоимость подписки: %d₽ в день(%d₽ в месяц)\n", common.PRICE_PER_DAY, common.PRICE_PER_DAY*30)

	if common.IsConfigActive(user) {
		expiryDate := common.FormatRussianDateFromUnix(user.ExpiryTime)
		text += fmt.Sprintf("—✅ Подписка активна до %s\n", expiryDate)
		text += "</blockquote>\n\n"

		text += "🚀 Если вам не понятно как подключиться, обратитесь в поддержку, мы отправим инструкцию и поможем\n\n"
		text += "1️⃣ Скачайте приложение по кнопке <u>Скачать приложение</u>. Выберите ваш телефон, <u>iOS</u> или <u>Android</u>\n"
		text += fmt.Sprintf("2️⃣ После установки нажмите <u>Подключить (%s)</u>, он импортирует подписку в %s", common.GetAppName(), common.GetAppName())
	} else {
		text += "</blockquote>\n\n"

		if common.TrialManager.CanUseTrial(user) {
			text += "🎁 У вас есть возможность попробовать наш сервис бесплатно!\n"
			text += fmt.Sprintf("На ваш баланс будет добавлено %d₽ для ознакомления с сервисом.\n", common.TRIAL_BALANCE_AMOUNT)
			text += "✨ Нажмите кнопку ниже, чтобы активировать пробный период."
		} else {
			if common.TARIFF_MODE_ENABLED {
				text += "🔐 У вас нет активного конфига для подключения\n"
				text += "💡 Выберите подходящий тариф и начните пользоваться безопасным интернетом!"
			} else {
				text += "🔐 У вас нет активного конфига для подключения\n"
				if user.Balance > float64(common.PRICE_PER_DAY) {
					text += "💡 Конфиг будет создан автоматически при наличии достаточного баланса!"
				} else {
					text += "💡 Пополните баланс для получения доступа к подписке!"
				}
			}
		}
	}

	log.Printf("SEND_MAIN_MENU: Текст меню для TelegramID=%d: %s", user.TelegramID, text)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
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

		if common.TARIFF_MODE_ENABLED {
			// Режим тарифов - показываем кнопку "Продлить"
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("📱 Скачать приложение", "download_app"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL(fmt.Sprintf("📱 Подключить (%s)", common.GetAppName()), redirectURL),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💳 Продлить", "extend"),
					tgbotapi.NewInlineKeyboardButtonData("💰 Пополнить", "topup"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🔐 Конфиг", "vpn"),
					tgbotapi.NewInlineKeyboardButtonData("🎯 Рефералы", "ref"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
				),
			)
		} else {
			// Режим автосписания - без кнопки "Продлить"
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("📱 Скачать приложение", "download_app"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL(fmt.Sprintf("📱 Подключить (%s)", common.GetAppName()), redirectURL),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💰 Пополнить", "topup"),
					tgbotapi.NewInlineKeyboardButtonData("🔐 Конфиг", "vpn"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🎯 Рефералы", "ref"),
					tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
				),
			)
		}
	} else {
		// Проверяем, может ли пользователь использовать пробный период
		if common.TrialManager.CanUseTrial(user) {
			if common.TARIFF_MODE_ENABLED {
				// Режим тарифов - показываем кнопку "Продлить"
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
				// Режим автосписания - без кнопки "Продлить"
				keyboard = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("📱 Скачать приложение", "download_app"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("🎁 Активировать пробный период", "activate_trial"),
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
		} else {
			if common.TARIFF_MODE_ENABLED {
				// Режим тарифов - показываем кнопку "Продлить"
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
			} else {
				// Режим автосписания - без кнопки "Продлить"
				keyboard = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("📱 Скачать приложение", "download_app"),
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
	}

	text := ""
	text = fmt.Sprintf("🌟 Добро пожаловать, %s!\n\n", user.FirstName)
	text += "<blockquote>"
	text += fmt.Sprintf("——💰 Ваш баланс: %.2f₽\n", user.Balance)
	text += fmt.Sprintf("——💳 Стоимость подписки: %d₽ в день(%d₽ в месяц)\n", common.PRICE_PER_DAY, common.PRICE_PER_DAY*30)

	if common.IsConfigActive(user) {
		expiryDate := common.FormatRussianDateFromUnix(user.ExpiryTime)
		text += fmt.Sprintf("——✅ Подписка активна до %s\n", expiryDate)
		text += "</blockquote>\n\n"

		text += "🚀 Если вам не понятно как подключиться, обратитесь в поддержку, мы отправим инструкцию и поможем\n\n"
		text += "1️⃣ Скачайте приложение по кнопке <u>Скачать приложение</u>. Выберите ваш телефон, <u>iOS</u> или <u>Android</u>\n"
		text += fmt.Sprintf("2️⃣ После установки нажмите <u>Подключить (%s)</u>, он импортирует подписку в %s", common.GetAppName(), common.GetAppName())
	} else {
		text += "</blockquote>\n\n"

		if common.TrialManager.CanUseTrial(user) {
			text += "🎁 У вас есть возможность попробовать наш сервис бесплатно!\n"
			text += fmt.Sprintf("На ваш баланс будет добавлено %d₽ для ознакомления с сервисом.\n", common.TRIAL_BALANCE_AMOUNT)
			text += "✨ Нажмите кнопку ниже, чтобы активировать пробный период."
		} else {
			if common.TARIFF_MODE_ENABLED {
				text += "🔐 У вас нет активного конфига для подключения\n"
				text += "💡 Выберите подходящий тариф и начните пользоваться безопасным интернетом!"
			} else {
				text += "🔐 У вас нет активного конфига для подключения\n"
				if user.Balance > float64(common.PRICE_PER_DAY) {
					text += "💡 Конфиг будет создан автоматически при наличии достаточного баланса!"
				} else {
					text += "💡 Пополните баланс для получения доступа к подписке!"
				}
			}
		}
	}
	log.Printf("EDIT_MAIN_MENU: Текст меню для TelegramID=%d: %s", user.TelegramID, text)

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ParseMode = "HTML"
	editMsg.ReplyMarkup = &keyboard
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("EDIT_MAIN_MENU: Ошибка редактирования сообщения для TelegramID=%d, MessageID=%d: %v", user.TelegramID, messageID, err)
	}
}
