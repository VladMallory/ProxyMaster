package main

import (
	"encoding/json"
	"fmt"
	"log"

	"bot/common"
	"bot/common/env"
)

func main() {
	// Загружаем .env
	cfg := env.MustLoad()
	cfg.ApplyToCommon()
	common.InitGlobals()

	// Инициализация логгера трафика
	if err := common.InitTrafficLogger(); err != nil {
		log.Fatalf("Ошибка инициализации логгера трафика: %v", err)
	}
	defer common.CloseTrafficLogger()

	// Инициализация базы данных
	if err := common.InitPostgreSQL(); err != nil {
		log.Fatalf("Ошибка инициализации PostgreSQL: %v", err)
	}
	defer common.GetDatabasePG().Close()

	// Получаем пользователя
	user, err := common.GetUserByTelegramID(836470509)
	if err != nil {
		log.Fatalf("Ошибка получения пользователя: %v", err)
	}
	log.Printf("Пользователь найден: %+v", user)

	// Логинимся в панель
	sessionCookie, err := common.Login()
	if err != nil {
		log.Fatalf("Ошибка авторизации в панели: %v", err)
	}

	// Получение inbound settings
	inbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Fatalf("Ошибка получения inbound settings: %v", err)
	}

	// Парсинг client settings
	var settings common.Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Fatalf("Ошибка парсинга client settings: %v", err)
	}

	// Поиск клиента
	var clientFound *common.Client
	for i := range settings.Clients {
		if settings.Clients[i].Email == user.Email {
			clientFound = &settings.Clients[i]
			break
		}
	}

	if clientFound != nil {
		fmt.Printf("Найден клиент: Email=%s, Enable=%t, TotalGB=%d, ExpiryTime=%d\n",
			clientFound.Email, clientFound.Enable, clientFound.TotalGB, clientFound.ExpiryTime)

		// Поиск статистики по клиенту
		trafficStats, err := common.GetClientTrafficStats(sessionCookie)
		if err != nil {
			log.Fatalf("Ошибка получения статистики трафика: %v", err)
		}

		var clientStats *common.TrafficStats
		for i := range trafficStats {
			if trafficStats[i].Email == user.Email {
				clientStats = &trafficStats[i]
				break
			}
		}

		if clientStats != nil {
			fmt.Printf("Статистика клиента: Up=%d, Down=%d, Total=%d\n",
				clientStats.Up, clientStats.Down, clientStats.Total)
		} else {
			fmt.Println("Статистика для клиента не найдена.")
		}

	} else {
		fmt.Println("Клиент не найден.")
	}
}
