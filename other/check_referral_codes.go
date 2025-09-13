package main

import (
	"log"

	"bot/common"
	"bot/referralLink"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("=== ПРОВЕРКА РЕФЕРАЛЬНЫХ КОДОВ В БД ===")

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

	// Проверяем код из логов
	problemCode := "REF873925520198"
	log.Printf("Проверяем проблемный код: %s", problemCode)

	service := referralLink.NewReferralService(db)

	// Проверяем валидность кода
	isValid := service.IsValidReferralCode(problemCode)
	log.Printf("Код '%s' валиден: %v", problemCode, isValid)

	// Пытаемся найти пригласившего
	referrer, err := service.GetReferrerByCode(problemCode)
	if err != nil {
		log.Printf("❌ Ошибка поиска пригласившего по коду '%s': %v", problemCode, err)
	} else {
		log.Printf("✅ Пригласивший найден: ID=%d, Name=%s", referrer.TelegramID, referrer.FirstName)
	}

	// Проверяем код с префиксом ref_
	codeWithPrefix := "ref_" + problemCode
	log.Printf("Проверяем код с префиксом: %s", codeWithPrefix)

	isValidWithPrefix := service.IsValidReferralCode(codeWithPrefix)
	log.Printf("Код '%s' валиден: %v", codeWithPrefix, isValidWithPrefix)

	// Пытаемся найти пригласившего с префиксом
	referrerWithPrefix, err := service.GetReferrerByCode(codeWithPrefix)
	if err != nil {
		log.Printf("❌ Ошибка поиска пригласившего по коду '%s': %v", codeWithPrefix, err)
	} else {
		log.Printf("✅ Пригласивший найден: ID=%d, Name=%s", referrerWithPrefix.TelegramID, referrerWithPrefix.FirstName)
	}

	// Проверяем, есть ли пользователь с таким ID
	userID := int64(873925520)
	log.Printf("Проверяем пользователя с ID: %d", userID)

	user, err := common.GetUserByTelegramID(userID)
	if err != nil {
		log.Printf("❌ Пользователь с ID %d не найден: %v", userID, err)
	} else {
		log.Printf("✅ Пользователь найден: ID=%d, Name=%s, ReferralCode=%s",
			user.TelegramID, user.FirstName, user.ReferralCode)
	}

	log.Println("=== ПРОВЕРКА ЗАВЕРШЕНА ===")
}
