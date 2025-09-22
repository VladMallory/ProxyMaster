package main

import (
	"log"

	"bot/common"
)

func main() {
	log.Printf("DEBUG_TRIAL_CONDITIONS: Проверка условий для пробного периода")

	// Инициализируем подключение к базе данных
	if err := common.InitMongoDB(); err != nil {
		log.Fatalf("DEBUG_TRIAL_CONDITIONS: Ошибка инициализации БД: %v", err)
	}
	defer common.DisconnectMongoDB()

	// Получаем всех пользователей
	users, err := common.GetAllUsers()
	if err != nil {
		log.Fatalf("DEBUG_TRIAL_CONDITIONS: Ошибка получения пользователей: %v", err)
	}

	log.Printf("DEBUG_TRIAL_CONDITIONS: Найдено пользователей: %d", len(users))

	for i, user := range users {
		log.Printf("DEBUG_TRIAL_CONDITIONS: ===== ПОЛЬЗОВАТЕЛЬ %d =====", i+1)
		log.Printf("DEBUG_TRIAL_CONDITIONS: TelegramID: %d", user.TelegramID)
		log.Printf("DEBUG_TRIAL_CONDITIONS: Username: %s", user.Username)
		log.Printf("DEBUG_TRIAL_CONDITIONS: FirstName: %s", user.FirstName)
		log.Printf("DEBUG_TRIAL_CONDITIONS: Balance: %.2f", user.Balance)
		log.Printf("DEBUG_TRIAL_CONDITIONS: HasActiveConfig: %v", user.HasActiveConfig)
		log.Printf("DEBUG_TRIAL_CONDITIONS: HasUsedTrial: %v", user.HasUsedTrial)
		log.Printf("DEBUG_TRIAL_CONDITIONS: ClientID: %s", user.ClientID)
		log.Printf("DEBUG_TRIAL_CONDITIONS: SubID: %s", user.SubID)
		log.Printf("DEBUG_TRIAL_CONDITIONS: ConfigCreatedAt: %v", user.ConfigCreatedAt)

		// Проверяем условия для пробного периода
		trialManager := common.NewTrialPeriodManager()
		canUseTrial := trialManager.CanUseTrial(&user)

		log.Printf("DEBUG_TRIAL_CONDITIONS: CanUseTrial: %v", canUseTrial)

		// Проверяем все условия из message_handler.go
		shouldOfferTrial := !user.HasActiveConfig && canUseTrial
		log.Printf("DEBUG_TRIAL_CONDITIONS: ShouldOfferTrial: %v", shouldOfferTrial)

		if !shouldOfferTrial {
			log.Printf("DEBUG_TRIAL_CONDITIONS: ❌ Пробный период НЕ будет предложен")
			if user.HasActiveConfig {
				log.Printf("DEBUG_TRIAL_CONDITIONS:   - Причина: HasActiveConfig = true")
			}
			if !canUseTrial {
				log.Printf("DEBUG_TRIAL_CONDITIONS:   - Причина: CanUseTrial = false (HasUsedTrial = %v)", user.HasUsedTrial)
			}
		} else {
			log.Printf("DEBUG_TRIAL_CONDITIONS: ✅ Пробный период БУДЕТ предложен")
		}

		log.Printf("DEBUG_TRIAL_CONDITIONS: =================================")
	}
}
