package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"bot/common"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Использование: go run main.go <telegram_id>")
		fmt.Println("Пример: go run main.go 873925520")
		return
	}

	telegramIDStr := os.Args[1]
	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Ошибка парсинга Telegram ID: %v", err)
	}

	fmt.Printf("=== Принудительное включение конфига для пользователя %d ===\n", telegramID)

	// Инициализируем глобальные переменные
	common.InitGlobals()

	// Получаем пользователя
	user, err := common.GetUserByTelegramID(telegramID)
	if err != nil {
		log.Fatalf("Ошибка получения пользователя %d: %v", telegramID, err)
	}

	fmt.Printf("Пользователь: %s (ID: %d)\n", user.FirstName, user.TelegramID)
	fmt.Printf("Баланс: %.2f₽\n", user.Balance)
	fmt.Printf("Активный конфиг: %v\n", user.HasActiveConfig)
	fmt.Printf("Email: %s\n", user.Email)

	if user.Balance <= 0 {
		fmt.Printf("❌ У пользователя нет баланса (%.2f₽)\n", user.Balance)
		return
	}

	canAffordDays := int(user.Balance / float64(common.PRICE_PER_DAY))
	if canAffordDays <= 0 {
		fmt.Printf("❌ Недостаточно средств для оплаты хотя бы одного дня (%.2f₽, нужно %d₽)\n",
			user.Balance, common.PRICE_PER_DAY)
		return
	}

	fmt.Printf("✅ Может купить: %d дней\n", canAffordDays)

	// Проверяем настройки
	fmt.Printf("\n=== Проверка настроек ===\n")
	fmt.Printf("AUTO_BILLING_ENABLED: %v\n", common.AUTO_BILLING_ENABLED)
	fmt.Printf("TARIFF_MODE_ENABLED: %v\n", common.TARIFF_MODE_ENABLED)

	if !common.AUTO_BILLING_ENABLED {
		fmt.Printf("❌ AUTO_BILLING_ENABLED отключено!\n")
		return
	}

	if common.TARIFF_MODE_ENABLED {
		fmt.Printf("❌ TARIFF_MODE_ENABLED включено!\n")
		return
	}

	// Если у пользователя уже есть активный конфиг
	if user.HasActiveConfig {
		fmt.Printf("\n✅ У пользователя уже есть активный конфиг\n")

		// Проверяем время истечения
		if user.ExpiryTime > 0 {
			expiryTime := time.UnixMilli(user.ExpiryTime)
			daysLeft := int(time.Until(expiryTime).Hours() / 24)
			fmt.Printf("Время истечения: %s (%d дней)\n",
				expiryTime.Format("2006-01-02 15:04:05"), daysLeft)

			if user.ExpiryTime <= time.Now().UnixMilli() {
				fmt.Printf("⚠️ Конфиг истек! Нужно продлить.\n")
			}
		}
		return
	}

	// Запускаем принудительный пересчет баланса
	fmt.Printf("\n=== Запуск принудительного пересчета баланса ===\n")

	// Создаем мок-сервис автосписания для тестирования
	fmt.Printf("🚀 Запуск ForceBalanceRecalculation...\n")
	common.ForceBalanceRecalculation(telegramID)

	// Ждем немного
	time.Sleep(2 * time.Second)

	// Проверяем результат
	updatedUser, err := common.GetUserByTelegramID(telegramID)
	if err != nil {
		log.Printf("Ошибка получения обновленного пользователя: %v", err)
		return
	}

	fmt.Printf("\n=== Результат ===\n")
	fmt.Printf("Активный конфиг: %v\n", updatedUser.HasActiveConfig)
	fmt.Printf("ClientID: %s\n", updatedUser.ClientID)
	fmt.Printf("SubID: %s\n", updatedUser.SubID)

	if updatedUser.HasActiveConfig {
		fmt.Printf("✅ Конфиг успешно создан!\n")

		if updatedUser.ExpiryTime > 0 {
			expiryTime := time.UnixMilli(updatedUser.ExpiryTime)
			daysLeft := int(time.Until(expiryTime).Hours() / 24)
			fmt.Printf("Время истечения: %s (%d дней)\n",
				expiryTime.Format("2006-01-02 15:04:05"), daysLeft)
		}
	} else {
		fmt.Printf("❌ Конфиг не был создан. Проверьте логи.\n")
		fmt.Printf("Возможные причины:\n")
		fmt.Printf("1. Ошибка авторизации в панели\n")
		fmt.Printf("2. Ошибка создания конфига через API\n")
		fmt.Printf("3. Проблема с автосписанием\n")
	}
}
