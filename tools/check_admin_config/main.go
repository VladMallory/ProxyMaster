package main

import (
	"encoding/json"
	"log"
	"time"

	"bot/common"
)

func main() {
	log.Println("=== ПРОВЕРКА КОНФИГА АДМИНИСТРАТОРА ===")

	common.InitGlobals()

	if err := common.InitPostgreSQL(); err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer common.DisconnectPostgreSQL()

	// Получаем пользователя из БД
	user, err := common.GetUserByTelegramID(873925520)
	if err != nil || user == nil {
		log.Fatalf("Пользователь не найден: %v", err)
	}

	log.Printf("Данные из БД:")
	log.Printf("  Email: %s", user.Email)
	log.Printf("  HasActiveConfig: %v", user.HasActiveConfig)
	log.Printf("  ExpiryTime: %d (%s)", user.ExpiryTime, time.UnixMilli(user.ExpiryTime).Format("2006-01-02 15:04:05"))
	log.Printf("  Balance: %.2f₽", user.Balance)

	// Авторизуемся в панели
	sessionCookie, err := common.Login()
	if err != nil {
		log.Fatalf("Ошибка авторизации: %v", err)
	}

	// Получаем inbound
	inbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Fatalf("Ошибка получения inbound: %v", err)
	}

	var settings common.Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Fatalf("Ошибка парсинга settings: %v", err)
	}

	// Ищем клиента в панели
	log.Println("\nПоиск конфига в панели...")
	for _, client := range settings.Clients {
		if client.Email == user.Email {
			log.Printf("\nКонфиг найден в панели:")
			log.Printf("  Email: %s", client.Email)
			log.Printf("  Enable: %v", client.Enable)
			log.Printf("  ExpiryTime: %d (%s)", client.ExpiryTime, time.UnixMilli(client.ExpiryTime).Format("2006-01-02 15:04:05"))
			log.Printf("  ID: %s", client.ID)
			log.Printf("  SubID: %s", client.SubID)

			// Проверяем текущее время
			now := time.Now()
			log.Printf("\nТекущее время: %s (%d мс)", now.Format("2006-01-02 15:04:05"), now.UnixMilli())

			if client.Enable {
				log.Println("\n✓ Конфиг ВКЛЮЧЕН в панели")
			} else {
				log.Println("\n✗ Конфиг ОТКЛЮЧЕН в панели")

				// Пытаемся включить
				log.Println("Попытка включить конфиг...")
				configManager := common.NewConfigManager(common.PANEL_URL, common.PANEL_USER, common.PANEL_PASS, common.INBOUND_ID)
				if err := configManager.EnableConfig(client.Email); err != nil {
					log.Printf("Ошибка включения: %v", err)
				} else {
					log.Println("✓ Конфиг успешно включен!")
				}
			}
			return
		}
	}

	log.Println("\n⚠ Конфиг НЕ НАЙДЕН в панели!")
}
