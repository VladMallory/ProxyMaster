package services

import (
	"log"
	"time"

	"bot/common"
	"bot/payments"
	paymentCommon "bot/payments/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// PaymentMonitorService отслеживает статус платежей и автоматически обрабатывает успешные
type PaymentMonitorService struct {
	paymentManager *payments.PaymentManager
	ticker         *time.Ticker
	interval       time.Duration
}

// NewPaymentMonitorService создает новый сервис мониторинга платежей
func NewPaymentMonitorService(paymentManager *payments.PaymentManager) *PaymentMonitorService {
	return &PaymentMonitorService{
		paymentManager: paymentManager,
		interval:       2 * time.Minute, // Проверяем каждые 2 минуты
	}
}

// Start запускает мониторинг платежей
func (pms *PaymentMonitorService) Start() {
	if pms.paymentManager == nil {
		log.Printf("PAYMENT_MONITOR: Платежная система не инициализирована, мониторинг отключен")
		return
	}

	log.Printf("PAYMENT_MONITOR: Запуск мониторинга платежей (интервал: %v)", pms.interval)

	pms.ticker = time.NewTicker(pms.interval)

	go func() {
		// Выполняем первую проверку сразу
		pms.checkPendingPayments()

		// Затем проверяем по расписанию
		for range pms.ticker.C {
			pms.checkPendingPayments()
		}
	}()
}

// Stop останавливает мониторинг платежей
func (pms *PaymentMonitorService) Stop() {
	if pms.ticker != nil {
		pms.ticker.Stop()
		log.Printf("PAYMENT_MONITOR: Мониторинг платежей остановлен")
	}
}

// checkPendingPayments проверяет все ожидающие платежи
func (pms *PaymentMonitorService) checkPendingPayments() {
	log.Printf("PAYMENT_MONITOR: Проверка ожидающих платежей...")

	// Здесь должна быть логика получения списка ожидающих платежей
	// Пока что это заглушка, так как в текущей системе нет хранения списка платежей

	// В реальной системе здесь был бы запрос к базе данных для получения
	// всех платежей со статусом "pending" старше определенного времени
	log.Printf("PAYMENT_MONITOR: Проверка завершена (функциональность требует доработки)")
}

// ProcessPaymentIfSucceeded проверяет и обрабатывает конкретный платеж
func (pms *PaymentMonitorService) ProcessPaymentIfSucceeded(paymentID string, userID int64) error {
	log.Printf("PAYMENT_MONITOR: Проверка платежа %s для пользователя %d", paymentID, userID)

	// Проверяем статус платежа
	paymentInfo, err := pms.paymentManager.CheckPaymentStatus(paymentCommon.PaymentMethodAPI, paymentID)
	if err != nil {
		log.Printf("PAYMENT_MONITOR: Ошибка проверки платежа %s: %v", paymentID, err)
		return err
	}

	// Если платеж успешен, обрабатываем его
	if paymentInfo.Status == paymentCommon.PaymentStatusSucceeded {
		log.Printf("PAYMENT_MONITOR: Платеж %s успешен, зачисляем средства", paymentID)

		// Зачисляем средства
		err = common.AddBalance(userID, paymentInfo.Amount)
		if err != nil {
			log.Printf("PAYMENT_MONITOR: Ошибка зачисления средств для платежа %s: %v", paymentID, err)
			return err
		}

		// Получаем обновленные данные пользователя
		user, err := common.GetUserByTelegramID(userID)
		if err != nil {
			log.Printf("PAYMENT_MONITOR: Ошибка получения данных пользователя %d: %v", userID, err)
			return err
		}

		log.Printf("PAYMENT_MONITOR: Средства успешно зачислены! Пользователь %d, новый баланс: %.2f₽", userID, user.Balance)

		// Отправляем уведомление пользователю
		if common.GlobalBot != nil {
			text := "✅ <b>Платеж обработан!</b>\n\n" +
				"💰 Пополнено: " + paymentCommon.FormatAmount(paymentInfo.Amount) + "\n" +
				"💳 Новый баланс: " + paymentCommon.FormatAmount(user.Balance) + "\n" +
				"🆔 ID платежа: " + paymentInfo.ID + "\n\n" +
				"Спасибо за пополнение!"

			msg := tgbotapi.NewMessage(userID, text)
			msg.ParseMode = "HTML"

			if _, err := common.GlobalBot.Send(msg); err != nil {
				log.Printf("PAYMENT_MONITOR: Ошибка отправки уведомления пользователю %d: %v", userID, err)
			} else {
				log.Printf("PAYMENT_MONITOR: Уведомление отправлено пользователю %d", userID)
			}
		}

		return nil
	}

	log.Printf("PAYMENT_MONITOR: Платеж %s имеет статус: %s", paymentID, paymentInfo.Status)
	return nil
}
