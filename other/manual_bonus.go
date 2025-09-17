package main

import (
	"log"

	"bot/common"
	"bot/referralLink"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("=== РУЧНОЕ НАЧИСЛЕНИЕ БОНУСОВ ===")

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

	log.Printf("Начисляем бонусы:")
	log.Printf("  Пригласивший: %d (Слава)", referrerID)
	log.Printf("  Приглашенный: %d (Vlad)", referredID)
	log.Printf("  Код: %s", referralCode)

	service := referralLink.NewReferralService(db)

	// Проверяем текущие балансы
	referrer, err := common.GetUserByTelegramID(referrerID)
	if err != nil {
		log.Fatalf("❌ Ошибка получения пригласившего: %v", err)
	}
	log.Printf("Баланс пригласившего (Слава): %.2f₽", referrer.Balance)

	referred, err := common.GetUserByTelegramID(referredID)
	if err != nil {
		log.Fatalf("❌ Ошибка получения приглашенного: %v", err)
	}
	log.Printf("Баланс приглашенного (Vlad): %.2f₽", referred.Balance)

	// Начисляем бонусы
	log.Println("\n=== НАЧИСЛЕНИЕ БОНУСОВ ===")
	err = service.AwardReferralBonuses(referrerID, referredID, referralCode)
	if err != nil {
		log.Printf("❌ Ошибка начисления бонусов: %v", err)
	} else {
		log.Printf("✅ Бонусы начислены успешно")
	}

	// Проверяем обновленные балансы
	log.Println("\n=== ПРОВЕРКА ОБНОВЛЕННЫХ БАЛАНСОВ ===")
	referrerAfter, err := common.GetUserByTelegramID(referrerID)
	if err != nil {
		log.Printf("❌ Ошибка получения пригласившего после начисления: %v", err)
	} else {
		log.Printf("Баланс пригласившего (Слава) после: %.2f₽ (+%.2f₽)",
			referrerAfter.Balance, referrerAfter.Balance-referrer.Balance)
	}

	referredAfter, err := common.GetUserByTelegramID(referredID)
	if err != nil {
		log.Printf("❌ Ошибка получения приглашенного после начисления: %v", err)
	} else {
		log.Printf("Баланс приглашенного (Vlad) после: %.2f₽ (+%.2f₽)",
			referredAfter.Balance, referredAfter.Balance-referred.Balance)
	}

	// Проверяем историю бонусов
	log.Println("\n=== ПРОВЕРКА ИСТОРИИ БОНУСОВ ===")
	history, err := service.GetReferralHistory(referrerID, 10)
	if err != nil {
		log.Printf("❌ Ошибка получения истории бонусов: %v", err)
	} else {
		log.Printf("✅ История бонусов для Славы (записей: %d):", len(history))
		for i, bonus := range history {
			log.Printf("   %d. %s: %.2f (тип: %s, код: %s)",
				i+1, bonus.Description, bonus.Amount, bonus.BonusType, bonus.ReferralCode)
		}
	}

	history2, err := service.GetReferralHistory(referredID, 10)
	if err != nil {
		log.Printf("❌ Ошибка получения истории бонусов: %v", err)
	} else {
		log.Printf("✅ История бонусов для Vlad (записей: %d):", len(history2))
		for i, bonus := range history2 {
			log.Printf("   %d. %s: %.2f (тип: %s, код: %s)",
				i+1, bonus.Description, bonus.Amount, bonus.BonusType, bonus.ReferralCode)
		}
	}

	log.Println("=== НАЧИСЛЕНИЕ ЗАВЕРШЕНО ===")
}
