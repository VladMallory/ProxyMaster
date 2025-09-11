package app

import (
	"log"
	"net/http"

	"bot/common"
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

	bot.Start()
}
