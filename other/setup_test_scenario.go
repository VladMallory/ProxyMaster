package main

import (
	"log"

	"bot/common"
	"bot/referralLink"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("=== НАСТРОЙКА ТЕСТОВОГО СЦЕНАРИЯ ===")

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

	log.Printf("Настраиваем тестовый сценарий:")
	log.Printf("  Пригласивший: %d (Слава) - код: %s", referrerID, referralCode)
	log.Printf("  Приглашенный: %d (Vlad) - будет переходить по ссылке", referredID)

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

	// Тестируем обработку реферального перехода
	log.Println("\n=== ТЕСТ ОБРАБОТКИ РЕФЕРАЛЬНОГО ПЕРЕХОДА ===")
	err = service.ProcessReferralTransition(referrerID, referredID, referralCode)
	if err != nil {
		log.Printf("❌ Ошибка обработки перехода: %v", err)
	} else {
		log.Printf("✅ Реферальный переход обработан успешно")
	}

	// Тестируем начисление бонусов
	log.Println("\n=== ТЕСТ НАЧИСЛЕНИЯ БОНУСОВ ===")
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

	// Проверяем статистику
	log.Println("\n=== ПРОВЕРКА СТАТИСТИКИ ===")
	stats, err := service.GetReferralStats(referrerID)
	if err != nil {
		log.Printf("❌ Ошибка получения статистики: %v", err)
	} else {
		log.Printf("✅ Статистика для Славы:")
		log.Printf("   Всего рефералов: %d", stats.TotalReferrals)
		log.Printf("   Успешных: %d", stats.SuccessfulReferrals)
		log.Printf("   Общие заработки: %.2f", stats.TotalEarnings)
	}

	log.Println("\n=== ТЕСТОВЫЙ СЦЕНАРИЙ НАСТРОЕН ===")
	log.Printf("Теперь Vlad может переходить по ссылке: https://t.me/aquavpn13_bot?start=ref_%s", referralCode)
}
