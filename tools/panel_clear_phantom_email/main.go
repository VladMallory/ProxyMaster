package main

import (
	"bot/common"
	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// PostgreSQL настройки
const (
	PG_HOST     = "localhost"
	PG_PORT     = "5432"
	PG_USER     = "vpn_bot_user"
	PG_PASSWORD = "your_secure_password"
	PG_DBNAME   = "vpn_bot"
)

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
	Remark   string `json:"remark"`
}

type Client struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	SubID      string `json:"subId"`
	Enable     bool   `json:"enable"`
	ExpiryTime int64  `json:"expiryTime"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type Settings struct {
	Clients []Client `json:"clients"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

type InboundResponse struct {
	Success bool    `json:"success"`
	Msg     string  `json:"msg"`
	Obj     Inbound `json:"obj"`
}

type UpdateResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

// Пользователь из базы данных
type User struct {
	TelegramID      int64   `json:"telegram_id"`
	Username        string  `json:"username"`
	FirstName       string  `json:"first_name"`
	HasActiveConfig bool    `json:"has_active_config"`
	ClientID        string  `json:"client_id"`
	SubID           string  `json:"sub_id"`
	Balance         float64 `json:"balance"`
}

func main() {
	log.Println("🧹 Запуск очистки призрачных пользователей из базы данных...")

	// Подключаемся к PostgreSQL
	db, err := connectToPostgreSQL()
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к PostgreSQL: %v", err)
	}
	defer db.Close()

	// Получаем всех пользователей из базы данных
	users, err := getUsersFromDB(db)
	if err != nil {
		log.Fatalf("❌ Ошибка получения пользователей: %v", err)
	}

	log.Printf("📊 Найдено пользователей в базе данных: %d", len(users))

	// Авторизуемся в панели
	sessionCookie, err := loginToPanel()
	if err != nil {
		log.Fatalf("❌ Ошибка авторизации в панели: %v", err)
	}

	// Получаем inbound из панели
	inbound, err := getInboundFromPanel(sessionCookie)
	if err != nil {
		log.Fatalf("❌ Ошибка получения inbound: %v", err)
	}

	// Парсим настройки
	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Fatalf("❌ Ошибка парсинга settings: %v", err)
	}

	log.Printf("📊 Найдено клиентов в панели: %d", len(settings.Clients))

	// Находим призрачные записи (пользователи из БД, которых нет в панели)
	ghostUsers := findGhostClients(users, settings.Clients)
	log.Printf("👻 Найдено призрачных пользователей: %d", len(ghostUsers))

	if len(ghostUsers) == 0 {
		log.Println("✅ Призрачных пользователей не найдено!")
		return
	}

	// Показываем призрачные записи
	for _, user := range ghostUsers {
		log.Printf("👻 Призрачный пользователь: ID=%d, Username=%s, ClientID=%s, SubID=%s",
			user.TelegramID, user.Username, user.ClientID, user.SubID)
	}

	// Спрашиваем подтверждение
	fmt.Print("\n❓ Удалить призрачных пользователей из базы данных? (y/N): ")
	var input string
	fmt.Scanln(&input)

	if strings.ToLower(input) != "y" && strings.ToLower(input) != "yes" {
		log.Println("❌ Операция отменена")
		return
	}

	// Удаляем призрачных пользователей из базы данных
	if err := removeGhostUsers(db, ghostUsers); err != nil {
		log.Fatalf("❌ Ошибка удаления призрачных пользователей: %v", err)
	}

	log.Printf("✅ Призрачные пользователи успешно удалены из базы данных! (%d записей)", len(ghostUsers))
}

// connectToPostgreSQL подключается к PostgreSQL
func connectToPostgreSQL() (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		PG_HOST, PG_PORT, PG_USER, PG_PASSWORD, PG_DBNAME)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// getUsersFromDB получает всех пользователей из базы данных
func getUsersFromDB(db *sql.DB) ([]User, error) {
	query := `
		SELECT telegram_id, username, first_name, has_active_config, 
		       COALESCE(client_id, '') as client_id, 
		       COALESCE(sub_id, '') as sub_id, 
		       balance 
		FROM users 
		WHERE has_active_config = true 
		AND (client_id IS NOT NULL AND client_id != '') 
		AND (sub_id IS NOT NULL AND sub_id != '')
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.TelegramID,
			&user.Username,
			&user.FirstName,
			&user.HasActiveConfig,
			&user.ClientID,
			&user.SubID,
			&user.Balance,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// loginToPanel авторизуется в панели 3x-ui
func loginToPanel() (string, error) {
	loginData := map[string]string{
		"username": common.PANEL_USER,
		"password": common.PANEL_PASS,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return "", err
	}

	log.Printf("🔐 Попытка авторизации в панели: %s", common.PANEL_URL+"login")
	log.Printf("🔐 Данные авторизации: %s", string(jsonData))

	req, err := http.NewRequest("POST", common.PANEL_URL+"login", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	log.Printf("🔐 Ответ сервера: status=%d, body=%s", resp.StatusCode, string(body))

	// Проверяем статус код
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("неверный статус код: %d, ответ: %s", resp.StatusCode, string(body))
	}

	// Если тело пустое, возможно это редирект - попробуем получить куки из заголовков
	if len(body) == 0 {
		log.Printf("⚠️ Пустой ответ, проверяем куки в заголовках")
		cookies := resp.Header.Get("Set-Cookie")
		log.Printf("🍪 Полученные куки: %s", cookies)

		if cookies != "" {
			// Ищем куку 3x-ui
			parts := strings.Split(cookies, ";")
			for _, part := range parts {
				if strings.HasPrefix(strings.TrimSpace(part), "3x-ui=") {
					cookie := strings.TrimSpace(part)
					log.Printf("✅ Найдена кука 3x-ui: %s", cookie)
					return cookie, nil
				}
			}
		}
		return "", fmt.Errorf("пустой ответ от сервера")
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		log.Printf("❌ Ошибка парсинга JSON: %v", err)
		log.Printf("❌ Тело ответа: %s", string(body))
		return "", fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	if !loginResp.Success {
		return "", fmt.Errorf("ошибка авторизации: %s", loginResp.Msg)
	}

	// Извлекаем куку из заголовков
	cookies := resp.Header.Get("Set-Cookie")
	log.Printf("🍪 Полученные куки: %s", cookies)

	if cookies == "" {
		return "", fmt.Errorf("кука не найдена в ответе")
	}

	// Ищем куку 3x-ui
	parts := strings.Split(cookies, ";")
	for _, part := range parts {
		if strings.HasPrefix(strings.TrimSpace(part), "3x-ui=") {
			cookie := strings.TrimSpace(part)
			log.Printf("✅ Найдена кука 3x-ui: %s", cookie)
			return cookie, nil
		}
	}

	return "", fmt.Errorf("кука 3x-ui не найдена")
}

// getInboundFromPanel получает inbound из панели
func getInboundFromPanel(sessionCookie string) (*Inbound, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%spanel/api/inbounds/get/%d", common.PANEL_URL, common.INBOUND_ID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Cookie", sessionCookie)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var inboundResp InboundResponse
	if err := json.Unmarshal(body, &inboundResp); err != nil {
		return nil, err
	}

	if !inboundResp.Success {
		return nil, fmt.Errorf("ошибка получения inbound: %s", inboundResp.Msg)
	}

	return &inboundResp.Obj, nil
}

// findGhostClients находит призрачные записи (пользователи из БД, которых нет в панели)
func findGhostClients(users []User, clients []Client) []User {
	var ghostUsers []User

	// Создаем карту клиентов из панели для быстрого поиска
	clientMap := make(map[string]bool)
	subIDMap := make(map[string]bool)
	for _, client := range clients {
		clientMap[client.ID] = true
		subIDMap[client.SubID] = true
	}

	// Проверяем каждого пользователя из базы данных
	for _, user := range users {
		// Проверяем, есть ли этот пользователь в панели
		found := false
		if clientMap[user.ClientID] || subIDMap[user.SubID] {
			found = true
		}

		// Если пользователь не найден в панели, это призрачная запись
		if !found {
			ghostUsers = append(ghostUsers, user)
		}
	}

	return ghostUsers
}

// removeGhostUsers удаляет призрачных пользователей из базы данных
func removeGhostUsers(db *sql.DB, ghostUsers []User) error {
	if len(ghostUsers) == 0 {
		return nil
	}

	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %v", err)
	}
	defer tx.Rollback()

	// Удаляем каждого призрачного пользователя
	for _, user := range ghostUsers {
		// Сначала обновляем статус активной конфигурации
		_, err = tx.Exec(`
			UPDATE users 
			SET has_active_config = false, 
			    client_id = NULL, 
			    sub_id = NULL 
			WHERE telegram_id = $1`, user.TelegramID)
		if err != nil {
			return fmt.Errorf("ошибка обновления пользователя %d: %v", user.TelegramID, err)
		}

		log.Printf("🗑️  Удален призрачный пользователь: ID=%d, Username=%s, ClientID=%s, SubID=%s",
			user.TelegramID, user.Username, user.ClientID, user.SubID)
	}

	// Подтверждаем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка подтверждения транзакции: %v", err)
	}

	return nil
}
