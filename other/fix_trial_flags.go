package main

import (
	"encoding/json"
	"log"

	"bot/common"
)

func main() {
	log.Printf("FIX_TRIAL_FLAGS: Исправление флагов пробного периода")

	// Инициализируем подключение к базе данных
	if err := common.InitMongoDB(); err != nil {
		log.Fatalf("FIX_TRIAL_FLAGS: Ошибка инициализации БД: %v", err)
	}
	defer common.DisconnectMongoDB()

	// Подключаемся к панели
	sessionCookie, err := common.Login()
	if err != nil {
		log.Fatalf("FIX_TRIAL_FLAGS: Ошибка авторизации в панели: %v", err)
	}

	// Получаем inbound
	inbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Fatalf("FIX_TRIAL_FLAGS: Ошибка получения inbound: %v", err)
	}

	// Парсим настройки
	var settings common.Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Fatalf("FIX_TRIAL_FLAGS: Ошибка парсинга настроек: %v", err)
	}

	log.Printf("FIX_TRIAL_FLAGS: Найдено клиентов в панели: %d", len(settings.Clients))

	// Получаем всех пользователей
	users, err := common.GetAllUsers()
	if err != nil {
		log.Fatalf("FIX_TRIAL_FLAGS: Ошибка получения пользователей: %v", err)
	}

	log.Printf("FIX_TRIAL_FLAGS: Найдено пользователей в БД: %d", len(users))

	fixedCount := 0

	for _, user := range users {
		// Проверяем, есть ли конфиг в панели для этого пользователя
		client := common.FindClientByTelegramID(settings.Clients, user.TelegramID)

		// Если пользователь использовал пробный период, но у него нет конфига в панели
		if user.HasUsedTrial && client == nil {
			log.Printf("FIX_TRIAL_FLAGS: Исправление для пользователя %d (%s)", user.TelegramID, user.Username)
			log.Printf("FIX_TRIAL_FLAGS:   - HasUsedTrial: %v -> false", user.HasUsedTrial)
			log.Printf("FIX_TRIAL_FLAGS:   - HasActiveConfig: %v -> false", user.HasActiveConfig)

			// Сбрасываем флаги
			err := common.ResetTrialFlag(user.TelegramID)
			if err != nil {
				log.Printf("FIX_TRIAL_FLAGS: ❌ Ошибка сброса флага для %d: %v", user.TelegramID, err)
			} else {
				log.Printf("FIX_TRIAL_FLAGS: ✅ Флаг сброшен для %d", user.TelegramID)
				fixedCount++
			}
		} else if user.HasUsedTrial && client != nil {
			log.Printf("FIX_TRIAL_FLAGS: Пользователь %d (%s) имеет конфиг в панели - оставляем как есть", user.TelegramID, user.Username)
		} else if !user.HasUsedTrial {
			log.Printf("FIX_TRIAL_FLAGS: Пользователь %d (%s) не использовал пробный период - оставляем как есть", user.TelegramID, user.Username)
		}
	}

	log.Printf("FIX_TRIAL_FLAGS: Исправлено пользователей: %d", fixedCount)
}
