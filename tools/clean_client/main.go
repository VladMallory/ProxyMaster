package main

import (
	"bufio"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"balance_client"
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

// Структуры для работы с панелью 3x-ui
type Inbound struct {
	ID       int    `json:"id"`
	Settings string `json:"settings"`
}

type Client struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	SubID      string `json:"subId"`
	Enable     bool   `json:"enable"`
	ExpiryTime int64  `json:"expiryTime"`
	Flow       string `json:"flow"`
	TotalGB    int    `json:"totalGB"`
	Reset      int    `json:"reset"`
	TgID       int64  `json:"tgId"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type Settings struct {
	Clients []Client `json:"clients"`
}

var db *sql.DB

// HTTP клиент для работы с панелью 3x-ui
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
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

// loginToPanel авторизуется в панели 3x-ui
func loginToPanel() (string, error) {
	loginData := map[string]string{
		"username": common.PANEL_USER,
		"password": common.PANEL_PASS,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации данных авторизации: %v", err)
	}

	req, err := http.NewRequest("POST", common.PANEL_URL+"login", strings.NewReader(string(jsonData)))
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

// getInbound получает inbound из панели
func getInbound(sessionCookie string) (*Inbound, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%spanel/api/inbounds/get/%d", common.PANEL_URL, common.INBOUND_ID), nil)
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

	// Логируем ответ для отладки
	log.Printf("GET_INBOUND: Ответ сервера: status=%d, body=%s", resp.StatusCode, string(body))

	var response struct {
		Success bool    `json:"success"`
		Msg     string  `json:"msg"`
		Obj     Inbound `json:"obj"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %v, body: %s", err, string(body))
	}

	if !response.Success {
		return nil, fmt.Errorf("неудачное получение inbound: %s", response.Msg)
	}

	return &response.Obj, nil
}

// updateInbound обновляет inbound в панели
func updateInbound(sessionCookie string, inbound Inbound) error {
	updateData, err := json.Marshal(inbound)
	if err != nil {
		return fmt.Errorf("ошибка сериализации inbound: %v", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%spanel/api/inbounds/update/%d", common.PANEL_URL, common.INBOUND_ID), strings.NewReader(string(updateData)))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса обновления: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", sessionCookie)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса обновления: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа обновления: %v", err)
	}

	var updateResponse struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}

	if err := json.Unmarshal(body, &updateResponse); err != nil {
		return fmt.Errorf("ошибка парсинга ответа обновления: %v", err)
	}

	if !updateResponse.Success {
		return fmt.Errorf("обновление inbound не удалось: %s", updateResponse.Msg)
	}

	return nil
}

// findClientByTelegramID находит клиента по Telegram ID
func findClientByTelegramID(clients []Client, telegramID int64) *Client {
	telegramIDStr := fmt.Sprintf("%d", telegramID)
	for _, client := range clients {
		if client.Email == telegramIDStr {
			return &client
		}
	}
	return nil
}

// showUsers показывает всех пользователей из базы данных, отсортированных по TelegramID
func showUsers() error {
	// SQL запрос для получения пользователей с лимитом 500 записей
	query := `
		SELECT telegram_id, username, first_name, last_name, balance, has_active_config, 
		       client_id, sub_id, email, expiry_time, created_at
		FROM users 
		LIMIT 500`

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer rows.Close()

	// Создаем слайс для хранения пользователей перед сортировкой
	var users []User

	// Собираем всех пользователей из результата запроса
	for rows.Next() {
		var user User
		var createdAt time.Time
		var clientID, subID, email sql.NullString
		var expiryTime sql.NullInt64

		err := rows.Scan(&user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
			&user.Balance, &user.HasActiveConfig, &clientID, &subID,
			&email, &expiryTime, &createdAt)

		if err != nil {
			log.Printf("Ошибка сканирования строки: %v", err)
			continue
		}

		// Обрабатываем NULL значения из базы данных
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

	// Сортируем пользователей по первой цифре TelegramID (1-9)
	sort.Slice(users, func(i, j int) bool {
		// Получаем первую цифру каждого TelegramID
		firstDigitI := getFirstDigit(users[i].TelegramID)
		firstDigitJ := getFirstDigit(users[j].TelegramID)

		// Если первые цифры одинаковые, сортируем по полному ID
		if firstDigitI == firstDigitJ {
			return users[i].TelegramID < users[j].TelegramID
		}

		// Сортируем по первой цифре (1-9)
		return firstDigitI < firstDigitJ
	})

	// Выводим заголовок таблицы
	fmt.Println("\n=== ПОЛЬЗОВАТЕЛИ (отсортированы по первой цифре TelegramID: 1-9) ===")
	fmt.Printf("%-15s %-20s %-15s %-10s %-15s %-20s %-15s\n",
		"TelegramID", "Username", "First Name", "Balance", "HasConfig", "ClientID", "SubID")
	fmt.Println(strings.Repeat("-", 120))

	// Выводим отсортированных пользователей
	for _, user := range users {
		// Определяем статус активной конфигурации
		hasConfig := "❌"
		if user.HasActiveConfig {
			hasConfig = "✅"
		}

		fmt.Printf("%-15d %-20s %-15s %-10.2f %-15s %-20s %-15s\n",
			user.TelegramID, user.Username, user.FirstName, user.Balance,
			hasConfig, user.ClientID, user.SubID)
	}

	return nil
}

// showPanelClients показывает клиентов из панели 3x-ui, отсортированных по TgID (TelegramID)
func showPanelClients() error {
	// Авторизуемся в панели 3x-ui
	sessionCookie, err := loginToPanel()
	if err != nil {
		return fmt.Errorf("ошибка авторизации в панели: %v", err)
	}

	// Получаем данные inbound из панели
	inbound, err := getInbound(sessionCookie)
	if err != nil {
		return fmt.Errorf("ошибка получения inbound: %v", err)
	}

	// Парсим настройки inbound для получения списка клиентов
	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return fmt.Errorf("ошибка парсинга settings: %v", err)
	}

	// Сортируем клиентов по первой цифре TgID (TelegramID) (1-9)
	sort.Slice(settings.Clients, func(i, j int) bool {
		// Получаем первую цифру каждого TgID
		firstDigitI := getFirstDigit(settings.Clients[i].TgID)
		firstDigitJ := getFirstDigit(settings.Clients[j].TgID)

		// Если первые цифры одинаковые, сортируем по полному ID
		if firstDigitI == firstDigitJ {
			return settings.Clients[i].TgID < settings.Clients[j].TgID
		}

		// Сортируем по первой цифре (1-9)
		return firstDigitI < firstDigitJ
	})

	// Выводим заголовок таблицы
	fmt.Println("\n=== КЛИЕНТЫ В ПАНЕЛИ (отсортированы по первой цифре TgID: 1-9) ===")
	fmt.Printf("%-20s %-15s %-10s %-20s %-15s %-15s\n",
		"Email", "Enable", "ExpiryTime", "ClientID", "SubID", "TgID")
	fmt.Println(strings.Repeat("-", 105))

	// Выводим отсортированных клиентов
	for _, client := range settings.Clients {
		// Определяем статус активности клиента
		enable := "❌"
		if client.Enable {
			enable = "✅"
		}

		// Форматируем время истечения подписки
		expiryTime := "Never"
		if client.ExpiryTime > 0 {
			expiryTime = time.UnixMilli(client.ExpiryTime).Format("2006-01-02 15:04")
		}

		fmt.Printf("%-20s %-15s %-20s %-20s %-15s %-15d\n",
			client.Email, enable, expiryTime, client.ID, client.SubID, client.TgID)
	}

	return nil
}

// removeClientFromPanel удаляет клиента из панели 3x-ui по его TelegramID
func removeClientFromPanel(telegramID int64) error {
	// Авторизуемся в панели для получения сессионной cookie
	sessionCookie, err := loginToPanel()
	if err != nil {
		return fmt.Errorf("ошибка авторизации в панели: %v", err)
	}

	// Получаем текущие настройки inbound из панели
	inbound, err := getInbound(sessionCookie)
	if err != nil {
		return fmt.Errorf("ошибка получения inbound: %v", err)
	}

	// Парсим JSON настройки для получения списка клиентов
	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return fmt.Errorf("ошибка парсинга settings: %v", err)
	}

	// Ищем клиента по TelegramID в списке клиентов панели
	clientToRemove := findClientByTelegramID(settings.Clients, telegramID)
	if clientToRemove == nil {
		return fmt.Errorf("клиент с TelegramID %d не найден в панели", telegramID)
	}

	// Создаем новый список клиентов без удаляемого клиента
	var newClients []Client
	for _, client := range settings.Clients {
		if client.Email != clientToRemove.Email {
			newClients = append(newClients, client)
		}
	}
	settings.Clients = newClients

	// Сериализуем обновленные настройки обратно в JSON
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("ошибка сериализации settings: %v", err)
	}
	inbound.Settings = string(settingsJSON)

	// Обновляем inbound в панели с новыми настройками
	if err := updateInbound(sessionCookie, *inbound); err != nil {
		return fmt.Errorf("ошибка обновления inbound: %v", err)
	}

	fmt.Printf("✅ Клиент %d успешно удален из панели\n", telegramID)
	return nil
}

// resetUserConfig сбрасывает конфигурацию пользователя в базе данных, очищая все VPN-связанные поля
func resetUserConfig(telegramID int64) error {
	// SQL запрос для сброса конфигурации пользователя
	query := `
		UPDATE users 
		SET has_active_config = false, 
		    client_id = '', 
		    sub_id = '', 
		    email = '', 
		    expiry_time = 0,
		    config_created_at = NULL
		WHERE telegram_id = $1`

	// Выполняем обновление в базе данных
	result, err := db.Exec(query, telegramID)
	if err != nil {
		return fmt.Errorf("ошибка обновления пользователя: %v", err)
	}

	// Проверяем, сколько строк было затронуто
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества обновленных строк: %v", err)
	}

	// Если ни одна строка не была обновлена, пользователь не найден
	if affected == 0 {
		return fmt.Errorf("пользователь с TelegramID %d не найден", telegramID)
	}

	fmt.Printf("✅ Конфиг пользователя %d сброшен в базе данных\n", telegramID)
	return nil
}

// deleteUserFromDatabase полностью удаляет пользователя из базы данных (необратимая операция)
func deleteUserFromDatabase(telegramID int64) error {
	// Сначала проверяем существование пользователя перед удалением
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE telegram_id = $1)`
	err := db.QueryRow(checkQuery, telegramID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("ошибка проверки существования пользователя: %v", err)
	}

	if !exists {
		return fmt.Errorf("пользователь с TelegramID %d не найден в базе данных", telegramID)
	}

	// Выполняем удаление пользователя из таблицы users
	query := `DELETE FROM users WHERE telegram_id = $1`
	result, err := db.Exec(query, telegramID)
	if err != nil {
		return fmt.Errorf("ошибка удаления пользователя: %v", err)
	}

	// Проверяем количество удаленных строк
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества удаленных строк: %v", err)
	}

	if affected == 0 {
		return fmt.Errorf("пользователь с TelegramID %d не был удален", telegramID)
	}

	fmt.Printf("✅ Пользователь %d полностью удален из базы данных\n", telegramID)
	return nil
}

// Вспомогательные функции

// getEnvOrDefault получает значение переменной окружения или возвращает значение по умолчанию
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// readUserInput читает пользовательский ввод из консоли и возвращает обрезанную строку
func readUserInput(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// getFirstDigit возвращает первую цифру числа для сортировки (1-9, затем 0)
func getFirstDigit(num int64) int {
	if num == 0 {
		return 10 // Помещаем 0 в конец
	}

	// Преобразуем число в строку и берем первый символ
	str := fmt.Sprintf("%d", num)
	firstChar := str[0]

	// Преобразуем обратно в число
	firstDigit := int(firstChar - '0')

	// Если первая цифра 0, помещаем в конец
	if firstDigit == 0 {
		return 10
	}

	return firstDigit
}

// main - главная функция программы, предоставляющая интерактивное меню для управления клиентами
func main() {
	// Загружаем переменные окружения из .env файла, если он существует
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Файл .env не найден, используем переменные окружения и значения по умолчанию")
	} else {
		log.Println("Настройки загружены из .env файла")
	}

	fmt.Println("=== Инструмент выборочного удаления клиентов ===")
	fmt.Println("Подключение к PostgreSQL...")

	// Инициализируем подключение к базе данных PostgreSQL
	if err := InitPostgreSQL(); err != nil {
		log.Fatalf("Ошибка подключения к PostgreSQL: %v", err)
	}
	// Гарантируем закрытие соединения при завершении программы
	defer DisconnectPostgreSQL()

	// Основной цикл программы - интерактивное меню
	for {
		fmt.Println("\nВыберите действие:")
		fmt.Println("1. Показать пользователей из базы данных")
		fmt.Println("2. Показать клиентов из панели 3x-ui")
		fmt.Println("3. Удалить клиента из панели")
		fmt.Println("4. Сбросить конфиг пользователя в базе данных")
		fmt.Println("5. Полная очистка (удалить из панели + сбросить в базе)")
		fmt.Println("6. Удалить пользователя из базы данных телеграм бота")
		fmt.Println("7. Управление балансом клиентов")
		fmt.Println("0. Выход")

		choice := readUserInput("Ваш выбор: ")

		switch choice {
		case "1":
			// Показать пользователей из базы данных (отсортированных по TelegramID)
			if err := showUsers(); err != nil {
				fmt.Printf("Ошибка: %v\n", err)
			}

		case "2":
			// Показать клиентов из панели 3x-ui (отсортированных по TgID)
			if err := showPanelClients(); err != nil {
				fmt.Printf("Ошибка: %v\n", err)
			}

		case "3":
			// Удаление клиента только из панели 3x-ui
			telegramIDStr := readUserInput("Введите TelegramID для удаления из панели: ")
			telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
			if err != nil {
				fmt.Printf("Ошибка парсинга TelegramID: %v\n", err)
				continue
			}

			confirm := readUserInput(fmt.Sprintf("Вы уверены, что хотите удалить клиента %d из панели? (yes/no): ", telegramID))
			if strings.ToLower(confirm) == "yes" {
				if err := removeClientFromPanel(telegramID); err != nil {
					fmt.Printf("Ошибка: %v\n", err)
				}
			} else {
				fmt.Println("Операция отменена")
			}

		case "4":
			// Сброс конфигурации пользователя в базе данных (без удаления из панели)
			telegramIDStr := readUserInput("Введите TelegramID для сброса конфига в базе: ")
			telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
			if err != nil {
				fmt.Printf("Ошибка парсинга TelegramID: %v\n", err)
				continue
			}

			confirm := readUserInput(fmt.Sprintf("Вы уверены, что хотите сбросить конфиг пользователя %d в базе? (yes/no): ", telegramID))
			if strings.ToLower(confirm) == "yes" {
				if err := resetUserConfig(telegramID); err != nil {
					fmt.Printf("Ошибка: %v\n", err)
				}
			} else {
				fmt.Println("Операция отменена")
			}

		case "5":
			// Полная очистка: удаление из панели + сброс в базе данных
			telegramIDStr := readUserInput("Введите TelegramID для полной очистки: ")
			telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
			if err != nil {
				fmt.Printf("Ошибка парсинга TelegramID: %v\n", err)
				continue
			}

			confirm := readUserInput(fmt.Sprintf("Вы уверены, что хотите ПОЛНОСТЬЮ очистить клиента %d? (yes/no): ", telegramID))
			if strings.ToLower(confirm) == "yes" {
				// Сначала удаляем клиента из панели 3x-ui
				if err := removeClientFromPanel(telegramID); err != nil {
					fmt.Printf("Ошибка удаления из панели: %v\n", err)
				}
				// Затем сбрасываем конфигурацию в базе данных
				if err := resetUserConfig(telegramID); err != nil {
					fmt.Printf("Ошибка сброса в базе: %v\n", err)
				} else {
					fmt.Printf("✅ Полная очистка клиента %d завершена\n", telegramID)
				}
			} else {
				fmt.Println("Операция отменена")
			}

		case "6":
			// Полное удаление пользователя из базы данных (ВНИМАНИЕ: необратимая операция)
			telegramIDStr := readUserInput("Введите TelegramID для удаления из базы данных: ")
			telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
			if err != nil {
				fmt.Printf("Ошибка парсинга TelegramID: %v\n", err)
				continue
			}

			confirm := readUserInput(fmt.Sprintf("⚠️ ВНИМАНИЕ! Это ПОЛНОСТЬЮ удалит пользователя %d из базы данных!\nВы уверены? (yes/no): ", telegramID))
			if strings.ToLower(confirm) == "yes" {
				if err := deleteUserFromDatabase(telegramID); err != nil {
					fmt.Printf("Ошибка: %v\n", err)
				}
			} else {
				fmt.Println("Операция отменена")
			}

		case "7":
			// Запуск модуля управления балансом клиентов
			balanceManager := balance_client.NewBalanceManager(db)
			balanceCLI := balance_client.NewBalanceCLI(balanceManager)
			balanceCLI.Run()

		case "0":
			// Завершение работы программы
			fmt.Println("До свидания!")
			return

		default:
			// Обработка неверного выбора
			fmt.Println("Неверный выбор. Попробуйте снова.")
		}
	}
}
