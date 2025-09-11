package common

import (
	"fmt"
	"log"
)

// Переменные для управления сервисом автосписания
var (
	autoBillingServicePtr interface{} // Ссылка на сервис автосписания
)

// SetAutoBillingService устанавливает ссылку на сервис автосписания
func SetAutoBillingService(service interface{}) {
	autoBillingServicePtr = service
}

// SwitchToTariffMode переключает на тарифный режим
func SwitchToTariffMode() {
	log.Printf("BILLING_MANAGER: Переключение на тарифный режим")
	TARIFF_MODE_ENABLED = true
	AUTO_BILLING_ENABLED = false

	// Останавливаем автосписание через интерфейс
	if autoBillingServicePtr != nil {
		if service, ok := autoBillingServicePtr.(interface{ Stop() }); ok {
			service.Stop()
		}
		autoBillingServicePtr = nil
	}

	log.Printf("BILLING_MANAGER: Переключение на тарифный режим завершено")
}

// SwitchToAutoBillingMode переключает на режим автосписания
func SwitchToAutoBillingMode() {
	log.Printf("BILLING_MANAGER: Переключение на режим автосписания")

	// Останавливаем старый сервис если есть
	if autoBillingServicePtr != nil {
		if service, ok := autoBillingServicePtr.(interface{ Stop() }); ok {
			service.Stop()
		}
		autoBillingServicePtr = nil
	}

	TARIFF_MODE_ENABLED = false
	AUTO_BILLING_ENABLED = true

	log.Printf("BILLING_MANAGER: Переключение на режим автосписания завершено")
	log.Printf("BILLING_MANAGER: Для полного переключения требуется перезапуск сервиса автосписания")
}

// GetBillingStatus возвращает текущий статус биллинга
func GetBillingStatus() string {
	status := "📊 Статус системы биллинга:\n\n"

	if TARIFF_MODE_ENABLED {
		status += "🎯 Режим: Тарифный\n"
		status += "💳 Описание: Пользователи покупают дни вручную\n"
		status += "🔄 Автосписание: Отключено\n"
	} else if AUTO_BILLING_ENABLED {
		status += "🤖 Режим: Автосписание\n"
		status += "💸 Описание: Ежедневное списание с баланса\n"
		status += "📅 Цена за день: " + formatPrice(PRICE_PER_DAY) + "\n"
		status += "⏰ Интервал пересчета: " + formatInterval(BALANCE_RECALC_INTERVAL) + "\n"
	} else {
		status += "❌ Режим: Неопределен\n"
		status += "⚠️ Описание: Оба режима отключены\n"
	}

	status += "\n🔧 Команды управления:\n"
	status += "/switch_tariff - Переключить на тарифы\n"
	status += "/switch_auto - Переключить на автосписание\n"
	status += "/billing_status - Показать текущий статус"

	return status
}

// formatPrice форматирует цену
func formatPrice(price int) string {
	return fmt.Sprintf("%d₽", price)
}

// formatInterval форматирует интервал времени
func formatInterval(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%d мин", minutes)
	}
	hours := minutes / 60
	if hours < 24 {
		return fmt.Sprintf("%d ч", hours)
	}
	days := hours / 24
	return fmt.Sprintf("%d дн", days)
}
