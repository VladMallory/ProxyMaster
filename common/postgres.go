package common

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

// GetDB возвращает глобальное соединение с базой данных
func GetDB() *sql.DB {
	return db
}

// Типы уже определены в types.go

// Константы для PostgreSQL
const (
	PG_HOST     = "localhost"            // Хост PostgreSQL сервера
	PG_PORT     = 5432                   // Порт PostgreSQL (обычно 5432)
	PG_USER     = "vpn_bot_user"         // Имя пользователя БД
	PG_PASSWORD = "your_secure_password" // Пароль пользователя БД
	PG_DBNAME   = "vpn_bot"              // Название базы данных
)

// InitPostgreSQL инициализирует подключение к PostgreSQL
func InitPostgreSQL() error {
	// Получаем настройки из переменных окружения, если они есть
	host := getEnvOrDefault("PG_HOST", PG_HOST)
	port := getEnvOrDefault("PG_PORT", "5432")
	user := getEnvOrDefault("PG_USER", PG_USER)
	password := getEnvOrDefault("PG_PASSWORD", PG_PASSWORD)
	dbname := getEnvOrDefault("PG_DBNAME", PG_DBNAME)

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		return fmt.Errorf("ошибка подключения к PostgreSQL: %v", err)
	}

	// Проверяем соединение
	if err = db.Ping(); err != nil {
		return fmt.Errorf("ошибка проверки соединения с PostgreSQL: %v", err)
	}

	// Настройки пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("PostgreSQL подключен успешно")

	// Логируем информацию о пользователях после подключения
	logUsersAfterConnectionPG()

	// Запускаем сервис очистки старых IP подключений
	go startCleanupService()

	return nil
}

// DisconnectPostgreSQL отключается от PostgreSQL
func DisconnectPostgreSQL() {
	if db != nil {
		db.Close()
		log.Println("PostgreSQL отключен")
	}
}

// GetDatabase возвращает объект базы данных (для совместимости)
func GetDatabasePG() *sql.DB {
	return db
}

// GetOrCreateUser получает или создает пользователя (thread-safe с UPSERT)
func GetOrCreateUserPG(telegramID int64, username, firstName, lastName string) (*User, error) {
	now := time.Now()

	// Используем PostgreSQL UPSERT для атомарного создания или получения пользователя
	query := `
		INSERT INTO users (telegram_id, username, first_name, last_name, balance, total_paid, 
						   configs_count, has_active_config, has_used_trial, created_at, updated_at,
						   referral_code, referred_by, referral_earnings, referral_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (telegram_id) DO UPDATE SET
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			updated_at = EXCLUDED.updated_at
		RETURNING telegram_id`

	var returnedTelegramID int64
	err := db.QueryRow(query, telegramID, username, firstName, lastName, 0.0, 0.0,
		0, false, false, now, now, nil, nil, 0.0, 0).Scan(&returnedTelegramID)
	if err != nil {
		return nil, fmt.Errorf("ошибка UPSERT пользователя: %v", err)
	}

	// Получаем пользователя из базы (гарантированно существует после UPSERT)
	user, err := GetUserByTelegramIDPG(telegramID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пользователя после UPSERT: %v", err)
	}

	if user == nil {
		return nil, fmt.Errorf("пользователь не найден после UPSERT")
	}

	// Если пользователь только что создан (баланс = 0 и конфигов нет), проверяем панель
	if user.Balance == 0.0 && !user.HasActiveConfig && user.ClientID == "" {
		log.Printf("POSTGRES: Новый пользователь %d, проверяем панель для синхронизации", telegramID)
		go func() {
			syncUserWithPanel(user)
		}()
		log.Printf("POSTGRES: Создан новый пользователь: %s (ID: %d)", firstName, telegramID)
	} else {
		log.Printf("POSTGRES: Получен существующий пользователь: %s (ID: %d)", firstName, telegramID)
	}

	return user, nil
}

// syncUserWithPanel синхронизирует пользователя с панелью 3x-ui
func syncUserWithPanel(user *User) {
	if user == nil {
		return
	}

	// Авторизуемся в панели
	sessionCookie, err := Login()
	if err != nil {
		log.Printf("SYNC_PANEL: Ошибка авторизации для пользователя %d: %v", user.TelegramID, err)
		return
	}

	// Получаем наш inbound
	targetInbound, err := GetInbound(sessionCookie)
	if err != nil {
		log.Printf("SYNC_PANEL: Ошибка получения inbound для пользователя %d: %v", user.TelegramID, err)
		return
	}

	if targetInbound == nil {
		log.Printf("SYNC_PANEL: Inbound с ID %d не найден для пользователя %d", INBOUND_ID, user.TelegramID)
		return
	}

	// Парсим settings
	var settings Settings
	if err := json.Unmarshal([]byte(targetInbound.Settings), &settings); err != nil {
		log.Printf("SYNC_PANEL: Ошибка парсинга settings для пользователя %d: %v", user.TelegramID, err)
		return
	}

	// Ищем клиента пользователя
	existingClient := FindClientByTelegramID(settings.Clients, user.TelegramID)
	if existingClient != nil {
		log.Printf("SYNC_PANEL: Найден конфиг в панели для пользователя %d, синхронизируем", user.TelegramID)

		// Обновляем данные пользователя из панели
		user.ClientID = existingClient.ID
		user.SubID = existingClient.SubID
		user.Email = existingClient.Email
		user.ExpiryTime = existingClient.ExpiryTime
		user.HasActiveConfig = existingClient.Enable && time.Now().UnixMilli() < existingClient.ExpiryTime
		user.UpdatedAt = time.Now()

		// Сохраняем в базу
		if err := UpdateUserPG(user); err != nil {
			log.Printf("SYNC_PANEL: Ошибка обновления пользователя %d: %v", user.TelegramID, err)
		} else {
			log.Printf("SYNC_PANEL: Пользователь %d успешно синхронизирован с панелью", user.TelegramID)
		}
	} else {
		log.Printf("SYNC_PANEL: Конфиг в панели для пользователя %d не найден", user.TelegramID)
	}
}

// GetUserByTelegramID получает пользователя по Telegram ID
func GetUserByTelegramIDPG(telegramID int64) (*User, error) {
	query := `
		SELECT telegram_id, username, first_name, last_name, balance, total_paid,
			   configs_count, has_active_config, client_id, sub_id, email,
			   config_created_at, expiry_time, has_used_trial, created_at, updated_at,
			   referral_code, referred_by, referral_earnings, referral_count
		FROM users WHERE telegram_id = $1`

	var user User
	var configCreatedAt sql.NullTime
	var clientID, subID, email, referralCode sql.NullString
	var expiryTime, referredBy sql.NullInt64

	err := db.QueryRow(query, telegramID).Scan(
		&user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
		&user.Balance, &user.TotalPaid, &user.ConfigsCount, &user.HasActiveConfig,
		&clientID, &subID, &email, &configCreatedAt,
		&expiryTime, &user.HasUsedTrial, &user.CreatedAt, &user.UpdatedAt,
		&referralCode, &referredBy, &user.ReferralEarnings, &user.ReferralCount,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пользователя: %v", err)
	}

	// Обработка NULL значений
	if configCreatedAt.Valid {
		user.ConfigCreatedAt = configCreatedAt.Time
	}
	if clientID.Valid {
		user.ClientID = clientID.String
	}
	if subID.Valid {
		user.SubID = subID.String
	}
	if email.Valid {
		user.Email = email.String
	}
	if expiryTime.Valid {
		user.ExpiryTime = expiryTime.Int64
	}
	if referralCode.Valid {
		user.ReferralCode = referralCode.String
	}
	if referredBy.Valid {
		user.ReferredBy = referredBy.Int64
	}

	return &user, nil
}

// GetAllUsers получает всех пользователей
func GetAllUsersPG() ([]User, error) {
	query := `
		SELECT telegram_id, username, first_name, last_name, balance, total_paid,
			   configs_count, has_active_config, client_id, sub_id, email,
			   config_created_at, expiry_time, has_used_trial, created_at, updated_at
		FROM users ORDER BY created_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения всех пользователей: %v", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var configCreatedAt sql.NullTime
		var clientID, subID, email sql.NullString
		var expiryTime sql.NullInt64

		err := rows.Scan(
			&user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
			&user.Balance, &user.TotalPaid, &user.ConfigsCount, &user.HasActiveConfig,
			&clientID, &subID, &email, &configCreatedAt,
			&expiryTime, &user.HasUsedTrial, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования пользователя: %v", err)
		}

		// Обработка NULL значений
		if configCreatedAt.Valid {
			user.ConfigCreatedAt = configCreatedAt.Time
		}
		if clientID.Valid {
			user.ClientID = clientID.String
		}
		if subID.Valid {
			user.SubID = subID.String
		}
		if email.Valid {
			user.Email = email.String
		}
		if expiryTime.Valid {
			user.ExpiryTime = expiryTime.Int64
		}

		users = append(users, user)
	}

	return users, nil
}

// GetUsersWithActiveConfigsPG получает всех пользователей с активными конфигами
func GetUsersWithActiveConfigsPG() ([]User, error) {
	query := `
		SELECT telegram_id, username, first_name, last_name, balance, total_paid,
			   configs_count, has_active_config, client_id, sub_id, email,
			   config_created_at, expiry_time, has_used_trial, created_at, updated_at,
			   referral_code, referred_by, referral_earnings, referral_count
		FROM users 
		WHERE has_active_config = true AND expiry_time > 0
		ORDER BY created_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пользователей с активными конфигами: %v", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var configCreatedAt sql.NullTime
		var clientID, subID, email, referralCode sql.NullString
		var expiryTime, referredBy sql.NullInt64

		err := rows.Scan(
			&user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
			&user.Balance, &user.TotalPaid, &user.ConfigsCount, &user.HasActiveConfig,
			&clientID, &subID, &email, &configCreatedAt, &expiryTime,
			&user.HasUsedTrial, &user.CreatedAt, &user.UpdatedAt,
			&referralCode, &referredBy, &user.ReferralEarnings, &user.ReferralCount,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования пользователя: %v", err)
		}

		if configCreatedAt.Valid {
			user.ConfigCreatedAt = configCreatedAt.Time
		}
		if clientID.Valid {
			user.ClientID = clientID.String
		}
		if subID.Valid {
			user.SubID = subID.String
		}
		if email.Valid {
			user.Email = email.String
		}
		if expiryTime.Valid {
			user.ExpiryTime = expiryTime.Int64
		}
		if referralCode.Valid {
			user.ReferralCode = referralCode.String
		}
		if referredBy.Valid {
			user.ReferredBy = referredBy.Int64
		}

		users = append(users, user)
	}

	return users, nil
}

// UpdateUser обновляет данные пользователя
func UpdateUserPG(user *User) error {
	query := `
		UPDATE users SET 
			username = $2, first_name = $3, last_name = $4, balance = $5,
			total_paid = $6, configs_count = $7, has_active_config = $8,
			client_id = $9, sub_id = $10, email = $11, config_created_at = $12,
			expiry_time = $13, has_used_trial = $14, updated_at = $15,
			referral_code = $16, referred_by = $17, referral_earnings = $18, referral_count = $19
		WHERE telegram_id = $1`

	var configCreatedAt interface{}
	if !user.ConfigCreatedAt.IsZero() {
		configCreatedAt = user.ConfigCreatedAt
	}

	_, err := db.Exec(query,
		user.TelegramID, user.Username, user.FirstName, user.LastName,
		user.Balance, user.TotalPaid, user.ConfigsCount, user.HasActiveConfig,
		nullIfEmpty(user.ClientID), nullIfEmpty(user.SubID), nullIfEmpty(user.Email),
		configCreatedAt, user.ExpiryTime, user.HasUsedTrial, time.Now(),
		nullIfEmpty(user.ReferralCode), user.ReferredBy, user.ReferralEarnings, user.ReferralCount,
	)
	if err != nil {
		return fmt.Errorf("ошибка обновления пользователя: %v", err)
	}

	return nil
}

// AddBalance добавляет баланс пользователю
func AddBalancePG(telegramID int64, amount float64) error {
	query := `
		UPDATE users SET 
			balance = balance + $2,
			total_paid = total_paid + $2,
			updated_at = $3
		WHERE telegram_id = $1`

	_, err := db.Exec(query, telegramID, amount, time.Now())
	if err != nil {
		return fmt.Errorf("ошибка добавления баланса: %v", err)
	}

	// Отправляем уведомление администратору о пополнении баланса
	user, err := GetUserByTelegramID(telegramID)
	if err != nil {
		log.Printf("POSTGRES: Ошибка получения данных пользователя %d для уведомления администратору: %v", telegramID, err)
	} else {
		SendBalanceTopupNotificationToAdmin(user, amount)
	}

	// Запускаем принудительный пересчет периода подписки после пополнения баланса
	// Добавляем небольшую задержку, чтобы база данных успела обновиться
	go func() {
		time.Sleep(100 * time.Millisecond) // 100ms задержка
		log.Printf("POSTGRES: Запуск принудительного пересчета после пополнения баланса для пользователя %d на сумму %.2f₽", telegramID, amount)

		// Сбрасываем статус "исчерпано" если он есть (только для пользователей с проблемами)
		sessionCookie, err := Login()
		if err != nil {
			log.Printf("POSTGRES: Ошибка авторизации для сброса статуса 'исчерпано' для пользователя %d: %v", telegramID, err)
		} else {
			log.Printf("POSTGRES: Попытка сброса статуса 'исчерпано' для пользователя %d после пополнения баланса", telegramID)
			if err := ForceResetDepletedStatus(sessionCookie, telegramID); err != nil {
				// Это нормально - значит у пользователя нет статуса "исчерпано" или конфига
				log.Printf("POSTGRES: Статус 'исчерпано' не найден для пользователя %d (это нормально): %v", telegramID, err)
			} else {
				log.Printf("POSTGRES: ✅ Статус 'исчерпано' успешно сброшен для пользователя %d после пополнения баланса", telegramID)
			}
		}

		ForceBalanceRecalculation(telegramID)
	}()

	return nil
}

// UpdateTrialFlag обновляет флаг использования пробного периода
func UpdateTrialFlagPG(telegramID int64) error {
	query := `UPDATE users SET has_used_trial = true WHERE telegram_id = $1`
	_, err := db.Exec(query, telegramID)
	if err != nil {
		return fmt.Errorf("ошибка обновления флага пробного периода: %v", err)
	}
	return nil
}

// ResetTrialFlag сбрасывает флаг использования пробного периода
func ResetTrialFlagPG(telegramID int64) error {
	query := `UPDATE users SET has_used_trial = false WHERE telegram_id = $1`
	_, err := db.Exec(query, telegramID)
	if err != nil {
		return fmt.Errorf("ошибка сброса флага пробного периода: %v", err)
	}
	return nil
}

// ClearAllUsers удаляет всех пользователей
func ClearAllUsersPG() error {
	_, err := db.Exec("DELETE FROM users")
	if err != nil {
		return fmt.Errorf("ошибка очистки пользователей: %v", err)
	}
	return nil
}

// ClearDatabase очищает всю базу данных
func ClearDatabasePG() error {
	tables := []string{"ip_violations", "ip_connections", "users", "traffic_configs"}

	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s", table)
		_, err := db.Exec(query)
		if err != nil {
			return fmt.Errorf("ошибка очистки таблицы %s: %v", table, err)
		}
	}

	// Восстанавливаем конфигурацию по умолчанию
	return createDefaultTrafficConfig()
}

// ResetAllTrialFlags сбрасывает флаги пробных периодов для всех пользователей
func ResetAllTrialFlagsPG() error {
	query := `UPDATE users SET has_used_trial = false, updated_at = $1`

	result, err := db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("ошибка сброса флагов пробных периодов: %v", err)
	}

	affected, _ := result.RowsAffected()
	log.Printf("Сброшены флаги пробных периодов для %d пользователей", affected)
	return nil
}

// GetTrafficConfig получает конфигурацию трафика
func GetTrafficConfigPG() *TrafficConfig {
	query := `
		SELECT enabled, daily_limit_gb, weekly_limit_gb, monthly_limit_gb, limit_gb, reset_days
		FROM traffic_configs WHERE id = 'default'`

	var config TrafficConfig
	err := db.QueryRow(query).Scan(
		&config.Enabled, &config.DailyLimitGB, &config.WeeklyLimitGB,
		&config.MonthlyLimitGB, &config.LimitGB, &config.ResetDays,
	)

	if err == sql.ErrNoRows {
		// Создаем конфигурацию по умолчанию
		config = TrafficConfig{
			Enabled:        true,
			DailyLimitGB:   0,
			WeeklyLimitGB:  0,
			MonthlyLimitGB: 0,
			LimitGB:        0,
			ResetDays:      30,
		}

		createDefaultTrafficConfig()
		log.Println("Создана конфигурация трафика по умолчанию")
	} else if err != nil {
		log.Printf("Ошибка получения конфигурации трафика: %v", err)
		// Возвращаем конфигурацию по умолчанию
		config = TrafficConfig{
			Enabled:        true,
			DailyLimitGB:   0,
			WeeklyLimitGB:  0,
			MonthlyLimitGB: 0,
			LimitGB:        0,
			ResetDays:      30,
		}
	}

	return &config
}

// SetTrafficConfig сохраняет конфигурацию трафика
func SetTrafficConfigPG(config *TrafficConfig) error {
	query := `
		INSERT INTO traffic_configs (id, enabled, daily_limit_gb, weekly_limit_gb, monthly_limit_gb, limit_gb, reset_days, updated_at)
		VALUES ('default', $1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			daily_limit_gb = EXCLUDED.daily_limit_gb,
			weekly_limit_gb = EXCLUDED.weekly_limit_gb,
			monthly_limit_gb = EXCLUDED.monthly_limit_gb,
			limit_gb = EXCLUDED.limit_gb,
			reset_days = EXCLUDED.reset_days,
			updated_at = EXCLUDED.updated_at`

	_, err := db.Exec(query,
		config.Enabled, config.DailyLimitGB, config.WeeklyLimitGB,
		config.MonthlyLimitGB, config.LimitGB, config.ResetDays, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("ошибка сохранения конфигурации трафика: %v", err)
	}

	return nil
}

// GetUsersStatistics получает статистику пользователей
func GetUsersStatisticsPG() (*UsersStatistics, error) {
	query := `SELECT * FROM get_users_statistics()`

	var stats UsersStatistics
	err := db.QueryRow(query).Scan(
		&stats.TotalUsers, &stats.PayingUsers, &stats.TrialAvailableUsers,
		&stats.TrialUsedUsers, &stats.InactiveUsers, &stats.ActiveConfigs,
		&stats.TotalRevenue, &stats.NewThisWeek, &stats.NewThisMonth,
		&stats.ConversionRate,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения статистики: %v", err)
	}

	return &stats, nil
}

// Вспомогательные функции

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func createDefaultTrafficConfig() error {
	query := `
		INSERT INTO traffic_configs (id, enabled, daily_limit_gb, weekly_limit_gb, monthly_limit_gb, limit_gb, reset_days)
		VALUES ('default', true, 0, 0, 0, 0, 30)
		ON CONFLICT (id) DO NOTHING`

	_, err := db.Exec(query)
	return err
}

// startCleanupService запускает сервис очистки старых IP подключений
func startCleanupService() {
	ticker := time.NewTicker(10 * time.Minute) // Каждые 10 минут
	defer ticker.Stop()

	for range ticker.C {
		cleanupOldConnections()
	}
}

// cleanupOldConnections очищает старые IP подключения (аналог TTL в MongoDB)
func cleanupOldConnections() {
	query := `SELECT cleanup_old_ip_connections()`
	var deletedCount int

	err := db.QueryRow(query).Scan(&deletedCount)
	if err != nil {
		log.Printf("Ошибка очистки старых подключений: %v", err)
	}
}

// logUsersAfterConnection выводит информацию о пользователях после подключения
func logUsersAfterConnectionPG() {
	users, err := GetAllUsersPG()
	if err != nil {
		log.Printf("INIT_POSTGRESQL: Ошибка получения пользователей: %v", err)
		return
	}

	log.Printf("INIT_POSTGRESQL: ========================================")
	log.Printf("INIT_POSTGRESQL: 📊 ИНФОРМАЦИЯ О ПОЛЬЗОВАТЕЛЯХ В БД")
	log.Printf("INIT_POSTGRESQL: ========================================")
	log.Printf("INIT_POSTGRESQL: Всего пользователей в базе: %d", len(users))

	if len(users) > 0 {
		// Выводим список пользователей (максимум 50)
		logUsersListPG(users)
	} else {
		log.Printf("INIT_POSTGRESQL: База данных пуста - пользователей нет")
	}

	log.Printf("INIT_POSTGRESQL: ========================================")
}

// logUsersListPG выводит список пользователей
func logUsersListPG(users []User) {
	displayCount := len(users)
	if displayCount > 50 {
		displayCount = 50
	}

	for i := 0; i < displayCount; i++ {
		user := users[i]
		status := "неактивен"
		if user.HasActiveConfig {
			status = "активен"
		}

		trialStatus := "доступен"
		if user.HasUsedTrial {
			trialStatus = "использован"
		}

		log.Printf("INIT_POSTGRESQL: %d) @%s (%s %s) - Баланс: %.2f₽, Статус: %s, Пробный: %s",
			i+1, user.Username, user.FirstName, user.LastName,
			user.Balance, status, trialStatus)
	}

	if len(users) > 50 {
		log.Printf("INIT_POSTGRESQL: ... и еще %d пользователей", len(users)-50)
	}
}

// BackupPostgreSQL создает бэкап базы данных
func BackupPostgreSQLPG() error {
	// Получаем настройки из переменных окружения
	host := getEnvOrDefault("PG_HOST", PG_HOST)
	port := getEnvOrDefault("PG_PORT", "5432")
	user := getEnvOrDefault("PG_USER", PG_USER)
	dbname := getEnvOrDefault("PG_DBNAME", PG_DBNAME)

	timestamp := time.Now().Format("20060102_150405")
	backupDir := fmt.Sprintf("backups/backupdb/backup_%s", timestamp)

	// Создаем директорию для бэкапа
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return fmt.Errorf("ошибка создания директории бэкапа: %v", err)
	}

	backupFile := fmt.Sprintf("%s/vpn_bot_backup.sql", backupDir)

	// Команда pg_dump
	cmd := fmt.Sprintf("PGPASSWORD='%s' pg_dump -h %s -p %s -U %s -d %s > %s",
		getEnvOrDefault("PG_PASSWORD", PG_PASSWORD), host, port, user, dbname, backupFile)

	err := executeCommand(cmd)
	if err != nil {
		return fmt.Errorf("ошибка создания бэкапа PostgreSQL: %v", err)
	}

	log.Printf("Бэкап PostgreSQL создан: %s", backupFile)
	return nil
}

// executeCommand выполняет системную команду
func executeCommand(cmd string) error {
	return exec.Command("sh", "-c", cmd).Run()
}

// RestorePostgreSQLPG восстанавливает базу данных из SQL файла
func RestorePostgreSQLPG() error {
	log.Printf("RESTORE_POSTGRESQL: ========================================")
	log.Printf("RESTORE_POSTGRESQL: НАЧАЛО ВОССТАНОВЛЕНИЯ БД ИЗ БЭКАПА")
	log.Printf("RESTORE_POSTGRESQL: ========================================")

	// Ищем последний бэкап в backupdb
	log.Printf("RESTORE_POSTGRESQL: Поиск последнего бэкапа в backupdb...")
	backupDir := "./backups/backupdb"
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("RESTORE_POSTGRESQL: ❌ Директория бэкапов не найдена, пропускаем восстановление")
			log.Printf("RESTORE_POSTGRESQL: ========================================")
			log.Printf("RESTORE_POSTGRESQL: ВОССТАНОВЛЕНИЕ ПРОПУЩЕНО - БЭКАПОВ НЕТ")
			log.Printf("RESTORE_POSTGRESQL: ========================================")
			return nil
		}
		return fmt.Errorf("ошибка чтения директории бэкапов: %v", err)
	}

	// Фильтруем директории бэкапов и сортируем по имени (последний по времени)
	var backupDirs []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "backup_") {
			backupDirs = append(backupDirs, entry.Name())
		}
	}

	if len(backupDirs) == 0 {
		log.Printf("RESTORE_POSTGRESQL: ❌ Бэкапы не найдены, пропускаем восстановление")
		log.Printf("RESTORE_POSTGRESQL: ========================================")
		log.Printf("RESTORE_POSTGRESQL: ВОССТАНОВЛЕНИЕ ПРОПУЩЕНО - БЭКАПОВ НЕТ")
		log.Printf("RESTORE_POSTGRESQL: ========================================")
		return nil
	}

	sort.Strings(backupDirs)
	latestBackup := backupDirs[len(backupDirs)-1]
	backupPath := filepath.Join(backupDir, latestBackup, "vpn_bot_backup.sql")

	log.Printf("RESTORE_POSTGRESQL: ✅ Найден последний бэкап: %s", backupPath)
	return restoreFromSQLFile(backupPath)
}

// restoreFromSQLFile восстанавливает данные из SQL файла
func restoreFromSQLFile(sqlFilePath string) error {
	log.Printf("RESTORE_FROM_SQL_FILE: Начало восстановления из %s", sqlFilePath)

	// Проверяем, существует ли файл
	if _, err := os.Stat(sqlFilePath); os.IsNotExist(err) {
		return fmt.Errorf("файл бэкапа не найден: %s", sqlFilePath)
	}

	// Получаем настройки подключения
	host := getEnvOrDefault("PG_HOST", PG_HOST)
	port := getEnvOrDefault("PG_PORT", "5432")
	user := getEnvOrDefault("PG_USER", PG_USER)
	password := getEnvOrDefault("PG_PASSWORD", PG_PASSWORD)
	dbname := getEnvOrDefault("PG_DBNAME", PG_DBNAME)

	// Команда psql для восстановления
	cmd := fmt.Sprintf("PGPASSWORD='%s' psql -h %s -p %s -U %s -d %s -f %s",
		password, host, port, user, dbname, sqlFilePath)

	log.Printf("RESTORE_FROM_SQL_FILE: Выполняем команду восстановления...")
	err := executeCommand(cmd)
	if err != nil {
		return fmt.Errorf("ошибка восстановления PostgreSQL: %v", err)
	}

	log.Printf("RESTORE_FROM_SQL_FILE: ✅ Данные успешно восстановлены из %s", sqlFilePath)
	log.Printf("RESTORE_FROM_SQL_FILE: ========================================")
	log.Printf("RESTORE_FROM_SQL_FILE: ВОССТАНОВЛЕНИЕ ЗАВЕРШЕНО УСПЕШНО")
	log.Printf("RESTORE_FROM_SQL_FILE: ========================================")
	return nil
}
