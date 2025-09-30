package main

import (
	"encoding/json"
	"log"
	"time"

	"bot/common"
)

const TEST_TELEGRAM_ID = 873925520

func main() {
	log.Println("=== ДИАГНОСТИКА И СБРОС КОНФИГА ===")

	common.InitGlobals()

	if err := common.InitPostgreSQL(); err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer common.DisconnectPostgreSQL()

	// Получаем пользователя из БД
	user, err := common.GetUserByTelegramID(TEST_TELEGRAM_ID)
	if err != nil || user == nil {
		log.Fatalf("Пользователь не найден: %v", err)
	}

	log.Println("\n========== ДАННЫЕ ИЗ БД ==========")
	log.Printf("Email: %s", user.Email)
	log.Printf("HasActiveConfig: %v", user.HasActiveConfig)
	log.Printf("ExpiryTime: %d", user.ExpiryTime)
	log.Printf("ExpiryTime (дата): %s", time.UnixMilli(user.ExpiryTime).Format("2006-01-02 15:04:05"))
	log.Printf("Balance: %.2f₽", user.Balance)
	log.Printf("ClientID: %s", user.ClientID)
	log.Printf("SubID: %s", user.SubID)

	// Авторизуемся в панели
	log.Println("\n========== АВТОРИЗАЦИЯ В ПАНЕЛИ ==========")
	sessionCookie, err := common.Login()
	if err != nil {
		log.Fatalf("Ошибка авторизации: %v", err)
	}
	log.Println("✓ Авторизация успешна")

	// Получаем inbound
	log.Println("\n========== ПОЛУЧЕНИЕ INBOUND ==========")
	inbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Fatalf("Ошибка получения inbound: %v", err)
	}

	var settings common.Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Fatalf("Ошибка парсинга settings: %v", err)
	}

	// Ищем клиента в панели
	log.Println("\n========== ПОИСК КЛИЕНТА В ПАНЕЛИ ==========")
	var foundClient *common.Client
	for i := range settings.Clients {
		if settings.Clients[i].Email == user.Email {
			foundClient = &settings.Clients[i]
			break
		}
	}

	if foundClient == nil {
		log.Fatalf("⚠ Клиент НЕ НАЙДЕН в панели!")
	}

	log.Println("✓ Клиент найден в панели")
	log.Printf("\n========== ТЕКУЩЕЕ СОСТОЯНИЕ КЛИЕНТА ==========")
	log.Printf("Email: %s", foundClient.Email)
	log.Printf("Enable: %v", foundClient.Enable)
	log.Printf("ID (UUID): %s", foundClient.ID)
	log.Printf("SubID: %s", foundClient.SubID)
	log.Printf("ExpiryTime: %d", foundClient.ExpiryTime)
	log.Printf("ExpiryTime (дата): %s", time.UnixMilli(foundClient.ExpiryTime).Format("2006-01-02 15:04:05"))
	log.Printf("Flow: %s", foundClient.Flow)
	log.Printf("TotalGB: %d", foundClient.TotalGB)
	log.Printf("Reset: %d", foundClient.Reset)
	log.Printf("LimitIP: %d", foundClient.LimitIP)

	// Проверяем текущее время
	now := time.Now()
	log.Printf("\n========== ПРОВЕРКА ВРЕМЕНИ ==========")
	log.Printf("Текущее время: %s", now.Format("2006-01-02 15:04:05"))
	log.Printf("Текущее время (мс): %d", now.UnixMilli())
	log.Printf("ExpiryTime клиента (мс): %d", foundClient.ExpiryTime)
	log.Printf("Разница: %d мс (%s)", foundClient.ExpiryTime-now.UnixMilli(),
		time.Duration(foundClient.ExpiryTime-now.UnixMilli())*time.Millisecond)

	if foundClient.ExpiryTime > now.UnixMilli() {
		log.Println("✓ Подписка АКТИВНА (не истекла)")
	} else {
		log.Println("✗ Подписка ИСТЕКЛА!")
	}

	// Дополнительные параметры
	log.Printf("\n========== ДОПОЛНИТЕЛЬНЫЕ ПАРАМЕТРЫ ==========")
	log.Printf("CreatedAt: %d (%s)", foundClient.CreatedAt, time.UnixMilli(foundClient.CreatedAt).Format("2006-01-02 15:04:05"))
	log.Printf("UpdatedAt: %d (%s)", foundClient.UpdatedAt, time.UnixMilli(foundClient.UpdatedAt).Format("2006-01-02 15:04:05"))

	// ЖЕСТКИЙ СБРОС КОНФИГА
	log.Printf("\n========== ЖЕСТКИЙ СБРОС КОНФИГА ==========")
	configManager := common.NewConfigManager(common.PANEL_URL, common.PANEL_USER, common.PANEL_PASS, common.INBOUND_ID)

	// Шаг 1: Отключаем конфиг
	log.Println("Шаг 1/2: Отключение конфига...")
	if err := configManager.DisableConfig(user.Email); err != nil {
		log.Printf("⚠ Ошибка отключения: %v", err)
	} else {
		log.Println("✓ Конфиг отключен")
	}

	// Ждем немного
	time.Sleep(2 * time.Second)

	// Шаг 2: Включаем конфиг обратно
	log.Println("Шаг 2/2: Включение конфига...")
	if err := configManager.EnableConfig(user.Email); err != nil {
		log.Printf("✗ Ошибка включения: %v", err)
	} else {
		log.Println("✓ Конфиг включен")
	}

	// Ждем обновления
	time.Sleep(2 * time.Second)

	// ПРОВЕРЯЕМ РЕЗУЛЬТАТ
	log.Printf("\n========== ПРОВЕРКА ПОСЛЕ СБРОСА ==========")
	inbound2, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Printf("⚠ Ошибка получения inbound: %v", err)
	} else {
		var settings2 common.Settings
		if err := json.Unmarshal([]byte(inbound2.Settings), &settings2); err != nil {
			log.Printf("⚠ Ошибка парсинга settings: %v", err)
		} else {
			for i := range settings2.Clients {
				if settings2.Clients[i].Email == user.Email {
					client := &settings2.Clients[i]
					log.Printf("Email: %s", client.Email)
					log.Printf("Enable: %v", client.Enable)
					log.Printf("ExpiryTime: %d (%s)", client.ExpiryTime,
						time.UnixMilli(client.ExpiryTime).Format("2006-01-02 15:04:05"))
					log.Printf("Reset: %d", client.Reset)
					log.Printf("TotalGB: %d", client.TotalGB)

					if client.Enable {
						log.Println("\n✓✓✓ КОНФИГ ВКЛЮЧЕН И ГОТОВ К РАБОТЕ ✓✓✓")
					} else {
						log.Println("\n✗✗✗ КОНФИГ ОТКЛЮЧЕН ✗✗✗")
					}
					break
				}
			}
		}
	}

	// Проверяем, может ли быть проблема с secondary inbound
	if common.SECONDARY_INBOUND_ENABLED {
		log.Printf("\n========== ПРОВЕРКА SECONDARY INBOUND ==========")
		log.Printf("SECONDARY_INBOUND_ID: %d", common.SECONDARY_INBOUND_ID)
		log.Println("Попытка включить конфиг в secondary inbound...")

		configManager2 := common.NewConfigManager(common.PANEL_URL, common.PANEL_USER, common.PANEL_PASS, common.SECONDARY_INBOUND_ID)
		if err := configManager2.EnableConfig(user.Email); err != nil {
			log.Printf("⚠ Ошибка включения в secondary inbound: %v", err)
		} else {
			log.Println("✓ Конфиг включен в secondary inbound")
		}
	}

	log.Println("\n========== ДИАГНОСТИКА ЗАВЕРШЕНА ==========")
	log.Println("\nПОПРОБУЙТЕ СЕЙЧАС:")
	log.Println("1. Обновите подписку в приложении")
	log.Println("2. Если не помогло - переимпортируйте конфиг")
	log.Println("3. Проверьте подключение")
}
