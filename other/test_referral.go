package main

import (
	"log"

	"bot/common"
	"bot/referralLink"
)

func main() {
	log.Println("🧪 Тестирование реферальной системы")

	// Инициализируем базу данных
	if err := common.InitPostgreSQL(); err != nil {
		log.Fatalf("Ошибка инициализации базы данных: %v", err)
	}
	defer common.GetDB().Close()

	// Инициализируем реферальную систему
	if err := referralLink.InitReferralSystem(common.GetDB(), nil); err != nil {
		log.Fatalf("Ошибка инициализации реферальной системы: %v", err)
	}

	// Тестируем генерацию реферальной ссылки для пользователя 5035512654
	telegramID := int64(5035512654)
	log.Printf("Генерация реферальной ссылки для пользователя %d", telegramID)

	linkInfo, err := referralLink.GlobalReferralManager.GetReferralLinkInfo(telegramID)
	if err != nil {
		log.Fatalf("Ошибка получения реферальной ссылки: %v", err)
	}

	log.Printf("✅ Реферальная ссылка: %s", linkInfo.ReferralLink)
	log.Printf("✅ Реферальный код: %s", linkInfo.ReferralCode)

	// Тестируем парсинг реферальной ссылки
	testText := "/start ref_" + linkInfo.ReferralCode
	log.Printf("Тестирование парсинга: '%s'", testText)

	isReferral := referralLink.GlobalReferralManager.IsReferralStart(testText)
	log.Printf("IsReferralStart: %v", isReferral)

	if isReferral {
		extractedCode := referralLink.GlobalReferralManager.ExtractReferralCode(testText)
		log.Printf("Извлеченный код: '%s'", extractedCode)
	}
}
