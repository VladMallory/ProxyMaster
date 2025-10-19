package main

import (
	"fmt"
	"log"
	"time"

	"bot/common"
)

func main() {
	fmt.Println("=== Тест системы проверки истекших подписок ===")

	// Инициализируем глобальные переменные
	common.InitGlobals()

	// Получаем всех пользователей с активными конфигами
	users, err := common.GetUsersWithActiveConfigs()
	if err != nil {
		log.Fatalf("Ошибка получения пользователей: %v", err)
	}

	fmt.Printf("Найдено пользователей с активными конфигами: %d\n", len(users))

	now := time.Now()
	expiredCount := 0

	fmt.Println("\n=== Проверка подписок ===")
	for _, user := range users {
		// Проверяем, истекла ли подписка
		if user.HasActiveConfig && user.ExpiryTime > 0 && user.ExpiryTime <= now.UnixMilli() {
			expiredCount++
			expiryTime := time.UnixMilli(user.ExpiryTime)
			fmt.Printf("❌ Истекшая подписка: %s (ID: %d, Email: %s, Истекла: %s)\n",
				user.FirstName, user.TelegramID, user.Email, expiryTime.Format("2006-01-02 15:04:05"))
		} else if user.HasActiveConfig {
			expiryTime := time.UnixMilli(user.ExpiryTime)
			if user.ExpiryTime > 0 {
				daysLeft := int(time.Until(expiryTime).Hours() / 24)
				fmt.Printf("✅ Активная подписка: %s (ID: %d, Email: %s, Осталось дней: %d)\n",
					user.FirstName, user.TelegramID, user.Email, daysLeft)
			} else {
				fmt.Printf("✅ Активная подписка без срока: %s (ID: %d, Email: %s)\n",
					user.FirstName, user.TelegramID, user.Email)
			}
		}
	}

	fmt.Printf("\n=== Результаты ===\n")
	fmt.Printf("Всего пользователей с активными конфигами: %d\n", len(users))
	fmt.Printf("Истекших подписок: %d\n", expiredCount)
	fmt.Printf("Активных подписок: %d\n", len(users)-expiredCount)

	if expiredCount > 0 {
		fmt.Printf("\n⚠️  ВНИМАНИЕ: Найдено %d истекших подписок, которые должны быть отключены!\n", expiredCount)
		fmt.Println("Запустите бота с включенным сервисом проверки истекших подписок для автоматического отключения.")
	} else {
		fmt.Println("\n✅ Все подписки в порядке!")
	}

	// Проверяем настройки
	fmt.Printf("\n=== Настройки ===\n")
	fmt.Printf("EXPIRED_SUBSCRIPTION_CHECK_ENABLED: %v\n", common.EXPIRED_SUBSCRIPTION_CHECK_ENABLED)
	fmt.Printf("EXPIRED_SUBSCRIPTION_CHECK_INTERVAL: %d минут\n", common.EXPIRED_SUBSCRIPTION_CHECK_INTERVAL)
}
