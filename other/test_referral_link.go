package main

import (
	"log"

	"bot/common"
	"bot/referralLink"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("=== ТЕСТ ПЕРЕХОДА ПО РЕФЕРАЛЬНОЙ ССЫЛКЕ ===")

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

	// ID пользователей
	referrerID := int64(5035512654) // Слава (пригласивший)
	referredID := int64(873925520)  // Vlad (приглашенный)
	referralCode := "5035512654654" // Код Славы

	log.Printf("Тестируем переход Vlad по реферальной ссылке:")
	log.Printf("  Пригласивший: %d (Слава) - код: %s", referrerID, referralCode)
	log.Printf("  Приглашенный: %d (Vlad) - переходит по ссылке", referredID)

	// Проверяем текущие балансы
	referrer, err := common.GetUserByTelegramID(referrerID)
	if err != nil {
		log.Fatalf("❌ Ошибка получения пригласившего: %v", err)
	}
	log.Printf("Баланс пригласившего (Слава) до: %.2f₽", referrer.Balance)

	referred, err := common.GetUserByTelegramID(referredID)
	if err != nil {
		log.Fatalf("❌ Ошибка получения приглашенного: %v", err)
	}
	log.Printf("Баланс приглашенного (Vlad) до: %.2f₽", referred.Balance)

	// Симулируем переход по реферальной ссылке
	log.Println("\n=== СИМУЛЯЦИЯ ПЕРЕХОДА ПО ССЫЛКЕ ===")

	// Создаем реферальный менеджер
	manager := referralLink.GlobalReferralManager

	// Проверяем, является ли это реферальным стартом
	referralText := "/start ref_" + referralCode
	log.Printf("Проверяем текст: '%s'", referralText)

	isReferralStart := manager.IsReferralStart(referralText)
	log.Printf("IsReferralStart('%s') = %v", referralText, isReferralStart)

	if isReferralStart {
		// Извлекаем реферальный код
		extractedCode := manager.ExtractReferralCode(referralText)
		log.Printf("Извлеченный код: '%s'", extractedCode)

		// Обрабатываем реферальный переход
		log.Printf("Обрабатываем реферальный переход...")
		manager.HandleStartCommand(0, referred, referralText)

		log.Printf("✅ Реферальный переход обработан")
	} else {
		log.Printf("❌ Текст не распознан как реферальный старт")
	}

	// Проверяем обновленные балансы
	log.Println("\n=== ПРОВЕРКА ОБНОВЛЕННЫХ БАЛАНСОВ ===")
	referrerAfter, err := common.GetUserByTelegramID(referrerID)
	if err != nil {
		log.Printf("❌ Ошибка получения пригласившего после перехода: %v", err)
	} else {
		log.Printf("Баланс пригласившего (Слава) после: %.2f₽ (+%.2f₽)",
			referrerAfter.Balance, referrerAfter.Balance-referrer.Balance)
	}

	referredAfter, err := common.GetUserByTelegramID(referredID)
	if err != nil {
		log.Printf("❌ Ошибка получения приглашенного после перехода: %v", err)
	} else {
		log.Printf("Баланс приглашенного (Vlad) после: %.2f₽ (+%.2f₽)",
			referredAfter.Balance, referredAfter.Balance-referred.Balance)
	}

	// Проверяем историю бонусов
	log.Println("\n=== ПРОВЕРКА ИСТОРИИ БОНУСОВ ===")
	service := referralLink.NewReferralService(db)

	history1, err := service.GetReferralHistory(referrerID, 10)
	if err != nil {
		log.Printf("❌ Ошибка получения истории бонусов для Славы: %v", err)
	} else {
		log.Printf("✅ История бонусов для Славы (записей: %d):", len(history1))
		for i, bonus := range history1 {
			log.Printf("   %d. %s: %.2f (тип: %s, код: %s)",
				i+1, bonus.Description, bonus.Amount, bonus.BonusType, bonus.ReferralCode)
		}
	}

	history2, err := service.GetReferralHistory(referredID, 10)
	if err != nil {
		log.Printf("❌ Ошибка получения истории бонусов для Vlad: %v", err)
	} else {
		log.Printf("✅ История бонусов для Vlad (записей: %d):", len(history2))
		for i, bonus := range history2 {
			log.Printf("   %d. %s: %.2f (тип: %s, код: %s)",
				i+1, bonus.Description, bonus.Amount, bonus.BonusType, bonus.ReferralCode)
		}
	}

	log.Println("=== ТЕСТ ЗАВЕРШЕН ===")
}
