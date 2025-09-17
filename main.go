package main

import (
	"log"
	"math/rand"
	"time"

	"bot/app"
	"bot/common"
	"bot/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Глобальная переменная для сервиса автосписания
var globalAutoBillingService *services.AutoBillingService

func main() {
	rand.Seed(time.Now().UnixNano())

	// Инициализируем глобальные переменные
	common.InitGlobals()

	// Инициализируем менеджер пробных периодов
	common.TrialManager = common.NewTrialPeriodManager()

	// Инициализируем IP ban логгер
	if err := common.InitIPBanLogger(); err != nil {
		log.Printf("Ошибка инициализации IP ban логгера: %v", err)
	} else {
		log.Printf("IP ban логгер успешно инициализирован")
	}

	// Инициализируем приложение
	app.InitializeApp()

	// Корректно отключаем MongoDB при завершении программы
	defer common.DisconnectMongoDB()

	// Запускаем IP Ban сервис если включен (в отдельной горутине)
	if common.IP_BAN_ENABLED {
		go startIPBanService()
	}

	// Запускаем автосписание если включено (в отдельной горутине)
	if common.AUTO_BILLING_ENABLED {
		go startAutoBillingService()
	}

	// Запускаем очистку дубликатов если включена (в отдельной горутине)
	if common.DUPLICATE_CLEANUP_ENABLED {
		go startDuplicateCleanupService()
	}

	// Запускаем Telegram бота (блокирующая функция)
	app.StartBot(common.BOT_TOKEN)
}

// startIPBanService запускает IP Ban сервис
func startIPBanService() {
	common.LogIPBanInfo("Запуск IP Ban сервиса...")

	// Создаем накопитель логов
	accumulator := common.NewLogAccumulator(common.ACCESS_LOG_PATH, common.IP_ACCUMULATED_PATH)

	// Запускаем накопитель логов
	if err := accumulator.Start(); err != nil {
		common.LogIPBanError("Ошибка запуска накопителя логов: %v", err)
		return
	}

	// Запускаем сервис очистки старых строк
	accumulator.StartCleanupService()
	common.LogIPBanInfo("Накопитель логов запущен")

	// Создаем анализатор логов (теперь работает с накопленным файлом)
	analyzer := common.NewLogAnalyzer(common.IP_ACCUMULATED_PATH)

	// Создаем менеджер конфигураций
	configManager := common.NewConfigManager(
		common.PANEL_URL,
		common.PANEL_USER,
		common.PANEL_PASS,
		common.INBOUND_ID,
	)

	// Создаем менеджер банов
	banManager := common.NewBanManager("/var/log/ip_bans.json")

	// Создаем менеджер iptables
	iptablesManager := common.NewIPTablesManager()

	// Ждем инициализации бота (увеличиваем время, так как запускаемся раньше)
	time.Sleep(5 * time.Second)

	// Получаем бот из глобальной переменной
	var bot *tgbotapi.BotAPI
	if common.GlobalBot != nil {
		bot = common.GlobalBot
		common.LogIPBanInfo("Бот получен из глобальной переменной")
	} else {
		common.LogIPBanWarning("Бот не инициализирован, уведомления отключены")
	}

	// Создаем IP Ban сервис
	service := common.NewIPBanService(
		analyzer,
		configManager,
		banManager,
		iptablesManager,
		common.MAX_IPS_PER_CONFIG,
		time.Duration(common.IP_CHECK_INTERVAL)*time.Minute,
		time.Duration(common.IP_BAN_GRACE_PERIOD)*time.Minute,
		bot,
	)

	// Запускаем сервис
	if err := service.Start(); err != nil {
		common.LogIPBanError("Ошибка запуска IP Ban сервиса: %v", err)
		return
	}

	common.LogIPBanInfo("IP Ban сервис успешно запущен")
}

// startAutoBillingService запускает сервис автосписания
func startAutoBillingService() {
	log.Printf("AUTO_BILLING: Запуск сервиса автосписания...")

	// Ждем инициализации бота
	time.Sleep(5 * time.Second)

	// Получаем бот из глобальной переменной
	var bot *tgbotapi.BotAPI
	if common.GlobalBot != nil {
		bot = common.GlobalBot
		log.Printf("AUTO_BILLING: Бот получен из глобальной переменной")
	} else {
		log.Printf("AUTO_BILLING: Бот не инициализирован, уведомления отключены")
	}

	// Создаем сервис автосписания
	globalAutoBillingService = services.NewAutoBillingService(bot)

	// Сохраняем ссылку на сервис в common
	common.SetAutoBillingService(globalAutoBillingService)

	// Запускаем сервис
	globalAutoBillingService.Start()

	log.Printf("AUTO_BILLING: Сервис автосписания успешно запущен")
}

// startDuplicateCleanupService запускает сервис очистки дубликатов
func startDuplicateCleanupService() {
	log.Printf("DUPLICATE_CLEANUP: Запуск сервиса очистки дубликатов...")

	// Ждем инициализации бота
	time.Sleep(5 * time.Second)

	// Получаем бот из глобальной переменной
	var bot *tgbotapi.BotAPI
	if common.GlobalBot != nil {
		bot = common.GlobalBot
		log.Printf("DUPLICATE_CLEANUP: Бот получен из глобальной переменной")
	} else {
		log.Printf("DUPLICATE_CLEANUP: Бот не инициализирован, уведомления об ошибках отключены")
	}

	// Создаем сервис очистки дубликатов
	duplicateCleanupService := services.NewDuplicateCleanupService(bot)

	// Запускаем сервис
	duplicateCleanupService.Start()

	log.Printf("DUPLICATE_CLEANUP: Сервис очистки дубликатов успешно запущен")
}

// stopAutoBillingService останавливает сервис автосписания
func stopAutoBillingService() {
	if globalAutoBillingService != nil {
		log.Printf("AUTO_BILLING: Остановка сервиса автосписания...")
		globalAutoBillingService.Stop()
		globalAutoBillingService = nil
		log.Printf("AUTO_BILLING: Сервис автосписания остановлен")
	}
}

// SwitchToTariffMode переключает на тарифный режим
func SwitchToTariffMode() {
	log.Printf("MAIN: Переключение на тарифный режим")
	common.TARIFF_MODE_ENABLED = true
	common.AUTO_BILLING_ENABLED = false
	stopAutoBillingService()
	log.Printf("MAIN: Переключение на тарифный режим завершено")
}

// SwitchToAutoBillingMode переключает на режим автосписания
func SwitchToAutoBillingMode() {
	log.Printf("MAIN: Переключение на режим автосписания")
	stopAutoBillingService()
	common.TARIFF_MODE_ENABLED = false
	common.AUTO_BILLING_ENABLED = true

	// Запускаем автосписание заново
	go startAutoBillingService()
	log.Printf("MAIN: Переключение на режим автосписания завершено")
}
