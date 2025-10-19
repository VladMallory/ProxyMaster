package powerOff

import (
	"log"

	"bot/common"
)

// InitializePowerOffSystem инициализирует всю систему безопасного выключения
func InitializePowerOffSystem() error {
	log.Printf("POWEROFF_SYSTEM: Начало инициализации, POWEROFF_SYSTEM_ENABLED = %v", common.POWEROFF_SYSTEM_ENABLED)
	if !common.POWEROFF_SYSTEM_ENABLED {
		log.Printf("POWEROFF_SYSTEM: Система безопасного выключения отключена в конфигурации")
		return nil
	}

	log.Printf("POWEROFF_SYSTEM: Инициализация системы безопасного выключения")

	// Инициализируем менеджер выключения
	log.Printf("POWEROFF_SYSTEM: Инициализация менеджера выключения")
	if err := InitializeShutdownManager(); err != nil {
		log.Printf("POWEROFF_SYSTEM: Ошибка инициализации менеджера выключения: %v", err)
		return err
	}
	log.Printf("POWEROFF_SYSTEM: Менеджер выключения инициализирован")

	// Инициализируем защитник платежей
	log.Printf("POWEROFF_SYSTEM: Инициализация защитника платежей")
	InitializePaymentGuard()
	log.Printf("POWEROFF_SYSTEM: Защитник платежей инициализирован")

	log.Printf("POWEROFF_SYSTEM: Система безопасного выключения успешно инициализирована")
	return nil
}

// GetSystemStatus возвращает статус системы безопасного выключения
func GetSystemStatus() map[string]interface{} {
	status := map[string]interface{}{
		"enabled":     common.POWEROFF_SYSTEM_ENABLED,
		"initialized": GlobalShutdownManager != nil,
	}

	if GlobalShutdownManager != nil {
		shutdownStatus := GlobalShutdownManager.GetStatus()
		status["shutdown_state"] = shutdownStatus.State.String()
		status["shutdown_in_progress"] = GlobalShutdownManager.IsShutdownInProgress()
		status["payment_blocked"] = GlobalShutdownManager.IsPaymentBlocked()
		status["active_payments"] = shutdownStatus.ActivePayments
		status["time_remaining"] = shutdownStatus.TimeRemaining
	}

	return status
}

// IsSystemEnabled проверяет, включена ли система безопасного выключения
func IsSystemEnabled() bool {
	return common.POWEROFF_SYSTEM_ENABLED
}

// IsInitialized проверяет, инициализирована ли система
func IsInitialized() bool {
	return GlobalShutdownManager != nil
}
