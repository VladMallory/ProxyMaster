package common

// Глобальная переменная для менеджера пробных периодов
var TrialManager *TrialPeriodManager

// InitGlobals инициализирует глобальные переменные
func InitGlobals() {
	TrialManager = NewTrialPeriodManager()
}
