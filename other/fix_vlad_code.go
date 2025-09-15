package main

import (
	"log"

	"bot/common"
	"bot/referralLink"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("=== ИСПРАВЛЕНИЕ КОДА VLAD ===")

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

	// ID пользователя Vlad
	userID := int64(873925520)
	log.Printf("Исправляем код для пользователя: %d", userID)

	// Получаем текущего пользователя
	user, err := common.GetUserByTelegramID(userID)
	if err != nil {
		log.Fatalf("❌ Пользователь с ID %d не найден: %v", userID, err)
	}

	log.Printf("✅ Пользователь найден: ID=%d, Name=%s, Current ReferralCode='%s'",
		user.TelegramID, user.FirstName, user.ReferralCode)

	// Исправляем код - убираем префикс ref_
	correctCode := "5035512654654"
	log.Printf("Устанавливаем правильный код: %s", correctCode)

	// Обновляем код в базе данных
	_, err = db.Exec(`
		UPDATE users 
		SET referral_code = $1, updated_at = NOW()
		WHERE telegram_id = $2
	`, correctCode, userID)

	if err != nil {
		log.Fatalf("❌ Ошибка обновления кода: %v", err)
	}

	log.Printf("✅ Код обновлен в базе данных")

	// Проверяем обновленного пользователя
	updatedUser, err := common.GetUserByTelegramID(userID)
	if err != nil {
		log.Fatalf("❌ Ошибка получения обновленного пользователя: %v", err)
	}

	log.Printf("✅ Обновленный пользователь: ID=%d, ReferralCode='%s'",
		updatedUser.TelegramID, updatedUser.ReferralCode)

	// Теперь тестируем реферальную систему
	service := referralLink.NewReferralService(db)

	// Проверяем валидность кода
	isValid := service.IsValidReferralCode(correctCode)
	log.Printf("Код '%s' валиден: %v", correctCode, isValid)

	// Пытаемся найти пригласившего
	referrer, err := service.GetReferrerByCode(correctCode)
	if err != nil {
		log.Printf("❌ Ошибка поиска пригласившего по коду '%s': %v", correctCode, err)
	} else {
		log.Printf("✅ Пригласивший найден: ID=%d, Name=%s, Balance=%.2f",
			referrer.TelegramID, referrer.FirstName, referrer.Balance)
	}

	// Тестируем обработку реферального перехода
	log.Println("\n=== ТЕСТ ОБРАБОТКИ РЕФЕРАЛЬНОГО ПЕРЕХОДА ===")
	referrerID := int64(5035512654) // Слава
	referredID := int64(873925520)  // Vlad

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

	// Проверяем обновленный баланс Vlad
	finalUser, err := common.GetUserByTelegramID(userID)
	if err != nil {
		log.Printf("❌ Ошибка получения финального пользователя: %v", err)
	} else {
		log.Printf("✅ Финальный баланс Vlad: %.2f₽", finalUser.Balance)
	}

	log.Println("=== ИСПРАВЛЕНИЕ ЗАВЕРШЕНО ===")
}
