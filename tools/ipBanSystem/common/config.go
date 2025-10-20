package common

// ============================================================================
// ГЛОБАЛЬНЫЕ ПЕРЕМЕННЫЕ КОНФИГУРАЦИИ
// ============================================================================

var (
	// ========== НАСТРОЙКИ ПАНЕЛИ УПРАВЛЕНИЯ ==========
	PANEL_URL  string // URL панели управления X-UI
	PANEL_USER string // Имя пользователя для авторизации в панели X-UI
	PANEL_PASS string // Пароль для авторизации в панели X-UI
	INBOUND_ID int    // ID входящего соединения в панели

	// ========== СИСТЕМА IP БАНА ==========
	MAX_IPS_PER_CONFIG   int    // Максимальное количество IP на конфиг
	ACCESS_LOG_PATH      string // Путь к файлу access.log
	IP_ACCUMULATED_PATH  string // Путь к файлу накопленных логов
	IP_BAN_LOG_PATH      string // Путь к файлу логов IP ban
	IP_SAVE_INTERVAL     int    // Интервал сохранения новых строк в минутах
	IP_CHECK_INTERVAL    int    // Интервал проверки IP в минутах
	IP_BAN_GRACE_PERIOD  int    // Период ожидания перед отключением в минутах
	IP_BAN_DURATION      int    // Длительность бана в минутах
	IP_COUNTER_RETENTION int    // Время хранения счетчиков IP в минутах
	IP_CLEANUP_INTERVAL  int    // Интервал очистки старых данных в часах
)

// ============================================================================
// ИНИЦИАЛИЗАЦИЯ КОНФИГУРАЦИИ
// ============================================================================

func init() {
	// ========== НАСТРОЙКИ ПАНЕЛИ УПРАВЛЕНИЯ ==========
	PANEL_URL = ""
	PANEL_USER = ""
	PANEL_PASS = ""
	INBOUND_ID = 2

	// ========== СИСТЕМА IP БАНА ==========
	MAX_IPS_PER_CONFIG = 20
	ACCESS_LOG_PATH = "/usr/local/x-ui/access.log"
	IP_ACCUMULATED_PATH = "/var/log/ip_accumulated.log"
	IP_BAN_LOG_PATH = "/root/bot/logs/ip_ban.log"
	IP_SAVE_INTERVAL = 25
	IP_CHECK_INTERVAL = 25
	IP_BAN_GRACE_PERIOD = 10
	IP_BAN_DURATION = 120
	IP_COUNTER_RETENTION = 20
	IP_CLEANUP_INTERVAL = 3
}
