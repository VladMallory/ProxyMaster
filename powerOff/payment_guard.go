package powerOff

import (
	"fmt"
	"log"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// PaymentGuard защищает платежи от создания во время выключения
type PaymentGuard struct {
	shutdownManager *ShutdownManager
}

// NewPaymentGuard создает новый защитник платежей
func NewPaymentGuard(shutdownManager *ShutdownManager) *PaymentGuard {
	return &PaymentGuard{
		shutdownManager: shutdownManager,
	}
}

// CheckPaymentAllowed проверяет, разрешено ли создание платежа
func (pg *PaymentGuard) CheckPaymentAllowed() (bool, string) {
	log.Printf("PAYMENT_GUARD: CheckPaymentAllowed вызвана, shutdownManager = %v", pg.shutdownManager != nil)
	if pg.shutdownManager == nil {
		log.Printf("PAYMENT_GUARD: shutdownManager is nil, разрешаем платеж")
		return true, ""
	}

	isBlocked := pg.shutdownManager.IsPaymentBlocked()
	log.Printf("PAYMENT_GUARD: IsPaymentBlocked = %v", isBlocked)

	if isBlocked {
		status := pg.shutdownManager.GetStatus()
		log.Printf("PAYMENT_GUARD: Платежи заблокированы, состояние: %s", status.State.String())
		message := "⚠️ <b>Платежи временно недоступны</b>\n\n"

		switch status.State {
		case ShutdownStatePreparation:
			message += "Бот готовится к выключению для технических работ.\n"
			message += fmt.Sprintf("Активных платежей: %d\n", status.ActivePayments)
			message += fmt.Sprintf("Время до выключения: %d сек\n\n", status.TimeRemaining)
			message += "Попробуйте позже."
		case ShutdownStateShuttingDown:
			message += "Бот выключается.\n"
			message += "Попробуйте позже."
		}

		return false, message
	}

	log.Printf("PAYMENT_GUARD: Платежи разрешены")
	return true, ""
}

// SendPaymentBlockedMessage отправляет сообщение о блокировке платежей
func (pg *PaymentGuard) SendPaymentBlockedMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	allowed, message := pg.CheckPaymentAllowed()
	if allowed {
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
		),
	)

	var editMsg tgbotapi.EditMessageTextConfig
	if messageID > 0 {
		editMsg = tgbotapi.NewEditMessageText(chatID, messageID, message)
		editMsg.ParseMode = "HTML"
		editMsg.ReplyMarkup = &keyboard
	} else {
		msg := tgbotapi.NewMessage(chatID, message)
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = &keyboard
		bot.Send(msg)
		return
	}

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("PAYMENT_GUARD: Ошибка отправки сообщения о блокировке платежей: %v", err)
	}
}

// WrapPaymentHandler оборачивает обработчик платежей для проверки блокировки
func (pg *PaymentGuard) WrapPaymentHandler(originalHandler func(*tgbotapi.BotAPI, int64, int, *common.User, int)) func(*tgbotapi.BotAPI, int64, int, *common.User, int) {
	return func(bot *tgbotapi.BotAPI, chatID int64, messageID int, user *common.User, amount int) {
		// Проверяем, разрешены ли платежи
		allowed, message := pg.CheckPaymentAllowed()
		if !allowed {
			log.Printf("PAYMENT_GUARD: Платеж заблокирован для пользователя %d", user.TelegramID)

			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
				),
			)

			editMsg := tgbotapi.NewEditMessageText(chatID, messageID, message)
			editMsg.ParseMode = "HTML"
			editMsg.ReplyMarkup = &keyboard

			if _, err := bot.Send(editMsg); err != nil {
				log.Printf("PAYMENT_GUARD: Ошибка отправки сообщения о блокировке: %v", err)
			}
			return
		}

		// Вызываем оригинальный обработчик
		originalHandler(bot, chatID, messageID, user, amount)
	}
}

// GlobalPaymentGuard глобальный экземпляр защитника платежей
var GlobalPaymentGuard *PaymentGuard

// InitializePaymentGuard инициализирует глобальный защитник платежей
func InitializePaymentGuard() {
	log.Printf("PAYMENT_GUARD: Начало инициализации, POWEROFF_SYSTEM_ENABLED = %v", common.POWEROFF_SYSTEM_ENABLED)
	if !common.POWEROFF_SYSTEM_ENABLED {
		log.Printf("PAYMENT_GUARD: Система защиты платежей отключена")
		return
	}

	log.Printf("PAYMENT_GUARD: GlobalShutdownManager = %v", GlobalShutdownManager != nil)
	if GlobalShutdownManager == nil {
		log.Printf("PAYMENT_GUARD: Менеджер выключения не инициализирован")
		return
	}

	GlobalPaymentGuard = NewPaymentGuard(GlobalShutdownManager)
	log.Printf("PAYMENT_GUARD: Защитник платежей инициализирован")
	log.Printf("PAYMENT_GUARD: GlobalPaymentGuard = %v", GlobalPaymentGuard != nil)
}

// CheckPaymentAllowedGlobal глобальная функция проверки разрешения платежей
func CheckPaymentAllowedGlobal() (bool, string) {
	log.Printf("PAYMENT_GUARD: CheckPaymentAllowedGlobal вызвана, GlobalPaymentGuard = %v", GlobalPaymentGuard != nil)
	if GlobalPaymentGuard == nil {
		log.Printf("PAYMENT_GUARD: GlobalPaymentGuard is nil, разрешаем платеж")
		return true, ""
	}
	allowed, message := GlobalPaymentGuard.CheckPaymentAllowed()
	log.Printf("PAYMENT_GUARD: Проверка платежа: allowed=%v, message='%s'", allowed, message)
	return allowed, message
}

// SendPaymentBlockedMessageGlobal глобальная функция отправки сообщения о блокировке
func SendPaymentBlockedMessageGlobal(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	if GlobalPaymentGuard == nil {
		return
	}
	GlobalPaymentGuard.SendPaymentBlockedMessage(bot, chatID, messageID)
}
