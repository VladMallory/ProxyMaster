package main

import (
	"log"
	"os"
	"strconv"

	"bot/common"
)

func main() {
	if len(os.Args) < 2 {
		log.Printf("Использование: go run main.go <telegram_id>")
		log.Printf("Пример: go run main.go 873925520")
		return
	}

	telegramIDStr := os.Args[1]
	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		log.Printf("❌ Ошибка парсинга Telegram ID: %v", err)
		return
	}

	log.Printf("=== ПРИНУДИТЕЛЬНАЯ СИНХРОНИЗАЦИЯ С ДОПОЛНИТЕЛЬНЫМ ИНБАУНДОМ ===")
	log.Printf("Telegram ID: %d", telegramID)
	log.Printf("SECONDARY_INBOUND_ENABLED: %v", common.SECONDARY_INBOUND_ENABLED)

	if !common.SECONDARY_INBOUND_ENABLED {
		log.Printf("❌ Дополнительный инбаунд отключен в конфигурации")
		return
	}

	// Получаем пользователя из базы данных
	user, err := common.GetUserByTelegramID(telegramID)
	if err != nil {
		log.Printf("❌ Ошибка получения пользователя: %v", err)
		return
	}

	if user == nil {
		log.Printf("❌ Пользователь с ID %d не найден в базе данных", telegramID)
		return
	}

	log.Printf("✅ Пользователь найден:")
	log.Printf("  Telegram ID: %d", user.TelegramID)
	log.Printf("  Username: %s", user.Username)
	log.Printf("  First Name: %s", user.FirstName)
	log.Printf("  HasActiveConfig: %v", user.HasActiveConfig)
	log.Printf("  HasActiveSecondaryConfig: %v", user.HasActiveSecondaryConfig)
	log.Printf("  Balance: %.2f₽", user.Balance)

	// Принудительная синхронизация с дополнительным инбаундом
	log.Printf("\n=== СИНХРОНИЗАЦИЯ С ДОПОЛНИТЕЛЬНЫМ ИНБАУНДОМ ===")
	err = common.SyncUserWithSecondaryPanel(user)
	if err != nil {
		log.Printf("❌ Ошибка синхронизации: %v", err)
		return
	}

	// Получаем обновленного пользователя
	updatedUser, err := common.GetUserByTelegramID(telegramID)
	if err != nil {
		log.Printf("❌ Ошибка получения обновленного пользователя: %v", err)
		return
	}

	log.Printf("\n=== РЕЗУЛЬТАТ СИНХРОНИЗАЦИИ ===")
	log.Printf("  HasActiveSecondaryConfig: %v", updatedUser.HasActiveSecondaryConfig)
	log.Printf("  SecondaryClientID: %s", updatedUser.SecondaryClientID)
	log.Printf("  SecondarySubID: %s", updatedUser.SecondarySubID)
	log.Printf("  SecondaryEmail: %s", updatedUser.SecondaryEmail)
	log.Printf("  SecondaryExpiryTime: %d", updatedUser.SecondaryExpiryTime)

	// Проверяем общий статус
	hasAnyActiveConfig := updatedUser.HasActiveConfig || updatedUser.HasActiveSecondaryConfig
	log.Printf("  HasAnyActiveConfig: %v", hasAnyActiveConfig)

	if updatedUser.HasActiveSecondaryConfig {
		log.Printf("✅ Синхронизация успешна! Пользователь найден в дополнительном инбаунде")

		// Получаем информацию о конфигах
		configInfo := common.GetActiveConfigInfo(updatedUser)
		log.Printf("\n=== ИНФОРМАЦИЯ О КОНФИГАХ ===")
		log.Printf("  Primary Active: %v", configInfo["primary_active"])
		log.Printf("  Secondary Active: %v", configInfo["secondary_active"])
		log.Printf("  Any Active: %v", configInfo["any_active"])

		if configInfo["secondary_config_url"] != "" {
			log.Printf("  Secondary Config URL: %s", configInfo["secondary_config_url"])
		}
	} else {
		log.Printf("⚠️ Пользователь не найден в дополнительном инбаунде")
		log.Printf("Возможные причины:")
		log.Printf("  1. Пользователь не создавал конфиг в дополнительном инбаунде")
		log.Printf("  2. Конфиг был удален или отключен")
		log.Printf("  3. Ошибка в логике синхронизации")
	}

	log.Printf("\n=== СИНХРОНИЗАЦИЯ ЗАВЕРШЕНА ===")
}
