package main

import (
	"log"
	"time"

	"./common"
)

func main() {
	log.Println("=== ТЕСТ МУЛЬТИПОДПИСОК ===")

	// Инициализируем конфигурацию
	common.Init()

	// Инициализируем базу данных
	if err := common.InitDB(); err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer common.CloseDB()

	// Тестируем получение доступных серверов
	log.Println("\n1. Тестирование получения доступных серверов...")
	servers, err := common.GetAvailableServers()
	if err != nil {
		log.Printf("Ошибка получения серверов: %v", err)
	} else {
		log.Printf("✅ Найдено серверов: %d", len(servers))
		for _, server := range servers {
			log.Printf("   - %s %s (%s) - %s", server.Flag, server.Name, server.Country, server.Protocol)
		}
	}

	// Тестируем создание состояния выбора серверов
	log.Println("\n2. Тестирование создания состояния выбора...")
	testUserID := int64(12345)
	state := common.ServerSelectionState{
		UserID:     testUserID,
		Selected:   []string{},
		MaxServers: common.MULTI_SUBSCRIPTION_MAX_SERVERS,
		Step:       "select",
	}

	err = common.SaveServerSelectionState(state)
	if err != nil {
		log.Printf("Ошибка сохранения состояния: %v", err)
	} else {
		log.Println("✅ Состояние выбора сохранено")
	}

	// Тестируем получение состояния
	log.Println("\n3. Тестирование получения состояния...")
	retrievedState, err := common.GetServerSelectionState(testUserID)
	if err != nil {
		log.Printf("Ошибка получения состояния: %v", err)
	} else {
		log.Printf("✅ Состояние получено: пользователь %d, шаг %s", retrievedState.UserID, retrievedState.Step)
	}

	// Тестируем создание мультиподписки (если есть серверы)
	if len(servers) > 0 {
		log.Println("\n4. Тестирование создания мультиподписки...")

		// Выбираем первые 2 сервера
		selectedServers := []string{}
		for i, server := range servers {
			if i >= 2 { // Максимум 2 сервера для теста
				break
			}
			selectedServers = append(selectedServers, server.ID)
		}

		log.Printf("Выбранные серверы: %v", selectedServers)

		// Создаем мультиподписку
		subscription, err := common.CreateMultiSubscription(testUserID, selectedServers)
		if err != nil {
			log.Printf("Ошибка создания мультиподписки: %v", err)
		} else {
			log.Printf("✅ Мультиподписка создана: %s", subscription.ID)
			log.Printf("   URL: %s", subscription.SubscriptionURL)
			log.Printf("   Серверов: %d", len(subscription.Servers))
			log.Printf("   Активна: %v", subscription.IsActive)
			log.Printf("   Истекает: %s", time.Unix(subscription.ExpiryTime, 0).Format("2006-01-02 15:04:05"))
		}

		// Тестируем получение мультиподписки
		log.Println("\n5. Тестирование получения мультиподписки...")
		if subscription != nil {
			retrievedSub, err := common.GetUserMultiSubscription(testUserID)
			if err != nil {
				log.Printf("Ошибка получения мультиподписки: %v", err)
			} else {
				log.Printf("✅ Мультиподписка получена: %s", retrievedSub.ID)
				log.Printf("   Серверов: %d", len(retrievedSub.Servers))
			}
		}
	} else {
		log.Println("\n4. Пропуск теста создания мультиподписки (нет серверов)")
	}

	// Тестируем очистку состояний
	log.Println("\n6. Тестирование очистки состояний...")
	err = common.CleanupExpiredServerSelectionStates()
	if err != nil {
		log.Printf("Ошибка очистки состояний: %v", err)
	} else {
		log.Println("✅ Очистка состояний выполнена")
	}

	// Тестируем настройки
	log.Println("\n7. Настройки мультиподписок:")
	log.Printf("   Включены: %v", common.MULTI_SUBSCRIPTION_ENABLED)
	log.Printf("   Максимум серверов: %d", common.MULTI_SUBSCRIPTION_MAX_SERVERS)
	log.Printf("   Базовый URL: %s", common.MULTI_SUBSCRIPTION_BASE_URL)
	log.Printf("   Интервал очистки: %d мин", common.MULTI_SUBSCRIPTION_CLEANUP_INTERVAL)
	log.Printf("   ID инбаунда: %d", common.MULTI_SERVER_INBOUND_ID)
	log.Printf("   Автосоздание клиентов: %v", common.MULTI_SERVER_AUTO_CREATE_CLIENTS)
	log.Printf("   Проверка существующих: %v", common.MULTI_SERVER_CHECK_EXISTING)
	log.Printf("   Дней действия: %d", common.MULTI_SERVER_DEFAULT_EXPIRY_DAYS)

	log.Println("\n=== ТЕСТ ЗАВЕРШЕН ===")
}
