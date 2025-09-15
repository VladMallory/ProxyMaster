package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// Константы для PostgreSQL (копируем из основного проекта)
const (
	PG_HOST     = "localhost"
	PG_PORT     = 5432
	PG_USER     = "your_db_user"
	PG_PASSWORD = "your_secure_password"
	PG_DBNAME   = "your_database_name"
)

// User структура для тестирования
type User struct {
	TelegramID      int64     `json:"telegram_id"`
	Username        string    `json:"username"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	Balance         float64   `json:"balance"`
	TotalPaid       float64   `json:"total_paid"`
	ConfigsCount    int       `json:"configs_count"`
	HasActiveConfig bool      `json:"has_active_config"`
	ClientID        string    `json:"client_id"`
	SubID           string    `json:"sub_id"`
	Email           string    `json:"email"`
	ConfigCreatedAt time.Time `json:"config_created_at"`
	ExpiryTime      int64     `json:"expiry_time"`
	HasUsedTrial    bool      `json:"has_used_trial"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TrafficConfig структура для тестирования
type TrafficConfig struct {
	Enabled        bool `json:"enabled"`
	DailyLimitGB   int  `json:"daily_limit_gb"`
	WeeklyLimitGB  int  `json:"weekly_limit_gb"`
	MonthlyLimitGB int  `json:"monthly_limit_gb"`
	LimitGB        int  `json:"limit_gb"`
	ResetDays      int  `json:"reset_days"`
}

// LoadTestConfig конфигурация нагрузочного тестирования
type LoadTestConfig struct {
	Duration        time.Duration // Продолжительность теста
	ConcurrentUsers int           // Количество одновременных пользователей
	ReadWeight      int           // Вес операций чтения (1-10)
	WriteWeight     int           // Вес операций записи (1-10)
	UpdateWeight    int           // Вес операций обновления (1-10)
}

// LoadTestStats статистика нагрузочного тестирования
type LoadTestStats struct {
	TotalOperations     int64
	ReadOperations      int64
	WriteOperations     int64
	UpdateOperations    int64
	Errors              int64
	AverageResponseTime time.Duration
	MaxResponseTime     time.Duration
	MinResponseTime     time.Duration
	StartTime           time.Time
	EndTime             time.Time
}

var (
	db         *sql.DB
	stats      LoadTestStats
	statsMutex sync.RWMutex
)

func main() {
	log.Println("🚀 Запуск нагрузочного тестирования базы данных")
	log.Println("================================================")

	// Инициализация базы данных
	if err := initDatabase(); err != nil {
		log.Fatalf("❌ Ошибка инициализации базы данных: %v", err)
	}
	defer db.Close()

	// Конфигурация теста
	config := LoadTestConfig{
		Duration:        5 * time.Minute, // 5 минут
		ConcurrentUsers: 200,             // 200 клиентов
		ReadWeight:      5,               // 50% операций чтения
		WriteWeight:     3,               // 30% операций записи
		UpdateWeight:    2,               // 20% операций обновления
	}

	log.Printf("📊 Конфигурация теста:")
	log.Printf("   - Продолжительность: %v", config.Duration)
	log.Printf("   - Одновременных пользователей: %d", config.ConcurrentUsers)
	log.Printf("   - Распределение операций: Чтение=%d, Запись=%d, Обновление=%d",
		config.ReadWeight, config.WriteWeight, config.UpdateWeight)

	// Запуск теста
	runLoadTest(config)

	// Вывод результатов
	printResults()
}

func initDatabase() error {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		PG_HOST, PG_PORT, PG_USER, PG_PASSWORD, PG_DBNAME)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		return fmt.Errorf("ошибка подключения к PostgreSQL: %v", err)
	}

	// Проверяем соединение
	if err = db.Ping(); err != nil {
		return fmt.Errorf("ошибка проверки соединения с PostgreSQL: %v", err)
	}

	// Настройки пула соединений для нагрузочного тестирования
	db.SetMaxOpenConns(50) // Увеличиваем для нагрузки
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(10 * time.Minute)

	log.Println("✅ Подключение к PostgreSQL установлено")
	return nil
}

func runLoadTest(config LoadTestConfig) {
	log.Println("🔥 Начинаем нагрузочное тестирование...")

	stats.StartTime = time.Now()

	var wg sync.WaitGroup
	stopChan := make(chan bool)

	// Запускаем горутины для каждого "клиента"
	for i := 0; i < config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			simulateClient(clientID, config, stopChan)
		}(i)
	}

	// Запускаем таймер
	go func() {
		time.Sleep(config.Duration)
		close(stopChan)
	}()

	// Ждем завершения всех горутин
	wg.Wait()

	stats.EndTime = time.Now()
	log.Println("✅ Нагрузочное тестирование завершено")
}

func simulateClient(clientID int, config LoadTestConfig, stopChan <-chan bool) {
	rand.Seed(time.Now().UnixNano() + int64(clientID))

	for {
		select {
		case <-stopChan:
			return
		default:
			// Выбираем тип операции на основе весов
			operation := selectOperation(config)

			// Выполняем операцию
			start := time.Now()
			err := performOperation(operation, clientID)
			duration := time.Since(start)

			// Обновляем статистику
			updateStats(operation, duration, err)

			// Небольшая пауза между операциями (имитация реального использования)
			time.Sleep(time.Duration(rand.Intn(100)+50) * time.Millisecond)
		}
	}
}

func selectOperation(config LoadTestConfig) string {
	totalWeight := config.ReadWeight + config.WriteWeight + config.UpdateWeight
	random := rand.Intn(totalWeight)

	if random < config.ReadWeight {
		return "read"
	} else if random < config.ReadWeight+config.WriteWeight {
		return "write"
	} else {
		return "update"
	}
}

func performOperation(operation string, clientID int) error {
	switch operation {
	case "read":
		return performReadOperation(clientID)
	case "write":
		return performWriteOperation(clientID)
	case "update":
		return performUpdateOperation(clientID)
	default:
		return fmt.Errorf("неизвестная операция: %s", operation)
	}
}

func performReadOperation(clientID int) error {
	// Читаем случайного пользователя
	query := `SELECT telegram_id, username, first_name, last_name, balance, total_paid,
			   configs_count, has_active_config, client_id, sub_id, email,
			   config_created_at, expiry_time, has_used_trial, created_at, updated_at
			   FROM users ORDER BY RANDOM() LIMIT 1`

	var user User
	var configCreatedAt sql.NullTime
	var clientIDStr, subID, email sql.NullString
	var expiryTime sql.NullInt64

	err := db.QueryRow(query).Scan(
		&user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
		&user.Balance, &user.TotalPaid, &user.ConfigsCount, &user.HasActiveConfig,
		&clientIDStr, &subID, &email, &configCreatedAt,
		&expiryTime, &user.HasUsedTrial, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Если пользователей нет, это не ошибка
		return nil
	}

	return err
}

func performWriteOperation(clientID int) error {
	// Создаем тестового пользователя
	telegramID := int64(1000000 + clientID + rand.Intn(10000))
	username := fmt.Sprintf("testuser_%d_%d", clientID, time.Now().Unix())
	firstName := fmt.Sprintf("Test%d", clientID)
	lastName := fmt.Sprintf("User%d", clientID)

	query := `
		INSERT INTO users (telegram_id, username, first_name, last_name, balance, total_paid, 
						   configs_count, has_active_config, has_used_trial, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (telegram_id) DO NOTHING`

	now := time.Now()
	_, err := db.Exec(query, telegramID, username, firstName, lastName,
		rand.Float64()*1000, rand.Float64()*500, rand.Intn(5),
		rand.Float64() < 0.3, rand.Float64() < 0.5, now, now)

	return err
}

func performUpdateOperation(clientID int) error {
	// Обновляем случайного пользователя
	query := `UPDATE users SET 
		balance = balance + $1,
		updated_at = $2
		WHERE telegram_id IN (
			SELECT telegram_id FROM users ORDER BY RANDOM() LIMIT 1
		)`

	balanceChange := (rand.Float64() - 0.5) * 100 // -50 до +50
	_, err := db.Exec(query, balanceChange, time.Now())

	return err
}

func updateStats(operation string, duration time.Duration, err error) {
	statsMutex.Lock()
	defer statsMutex.Unlock()

	stats.TotalOperations++

	switch operation {
	case "read":
		stats.ReadOperations++
	case "write":
		stats.WriteOperations++
	case "update":
		stats.UpdateOperations++
	}

	if err != nil {
		stats.Errors++
	}

	// Обновляем время ответа
	if stats.AverageResponseTime == 0 {
		stats.AverageResponseTime = duration
		stats.MaxResponseTime = duration
		stats.MinResponseTime = duration
	} else {
		// Простое скользящее среднее
		stats.AverageResponseTime = (stats.AverageResponseTime + duration) / 2

		if duration > stats.MaxResponseTime {
			stats.MaxResponseTime = duration
		}
		if duration < stats.MinResponseTime {
			stats.MinResponseTime = duration
		}
	}
}

func printResults() {
	statsMutex.RLock()
	defer statsMutex.RUnlock()

	duration := stats.EndTime.Sub(stats.StartTime)
	opsPerSecond := float64(stats.TotalOperations) / duration.Seconds()
	errorRate := float64(stats.Errors) / float64(stats.TotalOperations) * 100

	log.Println("\n📊 РЕЗУЛЬТАТЫ НАГРУЗОЧНОГО ТЕСТИРОВАНИЯ")
	log.Println("================================================")
	log.Printf("⏱️  Общее время теста: %v", duration)
	log.Printf("🔄 Всего операций: %d", stats.TotalOperations)
	log.Printf("⚡ Операций в секунду: %.2f", opsPerSecond)
	log.Printf("📖 Операций чтения: %d (%.1f%%)", stats.ReadOperations,
		float64(stats.ReadOperations)/float64(stats.TotalOperations)*100)
	log.Printf("✍️  Операций записи: %d (%.1f%%)", stats.WriteOperations,
		float64(stats.WriteOperations)/float64(stats.TotalOperations)*100)
	log.Printf("🔄 Операций обновления: %d (%.1f%%)", stats.UpdateOperations,
		float64(stats.UpdateOperations)/float64(stats.TotalOperations)*100)
	log.Printf("❌ Ошибок: %d (%.2f%%)", stats.Errors, errorRate)
	log.Printf("⏱️  Среднее время ответа: %v", stats.AverageResponseTime)
	log.Printf("⏱️  Максимальное время ответа: %v", stats.MaxResponseTime)
	log.Printf("⏱️  Минимальное время ответа: %v", stats.MinResponseTime)

	// Оценка производительности
	log.Println("\n🎯 ОЦЕНКА ПРОИЗВОДИТЕЛЬНОСТИ")
	log.Println("================================================")

	if opsPerSecond > 1000 {
		log.Println("🟢 ОТЛИЧНО: База данных справляется с высокой нагрузкой")
	} else if opsPerSecond > 500 {
		log.Println("🟡 ХОРОШО: База данных работает стабильно")
	} else if opsPerSecond > 100 {
		log.Println("🟠 УДОВЛЕТВОРИТЕЛЬНО: База данных работает, но может потребовать оптимизации")
	} else {
		log.Println("🔴 ПЛОХО: База данных не справляется с нагрузкой")
	}

	if errorRate < 1 {
		log.Println("🟢 ОТЛИЧНО: Очень низкий уровень ошибок")
	} else if errorRate < 5 {
		log.Println("🟡 ХОРОШО: Приемлемый уровень ошибок")
	} else {
		log.Println("🔴 ПЛОХО: Высокий уровень ошибок")
	}

	log.Println("\n✅ Нагрузочное тестирование завершено!")
}
