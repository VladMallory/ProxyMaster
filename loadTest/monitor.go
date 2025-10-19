package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// DatabaseMonitor мониторинг состояния базы данных
type DatabaseMonitor struct {
	db *sql.DB
}

// DatabaseStats статистика базы данных
type DatabaseStats struct {
	ActiveConnections int
	TotalConnections  int
	DatabaseSize      string
	TableSizes        map[string]string
	QueryStats        map[string]int64
	Timestamp         time.Time
}

func main() {
	log.Println("🔍 Запуск мониторинга базы данных")

	// Инициализация базы данных
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("❌ Ошибка инициализации базы данных: %v", err)
	}
	defer db.Close()

	monitor := &DatabaseMonitor{db: db}

	// Запускаем мониторинг каждые 10 секунд
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	log.Println("📊 Мониторинг запущен. Нажмите Ctrl+C для остановки...")

	for range ticker.C {
		stats, err := monitor.getDatabaseStats()
		if err != nil {
			log.Printf("❌ Ошибка получения статистики: %v", err)
			continue
		}

		monitor.printStats(stats)
	}
}

func initDatabase() (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		PG_HOST, PG_PORT, PG_USER, PG_PASSWORD, PG_DBNAME)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к PostgreSQL: %v", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка проверки соединения с PostgreSQL: %v", err)
	}

	return db, nil
}

func (m *DatabaseMonitor) getDatabaseStats() (*DatabaseStats, error) {
	stats := &DatabaseStats{
		TableSizes: make(map[string]string),
		QueryStats: make(map[string]int64),
		Timestamp:  time.Now(),
	}

	// Получаем информацию о соединениях
	if err := m.getConnectionStats(stats); err != nil {
		return nil, fmt.Errorf("ошибка получения статистики соединений: %v", err)
	}

	// Получаем размер базы данных
	if err := m.getDatabaseSize(stats); err != nil {
		return nil, fmt.Errorf("ошибка получения размера БД: %v", err)
	}

	// Получаем размеры таблиц
	if err := m.getTableSizes(stats); err != nil {
		return nil, fmt.Errorf("ошибка получения размеров таблиц: %v", err)
	}

	// Получаем статистику запросов
	if err := m.getQueryStats(stats); err != nil {
		return nil, fmt.Errorf("ошибка получения статистики запросов: %v", err)
	}

	return stats, nil
}

func (m *DatabaseMonitor) getConnectionStats(stats *DatabaseStats) error {
	query := `
		SELECT 
			(SELECT count(*) FROM pg_stat_activity WHERE state = 'active') as active_connections,
			(SELECT count(*) FROM pg_stat_activity) as total_connections`

	return m.db.QueryRow(query).Scan(&stats.ActiveConnections, &stats.TotalConnections)
}

func (m *DatabaseMonitor) getDatabaseSize(stats *DatabaseStats) error {
	query := `
		SELECT pg_size_pretty(pg_database_size(current_database())) as db_size`

	return m.db.QueryRow(query).Scan(&stats.DatabaseSize)
}

func (m *DatabaseMonitor) getTableSizes(stats *DatabaseStats) error {
	query := `
		SELECT 
			schemaname,
			tablename,
			pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
		FROM pg_tables 
		WHERE schemaname = 'public'
		ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC`

	rows, err := m.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table, size string
		if err := rows.Scan(&schema, &table, &size); err != nil {
			return err
		}
		stats.TableSizes[table] = size
	}

	return rows.Err()
}

func (m *DatabaseMonitor) getQueryStats(stats *DatabaseStats) error {
	// Получаем статистику по запросам
	queries := map[string]string{
		"SELECT": "SELECT count(*) FROM pg_stat_activity WHERE query LIKE 'SELECT%'",
		"INSERT": "SELECT count(*) FROM pg_stat_activity WHERE query LIKE 'INSERT%'",
		"UPDATE": "SELECT count(*) FROM pg_stat_activity WHERE query LIKE 'UPDATE%'",
		"DELETE": "SELECT count(*) FROM pg_stat_activity WHERE query LIKE 'DELETE%'",
	}

	for queryType, query := range queries {
		var count int64
		if err := m.db.QueryRow(query).Scan(&count); err != nil {
			stats.QueryStats[queryType] = 0
		} else {
			stats.QueryStats[queryType] = count
		}
	}

	return nil
}

func (m *DatabaseMonitor) printStats(stats *DatabaseStats) {
	log.Printf("\n🔍 МОНИТОРИНГ БД - %s", stats.Timestamp.Format("15:04:05"))
	log.Println("================================================")
	log.Printf("🔗 Активных соединений: %d", stats.ActiveConnections)
	log.Printf("🔗 Всего соединений: %d", stats.TotalConnections)
	log.Printf("💾 Размер БД: %s", stats.DatabaseSize)

	log.Println("\n📊 Размеры таблиц:")
	for table, size := range stats.TableSizes {
		log.Printf("   %s: %s", table, size)
	}

	log.Println("\n⚡ Активные запросы:")
	for queryType, count := range stats.QueryStats {
		if count > 0 {
			log.Printf("   %s: %d", queryType, count)
		}
	}

	// Проверяем состояние базы данных
	if stats.ActiveConnections > 50 {
		log.Println("⚠️  ВНИМАНИЕ: Высокое количество активных соединений!")
	}

	if stats.TotalConnections > 100 {
		log.Println("⚠️  ВНИМАНИЕ: Очень высокое количество соединений!")
	}
}
