package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"bot/common"
	"bot/payments"
	"bot/payments/promo"
	"bot/powerOff"
	"bot/referralLink"
	"bot/services"
	"bot/telegram_bot"
)

// InitializeApp инициализирует приложение
func InitializeApp() {
	log.Printf("APP: Инициализация приложения")

	// Запускаем HTTP сервер для обслуживания redirect файлов и API
	go func() {
		// Обработчик для старого redirect.html (для обратной совместимости)
		http.HandleFunc("/redirect.html", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("HTTP_SERVER: Redirect request: %s", r.URL.String())
			http.ServeFile(w, r, "redirect.html")
		})

		// Обработчик для redirect_happ.html
		http.HandleFunc("/redirect_happ.html", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("HTTP_SERVER: Happ redirect request: %s", r.URL.String())
			http.ServeFile(w, r, "importRedirect/redirect_happ.html")
		})

		// Обработчик для redirect_happ_test.html (тестовый файл для отладки Android)
		http.HandleFunc("/redirect_happ_test.html", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("HTTP_SERVER: Happ TEST redirect request: %s", r.URL.String())
			http.ServeFile(w, r, "importRedirect/redirect_happ_test.html")
		})

		// Обработчик для redirect_v2raytun.html
		http.HandleFunc("/redirect_v2raytun.html", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("HTTP_SERVER: v2raytun redirect request: %s", r.URL.String())
			http.ServeFile(w, r, "importRedirect/redirect_v2raytun.html")
		})

		// Обработчик для redirect_v2raytun_test.html (тестовый файл для отладки Android)
		http.HandleFunc("/redirect_v2raytun_test.html", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("HTTP_SERVER: v2raytun TEST redirect request: %s", r.URL.String())
			http.ServeFile(w, r, "importRedirect/redirect_v2raytun_test.html")
		})

		// Обработчик для callback-ов ЮКассы
		http.HandleFunc("/yukassa/callback", handleYukassaCallback)

		// ===== API ENDPOINT ДЛЯ ПОЛУЧЕНИЯ ДАННЫХ ПОДПИСКИ =====
		// Этот endpoint решает проблему с импортом подписок на Android
		// Проблема: CORS блокирует прямые запросы к серверу конфигураций
		// Решение: проксируем запросы через наш сервер
		// Использование: /api/subscription?url=SUBSCRIPTION_URL
		http.HandleFunc("/api/subscription", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("HTTP_SERVER: API subscription request: %s", r.URL.String())

			// Получаем URL подписки из параметров запроса
			url := r.URL.Query().Get("url")
			if url == "" {
				http.Error(w, "URL parameter is required", http.StatusBadRequest)
				return
			}

			// Делаем запрос к серверу конфигураций
			// Сервер возвращает base64-кодированные VLESS/VMESS конфигурации
			resp, err := http.Get(url)
			if err != nil {
				log.Printf("HTTP_SERVER: Error fetching subscription data: %v", err)
				http.Error(w, "Failed to fetch subscription data", http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			// Читаем ответ сервера
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("HTTP_SERVER: Error reading subscription data: %v", err)
				http.Error(w, "Failed to read subscription data", http.StatusInternalServerError)
				return
			}

			// Возвращаем данные как есть (base64-кодированные конфигурации)
			// CORS заголовки нужны для работы с JavaScript в браузере
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Write(body)
		})

		// API для мультиподписок
		// Использование: /api/multi-subscription?id=SUBSCRIPTION_ID
		http.HandleFunc("/api/multi-subscription", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("HTTP_SERVER: API multi-subscription request: %s", r.URL.String())

			// Получаем ID мультиподписки из параметров запроса
			subscriptionID := r.URL.Query().Get("id")
			if subscriptionID == "" {
				http.Error(w, "ID parameter is required", http.StatusBadRequest)
				return
			}

			// Получаем мультиподписку из базы данных
			subscription, err := getMultiSubscriptionByID(subscriptionID)
			if err != nil {
				log.Printf("HTTP_SERVER: Error getting multi-subscription %s: %v", subscriptionID, err)
				http.Error(w, "Multi-subscription not found", http.StatusNotFound)
				return
			}

			if subscription == nil || !subscription.IsActive {
				log.Printf("HTTP_SERVER: Multi-subscription %s not found or inactive", subscriptionID)
				http.Error(w, "Multi-subscription not found or inactive", http.StatusNotFound)
				return
			}

			// Генерируем конфигурации для всех серверов
			configs, err := generateMultiSubscriptionConfigs(subscription)
			if err != nil {
				log.Printf("HTTP_SERVER: Error generating multi-subscription configs: %v", err)
				http.Error(w, "Failed to generate configurations", http.StatusInternalServerError)
				return
			}

			// Возвращаем конфигурации
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Write([]byte(configs))
		})

		log.Printf("HTTP_SERVER: Запуск HTTP сервера на порту 8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Printf("HTTP_SERVER: Ошибка запуска сервера: %v", err)
		}
	}()

	// Восстанавливаем базу данных из последнего бэкапа
	log.Printf("APP: Запуск восстановления базы данных")
	if err := common.RestoreMongoDB(); err != nil {
		log.Fatal("APP: Ошибка восстановления базы данных:", err)
	}
	log.Printf("APP: Восстановление базы данных завершено")

	// Подключение к базе данных (теперь PostgreSQL)
	log.Printf("APP: Инициализация базы данных")
	if err := common.InitMongoDB(); err != nil {
		log.Fatal("APP: Ошибка инициализации базы данных:", err)
	}
	log.Printf("APP: База данных успешно инициализирована")

	// Запускаем сервисы
	log.Printf("APP: Запуск сервиса периодического бэкапа")
	services.StartPeriodicBackup()

	// Запускаем мониторинг трафика
	log.Printf("APP: Запуск мониторинга трафика")
	common.StartTrafficMonitoring()

	log.Printf("APP: Инициализация приложения завершена")
}

// handleYukassaCallback обрабатывает callback от ЮКассы
func handleYukassaCallback(w http.ResponseWriter, r *http.Request) {
	log.Printf("YUKASSA_CALLBACK: Получен callback от ЮКассы")

	// Проверяем метод запроса
	if r.Method != http.MethodPost {
		log.Printf("YUKASSA_CALLBACK: Неверный метод запроса: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("YUKASSA_CALLBACK: Ошибка чтения тела запроса: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("YUKASSA_CALLBACK: Получено тело запроса: %s", string(body))

	// Старый код ЮКассы удален - теперь используем Telegram Bot API
	log.Printf("YUKASSA_CALLBACK: Получен callback от старого API ЮКассы - игнорируем")

	// Возвращаем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]string{"status": "ok"}
	json.NewEncoder(w).Encode(response)

	log.Printf("YUKASSA_CALLBACK: Callback успешно обработан")
}

// StartBot запускает Telegram бота
func StartBot(token string) {
	log.Printf("APP: Запуск Telegram бота")

	bot, err := telegram_bot.NewBot(token)
	if err != nil {
		log.Fatal("APP: Ошибка инициализации бота:", err)
	}

	// Настраиваем команды бота
	if err := telegram_bot.SetBotCommands(bot.API); err != nil {
		log.Printf("APP: Ошибка настройки команд бота: %v", err)
	}

	// Сохраняем бот в глобальной переменной для IP Ban сервиса
	common.GlobalBot = bot.API
	log.Printf("APP: Бот сохранен в глобальной переменной для IP Ban сервиса")

	// Инициализируем платежную систему
	log.Printf("APP: Инициализация платежной системы")
	if err := payments.InitializePaymentManager(bot.API); err != nil {
		log.Printf("APP: Ошибка инициализации платежной системы: %v", err)
		log.Printf("APP: Платежи будут недоступны")
	} else {
		log.Printf("APP: Платежная система успешно инициализирована")

		// Регистрируем веб-хуки для платежной системы
		mux := http.DefaultServeMux
		payments.RegisterWebhookRoutes(mux, payments.GlobalPaymentManager)
		log.Printf("APP: Веб-хуки платежной системы зарегистрированы")

		// Проверяем необработанные платежи при запуске
		if payments.GlobalPaymentManager != nil {
			go func() {
				// Ждем 30 секунд после запуска, затем проверяем
				time.Sleep(30 * time.Second)
				log.Printf("APP: Проверка необработанных платежей при запуске")
				// Здесь будет вызов проверки необработанных платежей
			}()
		}
	}

	// Инициализируем систему безопасного выключения
	log.Printf("APP: Инициализация системы безопасного выключения")
	if err := powerOff.InitializePowerOffSystem(); err != nil {
		log.Printf("APP: Ошибка инициализации системы выключения: %v", err)
		log.Printf("APP: Система выключения будет недоступна")
	} else {
		log.Printf("APP: Система безопасного выключения успешно инициализирована")
	}

	// Инициализируем систему промокодов (независимо от платежной системы)
	log.Printf("APP: Инициализация системы промокодов")
	if err := promo.InitializePromoManager(); err != nil {
		log.Printf("APP: Ошибка инициализации системы промокодов: %v", err)
		log.Printf("APP: Промокоды будут недоступны")
	} else {
		log.Printf("APP: Система промокодов успешно инициализирована")
	}

	// Инициализируем реферальную систему
	log.Printf("APP: Инициализация реферальной системы")
	if err := referralLink.InitReferralSystem(common.GetDB(), bot.API); err != nil {
		log.Printf("APP: Ошибка инициализации реферальной системы: %v", err)
		log.Printf("APP: Реферальная система будет недоступна")
	} else {
		log.Printf("APP: Реферальная система успешно инициализирована")
	}

	// Запускаем систему уведомлений о подписке
	if common.NOTIFICATION_ENABLED {
		log.Printf("APP: Запуск системы уведомлений о подписке")
		notificationManager := telegram_bot.NewNotificationManager(bot.API)
		go notificationManager.StartNotificationScheduler()
		log.Printf("APP: Система уведомлений о подписке запущена")
	} else {
		log.Printf("APP: Система уведомлений о подписке отключена в конфигурации")
	}

	bot.Start()
}

// getMultiSubscriptionByID получает мультиподписку по ID
func getMultiSubscriptionByID(subscriptionID string) (*common.MultiSubscription, error) {
	log.Printf("GET_MULTI_SUBSCRIPTION_BY_ID: Получение мультиподписки %s", subscriptionID)

	query := `
		SELECT ms.id, ms.user_id, ms.subscription_url, ms.is_active, ms.created_at, ms.expiry_time,
		       COALESCE(
		           jsonb_agg(
		               jsonb_build_object(
		                   'id', s.id,
		                   'name', s.name,
		                   'country', s.country,
		                   'country_code', s.country_code,
		                   'flag', s.flag,
		                   'inbound_id', s.inbound_id,
		                   'config_url', s.config_url,
		                   'json_url', s.json_url,
		                   'protocol', s.protocol,
		                   'transport', s.transport,
		                   'enabled', s.enabled,
		                   'priority', s.priority
		               ) ORDER BY s.priority DESC, s.name
		           ) FILTER (WHERE s.id IS NOT NULL),
		           '[]'::jsonb
		       ) as servers
		FROM multi_subscriptions ms
		LEFT JOIN multi_subscription_servers mss ON ms.id = mss.subscription_id
		LEFT JOIN multi_servers s ON mss.server_id = s.id
		WHERE ms.id = $1
		GROUP BY ms.id, ms.subscription_url, ms.is_active, ms.created_at, ms.expiry_time, ms.user_id`

	var subscription common.MultiSubscription
	var serversJSON string

	err := common.GetDatabasePG().QueryRow(query, subscriptionID).Scan(
		&subscription.ID, &subscription.UserID, &subscription.SubscriptionURL, &subscription.IsActive,
		&subscription.CreatedAt, &subscription.ExpiryTime, &serversJSON,
	)
	if err == sql.ErrNoRows {
		log.Printf("GET_MULTI_SUBSCRIPTION_BY_ID: Мультиподписка %s не найдена", subscriptionID)
		return nil, nil
	}
	if err != nil {
		log.Printf("GET_MULTI_SUBSCRIPTION_BY_ID: Ошибка получения мультиподписки: %v", err)
		return nil, fmt.Errorf("ошибка получения мультиподписки: %v", err)
	}

	// Десериализуем серверы
	err = json.Unmarshal([]byte(serversJSON), &subscription.Servers)
	if err != nil {
		log.Printf("GET_MULTI_SUBSCRIPTION_BY_ID: Ошибка десериализации серверов: %v", err)
		return nil, fmt.Errorf("ошибка десериализации серверов: %v", err)
	}

	subscription.UpdatedAt = time.Now()

	log.Printf("GET_MULTI_SUBSCRIPTION_BY_ID: Мультиподписка получена: %s, серверов: %d", subscription.ID, len(subscription.Servers))
	return &subscription, nil
}

// generateMultiSubscriptionConfigs генерирует конфигурации для мультиподписки
func generateMultiSubscriptionConfigs(subscription *common.MultiSubscription) (string, error) {
	log.Printf("GENERATE_MULTI_SUBSCRIPTION_CONFIGS: Генерация конфигураций для мультиподписки %s", subscription.ID)

	var allConfigs []string

	for _, server := range subscription.Servers {
		// Получаем конфигурацию для каждого сервера
		config, err := getServerConfig(server)
		if err != nil {
			log.Printf("GENERATE_MULTI_SUBSCRIPTION_CONFIGS: Ошибка получения конфигурации сервера %s: %v", server.ID, err)
			continue
		}
		allConfigs = append(allConfigs, config)
	}

	if len(allConfigs) == 0 {
		return "", fmt.Errorf("не удалось получить конфигурации ни для одного сервера")
	}

	// Объединяем все конфигурации
	result := strings.Join(allConfigs, "\n")

	log.Printf("GENERATE_MULTI_SUBSCRIPTION_CONFIGS: Сгенерировано %d конфигураций", len(allConfigs))
	return result, nil
}

// getServerConfig получает конфигурацию для конкретного сервера
func getServerConfig(server common.Server) (string, error) {
	log.Printf("GET_SERVER_CONFIG: Получение конфигурации для сервера %s", server.ID)

	// Делаем запрос к серверу конфигураций
	resp, err := http.Get(server.ConfigURL)
	if err != nil {
		log.Printf("GET_SERVER_CONFIG: Ошибка запроса к серверу %s: %v", server.ID, err)
		return "", fmt.Errorf("ошибка запроса к серверу: %v", err)
	}
	defer resp.Body.Close()

	// Читаем ответ сервера
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("GET_SERVER_CONFIG: Ошибка чтения ответа сервера %s: %v", server.ID, err)
		return "", fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("GET_SERVER_CONFIG: Сервер %s вернул статус %d", server.ID, resp.StatusCode)
		return "", fmt.Errorf("сервер вернул статус %d", resp.StatusCode)
	}

	log.Printf("GET_SERVER_CONFIG: Конфигурация для сервера %s получена (%d байт)", server.ID, len(body))
	return string(body), nil
}
