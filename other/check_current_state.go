package main

import (
	"log"

	"bot/common"
	"bot/referralLink"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("=== ПРОВЕРКА ТЕКУЩЕГО СОСТОЯНИЯ ===")

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

	// Проверяем пользователей
	userID1 := int64(5035512654) // Слава
	userID2 := int64(873925520)  // Vlad

	log.Printf("Проверяем пользователя 1 (Слава): %d", userID1)
	user1, err := common.GetUserByTelegramID(userID1)
	if err != nil {
		log.Printf("❌ Пользователь %d не найден: %v", userID1, err)
	} else {
		log.Printf("✅ Пользователь 1: ID=%d, Name=%s, Balance=%.2f, ReferralCode='%s'",
			user1.TelegramID, user1.FirstName, user1.Balance, user1.ReferralCode)
	}

	log.Printf("Проверяем пользователя 2 (Vlad): %d", userID2)
	user2, err := common.GetUserByTelegramID(userID2)
	if err != nil {
		log.Printf("❌ Пользователь %d не найден: %v", userID2, err)
	} else {
		log.Printf("✅ Пользователь 2: ID=%d, Name=%s, Balance=%.2f, ReferralCode='%s'",
			user2.TelegramID, user2.FirstName, user2.Balance, user2.ReferralCode)
	}

	// Проверяем реферальный код
	service := referralLink.NewReferralService(db)
	referralCode := "5035512654654"

	log.Printf("Проверяем реферальный код: %s", referralCode)
	isValid := service.IsValidReferralCode(referralCode)
	log.Printf("Код '%s' валиден: %v", referralCode, isValid)

	// Пытаемся найти пригласившего
	referrer, err := service.GetReferrerByCode(referralCode)
	if err != nil {
		log.Printf("❌ Ошибка поиска пригласившего по коду '%s': %v", referralCode, err)
	} else {
		log.Printf("✅ Пригласивший найден: ID=%d, Name=%s, Balance=%.2f",
			referrer.TelegramID, referrer.FirstName, referrer.Balance)
	}

	// Проверяем историю бонусов
	log.Println("\n=== ПРОВЕРКА ИСТОРИИ БОНУСОВ ===")
	history1, err := service.GetReferralHistory(userID1, 10)
	if err != nil {
		log.Printf("❌ Ошибка получения истории бонусов для Славы: %v", err)
	} else {
		log.Printf("✅ История бонусов для Славы (записей: %d):", len(history1))
		for i, bonus := range history1 {
			log.Printf("   %d. %s: %.2f (тип: %s, код: %s)",
				i+1, bonus.Description, bonus.Amount, bonus.BonusType, bonus.ReferralCode)
		}
	}

	history2, err := service.GetReferralHistory(userID2, 10)
	if err != nil {
		log.Printf("❌ Ошибка получения истории бонусов для Vlad: %v", err)
	} else {
		log.Printf("✅ История бонусов для Vlad (записей: %d):", len(history2))
		for i, bonus := range history2 {
			log.Printf("   %d. %s: %.2f (тип: %s, код: %s)",
				i+1, bonus.Description, bonus.Amount, bonus.BonusType, bonus.ReferralCode)
		}
	}

	log.Println("=== ПРОВЕРКА ЗАВЕРШЕНА ===")
}
