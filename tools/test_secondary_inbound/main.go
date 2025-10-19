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

	"bot/common"
)

// Структуры для работы с панелью 3x-ui
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type InboundInfo struct {
	Success bool    `json:"success"`
	Msg     string  `json:"msg"`
	Obj     Inbound `json:"obj"`
}

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

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func main() {
	log.Printf("=== ТЕСТ ПОДКЛЮЧЕНИЯ К ДОПОЛНИТЕЛЬНОМУ ИНБАУНДУ ===")

	// Проверяем настройки
	log.Printf("SECONDARY_INBOUND_ENABLED: %v", common.SECONDARY_INBOUND_ENABLED)
	log.Printf("SECONDARY_INBOUND_ID: %d", common.SECONDARY_INBOUND_ID)
	log.Printf("PANEL_URL: %s", common.PANEL_URL)
	log.Printf("PANEL_USER: %s", common.PANEL_USER)

	if !common.SECONDARY_INBOUND_ENABLED {
		log.Printf("❌ Дополнительный инбаунд отключен в конфигурации")
		return
	}

	// Тест 1: Авторизация в панели
	log.Printf("\n=== ТЕСТ 1: АВТОРИЗАЦИЯ В ПАНЕЛИ ===")
	sessionCookie, err := login()
	if err != nil {
		log.Printf("❌ Ошибка авторизации: %v", err)
		return
	}
	log.Printf("✅ Успешная авторизация, получена сессионная кука")

	// Тест 2: Получение основного инбаунда (для сравнения)
	log.Printf("\n=== ТЕСТ 2: ПОЛУЧЕНИЕ ОСНОВНОГО ИНБАУНДА (ID=%d) ===", common.INBOUND_ID)
	primaryInbound, err := getInbound(sessionCookie, common.INBOUND_ID)
	if err != nil {
		log.Printf("❌ Ошибка получения основного инбаунда: %v", err)
	} else {
		log.Printf("✅ Основной инбаунд получен успешно")
		printInboundInfo(primaryInbound, "Основной")
	}

	// Тест 3: Получение дополнительного инбаунда
	log.Printf("\n=== ТЕСТ 3: ПОЛУЧЕНИЕ ДОПОЛНИТЕЛЬНОГО ИНБАУНДА (ID=%d) ===", common.SECONDARY_INBOUND_ID)
	secondaryInbound, err := getInbound(sessionCookie, common.SECONDARY_INBOUND_ID)
	if err != nil {
		log.Printf("❌ Ошибка получения дополнительного инбаунда: %v", err)
		return
	}
	log.Printf("✅ Дополнительный инбаунд получен успешно")
	printInboundInfo(secondaryInbound, "Дополнительный")

	// Тест 4: Поиск тестового пользователя в дополнительном инбаунде
	log.Printf("\n=== ТЕСТ 4: ПОИСК ПОЛЬЗОВАТЕЛЯ В ДОПОЛНИТЕЛЬНОМ ИНБАУНДЕ ===")
	testTelegramID := int64(873925520) // ID из логов
	findUserInSecondaryInbound(secondaryInbound, testTelegramID)

	// Тест 5: Создание тестового клиента в дополнительном инбаунде
	log.Printf("\n=== ТЕСТ 5: СОЗДАНИЕ ТЕСТОВОГО КЛИЕНТА ===")
	testClient := Client{
		ID:         "test-client-id-" + fmt.Sprintf("%d", time.Now().Unix()),
		Email:      fmt.Sprintf("%d@test.secondary", testTelegramID),
		SubID:      "test-sub-id-" + fmt.Sprintf("%d", time.Now().Unix()),
		Enable:     true,
		ExpiryTime: time.Now().AddDate(0, 0, 30).UnixMilli(), // 30 дней
		Flow:       "xtls-rprx-vision",
		TotalGB:    0,
		Reset:      0,
		TgID:       testTelegramID,
		CreatedAt:  time.Now().UnixMilli(),
		UpdatedAt:  time.Now().UnixMilli(),
	}

	err = createTestClient(sessionCookie, secondaryInbound, testClient)
	if err != nil {
		log.Printf("❌ Ошибка создания тестового клиента: %v", err)
	} else {
		log.Printf("✅ Тестовый клиент создан успешно")

		// Проверяем, что клиент появился
		log.Printf("\n=== ПРОВЕРКА СОЗДАННОГО КЛИЕНТА ===")
		updatedInbound, err := getInbound(sessionCookie, common.SECONDARY_INBOUND_ID)
		if err != nil {
			log.Printf("❌ Ошибка получения обновленного инбаунда: %v", err)
		} else {
			findUserInSecondaryInbound(updatedInbound, testTelegramID)
		}
	}

	log.Printf("\n=== ТЕСТ ЗАВЕРШЕН ===")
}

func login() (string, error) {
	loginData := LoginRequest{
		Username: common.PANEL_USER,
		Password: common.PANEL_PASS,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации данных авторизации: %v", err)
	}

	req, err := http.NewRequest("POST", common.PANEL_URL+"login", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("некорректный статус ответа: %d, body=%s", resp.StatusCode, string(body))
	}

	// Извлекаем сессионную куку
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "3x-ui" {
			return cookie.String(), nil
		}
	}

	return "", fmt.Errorf("сессионная кука не найдена в ответе")
}

func getInbound(sessionCookie string, inboundID int) (*Inbound, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%spanel/api/inbounds/get/%d", common.PANEL_URL, inboundID), nil)
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("некорректный статус ответа: %d, body=%s", resp.StatusCode, string(body))
	}

	var inboundInfo InboundInfo
	if err := json.Unmarshal(body, &inboundInfo); err != nil {
		return nil, fmt.Errorf("ошибка десериализации ответа: %v, body=%s", err, string(body))
	}

	if !inboundInfo.Success {
		return nil, fmt.Errorf("получение inbound не удалось: %s", inboundInfo.Msg)
	}

	return &inboundInfo.Obj, nil
}

func printInboundInfo(inbound *Inbound, prefix string) {
	log.Printf("%s инбаунд:", prefix)
	log.Printf("  ID: %d", inbound.ID)
	log.Printf("  Settings длина: %d символов", len(inbound.Settings))

	// Парсим settings
	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Printf("  ❌ Ошибка парсинга settings: %v", err)
		return
	}

	log.Printf("  Клиентов: %d", len(settings.Clients))

	// Показываем первые 5 клиентов
	maxClients := 5
	if len(settings.Clients) < maxClients {
		maxClients = len(settings.Clients)
	}

	for i := 0; i < maxClients; i++ {
		client := settings.Clients[i]
		log.Printf("    [%d] Email=%s, SubID=%s, Enable=%v, TgID=%d",
			i, client.Email, client.SubID, client.Enable, client.TgID)
	}

	if len(settings.Clients) > maxClients {
		log.Printf("    ... и еще %d клиентов", len(settings.Clients)-maxClients)
	}
}

func findUserInSecondaryInbound(inbound *Inbound, telegramID int64) {
	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Printf("❌ Ошибка парсинга settings: %v", err)
		return
	}

	log.Printf("Поиск пользователя %d среди %d клиентов:", telegramID, len(settings.Clients))

	found := false
	for i, client := range settings.Clients {
		// Проверяем по TgID
		if client.TgID == telegramID {
			log.Printf("✅ Найден по TgID: [%d] Email=%s, SubID=%s, Enable=%v",
				i, client.Email, client.SubID, client.Enable)
			found = true
		}

		// Проверяем по email (если email = telegramID)
		if client.Email == fmt.Sprintf("%d", telegramID) {
			log.Printf("✅ Найден по Email: [%d] Email=%s, SubID=%s, Enable=%v",
				i, client.Email, client.SubID, client.Enable)
			found = true
		}
	}

	if !found {
		log.Printf("❌ Пользователь %d не найден в дополнительном инбаунде", telegramID)
		log.Printf("Доступные клиенты:")
		for i, client := range settings.Clients {
			log.Printf("  [%d] Email=%s, SubID=%s, Enable=%v, TgID=%d",
				i, client.Email, client.SubID, client.Enable, client.TgID)
		}
	}
}

func createTestClient(sessionCookie string, inbound *Inbound, newClient Client) error {
	// Парсим settings
	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return fmt.Errorf("ошибка парсинга settings: %v", err)
	}

	// Добавляем нового клиента
	settings.Clients = append(settings.Clients, newClient)

	// Сериализуем обновлённые settings
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("ошибка сериализации settings: %v", err)
	}

	// Создаем обновленный inbound
	updatedInbound := *inbound
	updatedInbound.Settings = string(settingsJSON)

	// Обновляем inbound
	jsonData, err := json.Marshal(updatedInbound)
	if err != nil {
		return fmt.Errorf("ошибка сериализации данных: %v", err)
	}

	req, err := http.NewRequest("POST", common.PANEL_URL+"panel/api/inbounds/update/"+fmt.Sprintf("%d", inbound.ID), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", sessionCookie)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("некорректный статус ответа: %d, body=%s", resp.StatusCode, string(body))
	}

	var updateResp map[string]interface{}
	if err := json.Unmarshal(body, &updateResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %v", err)
	}

	if success, ok := updateResp["success"].(bool); !ok || !success {
		return fmt.Errorf("обновление inbound не удалось: %v", updateResp)
	}

	return nil
}
