package main

import (
	"database/sql"
	"log"
	"os"

	"bot/common"
	"bot/referralLink"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("=== ТЕСТ ИСПРАВЛЕННОЙ РЕФЕРАЛЬНОЙ СИСТЕМЫ ===")

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

	// Тест 1: Проверка конфигурации
	testConfiguration()

	// Тест 2: Создание тестовых пользователей
	referrerID, referredID := createTestUsers(db)

	// Тест 3: Генерация реферального кода
	referralCode := testGenerateReferralCode(db, referrerID)

	// Тест 4: Проверка валидности реферального кода
	testValidateReferralCode(db, referralCode)

	// Тест 5: Получение информации о пригласившем
	testGetReferrerByCode(db, referralCode)

	// Тест 6: Обработка реферального перехода
	testProcessReferralTransition(db, referrerID, referredID, referralCode)

	// Тест 7: Начисление бонусов
	testAwardReferralBonuses(db, referrerID, referredID, referralCode)

	// Тест 8: Получение статистики
	testGetReferralStats(db, referrerID)

	// Тест 9: Получение истории бонусов
	testGetReferralHistory(db, referrerID)

	// Тест 10: Получение информации о реферальной ссылке
	testGetReferralLinkInfo(db, referrerID)

	// Тест 11: Проверка формата ссылки
	testReferralLinkFormat(db, referrerID)

	log.Println("=== ВСЕ ТЕСТЫ ЗАВЕРШЕНЫ ===")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func testConfiguration() {
	log.Println("\n=== ТЕСТ 1: ПРОВЕРКА КОНФИГУРАЦИИ ===")

	log.Printf("REFERRAL_SYSTEM_ENABLED: %v", common.REFERRAL_SYSTEM_ENABLED)
	log.Printf("REFERRAL_BONUS_AMOUNT: %.2f", common.REFERRAL_BONUS_AMOUNT)
	log.Printf("REFERRAL_WELCOME_BONUS: %.2f", common.REFERRAL_WELCOME_BONUS)
	log.Printf("REFERRAL_LINK_BASE_URL: %s", common.REFERRAL_LINK_BASE_URL)
	log.Printf("REFERRAL_MIN_BALANCE_FOR_REF: %.2f", common.REFERRAL_MIN_BALANCE_FOR_REF)

	if !common.REFERRAL_SYSTEM_ENABLED {
		log.Println("❌ Реферальная система отключена!")
	} else {
		log.Println("✅ Реферальная система включена")
	}
}

func createTestUsers(db *sql.DB) (int64, int64) {
	log.Println("\n=== ТЕСТ 2: СОЗДАНИЕ ТЕСТОВЫХ ПОЛЬЗОВАТЕЛЕЙ ===")

	referrerID := int64(123456789)
	referredID := int64(987654321)

	// Создаем пригласившего
	_, err := db.Exec(`
		INSERT INTO users (telegram_id, username, first_name, balance, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (telegram_id) DO UPDATE SET
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			balance = EXCLUDED.balance,
			updated_at = NOW()
	`, referrerID, "test_referrer", "Test Referrer", 1000.0)

	if err != nil {
		log.Printf("❌ Ошибка создания пригласившего: %v", err)
	} else {
		log.Printf("✅ Пригласивший создан: ID=%d", referrerID)
	}

	// Создаем приглашенного
	_, err = db.Exec(`
		INSERT INTO users (telegram_id, username, first_name, balance, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (telegram_id) DO UPDATE SET
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			balance = EXCLUDED.balance,
			updated_at = NOW()
	`, referredID, "test_referred", "Test Referred", 0.0)

	if err != nil {
		log.Printf("❌ Ошибка создания приглашенного: %v", err)
	} else {
		log.Printf("✅ Приглашенный создан: ID=%d", referredID)
	}

	return referrerID, referredID
}

func testGenerateReferralCode(db *sql.DB, userID int64) string {
	log.Println("\n=== ТЕСТ 3: ГЕНЕРАЦИЯ РЕФЕРАЛЬНОГО КОДА ===")

	service := referralLink.NewReferralService(db)
	code, err := service.GenerateReferralCode(userID)

	if err != nil {
		log.Printf("❌ Ошибка генерации кода: %v", err)
		return ""
	}

	log.Printf("✅ Реферальный код сгенерирован: %s", code)
	return code
}

func testValidateReferralCode(db *sql.DB, code string) {
	log.Println("\n=== ТЕСТ 4: ПРОВЕРКА ВАЛИДНОСТИ РЕФЕРАЛЬНОГО КОДА ===")

	service := referralLink.NewReferralService(db)

	// Тестируем код с префиксом ref_
	codeWithPrefix := "ref_" + code
	isValidWithPrefix := service.IsValidReferralCode(codeWithPrefix)
	log.Printf("Код с префиксом '%s' валиден: %v", codeWithPrefix, isValidWithPrefix)

	// Тестируем код без префикса
	isValidWithoutPrefix := service.IsValidReferralCode(code)
	log.Printf("Код без префикса '%s' валиден: %v", code, isValidWithoutPrefix)

	if isValidWithPrefix || isValidWithoutPrefix {
		log.Printf("✅ Код валиден в любом формате")
	} else {
		log.Printf("❌ Код невалиден")
	}
}

func testGetReferrerByCode(db *sql.DB, code string) {
	log.Println("\n=== ТЕСТ 5: ПОЛУЧЕНИЕ ИНФОРМАЦИИ О ПРИГЛАСИВШЕМ ===")

	service := referralLink.NewReferralService(db)
	referrer, err := service.GetReferrerByCode(code)

	if err != nil {
		log.Printf("❌ Ошибка получения пригласившего: %v", err)
		return
	}

	log.Printf("✅ Пригласивший найден: ID=%d, Name=%s, Balance=%.2f",
		referrer.TelegramID, referrer.FirstName, referrer.Balance)
}

func testProcessReferralTransition(db *sql.DB, referrerID, referredID int64, code string) {
	log.Println("\n=== ТЕСТ 6: ОБРАБОТКА РЕФЕРАЛЬНОГО ПЕРЕХОДА ===")

	service := referralLink.NewReferralService(db)
	err := service.ProcessReferralTransition(referrerID, referredID, code)

	if err != nil {
		log.Printf("❌ Ошибка обработки перехода: %v", err)
	} else {
		log.Printf("✅ Реферальный переход обработан успешно")
	}
}

func testAwardReferralBonuses(db *sql.DB, referrerID, referredID int64, code string) {
	log.Println("\n=== ТЕСТ 7: НАЧИСЛЕНИЕ РЕФЕРАЛЬНЫХ БОНУСОВ ===")

	service := referralLink.NewReferralService(db)
	err := service.AwardReferralBonuses(referrerID, referredID, code)

	if err != nil {
		log.Printf("❌ Ошибка начисления бонусов: %v", err)
	} else {
		log.Printf("✅ Бонусы начислены успешно")
	}
}

func testGetReferralStats(db *sql.DB, userID int64) {
	log.Println("\n=== ТЕСТ 8: ПОЛУЧЕНИЕ СТАТИСТИКИ РЕФЕРАЛОВ ===")

	service := referralLink.NewReferralService(db)
	stats, err := service.GetReferralStats(userID)

	if err != nil {
		log.Printf("❌ Ошибка получения статистики: %v", err)
		return
	}

	log.Printf("✅ Статистика получена:")
	log.Printf("   Всего рефералов: %d", stats.TotalReferrals)
	log.Printf("   Успешных: %d", stats.SuccessfulReferrals)
	log.Printf("   Ожидающих: %d", stats.PendingReferrals)
	log.Printf("   Общие заработки: %.2f", stats.TotalEarnings)
}

func testGetReferralHistory(db *sql.DB, userID int64) {
	log.Println("\n=== ТЕСТ 9: ПОЛУЧЕНИЕ ИСТОРИИ БОНУСОВ ===")

	service := referralLink.NewReferralService(db)
	history, err := service.GetReferralHistory(userID, 10)

	if err != nil {
		log.Printf("❌ Ошибка получения истории: %v", err)
		return
	}

	log.Printf("✅ История бонусов получена (записей: %d):", len(history))
	for i, bonus := range history {
		log.Printf("   %d. %s: %.2f (тип: %s, код: %s)",
			i+1, bonus.Description, bonus.Amount, bonus.BonusType, bonus.ReferralCode)
	}
}

func testGetReferralLinkInfo(db *sql.DB, userID int64) {
	log.Println("\n=== ТЕСТ 10: ПОЛУЧЕНИЕ ИНФОРМАЦИИ О РЕФЕРАЛЬНОЙ ССЫЛКЕ ===")

	service := referralLink.NewReferralService(db)
	info, err := service.GetReferralLinkInfo(userID)

	if err != nil {
		log.Printf("❌ Ошибка получения информации о ссылке: %v", err)
		return
	}

	log.Printf("✅ Информация о ссылке получена:")
	log.Printf("   Код: %s", info.ReferralCode)
	log.Printf("   Ссылка: %s", info.ReferralLink)
	log.Printf("   Пользователь: %s (ID: %d)", info.FirstName, info.UserID)
	log.Printf("   Заработки: %.2f", info.Earnings)
	log.Printf("   Количество рефералов: %d", info.ReferralCount)
}

func testReferralLinkFormat(db *sql.DB, userID int64) {
	log.Println("\n=== ТЕСТ 11: ПРОВЕРКА ФОРМАТА ССЫЛКИ ===")

	service := referralLink.NewReferralService(db)
	info, err := service.GetReferralLinkInfo(userID)

	if err != nil {
		log.Printf("❌ Ошибка получения информации о ссылке: %v", err)
		return
	}

	expectedFormat := common.REFERRAL_LINK_BASE_URL + info.ReferralCode
	if info.ReferralLink == expectedFormat {
		log.Printf("✅ Формат ссылки правильный:")
		log.Printf("   Ожидаемый: %s", expectedFormat)
		log.Printf("   Полученный: %s", info.ReferralLink)
	} else {
		log.Printf("❌ Формат ссылки неправильный:")
		log.Printf("   Ожидаемый: %s", expectedFormat)
		log.Printf("   Полученный: %s", info.ReferralLink)
	}
}
