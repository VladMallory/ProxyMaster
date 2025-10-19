package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"bot/common"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Константы для PostgreSQL
const (
	PG_HOST = "localhost"
	PG_PORT = 5432

	PG_USER     = "vpn_bot_user"
	PG_PASSWORD = "your_secure_password"
	PG_DBNAME   = "vpn_bot"
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

// HTTP клиент для работы с панелью 3x-ui
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

// Структуры для работы с панелью 3x-ui
type Inbound struct {
	ID       int    `json:"id"`
	Settings string `json:"settings"`
}

type Client struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	SubID string `json:"subId"`
}

type Settings struct {
	Clients []Client `json:"clients"`
}

// InitPostgreSQL инициализирует соединение с PostgreSQL
func InitPostgreSQL() error {
	// Получаем настройки из переменных окружения, если они есть
	host := getEnvOrDefault("PG_HOST", "localhost")
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

// ClearAllData удаляет все данные из всех таблиц
func ClearAllData() error {
	log.Printf("CLEAR_ALL_DATA: Начало полной очистки базы данных")

	// Список всех таблиц в правильном порядке (сначала зависимые, потом основные)
	// Порядок важен для соблюдения внешних ключей
	tables := []string{
		"referral_bonuses",     // Зависит от users
		"referral_transitions", // Зависит от users
		"promo_usage",          // Зависит от promo_codes и users
		"promo_codes",          // Зависит от users (created_by)
		"ip_violations",        // Зависит от users
		"ip_connections",       // Зависит от users
		"users",                // Основная таблица
		"traffic_configs",      // Настройки трафика (очищаем, но потом восстанавливаем)
	}

	totalDeleted := 0
	clearedTables := 0

	for _, tableName := range tables {
		query := fmt.Sprintf("DELETE FROM %s", tableName)
		result, err := db.Exec(query)
		if err != nil {
			log.Printf("CLEAR_ALL_DATA: Ошибка очистки таблицы %s: %v", tableName, err)
			continue
		}
		affected, _ := result.RowsAffected()
		log.Printf("CLEAR_ALL_DATA: Очищена таблица %s: удалено %d записей", tableName, affected)
		totalDeleted += int(affected)
		clearedTables++
	}

	// Восстанавливаем конфигурацию по умолчанию
	log.Printf("CLEAR_ALL_DATA: Восстанавливаем конфигурацию по умолчанию")
	query := `
		INSERT INTO traffic_configs (id, enabled, daily_limit_gb, weekly_limit_gb, monthly_limit_gb, limit_gb, reset_days)
		VALUES ('default', true, 0, 0, 0, 0, 30)
		ON CONFLICT (id) DO NOTHING`
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("CLEAR_ALL_DATA: Ошибка восстановления конфигурации по умолчанию: %v", err)
	} else {
		log.Printf("CLEAR_ALL_DATA: Конфигурация по умолчанию восстановлена")
	}

	log.Printf("CLEAR_ALL_DATA: ✅ Полная очистка завершена")
	log.Printf("CLEAR_ALL_DATA: Очищено таблиц: %d, удалено записей: %d", clearedTables, totalDeleted)
	return nil
}

// ClearAllDataWithPanel удаляет все данные из базы данных И очищает панель 3x-ui
func ClearAllDataWithPanel() error {
	log.Printf("CLEAR_ALL_DATA_WITH_PANEL: Начало полной очистки базы данных и панели 3x-ui")

	// Сначала очищаем базу данных (теперь очищает ВСЕ таблицы)
	err := ClearAllData()
	if err != nil {
		log.Printf("CLEAR_ALL_DATA_WITH_PANEL: Ошибка очистки базы данных: %v", err)
		return fmt.Errorf("ошибка очистки базы данных: %v", err)
	}

	log.Printf("CLEAR_ALL_DATA_WITH_PANEL: База данных полностью очищена, теперь очищаем панель 3x-ui")

	// Теперь очищаем панель 3x-ui
	err = clearPanelClients()
	if err != nil {
		log.Printf("CLEAR_ALL_DATA_WITH_PANEL: Ошибка очистки панели 3x-ui: %v", err)
		// Не возвращаем ошибку, так как база уже очищена
		log.Printf("CLEAR_ALL_DATA_WITH_PANEL: Продолжаем, несмотря на ошибку очистки панели")
	} else {
		log.Printf("CLEAR_ALL_DATA_WITH_PANEL: Панель 3x-ui успешно очищена")
	}

	log.Printf("CLEAR_ALL_DATA_WITH_PANEL: ✅ Полная очистка базы данных и панели завершена")
	return nil
}

// clearPanelClients очищает всех клиентов из панели 3x-ui
func clearPanelClients() error {
	log.Printf("CLEAR_PANEL_CLIENTS: Начало очистки панели 3x-ui")

	// Авторизуемся в панели
	sessionCookie, err := loginToPanel(common.PANEL_URL, common.PANEL_USER, common.PANEL_PASS)
	if err != nil {
		return fmt.Errorf("ошибка авторизации в панели: %v", err)
	}

	// Получаем список всех inbound'ов
	inbounds, err := getAllInboundsFromPanel(sessionCookie, common.PANEL_URL)
	if err != nil {
		return fmt.Errorf("ошибка получения списка inbound'ов: %v", err)
	}

	// Очищаем клиентов из каждого inbound'а
	for _, inbound := range inbounds {
		err = clearInboundClients(sessionCookie, common.PANEL_URL, inbound.ID)
		if err != nil {
			log.Printf("CLEAR_PANEL_CLIENTS: Ошибка очистки inbound %d: %v", inbound.ID, err)
			continue
		}
		log.Printf("CLEAR_PANEL_CLIENTS: Inbound %d очищен", inbound.ID)
	}

	log.Printf("CLEAR_PANEL_CLIENTS: ✅ Панель 3x-ui очищена")
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

// loginToPanel авторизуется в панели 3x-ui
func loginToPanel(panelURL, username, password string) (string, error) {
	loginData := map[string]string{
		"username": username,
		"password": password,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации данных авторизации: %v", err)
	}

	req, err := http.NewRequest("POST", panelURL+"login", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса авторизации: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения запроса авторизации: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа авторизации: %v", err)
	}

	var loginResp struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}

	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", fmt.Errorf("ошибка парсинга ответа авторизации: %v", err)
	}

	if !loginResp.Success {
		return "", fmt.Errorf("неудачная авторизация: %s", loginResp.Msg)
	}

	// Извлекаем cookie из заголовков ответа
	var sessionCookie string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "3x-ui" {
			sessionCookie = cookie.String()
			break
		}
	}

	if sessionCookie == "" {
		return "", fmt.Errorf("не найдена cookie сессии")
	}

	return sessionCookie, nil
}

// getAllInboundsFromPanel получает список всех inbound'ов
func getAllInboundsFromPanel(sessionCookie, panelURL string) ([]Inbound, error) {
	req, err := http.NewRequest("GET", panelURL+"inbound/list", nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Cookie", sessionCookie)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	var response struct {
		Success bool      `json:"success"`
		Msg     string    `json:"msg"`
		Obj     []Inbound `json:"obj"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("неудачное получение списка inbound'ов: %s", response.Msg)
	}

	return response.Obj, nil
}

// clearInboundClients очищает клиентов из конкретного inbound'а
func clearInboundClients(sessionCookie, panelURL string, inboundID int) error {
	// Получаем inbound
	req, err := http.NewRequest("GET", fmt.Sprintf("%sinbound/get/%d", panelURL, inboundID), nil)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса получения inbound: %v", err)
	}

	req.Header.Set("Cookie", sessionCookie)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса получения inbound: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа получения inbound: %v", err)
	}

	var inboundResp struct {
		Success bool    `json:"success"`
		Msg     string  `json:"msg"`
		Obj     Inbound `json:"obj"`
	}

	if err := json.Unmarshal(body, &inboundResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа получения inbound: %v", err)
	}

	if !inboundResp.Success {
		return fmt.Errorf("неудачное получение inbound: %s", inboundResp.Msg)
	}

	// Парсим настройки
	var settings Settings
	if err := json.Unmarshal([]byte(inboundResp.Obj.Settings), &settings); err != nil {
		return fmt.Errorf("ошибка парсинга настроек: %v", err)
	}

	// Очищаем массив клиентов
	settings.Clients = []Client{}

	// Сериализуем обновленные настройки
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("ошибка сериализации настроек: %v", err)
	}
	inboundResp.Obj.Settings = string(settingsJSON)

	// Обновляем inbound
	updateData, err := json.Marshal(inboundResp.Obj)
	if err != nil {
		return fmt.Errorf("ошибка сериализации inbound: %v", err)
	}

	updateReq, err := http.NewRequest("POST", fmt.Sprintf("%sinbound/update/%d", panelURL, inboundID), bytes.NewBuffer(updateData))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса обновления inbound: %v", err)
	}

	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Cookie", sessionCookie)

	updateResp, err := httpClient.Do(updateReq)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса обновления inbound: %v", err)
	}
	defer updateResp.Body.Close()

	updateBody, err := io.ReadAll(updateResp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа обновления inbound: %v", err)
	}

	var updateResponse struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}

	if err := json.Unmarshal(updateBody, &updateResponse); err != nil {
		return fmt.Errorf("ошибка парсинга ответа обновления inbound: %v", err)
	}

	if !updateResponse.Success {
		return fmt.Errorf("обновление inbound не удалось: %s", updateResponse.Msg)
	}

	return nil
}

func main() {
	// Загружаем переменные из .env, если файл присутствует
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Файл .env не найден, используем переменные окружения и значения по умолчанию")
	} else {
		log.Println("Настройки загружены из .env файла")
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
		fmt.Println("6. Очистить всю базу данных (ВСЕ таблицы)")
		fmt.Println("7. Очистить базу данных И панель 3x-ui (ВСЕ таблицы)")
		fmt.Println("0. Выход")

		choice := readUserInput("Ваш выбор: ")

		switch choice {
		case "1":
			showAllUsers()

		case "2":
			fmt.Println("Функция сброса флагов не реализована в упрощенной версии")

		case "3":
			fmt.Println("Функция сброса флага не реализована в упрощенной версии")

		case "4":
			deleteUserByID()

		case "5":
			fmt.Println("Функция удаления всех пользователей не реализована в упрощенной версии")

		case "6":
			confirm := readUserInput("Вы уверены, что хотите очистить ВСЮ базу данных (ВСЕ таблицы: users, promo_codes, promo_usage, referral_bonuses, referral_transitions, ip_violations, ip_connections, traffic_configs)? (yes/no): ")
			if strings.ToLower(confirm) == "yes" {
				if err := ClearAllData(); err != nil {
					fmt.Printf("Ошибка: %v\n", err)
				} else {
					fmt.Println("✅ Вся база данных очищена (все таблицы)")
				}
			} else {
				fmt.Println("Операция отменена")
			}

		case "7":
			confirm := readUserInput("Вы уверены, что хотите очистить ВСЮ базу данных (ВСЕ таблицы) И панель 3x-ui? (yes/no): ")
			if strings.ToLower(confirm) == "yes" {
				if err := ClearAllDataWithPanel(); err != nil {
					fmt.Printf("Ошибка: %v\n", err)
				} else {
					fmt.Println("✅ Вся база данных и панель 3x-ui очищены")
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

// showAllUsers показывает всех пользователей из базы данных
func showAllUsers() {
	fmt.Println("\n=== ПОЛЬЗОВАТЕЛИ ===")

	query := `
		SELECT id, telegram_id, username, first_name, last_name, balance, 
		       total_paid, created_at, updated_at, has_active_config, 
		       client_id, email, sub_id, config_created_at, expiry_time, 
		       configs_count, has_used_trial
		FROM users 
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		fmt.Printf("Ошибка получения пользователей: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Printf("%-4s %-12s %-15s %-20s %-15s %-8s %-10s %-12s %-12s %-12s %-12s\n",
		"ID", "TelegramID", "Username", "First Name", "Last Name", "Balance",
		"TotalPaid", "HasConfig", "ClientID", "SubID", "CreatedAt")
	fmt.Println(strings.Repeat("-", 150))

	count := 0
	for rows.Next() {
		var user User
		var createdAt, updatedAt time.Time
		var configCreatedAt sql.NullTime
		var clientID, email, subID sql.NullString
		var expiryTime sql.NullInt64

		err := rows.Scan(
			&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
			&user.Balance, &user.TotalPaid, &createdAt, &updatedAt, &user.HasActiveConfig,
			&clientID, &email, &subID, &configCreatedAt, &expiryTime,
			&user.ConfigsCount, &user.HasUsedTrial,
		)
		if err != nil {
			fmt.Printf("Ошибка сканирования пользователя: %v\n", err)
			continue
		}

		user.CreatedAt = createdAt
		user.UpdatedAt = updatedAt
		if configCreatedAt.Valid {
			user.ConfigCreatedAt = configCreatedAt.Time
		}

		if clientID.Valid {
			user.ClientID = clientID.String
		}
		if email.Valid {
			user.Email = email.String
		}
		if subID.Valid {
			user.SubID = subID.String
		}
		if expiryTime.Valid {
			user.ExpiryTime = expiryTime.Int64
		}

		hasConfig := "❌"
		if user.HasActiveConfig {
			hasConfig = "✅"
		}

		clientIDStr := user.ClientID
		if len(clientIDStr) > 8 {
			clientIDStr = clientIDStr[:8] + "..."
		}

		subIDStr := user.SubID
		if len(subIDStr) > 8 {
			subIDStr = subIDStr[:8] + "..."
		}

		fmt.Printf("%-4d %-12d %-15s %-20s %-15s %-8.2f %-10.2f %-12s %-12s %-12s %-12s\n",
			user.ID, user.TelegramID, user.Username, user.FirstName, user.LastName,
			user.Balance, user.TotalPaid, hasConfig, clientIDStr, subIDStr,
			user.CreatedAt.Format("2006-01-02"))

		count++
	}

	if err = rows.Err(); err != nil {
		fmt.Printf("Ошибка итерации по пользователям: %v\n", err)
		return
	}

	fmt.Printf("\nВсего пользователей: %d\n", count)
}

// deleteUserByID удаляет пользователя по ID
func deleteUserByID() {
	// Сначала показываем всех пользователей
	showAllUsers()

	fmt.Print("\nВведите ID пользователя для удаления: ")
	var userID int64
	_, err := fmt.Scanf("%d", &userID)
	if err != nil {
		fmt.Printf("Ошибка ввода ID: %v\n", err)
		return
	}

	// Проверяем, существует ли пользователь
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)"
	err = db.QueryRow(query, userID).Scan(&exists)
	if err != nil {
		fmt.Printf("Ошибка проверки существования пользователя: %v\n", err)
		return
	}

	if !exists {
		fmt.Printf("Пользователь с ID %d не найден\n", userID)
		return
	}

	// Получаем информацию о пользователе
	var user User
	query = `
		SELECT id, telegram_id, username, first_name, last_name, balance, 
		       has_active_config, client_id, email, sub_id
		FROM users WHERE id = $1
	`

	var clientID, email, subID sql.NullString
	err = db.QueryRow(query, userID).Scan(
		&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
		&user.Balance, &user.HasActiveConfig, &clientID, &email, &subID,
	)
	if err != nil {
		fmt.Printf("Ошибка получения информации о пользователе: %v\n", err)
		return
	}

	if clientID.Valid {
		user.ClientID = clientID.String
	}
	if email.Valid {
		user.Email = email.String
	}
	if subID.Valid {
		user.SubID = subID.String
	}

	// Показываем информацию о пользователе
	fmt.Printf("\n=== ИНФОРМАЦИЯ О ПОЛЬЗОВАТЕЛЕ ===\n")
	fmt.Printf("ID: %d\n", user.ID)
	fmt.Printf("Telegram ID: %d\n", user.TelegramID)
	fmt.Printf("Username: %s\n", user.Username)
	fmt.Printf("First Name: %s\n", user.FirstName)
	fmt.Printf("Last Name: %s\n", user.LastName)
	fmt.Printf("Balance: %.2f\n", user.Balance)
	fmt.Printf("Has Active Config: %t\n", user.HasActiveConfig)
	fmt.Printf("Client ID: %s\n", user.ClientID)
	fmt.Printf("Email: %s\n", user.Email)
	fmt.Printf("Sub ID: %s\n", user.SubID)

	// Спрашиваем подтверждение
	confirm := readUserInput(fmt.Sprintf("\nВы уверены, что хотите удалить пользователя %s (ID: %d)? (yes/no): ", user.Username, userID))
	if strings.ToLower(confirm) != "yes" {
		fmt.Println("Удаление отменено")
		return
	}

	// Удаляем пользователя
	query = "DELETE FROM users WHERE id = $1"
	result, err := db.Exec(query, userID)
	if err != nil {
		fmt.Printf("Ошибка удаления пользователя: %v\n", err)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Ошибка получения количества удаленных строк: %v\n", err)
		return
	}

	if rowsAffected > 0 {
		fmt.Printf("✅ Пользователь %s (ID: %d) успешно удален из базы данных\n", user.Username, userID)
	} else {
		fmt.Printf("❌ Пользователь с ID %d не был удален\n", userID)
	}
}
