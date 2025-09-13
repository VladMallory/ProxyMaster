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

		// Обработчик для redirect_v2raytun.html
		http.HandleFunc("/redirect_v2raytun.html", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("HTTP_SERVER: v2raytun redirect request: %s", r.URL.String())
			http.ServeFile(w, r, "importRedirect/redirect_v2raytun.html")
		})

		// Обработчик для callback-ов ЮКассы
		http.HandleFunc("/yukassa/callback", handleYukassaCallback)

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

	// Мониторинг трафика и очистка конфигов временно отключены
	log.Printf("APP: Сервисы мониторинга трафика и очистки конфигов временно отключены")

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

	// Инициализируем систему промокодов (независимо от платежной системы)
	log.Printf("APP: Инициализация системы промокодов")
	if err := promo.InitializePromoManager(); err != nil {
		log.Printf("APP: Ошибка инициализации системы промокодов: %v", err)
		log.Printf("APP: Промокоды будут недоступны")
	} else {
		log.Printf("APP: Система промокодов успешно инициализирована")
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
