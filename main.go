package main

import (
	"log"
	"math/rand"
	"time"

	"bot/app"
	"bot/common"
	"bot/payments"
	"bot/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Глобальные переменные для сервисов
var globalAutoBillingService *services.AutoBillingService
var globalResetStatusChecker *services.ResetStatusChecker
var globalUniversalReminderService *services.UniversalReminderService
var globalExpiredSubscriptionService *services.ExpiredSubscriptionService
var globalDisabledConfigService *services.DisabledConfigService
var globalDepletedStatusChecker *services.DepletedStatusChecker

func main() {
	rand.Seed(time.Now().UnixNano())

	// Инициализируем глобальные переменные
	common.InitGlobals()

	// Инициализируем глобальный логгер платежей
	payments.InitGlobalPaymentLogger()

	// Инициализируем логгер консоли (должен быть первым, чтобы перехватывать весь вывод)
	if err := common.InitConsoleLogger(); err != nil {
		log.Printf("Ошибка инициализации логгера консоли: %v", err)
	}

	// Инициализируем менеджер пробных периодов
	common.TrialManager = common.NewTrialPeriodManager()

	// Инициализируем IP ban логгер
	if err := common.InitIPBanLogger(); err != nil {
		log.Printf("Ошибка инициализации IP ban логгера: %v", err)
	}

	// Инициализируем логгер трафика
	if err := common.InitTrafficLogger(); err != nil {
		log.Printf("Ошибка инициализации логгера трафика: %v", err)
	} else {
		log.Printf("Логгер трафика успешно инициализирован")
	}

	// Инициализируем логгер действий пользователей
	if err := common.InitUsersLogger(); err != nil {
		log.Printf("Ошибка инициализации логгера действий пользователей: %v", err)
	}

	// Инициализируем приложение
	app.InitializeApp()

	// Корректно отключаем MongoDB при завершении программы
	defer common.DisconnectMongoDB()

	// Закрываем логгер трафика при завершении программы
	defer common.CloseTrafficLogger()

	// Закрываем логгер консоли при завершении программы
	defer func() {
		if consoleLogger := common.GetConsoleLogger(); consoleLogger != nil {
			consoleLogger.Close()
		}
	}()

	// Закрываем логгер действий пользователей при завершении программы
	defer func() {
		if usersLogger := common.GetUsersLogger(); usersLogger != nil {
			usersLogger.Close()
		}
	}()

	// Останавливаем проверяльщик реферальных кодов при завершении программы
	defer common.StopReferralChecker()

	// Запускаем IP Ban сервис если включен (в отдельной горутине)
	// if common.IP_BAN_ENABLED {
	// 	go startIPBanService()
	// }

	// Запускаем автосписание если включено (в отдельной горутине)
	if common.AUTO_BILLING_ENABLED {
		go startAutoBillingService()
	}

	// Запускаем очистку дубликатов если включена (в отдельной горутине)
	if common.DUPLICATE_CLEANUP_ENABLED {
		go startDuplicateCleanupService()
	}

	// Запускаем проверку состояния reset если включена (в отдельной горутине)
	if common.RESET_STATUS_CHECK_ENABLED {
		go startResetStatusCheckerService()
	}

	// Запускаем автоматическую проверку реферальных кодов если включена (в отдельной горутине)
	if common.REFERRAL_CHECK_ENABLED {
		go startReferralCheckerService()
	}

	// Запускаем сервис универсальных напоминаний если включен (в отдельной горутине)
	if common.UNIVERSAL_REMINDER_ENABLED {
		go startUniversalReminderService()
	}

	// Запускаем сервис проверки истекших подписок если включен (в отдельной горутине)
	if common.EXPIRED_SUBSCRIPTION_CHECK_ENABLED {
		go startExpiredSubscriptionService()
	}

	// Запускаем сервис проверки отключенных конфигов если включен (в отдельной горутине)
	if common.DISABLED_CONFIG_CHECK_ENABLED {
		go startDisabledConfigService()
	}

	// Запускаем сервис проверки ложных состояний "исчерпано" если включен (в отдельной горутине)
	if common.DEPLETED_STATUS_CHECK_ENABLED {
		go startDepletedStatusCheckerService()
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

// startResetStatusCheckerService запускает сервис проверки состояния reset
func startResetStatusCheckerService() {
	log.Printf("RESET_STATUS_CHECKER: Запуск сервиса проверки состояния reset...")

	// Ждем инициализации бота
	time.Sleep(5 * time.Second)

	// Создаем сервис проверки состояния reset
	globalResetStatusChecker = services.NewResetStatusChecker()

	// Запускаем сервис
	globalResetStatusChecker.Start()

	log.Printf("RESET_STATUS_CHECKER: Сервис проверки состояния reset успешно запущен")
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

// startReferralCheckerService запускает сервис проверки реферальных кодов
func startReferralCheckerService() {
	log.Printf("MAIN: Запуск сервиса проверки реферальных кодов...")

	// Инициализируем проверяльщик реферальных кодов
	if err := common.InitReferralChecker(common.GetDB()); err != nil {
		log.Printf("MAIN: ❌ Ошибка инициализации проверяльщика реферальных кодов: %v", err)
		return
	}

	log.Printf("MAIN: ✅ Сервис проверки реферальных кодов успешно запущен")
}

// startUniversalReminderService запускает сервис универсальных напоминаний
func startUniversalReminderService() {
	log.Printf("UNIVERSAL_REMINDER: Запуск сервиса универсальных напоминаний...")

	// Ждем инициализации бота
	time.Sleep(5 * time.Second)

	// Получаем бот из глобальной переменной
	var bot *tgbotapi.BotAPI
	if common.GlobalBot != nil {
		bot = common.GlobalBot
		log.Printf("UNIVERSAL_REMINDER: Бот получен из глобальной переменной")
	} else {
		log.Printf("UNIVERSAL_REMINDER: Бот не инициализирован, уведомления отключены")
	}

	// Создаем сервис универсальных напоминаний
	globalUniversalReminderService = services.NewUniversalReminderService(bot)

	// Запускаем сервис
	globalUniversalReminderService.Start()

	log.Printf("UNIVERSAL_REMINDER: Сервис универсальных напоминаний успешно запущен")
}

// startExpiredSubscriptionService запускает сервис проверки истекших подписок
func startExpiredSubscriptionService() {
	log.Printf("EXPIRED_SUBSCRIPTION: Запуск сервиса проверки истекших подписок...")

	// Ждем инициализации бота
	time.Sleep(5 * time.Second)

	// Получаем бот из глобальной переменной
	var bot *tgbotapi.BotAPI
	if common.GlobalBot != nil {
		bot = common.GlobalBot
		log.Printf("EXPIRED_SUBSCRIPTION: Бот получен из глобальной переменной")
	} else {
		log.Printf("EXPIRED_SUBSCRIPTION: Бот не инициализирован, уведомления отключены")
	}

	// Создаем менеджер конфигураций
	configManager := common.NewConfigManager(
		common.PANEL_URL,
		common.PANEL_USER,
		common.PANEL_PASS,
		common.INBOUND_ID,
	)

	// Создаем сервис проверки истекших подписок
	globalExpiredSubscriptionService = services.NewExpiredSubscriptionService(bot, configManager)

	// Запускаем сервис
	globalExpiredSubscriptionService.Start()

	log.Printf("EXPIRED_SUBSCRIPTION: Сервис проверки истекших подписок успешно запущен")
}

// startDisabledConfigService запускает сервис проверки отключенных конфигов
func startDisabledConfigService() {
	log.Printf("DISABLED_CONFIG: Запуск сервиса проверки отключенных конфигов...")

	// Ждем инициализации бота
	time.Sleep(5 * time.Second)

	// Получаем бот из глобальной переменной
	var bot *tgbotapi.BotAPI
	if common.GlobalBot != nil {
		bot = common.GlobalBot
		log.Printf("DISABLED_CONFIG: Бот получен из глобальной переменной")
	} else {
		log.Printf("DISABLED_CONFIG: Бот не инициализирован, уведомления отключены")
	}

	// Создаем менеджер конфигураций
	configManager := common.NewConfigManager(
		common.PANEL_URL,
		common.PANEL_USER,
		common.PANEL_PASS,
		common.INBOUND_ID,
	)

	// Создаем сервис проверки отключенных конфигов
	globalDisabledConfigService = services.NewDisabledConfigService(bot, configManager)

	// Запускаем сервис
	globalDisabledConfigService.Start()

	log.Printf("DISABLED_CONFIG: Сервис проверки отключенных конфигов успешно запущен")
}

// startDepletedStatusCheckerService запускает сервис проверки ложных состояний "исчерпано"
func startDepletedStatusCheckerService() {
	log.Printf("DEPLETED_STATUS_CHECKER: Запуск сервиса проверки ложных состояний 'исчерпано'...")

	// Ждем инициализации бота
	time.Sleep(5 * time.Second)

	// Создаем сервис проверки ложных состояний "исчерпано"
	globalDepletedStatusChecker = services.NewDepletedStatusChecker()

	// Запускаем сервис
	globalDepletedStatusChecker.Start()

	log.Printf("DEPLETED_STATUS_CHECKER: Сервис проверки ложных состояний 'исчерпано' успешно запущен")
}
