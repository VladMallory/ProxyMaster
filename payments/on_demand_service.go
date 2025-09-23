package payments

import (
	"log"
	"time"

	"bot/common"
	paymentCommon "bot/payments/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// OnDemandPaymentService обрабатывает платежи по требованию
type OnDemandPaymentService struct {
	paymentManager *PaymentManager
}

// NewOnDemandPaymentService создает новый сервис обработки платежей по требованию
func NewOnDemandPaymentService(paymentManager *PaymentManager) *OnDemandPaymentService {
	return &OnDemandPaymentService{
		paymentManager: paymentManager,
	}
}

// StartPaymentMonitoring запускает мониторинг конкретного платежа
// Вызывается после создания платежа пользователем
func (odps *OnDemandPaymentService) StartPaymentMonitoring(paymentID string, userID int64, amount float64) {
	log.Printf("PAYMENT_ON_DEMAND: Запуск мониторинга платежа %s для пользователя %d", paymentID, userID)

	// Логируем создание платежа
	if err := GetGlobalPaymentLogger().LogPayment(paymentID, userID, amount, "pending"); err != nil {
		log.Printf("PAYMENT_ON_DEMAND: Ошибка логирования платежа %s: %v", paymentID, err)
	}

	// Запускаем мониторинг в отдельной горутине
	go odps.monitorPayment(paymentID, userID)
}

// monitorPayment мониторит конкретный платеж
func (odps *OnDemandPaymentService) monitorPayment(paymentID string, userID int64) {
	log.Printf("PAYMENT_ON_DEMAND: Начало мониторинга платежа %s", paymentID)

	// Создаем тикер для проверки каждые 30 секунд
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Максимальное время мониторинга - 10 минут
	timeout := time.NewTimer(10 * time.Minute)
	defer timeout.Stop()

	// Выполняем первую проверку сразу
	if odps.checkAndProcessPayment(paymentID, userID) {
		log.Printf("PAYMENT_ON_DEMAND: Платеж %s обработан при первой проверке", paymentID)
		return
	}

	// Продолжаем мониторинг
	for {
		select {
		case <-ticker.C:
			if odps.checkAndProcessPayment(paymentID, userID) {
				log.Printf("PAYMENT_ON_DEMAND: Платеж %s успешно обработан", paymentID)
				return
			}

		case <-timeout.C:
			log.Printf("PAYMENT_ON_DEMAND: Таймаут мониторинга платежа %s (10 минут)", paymentID)
			// Обновляем статус в логе
			GetGlobalPaymentLogger().UpdatePaymentStatus(paymentID, "timeout", false)
			return
		}
	}
}

// checkAndProcessPayment проверяет и обрабатывает платеж если он успешен
func (odps *OnDemandPaymentService) checkAndProcessPayment(paymentID string, userID int64) bool {
	if odps.paymentManager == nil {
		log.Printf("PAYMENT_ON_DEMAND: Платежная система не инициализирована")
		return false
	}

	// Проверяем статус платежа
	paymentInfo, err := odps.paymentManager.CheckPaymentStatus(paymentCommon.PaymentMethodAPI, paymentID)
	if err != nil {
		log.Printf("PAYMENT_ON_DEMAND: Ошибка проверки платежа %s: %v", paymentID, err)
		return false
	}

	log.Printf("PAYMENT_ON_DEMAND: Статус платежа %s: %s", paymentID, paymentInfo.Status)

	// Если платеж успешен, обрабатываем его
	if paymentInfo.Status == paymentCommon.PaymentStatusSucceeded {
		log.Printf("PAYMENT_ON_DEMAND: Платеж %s успешен, зачисляем средства", paymentID)

		// Зачисляем средства
		err = common.AddBalance(userID, paymentInfo.Amount)
		if err != nil {
			log.Printf("PAYMENT_ON_DEMAND: Ошибка зачисления средств для платежа %s: %v", paymentID, err)
			GetGlobalPaymentLogger().UpdatePaymentStatus(paymentID, "error_balance", false)
			return false
		}

		// Получаем обновленные данные пользователя
		user, err := common.GetUserByTelegramID(userID)
		if err != nil {
			log.Printf("PAYMENT_ON_DEMAND: Ошибка получения данных пользователя %d: %v", userID, err)
		}

		// Обновляем статус в логе
		GetGlobalPaymentLogger().UpdatePaymentStatus(paymentID, "succeeded", true)

		// Отправляем уведомление пользователю
		if common.GlobalBot != nil {
			text := "✅ <b>Платеж автоматически обработан!</b>\n\n" +
				"💰 Пополнено: " + paymentCommon.FormatAmount(paymentInfo.Amount) + "\n" +
				"💳 Новый баланс: " + paymentCommon.FormatAmount(user.Balance) + "\n" +
				"🆔 ID платежа: " + paymentInfo.ID + "\n\n" +
				"Спасибо за пополнение!"

			msg := tgbotapi.NewMessage(userID, text)
			msg.ParseMode = "HTML"

			if _, err := common.GlobalBot.Send(msg); err != nil {
				log.Printf("PAYMENT_ON_DEMAND: Ошибка отправки уведомления пользователю %d: %v", userID, err)
			} else {
				log.Printf("PAYMENT_ON_DEMAND: Уведомление отправлено пользователю %d", userID)
			}
		}

		return true
	}

	// Если платеж отменен или завершился с ошибкой
	if paymentInfo.Status == paymentCommon.PaymentStatusCanceled {
		log.Printf("PAYMENT_ON_DEMAND: Платеж %s отменен", paymentID)
		GetGlobalPaymentLogger().UpdatePaymentStatus(paymentID, "canceled", true)
		return true
	}

	// Платеж все еще в процессе
	return false
}

// GetPendingPayments возвращает список необработанных платежей
func (odps *OnDemandPaymentService) GetPendingPayments() ([]PaymentLogEntry, error) {
	return GetGlobalPaymentLogger().GetPendingPayments()
}

// CheckPendingPayments проверяет все необработанные платежи (вызывается периодически)
func (odps *OnDemandPaymentService) CheckPendingPayments() {
	pendingPayments, err := GetGlobalPaymentLogger().GetPendingPayments()
	if err != nil {
		log.Printf("PAYMENT_ON_DEMAND: Ошибка получения необработанных платежей: %v", err)
		return
	}

	if len(pendingPayments) == 0 {
		return
	}

	log.Printf("PAYMENT_ON_DEMAND: Найдено %d необработанных платежей", len(pendingPayments))

	for _, payment := range pendingPayments {
		log.Printf("PAYMENT_ON_DEMAND: Проверяем необработанный платеж %s", payment.PaymentID)
		odps.checkAndProcessPayment(payment.PaymentID, payment.UserID)
	}
}
