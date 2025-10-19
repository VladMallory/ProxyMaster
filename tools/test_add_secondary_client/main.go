package main

import (
	"log"

	"bot/common"
)

func main() {
	log.Printf("=== ТЕСТ ДОБАВЛЕНИЯ КЛИЕНТА В ДОПОЛНИТЕЛЬНЫЙ ИНБАУНД ===")
	log.Printf("SECONDARY_INBOUND_ID: %d", common.SECONDARY_INBOUND_ID)

	// Авторизация
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("❌ Ошибка авторизации: %v", err)
		return
	}
	log.Printf("✅ Авторизация успешна")

	// Создаем тестового пользователя
	testUser := &common.User{
		TelegramID: 999999999, // Тестовый ID
		Username:   "test_user",
		FirstName:  "Test",
		Balance:    100.0,
	}

	log.Printf("\n=== СОЗДАНИЕ ТЕСТОВОГО КЛИЕНТА ===")
	log.Printf("Тестовый пользователь: TelegramID=%d, Username=%s", testUser.TelegramID, testUser.Username)

	// Добавляем клиента в дополнительный инбаунд
	err = common.AddSecondaryClient(sessionCookie, testUser, 30)
	if err != nil {
		log.Printf("❌ Ошибка добавления клиента: %v", err)
		return
	}

	log.Printf("✅ Клиент добавлен успешно!")
	log.Printf("  SecondaryClientID: %s", testUser.SecondaryClientID)
	log.Printf("  SecondaryEmail: %s", testUser.SecondaryEmail)
	log.Printf("  SecondarySubID: %s", testUser.SecondarySubID)
	log.Printf("  HasActiveSecondaryConfig: %v", testUser.HasActiveSecondaryConfig)

	// Проверяем, что инбаунд все еще работает
	log.Printf("\n=== ПРОВЕРКА РАБОТОСПОСОБНОСТИ ===")
	secondaryInbound, err := common.GetSecondaryInbound(sessionCookie)
	if err != nil {
		log.Printf("❌ Инбаунд упал: %v", err)
		return
	}

	log.Printf("✅ Инбаунд работает! ID=%d, Protocol=%s, Port=%d",
		secondaryInbound.ID, secondaryInbound.Protocol, secondaryInbound.Port)

	log.Printf("\n=== ТЕСТ ЗАВЕРШЕН УСПЕШНО ===")
}
