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
	log.Printf("=== БЫСТРЫЙ ТЕСТ ДОБАВЛЕНИЯ КЛИЕНТА ===")
	log.Printf("SECONDARY_INBOUND_ID: %d", common.SECONDARY_INBOUND_ID)

	// Авторизация
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("❌ Ошибка авторизации: %v", err)
		return
	}
	log.Printf("✅ Авторизация успешна")

	// Получаем инбаунд
	inbound, err := getInbound(sessionCookie, common.SECONDARY_INBOUND_ID)
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
		log.Printf("  [%d] Email=%s, SubID=%s, Enable=%v, TgID=%v",
			i, client.Email, client.SubID, client.Enable, client.TgID)
	}

	// Тестируем разные варианты добавления клиента
	testCases := []struct {
		name   string
		client common.Client
	}{
		{
			name: "Минимальный клиент",
			client: common.Client{
				ID:        generateUUID(),
				Email:     "test_minimal@test.com",
				Enable:    true,
				TgID:      0,
				SubID:     generateSubID(),
				CreatedAt: time.Now().UnixMilli(),
				UpdatedAt: time.Now().UnixMilli(),
			},
		},
		{
			name: "Клиент с пустым Flow",
			client: common.Client{
				ID:        generateUUID(),
				Flow:      "",
				Email:     "test_empty_flow@test.com",
				Enable:    true,
				TgID:      0,
				SubID:     generateSubID(),
				CreatedAt: time.Now().UnixMilli(),
				UpdatedAt: time.Now().UnixMilli(),
			},
		},
		{
			name: "Клиент без Flow поля",
			client: common.Client{
				ID:        generateUUID(),
				Email:     "test_no_flow@test.com",
				Enable:    true,
				TgID:      0,
				SubID:     generateSubID(),
				CreatedAt: time.Now().UnixMilli(),
				UpdatedAt: time.Now().UnixMilli(),
			},
		},
	}

	for i, testCase := range testCases {
		log.Printf("\n=== ТЕСТ %d: %s ===", i+1, testCase.name)

		// Создаем копию settings
		testSettings := settings
		testSettings.Clients = append(testSettings.Clients, testCase.client)

		// Пытаемся обновить инбаунд
		err := updateInbound(sessionCookie, *inbound, testSettings)
		if err != nil {
			log.Printf("❌ Тест %d провален: %v", i+1, err)
		} else {
			log.Printf("✅ Тест %d успешен!", i+1)

			// Проверяем, что инбаунд все еще работает
			_, err = getInbound(sessionCookie, common.SECONDARY_INBOUND_ID)
			if err != nil {
				log.Printf("❌ Инбаунд упал после теста %d: %v", i+1, err)
				break
			} else {
				log.Printf("✅ Инбаунд работает после теста %d", i+1)
			}
		}

		// Небольшая пауза между тестами
		time.Sleep(1 * time.Second)
	}

	log.Printf("\n=== ТЕСТИРОВАНИЕ ЗАВЕРШЕНО ===")
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
