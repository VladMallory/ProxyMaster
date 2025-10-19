package payments

// PaymentLoggerInterface определяет интерфейс для логгера платежей
type PaymentLoggerInterface interface {
	LogPayment(paymentID string, userID int64, amount float64, status string) error
	UpdatePaymentStatus(paymentID string, status string, processed bool) error
	GetPendingPayments() ([]PaymentLogEntry, error)
	SetLogFile(logFile string)
	SetEnabled(enabled bool)
	GetLogFile() string
	IsEnabled() bool
}

// Глобальная переменная для интерфейса логгера
var GlobalPaymentLoggerInterface PaymentLoggerInterface

// SetGlobalPaymentLoggerInterface устанавливает глобальный интерфейс логгера
func SetGlobalPaymentLoggerInterface(logger PaymentLoggerInterface) {
	GlobalPaymentLoggerInterface = logger
}

// GetGlobalPaymentLoggerInterface возвращает глобальный интерфейс логгера
func GetGlobalPaymentLoggerInterface() PaymentLoggerInterface {
	return GlobalPaymentLoggerInterface
}

// LogPaymentToFile логирует платеж через глобальный интерфейс
func LogPaymentToFile(paymentID string, userID int64, amount float64, status string) error {
	if logger := GetGlobalPaymentLoggerInterface(); logger != nil {
		return logger.LogPayment(paymentID, userID, amount, status)
	}
	return nil
}
