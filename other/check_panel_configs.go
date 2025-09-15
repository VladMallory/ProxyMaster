package main

import (
	"encoding/json"
	"log"

	"bot/common"
)

func main() {
	log.Printf("CHECK_PANEL_CONFIGS: Проверка конфигов в панели")

	// Инициализируем подключение к базе данных
	if err := common.InitMongoDB(); err != nil {
		log.Fatalf("CHECK_PANEL_CONFIGS: Ошибка инициализации БД: %v", err)
	}
	defer common.DisconnectMongoDB()

	// Подключаемся к панели
	sessionCookie, err := common.Login()
	if err != nil {
		log.Fatalf("CHECK_PANEL_CONFIGS: Ошибка авторизации в панели: %v", err)
	}

	// Получаем inbound
	inbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Fatalf("CHECK_PANEL_CONFIGS: Ошибка получения inbound: %v", err)
	}

	log.Printf("CHECK_PANEL_CONFIGS: Inbound ID=%d, клиентов=%d", inbound.ID, len(inbound.Settings))

	// Парсим настройки
	var settings common.Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Fatalf("CHECK_PANEL_CONFIGS: Ошибка парсинга настроек: %v", err)
	}

	log.Printf("CHECK_PANEL_CONFIGS: Найдено клиентов в панели: %d", len(settings.Clients))

	for i, client := range settings.Clients {
		log.Printf("CHECK_PANEL_CONFIGS: ===== КЛИЕНТ %d =====", i+1)
		log.Printf("CHECK_PANEL_CONFIGS: Email: %s", client.Email)
		log.Printf("CHECK_PANEL_CONFIGS: ID: %s", client.ID)
		log.Printf("CHECK_PANEL_CONFIGS: SubID: %s", client.SubID)
		log.Printf("CHECK_PANEL_CONFIGS: Enable: %v", client.Enable)
		log.Printf("CHECK_PANEL_CONFIGS: ExpiryTime: %d", client.ExpiryTime)
		log.Printf("CHECK_PANEL_CONFIGS: CreatedAt: %d", client.CreatedAt)
		log.Printf("CHECK_PANEL_CONFIGS: =================================")
	}

	// Проверяем конкретных пользователей
	testUsers := []int64{873925520, 999999999}

	for _, telegramID := range testUsers {
		log.Printf("CHECK_PANEL_CONFIGS: Поиск конфига для пользователя %d", telegramID)
		client := common.FindClientByTelegramID(settings.Clients, telegramID)
		if client != nil {
			log.Printf("CHECK_PANEL_CONFIGS: ✅ Найден конфиг для %d: Email=%s, SubID=%s", telegramID, client.Email, client.SubID)
		} else {
			log.Printf("CHECK_PANEL_CONFIGS: ❌ Конфиг для %d не найден", telegramID)
		}
	}
}
