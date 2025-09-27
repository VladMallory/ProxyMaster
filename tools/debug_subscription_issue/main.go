package main

import (
	"fmt"
	"log"
	"time"

	"bot/common"
)

func main() {
	fmt.Println("=== Диагностика проблемы с включением конфига после пополнения ===")

	// Инициализируем глобальные переменные
	common.InitGlobals()

	// Проверяем настройки
	fmt.Println("\n=== Настройки системы ===")
	fmt.Printf("AUTO_BILLING_ENABLED: %v\n", common.AUTO_BILLING_ENABLED)
	fmt.Printf("TARIFF_MODE_ENABLED: %v\n", common.TARIFF_MODE_ENABLED)
	fmt.Printf("PRICE_PER_DAY: %d₽\n", common.PRICE_PER_DAY)
	fmt.Printf("EXPIRED_SUBSCRIPTION_CHECK_ENABLED: %v\n", common.EXPIRED_SUBSCRIPTION_CHECK_ENABLED)
	fmt.Printf("EXPIRED_SUBSCRIPTION_CHECK_INTERVAL: %d минут\n", common.EXPIRED_SUBSCRIPTION_CHECK_INTERVAL)

	// Получаем всех пользователей с активными конфигами
	users, err := common.GetUsersWithActiveConfigs()
	if err != nil {
		log.Fatalf("Ошибка получения пользователей: %v", err)
	}

	fmt.Printf("\n=== Пользователи с активными конфигами (%d) ===\n", len(users))

	now := time.Now()
	expiredCount := 0
	lowBalanceCount := 0
	normalCount := 0

	for _, user := range users {
		expiryTime := time.UnixMilli(user.ExpiryTime)
		daysLeft := int(time.Until(expiryTime).Hours() / 24)
		canAffordDays := int(user.Balance / float64(common.PRICE_PER_DAY))

		status := "✅ Активна"
		if user.ExpiryTime > 0 && user.ExpiryTime <= now.UnixMilli() {
			status = "❌ Истекла"
			expiredCount++
		} else if canAffordDays <= 0 {
			status = "⚠️ Нет средств"
			lowBalanceCount++
		} else {
			normalCount++
		}

		fmt.Printf("%s | %s (ID: %d) | Баланс: %.2f₽ | Дней доступно: %d | Истекает: %s (%d дней)\n",
			status, user.FirstName, user.TelegramID, user.Balance, canAffordDays,
			expiryTime.Format("2006-01-02 15:04"), daysLeft)
	}

	// Проверяем пользователей без активных конфигов, но с балансом
	fmt.Println("\n=== Пользователи БЕЗ активных конфигов, но с балансом ===")

	allUsers, err := common.GetAllUsers()
	if err != nil {
		log.Fatalf("Ошибка получения всех пользователей: %v", err)
	}

	usersWithoutConfig := 0
	for _, user := range allUsers {
		if !user.HasActiveConfig && user.Balance > 0 {
			canAffordDays := int(user.Balance / float64(common.PRICE_PER_DAY))
			if canAffordDays > 0 {
				usersWithoutConfig++
				fmt.Printf("🔍 %s (ID: %d) | Баланс: %.2f₽ | Может купить: %d дней | Email: %s\n",
					user.FirstName, user.TelegramID, user.Balance, canAffordDays, user.Email)
			}
		}
	}

	fmt.Printf("\n=== Результаты диагностики ===\n")
	fmt.Printf("Всего пользователей с активными конфигами: %d\n", len(users))
	fmt.Printf("  - Нормальные: %d\n", normalCount)
	fmt.Printf("  - Истекшие: %d\n", expiredCount)
	fmt.Printf("  - Без средств: %d\n", lowBalanceCount)
	fmt.Printf("Пользователей без конфигов, но с балансом: %d\n", usersWithoutConfig)

	if expiredCount > 0 {
		fmt.Printf("\n⚠️  ПРОБЛЕМА: Найдено %d истекших подписок!\n", expiredCount)
		fmt.Println("Эти подписки должны быть отключены системой проверки истекших подписок.")
	}

	if usersWithoutConfig > 0 {
		fmt.Printf("\n⚠️  ПРОБЛЕМА: Найдено %d пользователей с балансом, но без конфигов!\n", usersWithoutConfig)
		fmt.Println("Эти пользователи должны получить конфиги после пополнения баланса.")
		fmt.Println("Возможные причины:")
		fmt.Println("1. AUTO_BILLING_ENABLED = false")
		fmt.Println("2. TARIFF_MODE_ENABLED = true")
		fmt.Println("3. Баланс меньше PRICE_PER_DAY")
		fmt.Println("4. Ошибка в ForceBalanceRecalculation")
	}

	if expiredCount == 0 && usersWithoutConfig == 0 {
		fmt.Println("\n✅ Все в порядке! Проблем не найдено.")
	}

	fmt.Println("\n=== Рекомендации ===")
	if common.AUTO_BILLING_ENABLED && !common.TARIFF_MODE_ENABLED {
		fmt.Println("✅ Настройки автосписания корректны")
	} else {
		fmt.Println("❌ Проблема с настройками автосписания!")
		if !common.AUTO_BILLING_ENABLED {
			fmt.Println("   - Включите AUTO_BILLING_ENABLED = true")
		}
		if common.TARIFF_MODE_ENABLED {
			fmt.Println("   - Отключите TARIFF_MODE_ENABLED = false")
		}
	}

	fmt.Println("\nДля тестирования попробуйте:")
	fmt.Println("1. Пополнить баланс пользователю")
	fmt.Println("2. Нажать /start")
	fmt.Println("3. Проверить логи на предмет ошибок ForceBalanceRecalculation")
}
