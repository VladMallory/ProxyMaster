// Пакет common: глобальные переменные конфигурации и дефолтные значения.
// Источник истины:
//
//	Основной бот загружает конфигурацию из .env в main.go через env.MustLoad().ApplyToCommon(),
//	тем самым переопределяя значения, заданные здесь. Эти значения в init() служат
//	как дефолты и удобные примеры, а также поддержка для утилит из каталога tools/,
//	которые пока могут не подхватывать .env.
//
// Рекомендация:
//
//	Для утилит из tools/ добавить ранний вызов env.MustLoad().ApplyToCommon(),
//	тогда секреты и приватные данные можно безопасно удалить из init() и хранить только в .env.
//
// Безопасность:
//
//	Не храните реальные ключи и пароли в репозитории. Поместите их в .env, а здесь оставьте
//	пустые или демонстрационные значения, если необходимо.
package common

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// ============================================================================
// ГЛОБАЛЬНЫЕ ПЕРЕМЕННЫЕ КОНФИГУРАЦИИ
// ============================================================================

var (
	// ========== ОСНОВНЫЕ НАСТРОЙКИ БОТА ==========
	BOT_TOKEN    string // Токен Telegram бота (получается от @BotFather)
	ADMIN_ID     int64  // ID администратора в Telegram (для получения уведомлений и управления)
	SUPPORT_LINK string // Ссылка на поддержку (отображается пользователям при проблемах)

	// ========== НАСТРОЙКИ ПАНЕЛИ УПРАВЛЕНИЯ ==========
	PANEL_URL       string // URL панели управления X-UI (для создания/удаления конфигов)
	PANEL_USER      string // Имя пользователя для авторизации в панели X-UI
	PANEL_PASS      string // Пароль для авторизации в панели X-UI
	INBOUND_ID      int    // ID входящего соединения в панели (для привязки конфигов)
	CONFIG_BASE_URL string // Базовый URL для генерации ссылок на конфиги (используется в меню и редиректах)
	CONFIG_JSON_URL string // URL для получения JSON конфигураций (используется для импорта в приложения)
	REDIRECT_DOMAIN string // Домен для редиректа импорта (куда перенаправлять пользователей)
	REDIRECT_IMPORT string // Тип импорта: "happ" (HTTP API Proxy) или "v2raytun" (V2RayTun)

	// ========== НАСТРОЙКИ ДОПОЛНИТЕЛЬНОГО ИНБАУНДА ==========
	SECONDARY_INBOUND_ENABLED   bool   // Включена ли синхронизация с дополнительным инбаундом (для резервного сервера)
	SECONDARY_INBOUND_ID        int    // ID дополнительного инбаунда в панели (резервный сервер)
	SECONDARY_CONFIG_BASE_URL   string // Базовый URL для конфигураций дополнительного инбаунда
	SECONDARY_CONFIG_JSON_URL   string // URL для JSON конфигураций дополнительного инбаунда
	SECONDARY_REDIRECT_DOMAIN   string // Домен для редиректа импорта дополнительного инбаунда
	SECONDARY_REDIRECT_IMPORT   string // Тип импорта для дополнительного инбаунда (happ/v2raytun)
	SECONDARY_INBOUND_PROTOCOL  string // Протокол дополнительного инбаунда (vless, vmess, trojan, etc.)
	SECONDARY_INBOUND_TRANSPORT string // Транспорт дополнительного инбаунда (websocket, tcp, grpc, etc.)

	// ========== НАСТРОЙКИ ТРАФИКА И ПОДПИСОК ==========
	PRICE_PER_DAY          int  // Стоимость подписки за день (в рублях) - ежедневное списание при автосписании
	TRIAL_BALANCE_AMOUNT   int  // Сумма пробного баланса (в рублях) - добавляется новым пользователям на баланс
	TRAFFIC_LIMIT_GB       int  // Лимит трафика в гигабайтах - максимальный объем трафика на пользователя
	TRAFFIC_RESET_ENABLED  bool // Включен ли автоматический сброс трафика (обнуление статистики)
	TRAFFIC_RESET_INTERVAL int  // Интервал сброса трафика (в минутах) - как часто обнулять статистику
	TRAFFIC_CHECK_INTERVAL int  // Интервал проверки трафика (в минутах) - как часто проверять превышение лимитов
	SHOW_DATES_IN_CONFIGS  bool // Показывать ли даты в именах конфигов (true="123456789 до 2025 03 09", false="123456789")

	// ========== СИСТЕМА АВТОСПИСАНИЯ ==========
	AUTO_BILLING_ENABLED    bool // Включено ли автосписание (автоматическое списание средств за подписку)
	BALANCE_RECALC_INTERVAL int  // Интервал пересчета баланса (в минутах) - синхронизация баланса с периодом подписки
	TARIFF_MODE_ENABLED     bool // Включен ли режим тарифов (разные тарифы для разных пользователей)
	SYNC_DIAGNOSTIC_ENABLED bool // Включена ли диагностика рассинхронизации (поиск несоответствий в данных)

	// ========== УПРАВЛЕНИЕ ПОДПИСКАМИ ==========
	EXPIRED_SUBSCRIPTION_CHECK_ENABLED  bool // Включена ли автоматическая проверка истекших подписок (отключение конфигов)
	EXPIRED_SUBSCRIPTION_CHECK_INTERVAL int  // Интервал проверки истекших подписок (в минутах) - как часто проверять
	DISABLED_CONFIG_CHECK_ENABLED       bool // Включена ли автоматическая проверка отключенных конфигов (поиск неактивных)
	DISABLED_CONFIG_CHECK_INTERVAL      int  // Интервал проверки отключенных конфигов (в минутах) - как часто проверять

	// ========== СИСТЕМА IP БАНА ==========
	IP_BAN_ENABLED       bool   // Включена ли система IP бана (автоматическое отключение при превышении IP)
	MAX_IPS_PER_CONFIG   int    // Максимальное количество IP на конфиг (лимит IP адресов на пользователя)
	ACCESS_LOG_PATH      string // Путь к файлу access.log (логи доступа X-UI для анализа IP)
	IP_ACCUMULATED_PATH  string // Путь к файлу накопленных логов (агрегированная статистика IP)
	IP_BAN_LOG_PATH      string // Путь к файлу логов IP ban (история заблокированных пользователей)
	IP_SAVE_INTERVAL     int    // Интервал сохранения новых строк (в минутах) - частота анализа логов
	IP_CHECK_INTERVAL    int    // Интервал проверки IP (в минутах) - как часто проверять превышение лимитов
	IP_BAN_GRACE_PERIOD  int    // Период ожидания перед отключением (в минутах) - время на исправление
	IP_BAN_DURATION      int    // Длительность бана (в минутах, 0 = бесконечно) - когда разблокировать
	IP_COUNTER_RETENTION int    // Время хранения счетчиков IP (в минутах) - период хранения статистики
	IP_CLEANUP_INTERVAL  int    // Интервал очистки старых данных (в часах) - очистка устаревших данных

	// ========== РЕФЕРАЛЬНАЯ СИСТЕМА ==========
	REFERRAL_SYSTEM_ENABLED      bool    // Включена ли реферальная система (программа приглашений)
	REFERRAL_BONUS_AMOUNT        float64 // Бонус для пригласившего (в рублях) - награда за приглашение
	REFERRAL_WELCOME_BONUS       float64 // Бонус для приглашенного (в рублях) - бонус новому пользователю
	REFERRAL_LINK_BASE_URL       string  // Базовый URL для реферальных ссылок (шаблон ссылки)
	REFERRAL_MIN_BALANCE_FOR_REF float64 // Минимальный баланс для получения ссылки (условие для генерации)
	REFERRAL_CHECK_ENABLED       bool    // Включена ли проверка реферальных кодов (автоматическая проверка)
	REFERRAL_CHECK_INTERVAL      int     // Интервал проверки рефералов (в минутах) - частота проверок

	// ========== УВЕДОМЛЕНИЯ И НАПОМИНАНИЯ ==========

	// --- Уведомления пользователям (устаревшая система) ---
	NOTIFICATION_ENABLED        bool   // Включены ли уведомления о подписке (старая система с фиксированными сообщениями)
	NOTIFICATION_CHECK_INTERVAL int    // Интервал проверки подписок (в минутах) - как часто проверять пользователей
	NOTIFICATION_DAYS_BEFORE    []int  // За сколько дней отправлять уведомления (массив дней до истечения)
	NOTIFICATION_MESSAGE_1_DAY  string // Сообщение за 1 день до истечения (фиксированный текст)
	NOTIFICATION_MESSAGE_3_DAYS string // Сообщение за 3 дня до истечения (фиксированный текст)
	NOTIFICATION_MESSAGE_7_DAYS string // Сообщение за 7 дней до истечения (фиксированный текст)

	// --- Универсальные напоминания пользователям ---
	UNIVERSAL_REMINDER_ENABLED     bool   // Включены ли универсальные напоминания (новая система с динамическими сообщениями)
	UNIVERSAL_REMINDER_INTERVAL    int    // Интервал проверки (в минутах, 1440 = 24 часа) - как часто проверять
	UNIVERSAL_REMINDER_DAYS_BEFORE int    // За сколько дней начинать напоминания (по умолчанию 3) - начало уведомлений
	UNIVERSAL_REMINDER_MESSAGE     string // Универсальное сообщение с плейсхолдерами {DAYS} и {HOURS}
	UNIVERSAL_REMINDER_LOG_PATH    string // Путь к логу отправленных напоминаний (история всех отправок)

	// --- Уведомления администратору ---
	ADMIN_NOTIFICATIONS_ENABLED   bool // Включены ли уведомления для админа (общий переключатель)
	ADMIN_CONFIG_BLOCKING_ENABLED bool // Уведомления о блокировке конфигов (отключение пользователей)
	ADMIN_IP_BAN_ENABLED          bool // Уведомления о срабатывании IP ban (превышение лимита IP)
	ADMIN_BALANCE_TOPUP_ENABLED   bool // Уведомления о пополнении баланса (успешные платежи)
	ADMIN_REFERRAL_ENABLED        bool // Уведомления о новых рефералах (приглашения по ссылкам)
	ADMIN_REMINDER_ENABLED        bool // Уведомления о отправленных напоминаниях о подписке (отчеты о напоминаниях)

	// ========== СИСТЕМЫ ОЧИСТКИ И ПРОВЕРКИ ==========
	DUPLICATE_CLEANUP_ENABLED      bool // Включена ли очистка дубликатов (поиск и удаление повторяющихся конфигов)
	DUPLICATE_CLEANUP_INTERVAL     int  // Интервал очистки дубликатов (в минутах) - как часто искать дубли
	RESET_STATUS_CHECK_ENABLED     bool // Включена ли проверка состояния reset (поиск конфигов в состоянии reset)
	RESET_STATUS_CHECK_INTERVAL    int  // Интервал проверки reset (в минутах) - как часто проверять панель
	DEPLETED_STATUS_CHECK_ENABLED  bool // Включена ли проверка ложных состояний "исчерпано" (восстановление активных конфигов)
	DEPLETED_STATUS_CHECK_INTERVAL int  // Интервал проверки (в минутах) - как часто проверять ложные блокировки
	DEPLETED_ACTIVE_THRESHOLD      int  // Минимальное время активности для сброса (в часах) - критерий активности

	// ========== НАСТРОЙКИ ПЛАТЕЖЕЙ ==========
	// Telegram Bot API платежи
	YUKASSA_PROVIDER_TOKEN    string // Токен провайдера платежей от BotFather (для платежей через Telegram)
	TELEGRAM_PAYMENTS_ENABLED bool   // Включены ли платежи через Telegram Bot API (встроенные платежи)

	// Прямая интеграция с ЮКассой
	YUKASSA_SHOP_ID              string // ID магазина ЮКассы (получается при регистрации в ЮКассе)
	YUKASSA_SECRET_KEY           string // Секретный ключ ЮКассы (для авторизации API запросов)
	YUKASSA_API_PAYMENTS_ENABLED bool   // Включены ли платежи через API ЮКассы (прямые платежи)
	YUKASSA_WEBHOOK_URL          string // URL для уведомлений от ЮКассы (обработка статусов платежей)

	// Настройки чеков
	YUKASSA_RECEIPT_ENABLED bool   // Включена ли отправка чеков (обязательные чеки для ИП/ООО)
	YUKASSA_VAT_CODE        int    // Код НДС (1=без НДС, 2=НДС 0%, 3=НДС 10%, 4=НДС 20%)
	YUKASSA_PAYMENT_SUBJECT string // Предмет расчета (service=услуга, commodity=товар)
	YUKASSA_PAYMENT_MODE    string // Способ расчета (full_prepayment=полная предоплата)

	// ========== БЕЗОПАСНОЕ ВЫКЛЮЧЕНИЕ ==========
	POWEROFF_SYSTEM_ENABLED       bool // Включена ли система безопасного выключения (ожидание завершения платежей перед выключением)
	POWEROFF_PAYMENT_TIMEOUT      int  // Таймаут ожидания завершения платежей (в минутах) - максимальное время ожидания
	POWEROFF_CHECK_INTERVAL       int  // Интервал проверки активных платежей (в секундах) - как часто проверять
	POWEROFF_NOTIFICATION_ENABLED bool // Включены ли уведомления о выключении (уведомления пользователям о технических работах)

	// ========== ЛОГИРОВАНИЕ ==========
	CONSOLE_LOG_ENABLED bool   // Включено ли логирование консоли в файл (сохранение всех логов в файл)
	CONSOLE_LOG_PATH    string // Путь к файлу консольных логов (основной лог работы бота)
	USERS_LOG_ENABLED   bool   // Включено ли логирование действий пользователей (логи команд и действий)
	USERS_LOG_PATH      string // Путь к файлу логов пользователей (история действий пользователей)
	PAYMENT_LOG_ENABLED bool   // Включено ли логирование платежей (детальные логи всех платежей)
	PAYMENT_LOG_PATH    string // Путь к файлу логов платежей (история платежных операций)

	// Exhausted-лог: отдельный управляемый лог для изменений статуса "исчерпано"
	EXHAUSTED_LOG_ENABLED bool   // Включено ли логирование exhausted/depleted событий
	EXHAUSTED_LOG_PATH    string // Путь к файлу exhausted.log

	// ========== ГЛОБАЛЬНЫЕ ОБЪЕКТЫ ==========
	GlobalBot             *tgbotapi.BotAPI         // Глобальный экземпляр бота (используется сервисами для отправки сообщений)
	GlobalReferralManager ReferralManagerInterface // Глобальный реферальный менеджер (управление реферальной системой)
)

// ============================================================================
// ИНИЦИАЛИЗАЦИЯ КОНФИГУРАЦИИ
// ============================================================================

func init() {
	// ========== ОСНОВНЫЕ НАСТРОЙКИ БОТА ==========
	// Значения ниже — примеры/дефолты. Для продакшена они должны приходить из .env.
	// Значения загружаются из .env через env.MustLoad().ApplyToCommon()

	// ========== НАСТРОЙКИ ПАНЕЛИ УПРАВЛЕНИЯ ==========
	// Значения загружаются из .env через env.MustLoad().ApplyToCommon()

	// ========== НАСТРОЙКИ ДОПОЛНИТЕЛЬНОГО ИНБАУНДА ==========
	SECONDARY_INBOUND_ENABLED = false                  // Резервный сервер отключен
	SECONDARY_INBOUND_ID = 5                           // ID резервного инбаунда
	SECONDARY_CONFIG_BASE_URL = "https://.ru:123/asd/" // URL резервного сервера
	SECONDARY_CONFIG_JSON_URL = "https://.ru:123/asd/" // JSON резервного сервера
	SECONDARY_REDIRECT_DOMAIN = "im.domen.ru:8443"     // Домен резервного сервера
	SECONDARY_REDIRECT_IMPORT = "happ"                 // Тип импорта резервного
	SECONDARY_INBOUND_PROTOCOL = "vless"               // Протокол резервного
	SECONDARY_INBOUND_TRANSPORT = "websocket"          // Транспорт резервного

	// ========== НАСТРОЙКИ ТРАФИКА И ПОДПИСОК ==========
	PRICE_PER_DAY = 3              // Цена за день (₽)
	TRIAL_BALANCE_AMOUNT = 12      // Бонус новым пользователям (₽)
	TRAFFIC_LIMIT_GB = 140         // Лимит трафика (0=безлимит)
	TRAFFIC_RESET_ENABLED = true   // Автосброс трафика включен
	TRAFFIC_RESET_INTERVAL = 10080 // Интервал сброса (мин, 10080=7 дней)
	TRAFFIC_CHECK_INTERVAL = 1440  // Интервал проверки (мин, 1440=24ч)
	SHOW_DATES_IN_CONFIGS = false  // Даты в именах конфигов (false=только ID)

	// ========== СИСТЕМА АВТОСПИСАНИЯ ==========
	AUTO_BILLING_ENABLED = true    // Автосписание включено (списание ₽3/день)
	BALANCE_RECALC_INTERVAL = 30   // Интервал синхронизации баланса (мин)
	TARIFF_MODE_ENABLED = false    // Режим тарифов отключен (используется автосписание)
	SYNC_DIAGNOSTIC_ENABLED = true // Диагностика рассинхронизации включена

	// ========== УПРАВЛЕНИЕ ПОДПИСКАМИ ==========
	EXPIRED_SUBSCRIPTION_CHECK_ENABLED = true // Проверка истекших подписок включена
	EXPIRED_SUBSCRIPTION_CHECK_INTERVAL = 120 // отключение истекших подписок
	DISABLED_CONFIG_CHECK_ENABLED = true      // Проверка отключенных конфигов включена
	DISABLED_CONFIG_CHECK_INTERVAL = 120      // Интервал проверки (мин, 1440=24ч)

	// ========== СИСТЕМА IP БАНА ==========
	// IP_BAN_ENABLED = true                               // IP бан включен
	MAX_IPS_PER_CONFIG = 20                             // Максимум IP на конфиг
	ACCESS_LOG_PATH = "/usr/local/x-ui/access.log"      // Путь к логам X-UI
	IP_ACCUMULATED_PATH = "/var/log/ip_accumulated.log" // Путь к агрегированным логам
	IP_BAN_LOG_PATH = "/root/bot/logs/ip_ban.log"       // Путь к логам банов
	IP_SAVE_INTERVAL = 25                               // Интервал сохранения (мин)
	IP_CHECK_INTERVAL = 25                              // Интервал проверки (мин)
	IP_BAN_GRACE_PERIOD = 10                            // Период ожидания (мин)
	IP_BAN_DURATION = 120                               // Длительность бана (мин, 120=2ч)
	IP_COUNTER_RETENTION = 20                           // Хранение счетчиков (мин)
	IP_CLEANUP_INTERVAL = 3                             // Очистка старых данных (ч)

	// ========== РЕФЕРАЛЬНАЯ СИСТЕМА ==========
	REFERRAL_SYSTEM_ENABLED = true                           // Реферальная система включена
	REFERRAL_BONUS_AMOUNT = float64(PRICE_PER_DAY) * 30 * 5  // Бонус пригласившему (₽90 = 1 месяц)
	REFERRAL_WELCOME_BONUS = float64(PRICE_PER_DAY) * 30 * 1 // Бонус новому пользователю (₽90)
	REFERRAL_MIN_BALANCE_FOR_REF = 0.0                       // Минимальный баланс для получения ссылки (0=любой)
	REFERRAL_CHECK_ENABLED = true                            // Проверка реферальных ссылок включена
	REFERRAL_CHECK_INTERVAL = 1440                           // Интервал проверки (мин, 1440=24ч)

	// ========== УВЕДОМЛЕНИЯ И НАПОМИНАНИЯ ==========

	// --- Уведомления пользователям (устаревшая система) ---
	NOTIFICATION_ENABLED = false              // Старая система уведомлений отключена
	NOTIFICATION_CHECK_INTERVAL = 60          // Интервал проверки (мин)
	NOTIFICATION_DAYS_BEFORE = []int{1, 3, 7} // Дни для отправки уведомлений
	NOTIFICATION_MESSAGE_1_DAY = "⚠️ <b>Ваша подписка истекает завтра!</b>\n\n" +
		"Не забудьте продлить подписку, чтобы не потерять доступ к пидписке.\n\n" +
		"Нажмите /balance для просмотра баланса и продления."
	NOTIFICATION_MESSAGE_3_DAYS = "🔔 <b>Напоминание о подписке</b>\n\n" +
		"Ваша подписка истекает через 3 дня.\n" +
		"Рекомендуем продлить её заранее.\n\n" +
		"Нажмите /balance для просмотра баланса и продления."
	NOTIFICATION_MESSAGE_7_DAYS = "📅 <b>Уведомление о подписке</b>\n\n" +
		"Ваша подписка истекает через неделю.\n" +
		"Не забудьте пополнить баланс и продлить подписку.\n\n" +
		"Нажмите /balance для просмотра баланса и продления."

	// --- Универсальные напоминания пользователям ---
	UNIVERSAL_REMINDER_ENABLED = false // Универсальные напоминания отключены
	UNIVERSAL_REMINDER_INTERVAL = 1440 // Интервал проверки (мин, 1440=24ч)
	UNIVERSAL_REMINDER_DAYS_BEFORE = 3 // Начинать напоминания за 3 дня
	UNIVERSAL_REMINDER_MESSAGE = "⏰ <b>Напоминание о подписке</b>\n\n" +
		"До окончания вашей подписки осталось: <b>{DAYS} дней {HOURS} часов</b>\n\n" +
		"Не хотелось бы, чтобы вы потеряли доступ к любимому сервису.\n\n" +
		"💡 Продлить очень просто: нажмите кнопку `Пополнить` в боте, выберите СБП и удобный для вас банк.\n\n" +
		"🌟 Спасибо, что остаётесь с нами!"
	UNIVERSAL_REMINDER_LOG_PATH = "/root/bot/logs/reminders.log" // Путь к логу напоминаний

	// --- Уведомления администратору ---
	ADMIN_NOTIFICATIONS_ENABLED = true   // Уведомления админу включены
	ADMIN_CONFIG_BLOCKING_ENABLED = true // Уведомления о блокировке конфигов
	ADMIN_IP_BAN_ENABLED = true          // Уведомления о IP банах
	ADMIN_BALANCE_TOPUP_ENABLED = true   // Уведомления о пополнениях
	ADMIN_REFERRAL_ENABLED = true        // Уведомления о рефералах
	ADMIN_REMINDER_ENABLED = true        // Уведомления о напоминаниях

	// ========== СИСТЕМЫ ОЧИСТКИ И ПРОВЕРКИ ==========
	DUPLICATE_CLEANUP_ENABLED = true     // Очистка дубликатов включена
	DUPLICATE_CLEANUP_INTERVAL = 1440    // Интервал очистки (мин, 1400≈23ч)
	RESET_STATUS_CHECK_ENABLED = true    // Проверка статуса reset включена
	RESET_STATUS_CHECK_INTERVAL = 720    // Интервал проверки (мин, 1400≈23ч)
	DEPLETED_STATUS_CHECK_ENABLED = true // Проверка ложных блокировок включена
	DEPLETED_STATUS_CHECK_INTERVAL = 120 // Интервал проверки (мин)
	DEPLETED_ACTIVE_THRESHOLD = 2        // Минимальная активность для сброса (ч)

	// ========== НАСТРОЙКИ ПЛАТЕЖЕЙ ==========
	// Telegram Bot API платежи
	YUKASSA_PROVIDER_TOKEN = "YOUR_PROVIDER_TOKEN_HERE" // Токен провайдера (от BotFather)
	TELEGRAM_PAYMENTS_ENABLED = false                   // Telegram платежи отключены

	// Прямая интеграция с ЮКассой
	YUKASSA_API_PAYMENTS_ENABLED = true // API платежи ЮКассы включены
	// YUKASSA_SHOP_ID, YUKASSA_SECRET_KEY и YUKASSA_WEBHOOK_URL задаются в .env

	// Настройки чеков
	YUKASSA_RECEIPT_ENABLED = true           // Отправка чеков включена
	YUKASSA_VAT_CODE = 1                     // НДС: 1=без НДС
	YUKASSA_PAYMENT_SUBJECT = "service"      // Предмет: service=услуга
	YUKASSA_PAYMENT_MODE = "full_prepayment" // Способ: полная предоплата

	// ========== БЕЗОПАСНОЕ ВЫКЛЮЧЕНИЕ ==========
	POWEROFF_SYSTEM_ENABLED = true       // Система безопасного выключения включена
	POWEROFF_PAYMENT_TIMEOUT = 5         // Таймаут ожидания платежей (мин)
	POWEROFF_CHECK_INTERVAL = 10         // Интервал проверки активных платежей (сек)
	POWEROFF_NOTIFICATION_ENABLED = true // Уведомления о выключении включены

	// ========== ЛОГИРОВАНИЕ ==========
	CONSOLE_LOG_ENABLED = false                         // Логирование консоли отключено
	CONSOLE_LOG_PATH = "/root/bot/logs/console.log"     // Путь к консольным логам
	USERS_LOG_ENABLED = true                            // Логирование пользователей включено
	USERS_LOG_PATH = "/root/bot/logs/users.log"         // Путь к логам пользователей
	PAYMENT_LOG_ENABLED = true                          // Логирование платежей включено
	PAYMENT_LOG_PATH = "/root/bot/logs/pay.log"         // Путь к логам платежей
	EXHAUSTED_LOG_ENABLED = true                        // Логирование статуса исчерпано включено по умолчанию
	EXHAUSTED_LOG_PATH = "/root/bot/logs/exhausted.log" // Путь к exhausted.log
}
