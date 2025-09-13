package handlers

import (
	"fmt"
	"log"
	"strings"

	"bot/common"
	"bot/menus"
	"bot/payments/promo"
	"bot/referralLink"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleMessage обрабатывает входящие сообщения
func HandleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("HANDLE_MESSAGE: Обработка сообщения от TelegramID=%d, команда='%s'", message.From.ID, message.Command())

	telegramUser := message.From

	// Получаем или создаем пользователя
	username := ""
	if telegramUser.UserName != "" {
		username = telegramUser.UserName
	}

	firstName := telegramUser.FirstName
	lastName := ""
	if telegramUser.LastName != "" {
		lastName = telegramUser.LastName
	}

	user, err := common.GetOrCreateUser(telegramUser.ID, username, firstName, lastName)
	if err != nil {
		log.Printf("HANDLE_MESSAGE: Ошибка работы с пользователем TelegramID=%d: %v", telegramUser.ID, err)
		return
	}
	log.Printf("HANDLE_MESSAGE: Пользователь получен/создан: TelegramID=%d, Username=%s, FirstName=%s, LastName=%s", user.TelegramID, user.Username, user.FirstName, user.LastName)

	// Проверяем реферальную систему для команды /start
	var isReferralUser bool
	if message.IsCommand() && message.Command() == "start" && referralLink.GlobalReferralManager != nil {
		log.Printf("HANDLE_MESSAGE: Проверка реферальной системы для команды /start, текст: '%s'", message.Text)

		// Проверяем, является ли это реферальным стартом
		isReferralStart := referralLink.GlobalReferralManager.IsReferralStart(message.Text)
		log.Printf("HANDLE_MESSAGE: IsReferralStart('%s') = %v", message.Text, isReferralStart)

		if isReferralStart {
			// Извлекаем реферальный код
			referralCode := referralLink.GlobalReferralManager.ExtractReferralCode(message.Text)
			log.Printf("HANDLE_MESSAGE: Извлечен реферальный код: '%s'", referralCode)

			if referralCode != "" {
				// Убираем префикс "ref_" из кода перед сохранением
				cleanCode := strings.TrimPrefix(referralCode, "ref_")
				user.ReferralCode = cleanCode
				isReferralUser = true
				log.Printf("HANDLE_MESSAGE: Сохранен реферальный код %s (очищенный от %s) для пользователя %d", cleanCode, referralCode, user.TelegramID)

				// Обрабатываем реферальный переход
				log.Printf("HANDLE_MESSAGE: Вызов HandleStartCommand для обработки реферального кода")
				referralLink.GlobalReferralManager.HandleStartCommand(message.Chat.ID, user, message.Text)

				// Всегда отправляем реферальное сообщение для реферальных пользователей
				referralMessage := "🎉 <b>Реферальная ссылка активирована!</b>\n\n"
				referralMessage += "💰 <b>Вам зачислены деньги на баланс!</b>\n"
				referralMessage += "🎁 <b>Приветственный бонус:</b> " + fmt.Sprintf("%.0f", common.REFERRAL_WELCOME_BONUS) + "₽\n\n"
				referralMessage += "Спасибо, что присоединились к нашему сервису!\n"
				referralMessage += "Используйте кнопки ниже для управления аккаунтом."

				// Создаем клавиатуру для реферального пользователя
				keyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("💰 Баланс", "balance"),
						tgbotapi.NewInlineKeyboardButtonData("🔧 VPN", "vpn"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("💳 Пополнить", "topup"),
						tgbotapi.NewInlineKeyboardButtonData("🎯 Рефералы", "ref"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("📱 Скачать приложение", "download_app"),
					),
				)

				msg := tgbotapi.NewMessage(message.Chat.ID, referralMessage)
				msg.ParseMode = "HTML"
				msg.ReplyMarkup = &keyboard

				if _, err := bot.Send(msg); err != nil {
					log.Printf("HANDLE_MESSAGE: Ошибка отправки реферального сообщения: %v", err)
				} else {
					log.Printf("HANDLE_MESSAGE: ✅ Реферальное сообщение отправлено пользователю %d", user.TelegramID)
				}
				return
			} else {
				log.Printf("HANDLE_MESSAGE: Реферальный код пустой, пропускаем обработку")
			}
		} else {
			log.Printf("HANDLE_MESSAGE: Команда /start без реферального кода")
		}
	} else if message.IsCommand() && message.Command() == "start" {
		log.Printf("HANDLE_MESSAGE: GlobalReferralManager не инициализирован, реферальная система недоступна")
	}

	// Проверяем, является ли это первым сообщением от пользователя (команда /start)
	// и предлагаем пробный период, если пользователь новый (НО НЕ реферальный)
	if message.IsCommand() && message.Command() == "start" && !user.HasActiveConfig && common.TrialManager.CanUseTrial(user) && !isReferralUser {
		log.Printf("HANDLE_MESSAGE: Предложение пробного периода новому пользователю TelegramID=%d", telegramUser.ID)
		common.TrialManager.HandleTrialPeriod(bot, user, message.Chat.ID)
		return
	}

	if message.IsCommand() {
		// Проверяем, является ли это команда промокодов
		log.Printf("HANDLE_MESSAGE: Проверка команды: %s, GlobalPromoManager: %v", message.Command(), promo.GlobalPromoManager != nil)
		if promo.GlobalPromoManager != nil {
			log.Printf("HANDLE_MESSAGE: IsPromoCommand(%s): %v", message.Command(), promo.GlobalPromoManager.IsPromoCommand(message.Command()))
			if promo.GlobalPromoManager.IsPromoCommand(message.Command()) {
				log.Printf("HANDLE_MESSAGE: Обработка команды промокодов: %s от пользователя %d", message.Command(), message.From.ID)
				args := strings.Fields(message.Text)[1:] // Убираем команду из аргументов
				err := promo.GlobalPromoManager.HandleCommand(message.Chat.ID, message.From.ID, message.Command(), args)
				if err != nil {
					log.Printf("HANDLE_MESSAGE: Ошибка обработки команды промокодов %s: %v", message.Command(), err)
				} else {
					log.Printf("HANDLE_MESSAGE: Команда промокодов %s успешно обработана", message.Command())
				}
				return
			}
		} else {
			log.Printf("HANDLE_MESSAGE: GlobalPromoManager is nil!")
		}

		handleCommand(bot, message, user)
	}
}

// handleCommand обрабатывает команды
func handleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, user *common.User) {
	telegramUser := message.From

	switch message.Command() {
	case "start":
		log.Printf("HANDLE_MESSAGE: Выполнение команды /start для TelegramID=%d", telegramUser.ID)
		menus.SendMainMenu(bot, message.Chat.ID, user)
	case "balance":
		log.Printf("HANDLE_MESSAGE: Выполнение команды /balance для TelegramID=%d", telegramUser.ID)
		menus.SendBalance(bot, message.Chat.ID, user)
	case "debug":
		handleDebugCommand(bot, message, user)
	case "backup":
		handleBackupCommand(bot, message)
	case "traffic":
		handleTrafficCommand(bot, message)
	case "trial":
		handleTrialCommand(bot, message)
	case "reset_trial":
		handleResetTrialCommand(bot, message)
	case "users":
		HandleUsersCommand(bot, message)
	case "users10", "users50", "users100", "users200", "users400", "users500", "users1000", "users5000":
		HandleUsersLimitCommand(bot, message)
	case "clear_users":
		handleClearUsersCommand(bot, message)
	case "confirm_clear_users":
		handleConfirmClearUsersCommand(bot, message)
	case "clear_database":
		handleClearDatabaseCommand(bot, message)
	case "confirm_clear_database":
		handleConfirmClearDatabaseCommand(bot, message)
	case "reset_ip_counters":
		handleResetIPCountersCommand(bot, message)
	case "switch_tariff":
		handleSwitchTariffCommand(bot, message)
	case "switch_auto":
		handleSwitchAutoCommand(bot, message)
	case "billing_status":
		handleBillingStatusCommand(bot, message)
	case "ref":
		handleRefCommand(bot, message, user)
	}
}

// handleDebugCommand обрабатывает команду /debug
func handleDebugCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, user *common.User) {
	log.Printf("HANDLE_MESSAGE: Выполнение команды /debug для TelegramID=%d", message.From.ID)

	if user.HasActiveConfig && user.SubID != "" {
		debugText := fmt.Sprintf("🔧 Debug Info:\n\n"+
			"SubID: %s\n"+
			"ClientID: %s\n"+
			"Email: %s\n"+
			"Subscription URL: %s%s\n"+
			"JSON URL (old): %s%s",
			user.SubID, user.ClientID, user.Email,
			common.CONFIG_BASE_URL, user.SubID,
			common.CONFIG_JSON_URL, user.SubID)
		msg := tgbotapi.NewMessage(message.Chat.ID, debugText)
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки debug-сообщения для TelegramID=%d: %v", message.From.ID, err)
		}
	} else {
		log.Printf("HANDLE_MESSAGE: Пользователь TelegramID=%d не имеет активного конфига или SubID", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🔧 Нет активного конфига для отладки")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения для TelegramID=%d: %v", message.From.ID, err)
		}
	}
}

// handleBackupCommand обрабатывает команду /backup
func handleBackupCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("HANDLE_MESSAGE: Выполнение команды /backup для TelegramID=%d", message.From.ID)

	if message.From.ID == common.ADMIN_ID {
		log.Printf("HANDLE_MESSAGE: Вызов BackupMongoDB для TelegramID=%d", message.From.ID)
		if err := common.BackupMongoDB(); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка создания бэкапа для TelegramID=%d: %v", message.From.ID, err)
			msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка создания бэкапа: %v", err))
			if _, err := bot.Send(msg); err != nil {
				log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения об ошибке бэкапа для TelegramID=%d: %v", message.From.ID, err)
			}
		} else {
			log.Printf("HANDLE_MESSAGE: Бэкап успешно создан для TelegramID=%d", message.From.ID)
			msg := tgbotapi.NewMessage(message.Chat.ID, "✅ Бэкап успешно создан")
			if _, err := bot.Send(msg); err != nil {
				log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения об успехе бэкапа для TelegramID=%d: %v", message.From.ID, err)
			}
		}
	} else {
		log.Printf("HANDLE_MESSAGE: Пользователь TelegramID=%d не является админом для команды /backup", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚫 Доступ запрещён")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения о запрете для TelegramID=%d: %v", message.From.ID, err)
		}
	}
}

// handleTrafficCommand обрабатывает команду /traffic
func handleTrafficCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("HANDLE_MESSAGE: Выполнение команды /traffic для TelegramID=%d", message.From.ID)

	if message.From.ID == common.ADMIN_ID {
		common.ShowTrafficConfig(bot, message.Chat.ID)
	} else {
		log.Printf("HANDLE_MESSAGE: Пользователь TelegramID=%d не является админом для команды /traffic", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚫 Доступ запрещён")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения о запрете для TelegramID=%d: %v", message.From.ID, err)
		}
	}
}

// handleTrialCommand обрабатывает команду /trial
func handleTrialCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("HANDLE_MESSAGE: Выполнение команды /trial для TelegramID=%d", message.From.ID)

	if message.From.ID == common.ADMIN_ID {
		text := common.TrialManager.GetTrialPeriodInfo()
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки информации о пробном периоде для TelegramID=%d: %v", message.From.ID, err)
		}
	} else {
		log.Printf("HANDLE_MESSAGE: Пользователь TelegramID=%d не является админом для команды /trial", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚫 Доступ запрещён")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения о запрете для TelegramID=%d: %v", message.From.ID, err)
		}
	}
}

// handleResetTrialCommand обрабатывает команду /reset_trial
func handleResetTrialCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("HANDLE_MESSAGE: Выполнение команды /reset_trial для TelegramID=%d", message.From.ID)

	if message.From.ID == common.ADMIN_ID {
		if err := common.ResetAllTrialFlags(); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка сброса пробных периодов для TelegramID=%d: %v", message.From.ID, err)
			msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка сброса пробных периодов: %v", err))
			if _, err := bot.Send(msg); err != nil {
				log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения об ошибке для TelegramID=%d: %v", message.From.ID, err)
			}
		} else {
			log.Printf("HANDLE_MESSAGE: Пробные периоды успешно сброшены для TelegramID=%d", message.From.ID)
			msg := tgbotapi.NewMessage(message.Chat.ID, "✅ Пробные периоды успешно сброшены для всех пользователей!")
			if _, err := bot.Send(msg); err != nil {
				log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения об успехе для TelegramID=%d: %v", message.From.ID, err)
			}
		}
	} else {
		log.Printf("HANDLE_MESSAGE: Пользователь TelegramID=%d не является админом для команды /reset_trial", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚫 Доступ запрещён")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения о запрете для TelegramID=%d: %v", message.From.ID, err)
		}
	}
}

// handleClearUsersCommand обрабатывает команду /clear_users
func handleClearUsersCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("HANDLE_MESSAGE: Выполнение команды /clear_users для TelegramID=%d", message.From.ID)

	if message.From.ID == common.ADMIN_ID {
		msg := tgbotapi.NewMessage(message.Chat.ID, "⚠️ ВНИМАНИЕ! Это удалит ВСЕХ пользователей из базы данных!\n\n"+
			"Для подтверждения отправьте: /confirm_clear_users")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки предупреждения для TelegramID=%d: %v", message.From.ID, err)
		}
	} else {
		log.Printf("HANDLE_MESSAGE: Пользователь TelegramID=%d не является админом для команды /clear_users", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚫 Доступ запрещён")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения о запрете для TelegramID=%d: %v", message.From.ID, err)
		}
	}
}

// handleConfirmClearUsersCommand обрабатывает команду /confirm_clear_users
func handleConfirmClearUsersCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("HANDLE_MESSAGE: Выполнение команды /confirm_clear_users для TelegramID=%d", message.From.ID)

	if message.From.ID == common.ADMIN_ID {
		if err := common.ClearAllUsers(); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка очистки пользователей для TelegramID=%d: %v", message.From.ID, err)
			msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка очистки пользователей: %v", err))
			if _, err := bot.Send(msg); err != nil {
				log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения об ошибке для TelegramID=%d: %v", message.From.ID, err)
			}
		} else {
			log.Printf("HANDLE_MESSAGE: Пользователи успешно очищены для TelegramID=%d", message.From.ID)
			msg := tgbotapi.NewMessage(message.Chat.ID, "✅ Все пользователи успешно удалены из базы данных!\n\n"+
				"Теперь все новые пользователи смогут получить пробный период.")
			if _, err := bot.Send(msg); err != nil {
				log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения об успехе для TelegramID=%d: %v", message.From.ID, err)
			}
		}
	} else {
		log.Printf("HANDLE_MESSAGE: Пользователь TelegramID=%d не является админом для команды /confirm_clear_users", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚫 Доступ запрещён")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения о запрете для TelegramID=%d: %v", message.From.ID, err)
		}
	}
}

// handleClearDatabaseCommand обрабатывает команду /clear_database
func handleClearDatabaseCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("HANDLE_MESSAGE: Выполнение команды /clear_database для TelegramID=%d", message.From.ID)

	if message.From.ID == common.ADMIN_ID {
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚨 КРИТИЧЕСКОЕ ВНИМАНИЕ! Это удалит ВСЕ данные из базы данных!\n\n"+
			"Для подтверждения отправьте: /confirm_clear_database")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки предупреждения для TelegramID=%d: %v", message.From.ID, err)
		}
	} else {
		log.Printf("HANDLE_MESSAGE: Пользователь TelegramID=%d не является админом для команды /clear_database", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚫 Доступ запрещён")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения о запрете для TelegramID=%d: %v", message.From.ID, err)
		}
	}
}

// handleConfirmClearDatabaseCommand обрабатывает команду /confirm_clear_database
func handleConfirmClearDatabaseCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("HANDLE_MESSAGE: Выполнение команды /confirm_clear_database для TelegramID=%d", message.From.ID)

	if message.From.ID == common.ADMIN_ID {
		if err := common.ClearDatabase(); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка очистки базы данных для TelegramID=%d: %v", message.From.ID, err)
			msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка очистки базы данных: %v", err))
			if _, err := bot.Send(msg); err != nil {
				log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения об ошибке для TelegramID=%d: %v", message.From.ID, err)
			}
		} else {
			log.Printf("HANDLE_MESSAGE: База данных успешно очищена для TelegramID=%d", message.From.ID)
			msg := tgbotapi.NewMessage(message.Chat.ID, "✅ База данных полностью очищена!\n\n"+
				"Все данные удалены. Бот готов к новому запуску.")
			if _, err := bot.Send(msg); err != nil {
				log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения об успехе для TelegramID=%d: %v", message.From.ID, err)
			}
		}
	} else {
		log.Printf("HANDLE_MESSAGE: Пользователь TelegramID=%d не является админом для команды /confirm_clear_database", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚫 Доступ запрещён")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения о запрете для TelegramID=%d: %v", message.From.ID, err)
		}
	}
}

// handleResetIPCountersCommand обрабатывает команду /reset_ip_counters
func handleResetIPCountersCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("HANDLE_MESSAGE: Выполнение команды /reset_ip_counters для TelegramID=%d", message.From.ID)

	if message.From.ID == common.ADMIN_ID {
		// Создаем новый анализатор и сбрасываем счетчики
		analyzer := common.NewLogAnalyzer(common.ACCESS_LOG_PATH)
		analyzer.ResetStats()

		log.Printf("HANDLE_MESSAGE: Счетчики IP адресов сброшены для TelegramID=%d", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🔄 Все счетчики IP адресов сброшены!\n\n"+
			"Система мониторинга начнет анализ заново.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения об успехе для TelegramID=%d: %v", message.From.ID, err)
		}
	} else {
		log.Printf("HANDLE_MESSAGE: Пользователь TelegramID=%d не является админом для команды /reset_ip_counters", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚫 Доступ запрещён")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения о запрете для TelegramID=%d: %v", message.From.ID, err)
		}
	}
}

// handleSwitchTariffCommand обрабатывает команду /switch_tariff
func handleSwitchTariffCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("HANDLE_MESSAGE: Выполнение команды /switch_tariff для TelegramID=%d", message.From.ID)

	if message.From.ID == common.ADMIN_ID {
		common.SwitchToTariffMode()

		msg := tgbotapi.NewMessage(message.Chat.ID,
			"✅ Переключение на тарифный режим выполнено!\n\n"+
				"🎯 Теперь активен тарифный режим:\n"+
				"• Пользователи покупают дни вручную\n"+
				"• Автосписание отключено\n"+
				"• Показываются кнопки выбора тарифов\n\n"+
				"Используйте /billing_status для проверки статуса")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения для TelegramID=%d: %v", message.From.ID, err)
		}
	} else {
		log.Printf("HANDLE_MESSAGE: Пользователь TelegramID=%d не является админом для команды /switch_tariff", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚫 Доступ запрещён")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения о запрете для TelegramID=%d: %v", message.From.ID, err)
		}
	}
}

// handleSwitchAutoCommand обрабатывает команду /switch_auto
func handleSwitchAutoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("HANDLE_MESSAGE: Выполнение команды /switch_auto для TelegramID=%d", message.From.ID)

	if message.From.ID == common.ADMIN_ID {
		common.SwitchToAutoBillingMode()

		msg := tgbotapi.NewMessage(message.Chat.ID,
			"✅ Переключение на автосписание выполнено!\n\n"+
				"🤖 Теперь активен режим автосписания:\n"+
				"• Ежедневное списание с баланса\n"+
				"• Автоматический пересчет дней\n"+
				"• Кнопки тарифов скрыты\n\n"+
				"⚠️ Для полного переключения требуется перезапуск бота!\n\n"+
				"Используйте /billing_status для проверки статуса")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения для TelegramID=%d: %v", message.From.ID, err)
		}
	} else {
		log.Printf("HANDLE_MESSAGE: Пользователь TelegramID=%d не является админом для команды /switch_auto", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚫 Доступ запрещён")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения о запрете для TelegramID=%d: %v", message.From.ID, err)
		}
	}
}

// handleBillingStatusCommand обрабатывает команду /billing_status
func handleBillingStatusCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("HANDLE_MESSAGE: Выполнение команды /billing_status для TelegramID=%d", message.From.ID)

	if message.From.ID == common.ADMIN_ID {
		status := common.GetBillingStatus()

		msg := tgbotapi.NewMessage(message.Chat.ID, status)
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки статуса для TelegramID=%d: %v", message.From.ID, err)
		}
	} else {
		log.Printf("HANDLE_MESSAGE: Пользователь TelegramID=%d не является админом для команды /billing_status", message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚫 Доступ запрещён")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("HANDLE_MESSAGE: Ошибка отправки сообщения о запрете для TelegramID=%d: %v", message.From.ID, err)
		}
	}
}

// handleRefCommand обрабатывает команду /ref
func handleRefCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, user *common.User) {
	log.Printf("HANDLE_MESSAGE: Выполнение команды /ref для TelegramID=%d", message.From.ID)

	// Проверяем, включена ли реферальная система
	if !common.REFERRAL_SYSTEM_ENABLED {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Реферальная система временно отключена")
		bot.Send(msg)
		return
	}

	// Используем глобальный менеджер рефералов
	if referralLink.GlobalReferralManager != nil {
		referralLink.GlobalReferralManager.HandleCommand(message.Chat.ID, user, "ref")
	} else {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Реферальная система не инициализирована")
		bot.Send(msg)
	}
}
