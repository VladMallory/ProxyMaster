package main

import (
	"log"

	"bot/common"
	"bot/referralLink"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("=== ТЕСТ ПРАВИЛЬНОЙ РЕФЕРАЛЬНОЙ ССЫЛКИ ===")

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

	// ID пригласившего и приглашенного
	referrerID := int64(873925520)  // Vlad
	referredID := int64(5035512654) // Слава

	// Правильный реферальный код
	correctCode := "873925520520"
	log.Printf("Тестируем правильный код: %s", correctCode)

	service := referralLink.NewReferralService(db)

	// Проверяем валидность правильного кода
	isValid := service.IsValidReferralCode(correctCode)
	log.Printf("Код '%s' валиден: %v", correctCode, isValid)

	// Проверяем валидность кода с префиксом
	codeWithPrefix := "ref_" + correctCode
	isValidWithPrefix := service.IsValidReferralCode(codeWithPrefix)
	log.Printf("Код '%s' валиден: %v", codeWithPrefix, isValidWithPrefix)

	// Пытаемся найти пригласившего по правильному коду
	referrer, err := service.GetReferrerByCode(correctCode)
	if err != nil {
		log.Printf("❌ Ошибка поиска пригласившего по коду '%s': %v", correctCode, err)
	} else {
		log.Printf("✅ Пригласивший найден: ID=%d, Name=%s, Balance=%.2f",
			referrer.TelegramID, referrer.FirstName, referrer.Balance)
	}

	// Тестируем обработку реферального перехода
	log.Println("\n=== ТЕСТ ОБРАБОТКИ РЕФЕРАЛЬНОГО ПЕРЕХОДА ===")
	err = service.ProcessReferralTransition(referrerID, referredID, correctCode)
	if err != nil {
		log.Printf("❌ Ошибка обработки перехода: %v", err)
	} else {
		log.Printf("✅ Реферальный переход обработан успешно")
	}

	// Тестируем начисление бонусов
	log.Println("\n=== ТЕСТ НАЧИСЛЕНИЯ БОНУСОВ ===")
	err = service.AwardReferralBonuses(referrerID, referredID, correctCode)
	if err != nil {
		log.Printf("❌ Ошибка начисления бонусов: %v", err)
	} else {
		log.Printf("✅ Бонусы начислены успешно")
	}

	// Проверяем статистику
	log.Println("\n=== ПРОВЕРКА СТАТИСТИКИ ===")
	stats, err := service.GetReferralStats(referrerID)
	if err != nil {
		log.Printf("❌ Ошибка получения статистики: %v", err)
	} else {
		log.Printf("✅ Статистика пригласившего:")
		log.Printf("   Всего рефералов: %d", stats.TotalReferrals)
		log.Printf("   Успешных: %d", stats.SuccessfulReferrals)
		log.Printf("   Общие заработки: %.2f", stats.TotalEarnings)
	}

	log.Println("=== ТЕСТ ЗАВЕРШЕН ===")
}
