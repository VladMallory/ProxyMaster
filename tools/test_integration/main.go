package main

import (
	"log"

	"bot/common"
)

func main() {
	log.Printf("=== ТЕСТ ИНТЕГРАЦИИ СИНХРОНИЗАЦИИ С ДОПОЛНИТЕЛЬНЫМ ИНБАУНДОМ ===")

	// Создаем тестового пользователя с активным основным конфигом
	testUser := &common.User{
		TelegramID:               888888888, // Тестовый ID
		Username:                 "integration_test_user",
		FirstName:                "Integration Test",
		Balance:                  100.0,
		HasActiveConfig:          true,
		ClientID:                 "test-main-client-id",
		Email:                    "888888888",
		SubID:                    "test-main-sub-id",
		ExpiryTime:               1761686400000, // Через 30 дней
		HasActiveSecondaryConfig: false,         // Пока нет дополнительного конфига
	}

	log.Printf("Тестовый пользователь:")
	log.Printf("  TelegramID: %d", testUser.TelegramID)
	log.Printf("  HasActiveConfig: %v", testUser.HasActiveConfig)
	log.Printf("  Email: %s", testUser.Email)
	log.Printf("  HasActiveSecondaryConfig: %v", testUser.HasActiveSecondaryConfig)
	log.Printf("  SecondaryEmail: %s", testUser.SecondaryEmail)

	// Проверяем HasAnyActiveConfig до синхронизации
	hasAnyConfigBefore := common.HasAnyActiveConfig(testUser)
	log.Printf("\n=== ДО СИНХРОНИЗАЦИИ ===")
	log.Printf("HasAnyActiveConfig: %v", hasAnyConfigBefore)

	// Выполняем синхронизацию с дополнительным инбаундом
	log.Printf("\n=== СИНХРОНИЗАЦИЯ С ДОПОЛНИТЕЛЬНЫМ ИНБАУНДОМ ===")
	err := common.SyncUserWithSecondaryPanel(testUser)
	if err != nil {
		log.Printf("❌ Ошибка синхронизации: %v", err)
		return
	}

	log.Printf("✅ Синхронизация завершена")

	// Проверяем результат
	log.Printf("\n=== ПОСЛЕ СИНХРОНИЗАЦИИ ===")
	log.Printf("HasActiveSecondaryConfig: %v", testUser.HasActiveSecondaryConfig)
	log.Printf("SecondaryEmail: %s", testUser.SecondaryEmail)
	log.Printf("SecondarySubID: %s", testUser.SecondarySubID)
	log.Printf("SecondaryClientID: %s", testUser.SecondaryClientID)

	// Проверяем HasAnyActiveConfig после синхронизации
	hasAnyConfigAfter := common.HasAnyActiveConfig(testUser)
	log.Printf("HasAnyActiveConfig: %v", hasAnyConfigAfter)

	// Проверяем URL конфигурации
	secondaryConfigURL := common.GetSecondaryConfigURL(testUser)
	log.Printf("SecondaryConfigURL: %s", secondaryConfigURL)

	// Проверяем, что дополнительный конфиг активен
	isSecondaryActive := common.IsSecondaryConfigActive(testUser)
	log.Printf("IsSecondaryConfigActive: %v", isSecondaryActive)

	log.Printf("\n=== РЕЗУЛЬТАТ ===")
	if testUser.HasActiveSecondaryConfig && testUser.SecondaryEmail == "888888888_1" {
		log.Printf("✅ ИНТЕГРАЦИЯ РАБОТАЕТ КОРРЕКТНО!")
		log.Printf("   - Основной конфиг: %s", testUser.Email)
		log.Printf("   - Дополнительный конфиг: %s", testUser.SecondaryEmail)
		log.Printf("   - HasAnyActiveConfig: %v", hasAnyConfigAfter)
	} else {
		log.Printf("❌ ИНТЕГРАЦИЯ НЕ РАБОТАЕТ!")
		log.Printf("   - HasActiveSecondaryConfig: %v", testUser.HasActiveSecondaryConfig)
		log.Printf("   - SecondaryEmail: %s", testUser.SecondaryEmail)
	}
}
