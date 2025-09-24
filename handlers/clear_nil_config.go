package handlers

import (
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

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	_ "github.com/mattn/go-sqlite3"
)

// Путь к SQLite базе данных панели
const (
	PANEL_DB_PATH = "/etc/x-ui/x-ui.db"
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

// Структура для записи в client_traffics
type ClientTraffic struct {
	ID         int    `db:"id"`
	InboundID  int    `db:"inbound_id"`
	Enable     bool   `db:"enable"`
	Email      string `db:"email"`
	Up         int64  `db:"up"`
	Down       int64  `db:"down"`
	AllTime    int64  `db:"all_time"`
	ExpiryTime int64  `db:"expiry_time"`
	Total      int64  `db:"total"`
	Reset      int    `db:"reset"`
	LastOnline int64  `db:"last_online"`
}

// clearOrphanedRecords выполняет очистку сиротских записей из базы данных панели
func clearOrphanedRecords(bot *tgbotapi.BotAPI, chatID int64) (int, error) {
	log.Printf("CLEAR_ORPHANED_RECORDS: Начало очистки сиротских записей")

	// Подключаемся к SQLite базе данных панели
	panelDB, err := connectToPanelDB()
	if err != nil {
		sendMessage(bot, chatID, fmt.Sprintf("❌ Ошибка подключения к БД панели: %v", err))
		return 0, fmt.Errorf("ошибка подключения к БД панели: %v", err)
	}
	defer panelDB.Close()

	// Получаем данные из панели через API
	sessionCookie, err := loginToPanel()
	if err != nil {
		sendMessage(bot, chatID, fmt.Sprintf("❌ Ошибка авторизации в панели: %v", err))
		return 0, fmt.Errorf("ошибка авторизации в панели: %v", err)
	}

	inbound, err := getInboundFromPanel(sessionCookie)
	if err != nil {
		sendMessage(bot, chatID, fmt.Sprintf("❌ Ошибка получения inbound: %v", err))
		return 0, fmt.Errorf("ошибка получения inbound: %v", err)
	}

	// Парсим настройки
	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		sendMessage(bot, chatID, fmt.Sprintf("❌ Ошибка парсинга settings: %v", err))
		return 0, fmt.Errorf("ошибка парсинга settings: %v", err)
	}

	sendMessage(bot, chatID, fmt.Sprintf("📊 Найдено клиентов в панели через API: %d", len(settings.Clients)))

	// Получаем все записи из client_traffics
	clientTraffics, err := getClientTraffics(panelDB)
	if err != nil {
		sendMessage(bot, chatID, fmt.Sprintf("❌ Ошибка получения client_traffics: %v", err))
		return 0, fmt.Errorf("ошибка получения client_traffics: %v", err)
	}

	sendMessage(bot, chatID, fmt.Sprintf("📊 Найдено записей в client_traffics: %d", len(clientTraffics)))

	// Находим записи, которые есть в client_traffics, но нет в API
	orphanedRecords := findOrphanedRecords(clientTraffics, settings.Clients)
	sendMessage(bot, chatID, fmt.Sprintf("👻 Найдено сиротских записей: %d", len(orphanedRecords)))

	if len(orphanedRecords) == 0 {
		sendMessage(bot, chatID, "✅ Сиротских записей не найдено!")
		return 0, nil
	}

	// Показываем сиротские записи
	for _, record := range orphanedRecords {
		sendMessage(bot, chatID, fmt.Sprintf("👻 Сиротская запись: ID=%d, Email=%s, Up=%d, Down=%d, Total=%d, ExpiryTime=%d",
			record.ID, record.Email, record.Up, record.Down, record.Total, record.ExpiryTime))
	}

	// Удаляем сиротские записи
	if err := removeOrphanedRecords(panelDB, orphanedRecords); err != nil {
		sendMessage(bot, chatID, fmt.Sprintf("❌ Ошибка удаления сиротских записей: %v", err))
		return 0, fmt.Errorf("ошибка удаления сиротских записей: %v", err)
	}

	log.Printf("CLEAR_ORPHANED_RECORDS: Успешно удалено %d сиротских записей", len(orphanedRecords))
	return len(orphanedRecords), nil
}

// connectToPanelDB подключается к SQLite базе данных панели
func connectToPanelDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", PANEL_DB_PATH)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
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

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("неверный статус код: %d, ответ: %s", resp.StatusCode, string(body))
	}

	// Извлекаем куку из заголовков
	cookies := resp.Header.Get("Set-Cookie")
	if cookies == "" {
		return "", fmt.Errorf("кука не найдена в ответе")
	}

	// Ищем куку 3x-ui
	parts := strings.Split(cookies, ";")
	for _, part := range parts {
		if strings.HasPrefix(strings.TrimSpace(part), "3x-ui=") {
			cookie := strings.TrimSpace(part)
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

// getClientTraffics получает все записи из client_traffics
func getClientTraffics(db *sql.DB) ([]ClientTraffic, error) {
	query := `SELECT id, inbound_id, enable, email, up, down, all_time, expiry_time, total, reset, last_online FROM client_traffics ORDER BY id`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ClientTraffic
	for rows.Next() {
		var record ClientTraffic
		err := rows.Scan(
			&record.ID,
			&record.InboundID,
			&record.Enable,
			&record.Email,
			&record.Up,
			&record.Down,
			&record.AllTime,
			&record.ExpiryTime,
			&record.Total,
			&record.Reset,
			&record.LastOnline,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

// findOrphanedRecords находит записи, которые есть в client_traffics, но нет в API
func findOrphanedRecords(traffics []ClientTraffic, clients []Client) []ClientTraffic {
	// Создаем карту клиентов из API для быстрого поиска
	// Учитываем разные форматы email'ов
	clientMap := make(map[string]bool)
	for _, client := range clients {
		clientMap[client.Email] = true

		// Если email в сложном формате (tg_telegramID_...), добавляем и простой Telegram ID
		if strings.HasPrefix(client.Email, "tg_") {
			parts := strings.Split(client.Email, "_")
			if len(parts) >= 2 {
				telegramID := parts[1]
				clientMap[telegramID] = true
			}
		}
	}

	var orphaned []ClientTraffic
	for _, traffic := range traffics {
		// Проверяем как точное совпадение, так и извлеченный Telegram ID
		isFound := clientMap[traffic.Email]

		// Если не найдено и email в сложном формате, проверяем извлеченный Telegram ID
		if !isFound && strings.HasPrefix(traffic.Email, "tg_") {
			parts := strings.Split(traffic.Email, "_")
			if len(parts) >= 2 {
				telegramID := parts[1]
				isFound = clientMap[telegramID]
			}
		}

		// Если не найдено и это простой Telegram ID, проверяем сложные форматы
		if !isFound && !strings.HasPrefix(traffic.Email, "tg_") {
			// Проверяем, есть ли клиент с таким Telegram ID в сложном формате
			for _, client := range clients {
				if strings.HasPrefix(client.Email, "tg_") {
					parts := strings.Split(client.Email, "_")
					if len(parts) >= 2 && parts[1] == traffic.Email {
						isFound = true
						break
					}
				}
			}
		}

		if !isFound {
			orphaned = append(orphaned, traffic)
		}
	}

	return orphaned
}

// removeOrphanedRecords удаляет сиротские записи из client_traffics
func removeOrphanedRecords(db *sql.DB, records []ClientTraffic) error {
	if len(records) == 0 {
		return nil
	}

	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %v", err)
	}
	defer tx.Rollback()

	// Удаляем каждую сиротскую запись
	for _, record := range records {
		_, err = tx.Exec(`DELETE FROM client_traffics WHERE id = ?`, record.ID)
		if err != nil {
			return fmt.Errorf("ошибка удаления записи %d: %v", record.ID, err)
		}

		log.Printf("🗑️  Удалена сиротская запись: ID=%d, Email=%s", record.ID, record.Email)
	}

	// Подтверждаем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка подтверждения транзакции: %v", err)
	}

	return nil
}

// sendMessage отправляет сообщение в телеграм чат
func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("CLEAR_ORPHANED_RECORDS: Ошибка отправки сообщения: %v", err)
	}
}
