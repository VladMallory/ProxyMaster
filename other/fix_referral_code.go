package main

import (
	"log"

	"bot/common"
	"bot/referralLink"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("=== ИСПРАВЛЕНИЕ РЕФЕРАЛЬНОГО КОДА ===")

	// Инициализируем глобальные переменные
	common.InitGlobals()

	// Инициализация базы данных через common
	err := common.InitPostgreSQL()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer common.DisconnectPostgreSQL()

	// Получаем подключение к БД
	db := common.GetDB()

	// Инициализация реферальной системы
	err = referralLink.InitReferralSystem(db, nil)
	if err != nil {
		log.Fatalf("Ошибка инициализации реферальной системы: %v", err)
	}

	// ID пользователя, для которого нужно сгенерировать код
	userID := int64(873925520)
	log.Printf("Генерируем реферальный код для пользователя: %d", userID)

	// Получаем информацию о пользователе
	user, err := common.GetUserByTelegramID(userID)
	if err != nil {
		log.Fatalf("❌ Пользователь с ID %d не найден: %v", userID, err)
	}

	log.Printf("✅ Пользователь найден: ID=%d, Name=%s, Current ReferralCode='%s'",
		user.TelegramID, user.FirstName, user.ReferralCode)

	// Генерируем реферальный код
	service := referralLink.NewReferralService(db)
	code, err := service.GenerateReferralCode(userID)
	if err != nil {
		log.Fatalf("❌ Ошибка генерации реферального кода: %v", err)
	}

	log.Printf("✅ Реферальный код сгенерирован: %s", code)

	// Проверяем, что код сохранился
	updatedUser, err := common.GetUserByTelegramID(userID)
	if err != nil {
		log.Fatalf("❌ Ошибка получения обновленного пользователя: %v", err)
	}

	log.Printf("✅ Обновленный пользователь: ID=%d, ReferralCode='%s'",
		updatedUser.TelegramID, updatedUser.ReferralCode)

	// Проверяем валидность кода
	isValid := service.IsValidReferralCode(code)
	log.Printf("Код '%s' валиден: %v", code, isValid)

	// Проверяем валидность кода с префиксом
	codeWithPrefix := "ref_" + code
	isValidWithPrefix := service.IsValidReferralCode(codeWithPrefix)
	log.Printf("Код '%s' валиден: %v", codeWithPrefix, isValidWithPrefix)

	// Получаем информацию о реферальной ссылке
	info, err := service.GetReferralLinkInfo(userID)
	if err != nil {
		log.Printf("❌ Ошибка получения информации о ссылке: %v", err)
	} else {
		log.Printf("✅ Информация о ссылке:")
		log.Printf("   Код: %s", info.ReferralCode)
		log.Printf("   Ссылка: %s", info.ReferralLink)
	}

	log.Println("=== ИСПРАВЛЕНИЕ ЗАВЕРШЕНО ===")
}
