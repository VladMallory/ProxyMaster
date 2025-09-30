package main

import (
	"encoding/json"
	"log"
	"time"

	"bot/common"
)

func main() {
	log.Println("=== ЭКСТРЕННОЕ ВКЛЮЧЕНИЕ ВСЕХ КОНФИГОВ ===")

	// Инициализируем глобальные переменные
	common.InitGlobals()

	// Инициализируем базу данных
	if err := common.InitPostgreSQL(); err != nil {
		log.Fatalf("Ошибка инициализации базы данных: %v", err)
	}
	defer common.DisconnectPostgreSQL()

	// Авторизуемся в панели
	log.Println("Авторизация в панели...")
	sessionCookie, err := common.Login()
	if err != nil {
		log.Fatalf("Ошибка авторизации: %v", err)
	}
	log.Println("✓ Авторизация успешна")

	// Получаем inbound
	log.Println("Получение inbound...")
	targetInbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Fatalf("Ошибка получения inbound: %v", err)
	}

	if targetInbound == nil {
		log.Fatalf("Inbound с ID %d не найден", common.INBOUND_ID)
	}
	log.Println("✓ Inbound получен")

	// Парсим settings
	var settings common.Settings
	if err := json.Unmarshal([]byte(targetInbound.Settings), &settings); err != nil {
		log.Fatalf("Ошибка парсинга settings: %v", err)
	}

	log.Printf("Найдено клиентов в панели: %d\n", len(settings.Clients))

	// Получаем всех пользователей из базы данных
	users, err := common.GetAllUsers()
	if err != nil {
		log.Fatalf("Ошибка получения пользователей из БД: %v", err)
	}

	// Создаем карту пользователей для быстрого поиска
	userMap := make(map[string]*common.User)
	for i := range users {
		if users[i].Email != "" {
			userMap[users[i].Email] = &users[i]
		}
	}

	log.Printf("Найдено пользователей в БД: %d\n", len(users))

	// Проверяем и включаем отключенные конфиги
	now := time.Now()
	enabledCount := 0
	skippedCount := 0
	alreadyEnabledCount := 0

	for _, client := range settings.Clients {
		user, exists := userMap[client.Email]
		if !exists {
			log.Printf("⚠ Пропуск клиента %s - не найден в БД", client.Email)
			skippedCount++
			continue
		}

		// Проверяем, истекла ли подписка
		isExpired := false
		if client.ExpiryTime > 0 && client.ExpiryTime <= now.UnixMilli() {
			isExpired = true
		}

		// Если конфиг уже включен - пропускаем
		if client.Enable {
			alreadyEnabledCount++
			continue
		}

		// Если конфиг отключен, но подписка активна - включаем
		if !isExpired && user.HasActiveConfig {
			log.Printf("→ Включаем конфиг для %s (ID: %d) - подписка активна до %s",
				user.FirstName, user.TelegramID,
				time.UnixMilli(client.ExpiryTime).Format("2006-01-02 15:04"))

			// Создаем ConfigManager для выполнения операций
			configManager := common.NewConfigManager(common.PANEL_URL, common.PANEL_USER, common.PANEL_PASS, common.INBOUND_ID)
			err := configManager.EnableConfig(client.Email)
			if err != nil {
				log.Printf("✗ Ошибка включения конфига %s: %v", client.Email, err)
			} else {
				log.Printf("✓ Конфиг включен для %s", user.FirstName)
				enabledCount++
			}
		} else {
			if isExpired {
				log.Printf("⊘ Пропуск %s - подписка истекла %s",
					user.FirstName,
					time.UnixMilli(client.ExpiryTime).Format("2006-01-02 15:04"))
			} else {
				log.Printf("⊘ Пропуск %s - неактивна в БД (has_active_config=false)", user.FirstName)
			}
			skippedCount++
		}

		time.Sleep(100 * time.Millisecond) // Небольшая задержка между запросами
	}

	log.Println("\n=== ИТОГИ ===")
	log.Printf("Всего клиентов в панели: %d\n", len(settings.Clients))
	log.Printf("Уже включены: %d\n", alreadyEnabledCount)
	log.Printf("Включено конфигов: %d\n", enabledCount)
	log.Printf("Пропущено: %d\n", skippedCount)
	log.Println("\n✓ Операция завершена")
}
