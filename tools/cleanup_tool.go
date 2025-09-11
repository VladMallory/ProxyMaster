package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Константы для PostgreSQL
const (
	PG_HOST = "localhost"
	PG_PORT = 5432

	PG_USER     = "your_db_user"
	PG_PASSWORD = "your_secure_password"
	PG_DBNAME   = "your_database_name"
)

// User структура пользователя
type User struct {
	ID              int64     `db:"id"`
	TelegramID      int64     `db:"telegram_id"`
	Username        string    `db:"username"`
	FirstName       string    `db:"first_name"`
	LastName        string    `db:"last_name"`
	Balance         float64   `db:"balance"`
	TotalPaid       float64   `db:"total_paid"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
	HasActiveConfig bool      `db:"has_active_config"`
	ClientID        string    `db:"client_id"`
	Email           string    `db:"email"`
	SubID           string    `db:"sub_id"`
	ConfigCreatedAt time.Time `db:"config_created_at"`
	ExpiryTime      int64     `db:"expiry_time"`
	ConfigsCount    int       `db:"configs_count"`
	HasUsedTrial    bool      `db:"has_used_trial"`
}

var db *sql.DB

// InitPostgreSQL инициализирует соединение с PostgreSQL
func InitPostgreSQL() error {
	// Получаем настройки из переменных окружения, если они есть
	host := getEnvOrDefault("PG_HOST", "localhost")
	port := getEnvOrDefault("PG_PORT", "5432")
	user := getEnvOrDefault("PG_USER", PG_USER)
	password := getEnvOrDefault("PG_PASSWORD", PG_PASSWORD)
	dbname := getEnvOrDefault("PG_DBNAME", PG_DBNAME)
	sslMode := getEnvOrDefault("PG_SSLMODE", "disable")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslMode)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		return fmt.Errorf("ошибка подключения к PostgreSQL: %v", err)
	}

	if err = db.Ping(); err != nil {
		return fmt.Errorf("ошибка проверки соединения с PostgreSQL: %v", err)
	}

	log.Println("Успешно подключено к PostgreSQL")
	return nil
}

// DisconnectPostgreSQL закрывает соединение с PostgreSQL
func DisconnectPostgreSQL() {
	if db != nil {
		if err := db.Close(); err != nil {
			log.Printf("Ошибка отключения от PostgreSQL: %v", err)
		}
	}
}

// ResetAllTrialFlags сбрасывает флаг HasUsedTrial у всех пользователей
func ResetAllTrialFlags() error {
	query := `UPDATE users SET has_used_trial = false, updated_at = $1`

	result, err := db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("ошибка сброса флагов пробных периодов: %v", err)
	}

	affected, _ := result.RowsAffected()
	log.Printf("Сброшены флаги пробных периодов для %d пользователей", affected)
	return nil
}

// ResetUserTrialFlag сбрасывает флаг HasUsedTrial у конкретного пользователя
func ResetUserTrialFlag(telegramID int64) error {
	query := `UPDATE users SET has_used_trial = false, updated_at = $1 WHERE telegram_id = $2`

	result, err := db.Exec(query, time.Now(), telegramID)
	if err != nil {
		return fmt.Errorf("ошибка сброса флага пробного периода для пользователя %d: %v", telegramID, err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("пользователь с Telegram ID %d не найден", telegramID)
	}

	log.Printf("Сброшен флаг пробного периода для пользователя %d", telegramID)
	return nil
}

// DeleteUser удаляет пользователя
func DeleteUser(telegramID int64) error {
	query := `DELETE FROM users WHERE telegram_id = $1`

	result, err := db.Exec(query, telegramID)
	if err != nil {
		return fmt.Errorf("ошибка удаления пользователя %d: %v", telegramID, err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("пользователь с Telegram ID %d не найден", telegramID)
	}

	log.Printf("Удален пользователь %d", telegramID)
	return nil
}

// GetAllUsers получает всех пользователей
func GetAllUsers() ([]User, error) {
	query := `
		SELECT id, telegram_id, username, first_name, last_name, balance, total_paid,
			   created_at, updated_at, has_active_config, 
			   COALESCE(client_id, ''), COALESCE(email, ''), COALESCE(sub_id, ''),
			   COALESCE(config_created_at, '1970-01-01'::timestamp), 
			   COALESCE(expiry_time, 0), configs_count, has_used_trial
		FROM users ORDER BY created_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка пользователей: %v", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
			&user.Balance, &user.TotalPaid, &user.CreatedAt, &user.UpdatedAt,
			&user.HasActiveConfig, &user.ClientID, &user.Email, &user.SubID,
			&user.ConfigCreatedAt, &user.ExpiryTime, &user.ConfigsCount, &user.HasUsedTrial,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования пользователя: %v", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// ShowUsers показывает всех пользователей
func ShowUsers() error {
	users, err := GetAllUsers()
	if err != nil {
		return err
	}

	fmt.Printf("\n=== СПИСОК ПОЛЬЗОВАТЕЛЕЙ ===\n")
	fmt.Printf("Всего пользователей: %d\n\n", len(users))

	if len(users) == 0 {
		fmt.Println("Пользователей нет")
		return nil
	}

	for i, user := range users {
		status := "неактивен"
		if user.HasActiveConfig {
			status = "активен"
		}

		trialStatus := "доступен"
		if user.HasUsedTrial {
			trialStatus = "использован"
		}

		fmt.Printf("%d) Telegram ID: %d\n", i+1, user.TelegramID)
		fmt.Printf("   Имя: %s %s (@%s)\n", user.FirstName, user.LastName, user.Username)
		fmt.Printf("   Баланс: %.2f₽ (всего оплачено: %.2f₽)\n", user.Balance, user.TotalPaid)
		fmt.Printf("   Статус: %s | Пробный период: %s\n", status, trialStatus)
		fmt.Printf("   Создан: %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))
		if user.HasActiveConfig {
			fmt.Printf("   Email: %s | SubID: %s\n", user.Email, user.SubID)
		}
		fmt.Println("   " + strings.Repeat("-", 50))
	}

	return nil
}

// ClearAllUsers удаляет всех пользователей
func ClearAllUsers() error {
	result, err := db.Exec("DELETE FROM users")
	if err != nil {
		return fmt.Errorf("ошибка очистки базы данных: %v", err)
	}

	affected, _ := result.RowsAffected()
	log.Printf("Удалено пользователей: %d", affected)
	return nil
}

// ClearAllData удаляет все данные из всех таблиц
func ClearAllData() error {
	tables := []string{"ip_violations", "ip_connections", "users"}

	totalDeleted := 0
	for _, tableName := range tables {
		query := fmt.Sprintf("DELETE FROM %s", tableName)
		result, err := db.Exec(query)
		if err != nil {
			log.Printf("Ошибка очистки таблицы %s: %v", tableName, err)
			continue
		}
		affected, _ := result.RowsAffected()
		log.Printf("Очищена таблица %s: удалено %d записей", tableName, affected)
		totalDeleted += int(affected)
	}

	// Восстанавливаем конфигурацию по умолчанию
	query := `
		INSERT INTO traffic_configs (id, enabled, daily_limit_gb, weekly_limit_gb, monthly_limit_gb, limit_gb, reset_days)
		VALUES ('default', true, 0, 0, 0, 0, 30)
		ON CONFLICT (id) DO NOTHING`
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Ошибка восстановления конфигурации по умолчанию: %v", err)
	}

	log.Printf("Всего удалено записей: %d", totalDeleted)
	return nil
}

// Вспомогательные функции

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func readUserInput(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func main() {
	// Загружаем переменные из .env, если файл присутствует (в корне или в tools/)
	if err := godotenv.Load(".env", "tools/.env"); err != nil {
		log.Println(".env не найден, используем переменные окружения и значения по умолчанию")
	}

	fmt.Println("=== PostgreSQL VPN Bot Cleanup Tool ===")
	fmt.Println("Подключение к PostgreSQL...")

	if err := InitPostgreSQL(); err != nil {
		log.Fatalf("Ошибка подключения к PostgreSQL: %v", err)
	}
	defer DisconnectPostgreSQL()

	for {
		fmt.Println("\nВыберите действие:")
		fmt.Println("1. Показать всех пользователей")
		fmt.Println("2. Сбросить флаги пробных периодов у всех пользователей")
		fmt.Println("3. Сбросить флаг пробного периода у конкретного пользователя")
		fmt.Println("4. Удалить конкретного пользователя")
		fmt.Println("5. Удалить всех пользователей")
		fmt.Println("6. Очистить всю базу данных")
		fmt.Println("0. Выход")

		choice := readUserInput("Ваш выбор: ")

		switch choice {
		case "1":
			if err := ShowUsers(); err != nil {
				fmt.Printf("Ошибка: %v\n", err)
			}

		case "2":
			confirm := readUserInput("Вы уверены, что хотите сбросить флаги пробных периодов у ВСЕХ пользователей? (yes/no): ")
			if strings.ToLower(confirm) == "yes" {
				if err := ResetAllTrialFlags(); err != nil {
					fmt.Printf("Ошибка: %v\n", err)
				} else {
					fmt.Println("✅ Флаги пробных периодов сброшены у всех пользователей")
				}
			} else {
				fmt.Println("Операция отменена")
			}

		case "3":
			telegramIDStr := readUserInput("Введите Telegram ID пользователя: ")
			telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
			if err != nil {
				fmt.Printf("Ошибка: некорректный Telegram ID: %v\n", err)
				continue
			}

			if err := ResetUserTrialFlag(telegramID); err != nil {
				fmt.Printf("Ошибка: %v\n", err)
			} else {
				fmt.Printf("✅ Флаг пробного периода сброшен для пользователя %d\n", telegramID)
			}

		case "4":
			telegramIDStr := readUserInput("Введите Telegram ID пользователя для удаления: ")
			telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
			if err != nil {
				fmt.Printf("Ошибка: некорректный Telegram ID: %v\n", err)
				continue
			}

			confirm := readUserInput(fmt.Sprintf("Вы уверены, что хотите удалить пользователя %d? (yes/no): ", telegramID))
			if strings.ToLower(confirm) == "yes" {
				if err := DeleteUser(telegramID); err != nil {
					fmt.Printf("Ошибка: %v\n", err)
				} else {
					fmt.Printf("✅ Пользователь %d удален\n", telegramID)
				}
			} else {
				fmt.Println("Операция отменена")
			}

		case "5":
			confirm := readUserInput("Вы уверены, что хотите удалить ВСЕХ пользователей? (yes/no): ")
			if strings.ToLower(confirm) == "yes" {
				if err := ClearAllUsers(); err != nil {
					fmt.Printf("Ошибка: %v\n", err)
				} else {
					fmt.Println("✅ Все пользователи удалены")
				}
			} else {
				fmt.Println("Операция отменена")
			}

		case "6":
			confirm := readUserInput("Вы уверены, что хотите очистить ВСЮ базу данных? (yes/no): ")
			if strings.ToLower(confirm) == "yes" {
				if err := ClearAllData(); err != nil {
					fmt.Printf("Ошибка: %v\n", err)
				} else {
					fmt.Println("✅ База данных очищена")
				}
			} else {
				fmt.Println("Операция отменена")
			}

		case "0":
			fmt.Println("До свидания!")
			return

		default:
			fmt.Println("Неверный выбор. Попробуйте снова.")
		}
	}
}
