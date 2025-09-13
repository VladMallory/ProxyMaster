package main

import (
	"log"

	"bot/common"
	"bot/referralLink"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("=== ОТЛАДКА ПРОБЛЕМЫ С БАЛАНСОМ ===")

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

	log.Printf("Проверяем пользователя 1: %d", userID1)
	user1, err := common.GetUserByTelegramID(userID1)
	if err != nil {
		log.Printf("❌ Пользователь %d не найден: %v", userID1, err)
	} else {
		log.Printf("✅ Пользователь 1: ID=%d, Name=%s, Balance=%.2f, ReferralCode='%s'",
			user1.TelegramID, user1.FirstName, user1.Balance, user1.ReferralCode)
	}

	log.Printf("Проверяем пользователя 2: %d", userID2)
	user2, err := common.GetUserByTelegramID(userID2)
	if err != nil {
		log.Printf("❌ Пользователь %d не найден: %v", userID2, err)
	} else {
		log.Printf("✅ Пользователь 2: ID=%d, Name=%s, Balance=%.2f, ReferralCode='%s'",
			user2.TelegramID, user2.FirstName, user2.Balance, user2.ReferralCode)
	}

	// Проверяем код из логов
	problemCode := "5035512654654"
	log.Printf("Проверяем проблемный код: %s", problemCode)

	service := referralLink.NewReferralService(db)

	// Проверяем валидность кода
	isValid := service.IsValidReferralCode(problemCode)
	log.Printf("Код '%s' валиден: %v", problemCode, isValid)

	// Проверяем валидность кода с префиксом
	codeWithPrefix := "ref_" + problemCode
	isValidWithPrefix := service.IsValidReferralCode(codeWithPrefix)
	log.Printf("Код '%s' валиден: %v", codeWithPrefix, isValidWithPrefix)

	// Пытаемся найти пригласившего
	referrer, err := service.GetReferrerByCode(problemCode)
	if err != nil {
		log.Printf("❌ Ошибка поиска пригласившего по коду '%s': %v", problemCode, err)
	} else {
		log.Printf("✅ Пригласивший найден: ID=%d, Name=%s, Balance=%.2f",
			referrer.TelegramID, referrer.FirstName, referrer.Balance)
	}

	// Проверяем историю бонусов для Vlad
	log.Println("\n=== ПРОВЕРКА ИСТОРИИ БОНУСОВ ДЛЯ VLAD ===")
	history, err := service.GetReferralHistory(userID2, 10)
	if err != nil {
		log.Printf("❌ Ошибка получения истории бонусов: %v", err)
	} else {
		log.Printf("✅ История бонусов для Vlad (записей: %d):", len(history))
		for i, bonus := range history {
			log.Printf("   %d. %s: %.2f (тип: %s, код: %s)",
				i+1, bonus.Description, bonus.Amount, bonus.BonusType, bonus.ReferralCode)
		}
	}

	// Проверяем статистику для Славы
	log.Println("\n=== ПРОВЕРКА СТАТИСТИКИ ДЛЯ СЛАВЫ ===")
	stats, err := service.GetReferralStats(userID1)
	if err != nil {
		log.Printf("❌ Ошибка получения статистики: %v", err)
	} else {
		log.Printf("✅ Статистика для Славы:")
		log.Printf("   Всего рефералов: %d", stats.TotalReferrals)
		log.Printf("   Успешных: %d", stats.SuccessfulReferrals)
		log.Printf("   Общие заработки: %.2f", stats.TotalEarnings)
	}

	log.Println("=== ОТЛАДКА ЗАВЕРШЕНА ===")
}
