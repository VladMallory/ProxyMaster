package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"bot/common"
)

func main() {
	log.Printf("=== ТЕСТ ДОБАВЛЕНИЯ КЛИЕНТА В ИНБАУНД ID 5 ===")

	// Авторизация
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("❌ Ошибка авторизации: %v", err)
		return
	}
	log.Printf("✅ Авторизация успешна")

	// Получаем инбаунд ID 5
	inbound, err := getInbound(sessionCookie, 5)
	if err != nil {
		log.Printf("❌ Ошибка получения инбаунда: %v", err)
		return
	}

	// Парсим settings
	var settings common.Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Printf("❌ Ошибка парсинга settings: %v", err)
		return
	}

	log.Printf("✅ Инбаунд получен, клиентов: %d", len(settings.Clients))

	// Показываем существующих клиентов
	log.Printf("Существующие клиенты:")
	for i, client := range settings.Clients {
		log.Printf("  [%d] Email=%s, SubID=%s, Enable=%v, TgID=%v, Flow='%s'",
			i, client.Email, client.SubID, client.Enable, client.TgID, client.Flow)
	}

	// Создаем тестового клиента
	testClient := common.Client{
		ID:        generateUUID(),
		Flow:      "", // Пустой flow для VLESS WebSocket
		Email:     "test_client_123",
		Enable:    true,
		TgID:      0,
		SubID:     generateSubID(),
		CreatedAt: time.Now().UnixMilli(),
		UpdatedAt: time.Now().UnixMilli(),
	}

	log.Printf("\n=== ДОБАВЛЕНИЕ ТЕСТОВОГО КЛИЕНТА ===")
	log.Printf("Новый клиент: Email=%s, SubID=%s, Flow='%s'",
		testClient.Email, testClient.SubID, testClient.Flow)

	// Добавляем клиента
	settings.Clients = append(settings.Clients, testClient)

	// Обновляем инбаунд
	err = updateInbound(sessionCookie, *inbound, settings)
	if err != nil {
		log.Printf("❌ Ошибка обновления инбаунда: %v", err)
		return
	}

	log.Printf("✅ Клиент добавлен успешно!")

	// Проверяем, что инбаунд все еще работает
	log.Printf("\n=== ПРОВЕРКА РАБОТОСПОСОБНОСТИ ===")
	updatedInbound, err := getInbound(sessionCookie, 5)
	if err != nil {
		log.Printf("❌ Инбаунд упал: %v", err)
		return
	}

	// Парсим обновленные settings
	var updatedSettings common.Settings
	if err := json.Unmarshal([]byte(updatedInbound.Settings), &updatedSettings); err != nil {
		log.Printf("❌ Ошибка парсинга обновленных settings: %v", err)
		return
	}

	log.Printf("✅ Инбаунд работает! Клиентов: %d", len(updatedSettings.Clients))
	log.Printf("Последний клиент: Email=%s, SubID=%s, Enable=%v",
		updatedSettings.Clients[len(updatedSettings.Clients)-1].Email,
		updatedSettings.Clients[len(updatedSettings.Clients)-1].SubID,
		updatedSettings.Clients[len(updatedSettings.Clients)-1].Enable)

	log.Printf("\n=== ТЕСТ ЗАВЕРШЕН УСПЕШНО ===")
}

// Вспомогательные функции
func getInbound(sessionCookie string, inboundID int) (*common.Inbound, error) {
	url := fmt.Sprintf("%s/api/inbounds/%d", common.PANEL_URL, inboundID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", sessionCookie)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Printf("GET_INBOUND: Ответ сервера: status=%d, body=%s", resp.StatusCode, string(body))

	var response struct {
		Success bool           `json:"success"`
		Msg     string         `json:"msg"`
		Obj     common.Inbound `json:"obj"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("GET_INBOUND: Ошибка десериализации: %v, body=%s", err, string(body))
		return nil, err
	}

	if !response.Success {
		return nil, fmt.Errorf("получение инбаунда не удалось: %s", response.Msg)
	}

	return &response.Obj, nil
}

func updateInbound(sessionCookie string, inbound common.Inbound, settings common.Settings) error {
	// Сериализуем settings
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	inbound.Settings = string(settingsJSON)

	url := fmt.Sprintf("%s/api/inbounds/%d", common.PANEL_URL, inbound.ID)

	jsonData, err := json.Marshal(inbound)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", sessionCookie)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if !response.Success {
		return fmt.Errorf("обновление инбаунда не удалось: %s", response.Msg)
	}

	return nil
}

func generateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func generateSubID() string {
	return fmt.Sprintf("test-%d", time.Now().Unix())
}
