package powerOff

import (
	"fmt"
	"log"
	"sync"
	"time"

	"bot/common"
	"bot/payments"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// GlobalShutdownManager глобальный экземпляр менеджера выключения
var GlobalShutdownManager *ShutdownManager
var shutdownMutex sync.RWMutex

// NewShutdownManager создает новый менеджер выключения
func NewShutdownManager() *ShutdownManager {
	return &ShutdownManager{
		State:               ShutdownStateNormal,
		PaymentTimeout:      common.POWEROFF_PAYMENT_TIMEOUT,
		CheckInterval:       common.POWEROFF_CHECK_INTERVAL,
		NotificationEnabled: common.POWEROFF_NOTIFICATION_ENABLED,
	}
}

// InitializeShutdownManager инициализирует глобальный менеджер выключения
func InitializeShutdownManager() error {
	if !common.POWEROFF_SYSTEM_ENABLED {
		log.Printf("POWEROFF: Система безопасного выключения отключена в конфигурации")
		return nil
	}

	GlobalShutdownManager = NewShutdownManager()
	log.Printf("POWEROFF: Менеджер безопасного выключения инициализирован")
	return nil
}

// RequestShutdown запрашивает безопасное выключение
func (sm *ShutdownManager) RequestShutdown(adminID int64) error {
	shutdownMutex.Lock()
	defer shutdownMutex.Unlock()

	log.Printf("POWEROFF: RequestShutdown вызвана, текущее состояние: %s", sm.State.String())
	if sm.State != ShutdownStateNormal {
		log.Printf("POWEROFF: Процесс выключения уже запущен, состояние: %s", sm.State.String())
		return fmt.Errorf("процесс выключения уже запущен")
	}

	sm.State = ShutdownStatePreparation
	sm.RequestedBy = adminID
	sm.RequestedAt = time.Now()

	log.Printf("POWEROFF: Запрошено выключение администратором %d, новое состояние: %s", adminID, sm.State.String())
	return nil
}

// GetStatus возвращает текущий статус выключения
func (sm *ShutdownManager) GetStatus() ShutdownStatus {
	shutdownMutex.RLock()
	defer shutdownMutex.RUnlock()

	status := ShutdownStatus{
		State:       sm.State,
		RequestedBy: sm.RequestedBy,
		RequestedAt: sm.RequestedAt,
		CanShutdown: sm.State == ShutdownStateNormal,
	}

	// Подсчитываем активные платежи
	if payments.GlobalPaymentManager != nil {
		onDemandService := payments.GlobalPaymentManager.GetOnDemandService()
		if onDemandService != nil {
			pendingPayments, err := onDemandService.GetPendingPayments()
			if err == nil {
				status.ActivePayments = len(pendingPayments)
			}
		}
	}

	// Вычисляем оставшееся время до принудительного выключения
	if sm.State == ShutdownStatePreparation {
		elapsed := time.Since(sm.RequestedAt)
		timeout := time.Duration(sm.PaymentTimeout) * time.Minute
		remaining := timeout - elapsed
		if remaining > 0 {
			status.TimeRemaining = int(remaining.Seconds())
		} else {
			status.TimeRemaining = 0
		}
	}

	return status
}

// IsShutdownInProgress проверяет, идет ли процесс выключения
func (sm *ShutdownManager) IsShutdownInProgress() bool {
	shutdownMutex.RLock()
	defer shutdownMutex.RUnlock()
	return sm.State != ShutdownStateNormal
}

// IsPaymentBlocked проверяет, заблокированы ли платежи
func (sm *ShutdownManager) IsPaymentBlocked() bool {
	shutdownMutex.RLock()
	defer shutdownMutex.RUnlock()
	return sm.State == ShutdownStatePreparation || sm.State == ShutdownStateShuttingDown
}

// CancelShutdown отменяет процесс выключения
func (sm *ShutdownManager) CancelShutdown(adminID int64) error {
	shutdownMutex.Lock()
	defer shutdownMutex.Unlock()

	if sm.State == ShutdownStateNormal {
		return fmt.Errorf("процесс выключения не запущен")
	}

	if sm.RequestedBy != adminID {
		return fmt.Errorf("только администратор, запросивший выключение, может его отменить")
	}

	sm.State = ShutdownStateNormal
	sm.RequestedBy = 0
	sm.RequestedAt = time.Time{}

	log.Printf("POWEROFF: Выключение отменено администратором %d", adminID)
	return nil
}

// StartShutdownProcess запускает процесс выключения
func (sm *ShutdownManager) StartShutdownProcess(bot *tgbotapi.BotAPI, adminChatID int64) {
	log.Printf("POWEROFF: Начало процесса выключения")

	// Проверяем активные платежи ДО изменения состояния
	activePayments := sm.getActivePayments()
	log.Printf("POWEROFF: Найдено %d активных платежей", len(activePayments))

	if len(activePayments) > 0 {
		// Есть активные платежи - остаемся в состоянии Preparation для ожидания
		log.Printf("POWEROFF: Ожидание завершения %d платежей", len(activePayments))
		sm.waitForPaymentsCompletion(bot, adminChatID, activePayments)
	} else {
		// Нет активных платежей - сразу выключаемся
		shutdownMutex.Lock()
		sm.State = ShutdownStateShuttingDown
		shutdownMutex.Unlock()

		sm.performShutdown(bot, adminChatID)
	}
}

// getActivePayments получает список активных платежей
func (sm *ShutdownManager) getActivePayments() []PaymentInfo {
	log.Printf("POWEROFF: getActivePayments вызвана")
	if payments.GlobalPaymentManager == nil {
		log.Printf("POWEROFF: GlobalPaymentManager is nil")
		return []PaymentInfo{}
	}

	onDemandService := payments.GlobalPaymentManager.GetOnDemandService()
	if onDemandService == nil {
		log.Printf("POWEROFF: onDemandService is nil")
		return []PaymentInfo{}
	}

	pendingPayments, err := onDemandService.GetPendingPayments()
	if err != nil {
		log.Printf("POWEROFF: Ошибка получения активных платежей: %v", err)
		return []PaymentInfo{}
	}

	log.Printf("POWEROFF: Получено %d pending платежей", len(pendingPayments))

	var paymentInfos []PaymentInfo
	for _, payment := range pendingPayments {
		paymentInfos = append(paymentInfos, PaymentInfo{
			PaymentID: payment.PaymentID,
			UserID:    payment.UserID,
			Amount:    payment.Amount,
			Status:    payment.Status,
			CreatedAt: payment.CreatedAt,
		})
	}

	log.Printf("POWEROFF: Возвращаем %d активных платежей", len(paymentInfos))
	return paymentInfos
}

// waitForPaymentsCompletion ожидает завершения активных платежей
func (sm *ShutdownManager) waitForPaymentsCompletion(bot *tgbotapi.BotAPI, adminChatID int64, initialPayments []PaymentInfo) {
	log.Printf("POWEROFF: Ожидание завершения %d платежей", len(initialPayments))

	// Отправляем уведомление админу
	if sm.NotificationEnabled {
		msg := tgbotapi.NewMessage(adminChatID,
			fmt.Sprintf("⏳ Найдено %d активных платежей. Ожидаем их завершения...", len(initialPayments)))
		bot.Send(msg)
	}

	ticker := time.NewTicker(time.Duration(sm.CheckInterval) * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(time.Duration(sm.PaymentTimeout) * time.Minute)
	defer timeout.Stop()

	lastNotification := time.Now()

	for {
		select {
		case <-ticker.C:
			// Проверяем текущие активные платежи
			log.Printf("POWEROFF: Проверка активных платежей (ticker)")
			currentPayments := sm.getActivePayments()

			if len(currentPayments) == 0 {
				// Все платежи завершены
				log.Printf("POWEROFF: Все платежи завершены, выключаем бота")

				// Переводим в состояние выключения
				shutdownMutex.Lock()
				sm.State = ShutdownStateShuttingDown
				shutdownMutex.Unlock()

				if sm.NotificationEnabled {
					msg := tgbotapi.NewMessage(adminChatID, "✅ Все платежи завершены. Выключаем бота...")
					bot.Send(msg)
				}
				sm.performShutdown(bot, adminChatID)
				return
			}

			// Отправляем периодические уведомления (не чаще чем раз в минуту)
			if sm.NotificationEnabled && time.Since(lastNotification) > time.Minute {
				msg := tgbotapi.NewMessage(adminChatID,
					fmt.Sprintf("⏳ Осталось %d активных платежей...", len(currentPayments)))
				bot.Send(msg)
				lastNotification = time.Now()
			}

		case <-timeout.C:
			// Таймаут - принудительно выключаемся
			log.Printf("POWEROFF: Таймаут ожидания платежей, принудительное выключение")

			// Переводим в состояние выключения
			shutdownMutex.Lock()
			sm.State = ShutdownStateShuttingDown
			shutdownMutex.Unlock()

			if sm.NotificationEnabled {
				msg := tgbotapi.NewMessage(adminChatID, "⏰ Таймаут ожидания. Выключаем бота принудительно...")
				bot.Send(msg)
			}
			sm.performShutdown(bot, adminChatID)
			return
		}
	}
}

// performShutdown выполняет безопасное выключение
func (sm *ShutdownManager) performShutdown(bot *tgbotapi.BotAPI, adminChatID int64) {
	log.Printf("POWEROFF: Выполнение безопасного выключения")

	// Уведомляем админа
	if sm.NotificationEnabled {
		msg := tgbotapi.NewMessage(adminChatID, "🔌 Бот выключается безопасно...")
		bot.Send(msg)
	}

	// Останавливаем сервисы
	sm.stopServices()

	// Закрываем соединения с БД
	common.DisconnectMongoDB()

	// Завершаем работу
	log.Println("POWEROFF: Graceful shutdown завершен")
	// Здесь можно добавить вызов os.Exit(0) или передать сигнал в main
}

// stopServices останавливает все сервисы
func (sm *ShutdownManager) stopServices() {
	log.Printf("POWEROFF: Остановка сервисов...")

	// Здесь можно добавить остановку других сервисов
	// Например, если есть глобальные сервисы, которые нужно остановить

	log.Printf("POWEROFF: Сервисы остановлены")
}
