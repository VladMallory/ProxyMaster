package powerOff

import "time"

// ShutdownState представляет состояние системы выключения
type ShutdownState int

const (
	// ShutdownStateNormal - нормальная работа
	ShutdownStateNormal ShutdownState = iota
	// ShutdownStatePreparation - режим подготовки к выключению
	ShutdownStatePreparation
	// ShutdownStateShuttingDown - процесс выключения
	ShutdownStateShuttingDown
)

// String возвращает строковое представление состояния
func (s ShutdownState) String() string {
	switch s {
	case ShutdownStateNormal:
		return "normal"
	case ShutdownStatePreparation:
		return "preparation"
	case ShutdownStateShuttingDown:
		return "shutting_down"
	default:
		return "unknown"
	}
}

// ShutdownManager управляет процессом безопасного выключения
type ShutdownManager struct {
	State               ShutdownState
	RequestedBy         int64     // ID администратора, запросившего выключение
	RequestedAt         time.Time // Время запроса выключения
	PaymentTimeout      int       // Таймаут ожидания платежей в минутах
	CheckInterval       int       // Интервал проверки в секундах
	NotificationEnabled bool      // Включены ли уведомления
}

// PaymentInfo информация об активном платеже
type PaymentInfo struct {
	PaymentID string    `json:"payment_id"`
	UserID    int64     `json:"user_id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ShutdownStatus статус процесса выключения
type ShutdownStatus struct {
	State          ShutdownState `json:"state"`
	RequestedBy    int64         `json:"requested_by"`
	RequestedAt    time.Time     `json:"requested_at"`
	ActivePayments int           `json:"active_payments"`
	CanShutdown    bool          `json:"can_shutdown"`
	TimeRemaining  int           `json:"time_remaining"` // секунды до принудительного выключения
}
