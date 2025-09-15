package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Константы для подключения к панели 3x-ui (из config.go)
const (
	PANEL_URL  = "https://st.xn--80aag4arlz.xn--p1ai:43690/EaM116GU4tWl0jrKPC/"
	PANEL_USER = "FjuyaaiVMbLwkUL7n8KhxzNMrJ4HMWhBdRj6"
	PANEL_PASS = "AK9UxfugFtFr43DNkcPtYteQ8pYirFGJQ4FG"
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
	ID    string `json:"id"`
	Email string `json:"email"`
	SubID string `json:"subId"`
}

type Settings struct {
	Clients []Client `json:"clients"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

type InboundListResponse struct {
	Success bool      `json:"success"`
	Msg     string    `json:"msg"`
	Obj     []Inbound `json:"obj"`
}

func main() {
	fmt.Println("=== 3x-ui Panel Test (ТЕСТОВЫЙ РЕЖИМ) ===")
	fmt.Println("Эта программа ТОЛЬКО показывает информацию о панели 3x-ui")
	fmt.Println("НИЧЕГО НЕ УДАЛЯЕТ!")
	fmt.Println()

	// Проверяем подключение к панели
	fmt.Println("🔍 Проверяем подключение к панели...")
	sessionCookie, err := loginToPanel()
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к панели: %v", err)
	}
	fmt.Println("✅ Успешно подключились к панели")

	// Получаем список всех inbound'ов
	fmt.Println("📋 Получаем список inbound'ов...")
	inbounds, err := getAllInbounds(sessionCookie)
	if err != nil {
		log.Fatalf("❌ Ошибка получения списка inbound'ов: %v", err)
	}
	fmt.Printf("✅ Найдено %d inbound'ов\n", len(inbounds))

	if len(inbounds) == 0 {
		fmt.Println("ℹ️  Inbound'ов не найдено")
		return
	}

	// Показываем подробную информацию о найденных inbound'ах
	fmt.Println("\n📊 Подробная информация о найденных inbound'ах:")
	totalClients := 0

	for i, inbound := range inbounds {
		clients := getClientsFromSettings(inbound.Settings)
		clientCount := len(clients)
		totalClients += clientCount

		fmt.Printf("\n  %d. Inbound ID: %d\n", i+1, inbound.ID)
		fmt.Printf("     Remark: %s\n", inbound.Remark)
		fmt.Printf("     Клиентов: %d\n", clientCount)

		if clientCount > 0 {
			fmt.Printf("     Клиенты:\n")
			for j, client := range clients {
				fmt.Printf("       %d. ID: %s, Email: %s, SubID: %s\n",
					j+1, client.ID, client.Email, client.SubID)
			}
		}
	}

	fmt.Printf("\n📈 Итого: %d inbound'ов, %d клиентов\n", len(inbounds), totalClients)
	fmt.Println("\n✅ Тест завершен успешно!")
	fmt.Println("💡 Для реальной очистки используйте: ./clean_panel")
}

// loginToPanel авторизуется в панели 3x-ui
func loginToPanel() (string, error) {
	loginData := map[string]string{
		"username": PANEL_USER,
		"password": PANEL_PASS,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации данных авторизации: %v", err)
	}

	req, err := http.NewRequest("POST", PANEL_URL+"login", bytes.NewBuffer(jsonData))
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

	var loginResp LoginResponse
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

// getAllInbounds получает список всех inbound'ов
func getAllInbounds(sessionCookie string) ([]Inbound, error) {
	req, err := http.NewRequest("GET", PANEL_URL+"inbound/list", nil)
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

	var response InboundListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("неудачное получение списка inbound'ов: %s", response.Msg)
	}

	return response.Obj, nil
}

// getClientsFromSettings извлекает клиентов из настроек
func getClientsFromSettings(settingsJSON string) []Client {
	var settings Settings
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		return []Client{}
	}
	return settings.Clients
}
