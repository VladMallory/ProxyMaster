package main

import (
	"log"

	"bot/common"
	"bot/referralLink"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("=== ФИНАЛЬНЫЙ ТЕСТ РЕФЕРАЛЬНОЙ СИСТЕМЫ ===")

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

	log.Println("✅ Реферальная система инициализирована")

	// Создаем нового пользователя для теста
	userID := int64(999888777)

	// Удаляем пользователя если существует
	db.Exec(`DELETE FROM users WHERE telegram_id = $1`, userID)

	// Создаем нового пользователя
	_, err = db.Exec(`
		INSERT INTO users (telegram_id, username, first_name, balance, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, userID, "test_user_new", "Test User New", 1000.0)

	if err != nil {
		log.Printf("❌ Ошибка создания пользователя: %v", err)
		return
	}

	log.Printf("✅ Новый пользователь создан: ID=%d", userID)

	// Тестируем генерацию реферального кода
	service := referralLink.NewReferralService(db)
	code, err := service.GenerateReferralCode(userID)

	if err != nil {
		log.Printf("❌ Ошибка генерации кода: %v", err)
		return
	}

	log.Printf("✅ Реферальный код сгенерирован: %s", code)

	// Тестируем получение информации о ссылке
	info, err := service.GetReferralLinkInfo(userID)
	if err != nil {
		log.Printf("❌ Ошибка получения информации о ссылке: %v", err)
		return
	}

	log.Printf("✅ Информация о ссылке получена:")
	log.Printf("   Код: %s", info.ReferralCode)
	log.Printf("   Ссылка: %s", info.ReferralLink)
	log.Printf("   Ожидаемая ссылка: %s", common.REFERRAL_LINK_BASE_URL+info.ReferralCode)

	// Проверяем, что ссылка создается правильно
	expectedLink := common.REFERRAL_LINK_BASE_URL + info.ReferralCode
	if info.ReferralLink == expectedLink {
		log.Printf("✅ Ссылка создается правильно!")
		log.Printf("   Формат: https://t.me/aquavpn13_bot?start=ref_XXXXX")
		log.Printf("   Без двойных префиксов!")
	} else {
		log.Printf("❌ Ссылка создается неправильно:")
		log.Printf("   Ожидаемая: %s", expectedLink)
		log.Printf("   Полученная: %s", info.ReferralLink)
	}

	// Тестируем валидацию кода
	log.Printf("\n=== ТЕСТ ВАЛИДАЦИИ КОДА ===")

	// Тест с полной ссылкой
	fullCode := "ref_" + info.ReferralCode
	isValidFull := service.IsValidReferralCode(fullCode)
	log.Printf("Код с полной ссылкой '%s' валиден: %v", fullCode, isValidFull)

	// Тест с кодом без префикса
	isValidCode := service.IsValidReferralCode(info.ReferralCode)
	log.Printf("Код без префикса '%s' валиден: %v", info.ReferralCode, isValidCode)

	if isValidFull && isValidCode {
		log.Printf("✅ Валидация работает правильно!")
	} else {
		log.Printf("❌ Проблема с валидацией кода")
	}

	log.Println("\n=== ТЕСТ ЗАВЕРШЕН ===")
}
