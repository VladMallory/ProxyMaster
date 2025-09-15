package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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

type InboundGetResponse struct {
	Success bool    `json:"success"`
	Msg     string  `json:"msg"`
	Obj     Inbound `json:"obj"`
}

type InboundUpdateResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

func main() {
	fmt.Println("=== 3x-ui Panel Cleaner ===")
	fmt.Println("Эта программа очистит всех клиентов из панели 3x-ui")
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
		fmt.Println("ℹ️  Inbound'ов не найдено, нечего очищать")
		return
	}

	// Показываем информацию о найденных inbound'ах
	fmt.Println("\n📊 Информация о найденных inbound'ах:")
	for i, inbound := range inbounds {
		clientCount := getClientCount(inbound.Settings)
		fmt.Printf("  %d. ID: %d, Remark: %s, Клиентов: %d\n",
			i+1, inbound.ID, inbound.Remark, clientCount)
	}

	// Спрашиваем подтверждение
	fmt.Print("\n⚠️  ВНИМАНИЕ! Это действие удалит ВСЕХ клиентов из ВСЕХ inbound'ов!\n")
	fmt.Print("Продолжить? (yes/no): ")

	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "yes" {
		fmt.Println("❌ Операция отменена")
		return
	}

	// Очищаем клиентов из каждого inbound'а
	fmt.Println("\n🧹 Начинаем очистку...")
	totalCleared := 0

	for _, inbound := range inbounds {
		fmt.Printf("Очищаем inbound %d (%s)... ", inbound.ID, inbound.Remark)

		clientCount := getClientCount(inbound.Settings)
		err := clearInboundClients(sessionCookie, inbound.ID)
		if err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)
			continue
		}

		fmt.Printf("✅ Очищено %d клиентов\n", clientCount)
		totalCleared += clientCount
	}

	fmt.Printf("\n🎉 Очистка завершена! Всего удалено клиентов: %d\n", totalCleared)
}

// LoginRequest структура для авторизации
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginToPanel авторизуется в панели 3x-ui
func loginToPanel() (string, error) {
	loginData := LoginRequest{
		Username: PANEL_USER,
		Password: PANEL_PASS,
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

	// Извлекаем куку из заголовков ответа
	for _, cookie := range resp.Header.Values("Set-Cookie") {
		if strings.Contains(cookie, "3x-ui=") {
			sessionCookie := strings.Split(cookie, ";")[0]
			return sessionCookie, nil
		}
	}

	return "", fmt.Errorf("кука сессии не найдена")
}

// getAllInbounds получает список всех inbound'ов
func getAllInbounds(sessionCookie string) ([]Inbound, error) {
	req, err := http.NewRequest("GET", PANEL_URL+"panel/api/inbounds", nil)
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

	// Отладочная информация
	fmt.Printf("DEBUG: Статус ответа: %d\n", resp.StatusCode)
	fmt.Printf("DEBUG: Тело ответа: %s\n", string(body))

	var response InboundListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("неудачное получение списка inbound'ов: %s", response.Msg)
	}

	return response.Obj, nil
}

// getClientCount подсчитывает количество клиентов в настройках
func getClientCount(settingsJSON string) int {
	var settings Settings
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		return 0
	}
	return len(settings.Clients)
}

// clearInboundClients очищает клиентов из конкретного inbound'а
func clearInboundClients(sessionCookie string, inboundID int) error {
	// Получаем inbound
	req, err := http.NewRequest("GET", fmt.Sprintf("%spanel/api/inbounds/get/%d", PANEL_URL, inboundID), nil)
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

	var inboundResp InboundGetResponse
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

	updateReq, err := http.NewRequest("POST", fmt.Sprintf("%spanel/api/inbounds/update/%d", PANEL_URL, inboundID), bytes.NewBuffer(updateData))
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

	var updateResponse InboundUpdateResponse
	if err := json.Unmarshal(updateBody, &updateResponse); err != nil {
		return fmt.Errorf("ошибка парсинга ответа обновления inbound: %v", err)
	}

	if !updateResponse.Success {
		return fmt.Errorf("обновление inbound не удалось: %s", updateResponse.Msg)
	}

	return nil
}
