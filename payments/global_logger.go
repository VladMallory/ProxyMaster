package payments

// Глобальный логгер платежей
var GlobalPaymentLogger *PaymentLogger

// InitGlobalPaymentLogger инициализирует глобальный логгер платежей
func InitGlobalPaymentLogger() {
	GlobalPaymentLogger = NewPaymentLogger()
	SetGlobalPaymentLoggerInterface(GlobalPaymentLogger)
}

// GetGlobalPaymentLogger возвращает глобальный логгер платежей
func GetGlobalPaymentLogger() *PaymentLogger {
	if GlobalPaymentLogger == nil {
		InitGlobalPaymentLogger()
	}
	return GlobalPaymentLogger
}

// UpdateGlobalPaymentLoggerConfig обновляет настройки глобального логгера из конфига
func UpdateGlobalPaymentLoggerConfig() {
	if GlobalPaymentLogger != nil {
		GlobalPaymentLogger.updateFromConfig()
	}
}
