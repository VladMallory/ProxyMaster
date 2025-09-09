package common

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// Глобальные переменные конфигурации
var (
	BOT_TOKEN              string
	ADMIN_ID               int64
	PANEL_URL              string
	PANEL_USER             string
	PANEL_PASS             string
	INBOUND_ID             int
	CONFIG_BASE_URL        string
	CONFIG_JSON_URL        string
	REDIRECT_DOMAIN        string
	PRICE_PER_DAY          int
	TRIAL_PERIOD_DAYS      int
	TRAFFIC_LIMIT_GB       int
	TRAFFIC_RESET_ENABLED  bool
	TRAFFIC_RESET_INTERVAL int
	TRAFFIC_CHECK_INTERVAL int
	SHOW_DATES_IN_CONFIGS  bool
	SUPPORT_LINK           string

	// IP Ban система - управление конфигами на основе количества IP адресов
	IP_BAN_ENABLED       bool   // Включена ли система IP бана
	MAX_IPS_PER_CONFIG   int    // Максимальное количество IP адресов на один конфиг
	ACCESS_LOG_PATH      string // Путь к файлу access.log
	IP_ACCUMULATED_PATH  string // Путь к файлу накопленных логов
	IP_SAVE_INTERVAL     int    // Интервал сохранения новых строк в минутах
	IP_CHECK_INTERVAL    int    // Интервал проверки IP в минутах
	IP_BAN_GRACE_PERIOD  int    // Период ожидания в минутах перед отключением конфига
	IP_BAN_DURATION      int    // Длительность бана в минутах (0 = бесконечно)
	IP_COUNTER_RETENTION int    // Время хранения счетчиков IP в минутах (0 = бесконечно)
	IP_CLEANUP_INTERVAL  int    // Интервал очистки старых данных в часах

	// Глобальный бот для отправки уведомлений
	GlobalBot *tgbotapi.BotAPI // Глобальный экземпляр бота
)

// Инициализация глобальных переменных конфигурации
func init() {
	// Эти значения должны быть загружены из переменных окружения или конфигурационного файла
	BOT_TOKEN = "8250593221:AAGbCd_CfxvYCzbHgnG5iVWpz0ujsOMnOkY"
	ADMIN_ID = 873925520 // Замените на ваш Telegram ID

	// URL для панели управления
	PANEL_URL = "https://status.moment-was-da.ru:57578/UV7FVRXd61xso1XVRT/"
	PANEL_USER = "C7QKEujq7qzhtFxz"
	PANEL_PASS = "cXFMhAHUk7FMEwoD"
	INBOUND_ID = 5

	// URL для конфигураций
	CONFIG_BASE_URL = "https://status.moment-was-da.ru:2096/sub/"
	CONFIG_JSON_URL = "https://status.moment-was-da.ru:2096/json/"

	REDIRECT_DOMAIN = "status.moment-was-da.ru:8081" // редирект для импорта подписки в Happ

	// ---TRAFIC---
	TRAFFIC_RESET_ENABLED = true  // включен ли автоматический сброс трафика
	PRICE_PER_DAY = 10            // стоимость подписки за день
	TRIAL_PERIOD_DAYS = 1         // дни пробного периода
	TRAFFIC_LIMIT_GB = 2          // лимиты трафика. Срок действия указывается в TRAFFIC_RESET_INTERVAL
	TRAFFIC_RESET_INTERVAL = 2880 // через сколько минут трафик пользователя будет обнулен
	TRAFFIC_CHECK_INTERVAL = 1440 // как часто бот проверяет, не превысил ли пользователь лимит трафика

	// ---IP Ban---
	IP_BAN_ENABLED = true                               // Включена ли система IP бана
	ACCESS_LOG_PATH = "/usr/local/x-ui/access.log"      // Путь к файлу access.log
	IP_ACCUMULATED_PATH = "/var/log/ip_accumulated.log" // Путь к файлу накопленных логов
	MAX_IPS_PER_CONFIG = 1                              // Максимальное количество IP адресов на один конфиг (если больше - отключается)
	IP_SAVE_INTERVAL = 25                               // Интервал сохранения новых строк в минутах
	IP_CHECK_INTERVAL = 25                              // Интервал проверки IP в минутах
	IP_BAN_DURATION = 120                               // Длительность бана в минутах (1 час)
	IP_COUNTER_RETENTION = 90                           // Время хранения счетчиков IP в минутах (3 часа - в 3 раза больше времени бана для стабильной работы)
	IP_CLEANUP_INTERVAL = 1                             // Интервал очистки старых данных в часах
	// не используется
	IP_BAN_GRACE_PERIOD = 10 // Период ожидания в минутах перед отключением конфига (10 минут). Сейчас не работает

	// ---ПРОЧИЕ НАСТРОЙКИ---
	// Показывать ли даты в именах конфигов - если true, то в email будет
	// "123456789 до 2025 03 09", если false, то просто "123456789"
	SHOW_DATES_IN_CONFIGS = false

	// Ссылка на поддержку - куда направлять пользователей при нажатии на кнопку "Поддержка"
	SUPPORT_LINK = "https://t.me/BloknotaNet"
}
