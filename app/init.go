package app

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
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
