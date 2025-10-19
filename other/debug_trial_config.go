//go:build tools
// +build tools

package main

import (
	"encoding/json"
	"fmt"
	"log"

	"bot/common"
)

func main() {
	log.Printf("DEBUG_TRIAL: Начало отладки создания конфига для пробного периода")

	// Инициализируем подключение к базе данных
	if err := common.InitMongoDB(); err != nil {
		log.Fatalf("DEBUG_TRIAL: Ошибка инициализации БД: %v", err)
	}
	defer common.DisconnectMongoDB()

	// Создаем тестового пользователя
	user, err := common.GetOrCreateUser(999999999, "testuser", "Test", "User")
	if err != nil {
		log.Fatalf("DEBUG_TRIAL: Ошибка создания тестового пользователя: %v", err)
	}

	log.Printf("DEBUG_TRIAL: Тестовый пользователь создан: %+v", user)

	// Проверяем подключение к панели
	log.Printf("DEBUG_TRIAL: Проверка подключения к панели...")
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("DEBUG_TRIAL: ❌ Ошибка авторизации в панели: %v", err)
		return
	}
	log.Printf("DEBUG_TRIAL: ✅ Успешная авторизация в панели")

	// Получаем inbound
	log.Printf("DEBUG_TRIAL: Получение inbound ID=%d...", common.INBOUND_ID)
	inbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Printf("DEBUG_TRIAL: ❌ Ошибка получения inbound: %v", err)
		return
	}
	log.Printf("DEBUG_TRIAL: ✅ Inbound получен: ID=%d, Settings length=%d", inbound.ID, len(inbound.Settings))

	// Парсим настройки
	var settings common.Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Printf("DEBUG_TRIAL: ❌ Ошибка парсинга настроек: %v", err)
		return
	}
	log.Printf("DEBUG_TRIAL: ✅ Настройки распарсены: клиентов=%d", len(settings.Clients))

	// Проверяем, есть ли уже клиент с таким TelegramID
	existingClient := common.FindClientByTelegramID(settings.Clients, user.TelegramID)
	if existingClient != nil {
		log.Printf("DEBUG_TRIAL: ⚠️ Клиент уже существует: %+v", existingClient)
	} else {
		log.Printf("DEBUG_TRIAL: ✅ Клиент не найден, можно создавать нового")
	}

	// Пробуем создать конфиг для пробного периода
	log.Printf("DEBUG_TRIAL: Попытка создания конфига для пробного периода...")
	trialDays := common.TRIAL_BALANCE_AMOUNT / common.PRICE_PER_DAY
	log.Printf("DEBUG_TRIAL: Пробный период: %d дней (баланс=%d, цена за день=%d)", trialDays, common.TRIAL_BALANCE_AMOUNT, common.PRICE_PER_DAY)

	err = common.AddTrialClient(sessionCookie, user, trialDays)
	if err != nil {
		log.Printf("DEBUG_TRIAL: ❌ Ошибка создания конфига: %v", err)
		return
	}

	log.Printf("DEBUG_TRIAL: ✅ Конфиг успешно создан!")
	log.Printf("DEBUG_TRIAL: Пользователь после создания: %+v", user)

	// Проверяем URL конфига
	if user.SubID != "" {
		configURL := fmt.Sprintf("%s%s", common.CONFIG_BASE_URL, user.SubID)
		log.Printf("DEBUG_TRIAL: URL конфига: %s", configURL)
	} else {
		log.Printf("DEBUG_TRIAL: ❌ SubID пустой!")
	}
}
