package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"bot/common"
)

func main() {
	log.Printf("=== ПРОСТОЙ ТЕСТ ИНБАУНДОВ ===")

	// Авторизация
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("❌ Ошибка авторизации: %v", err)
		return
	}
	log.Printf("✅ Авторизация успешна")

	// Получаем основной инбаунд
	log.Printf("\n=== ОСНОВНОЙ ИНБАУНД ===")
	primaryInbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Printf("❌ Ошибка получения основного инбаунда: %v", err)
	} else {
		log.Printf("✅ Основной инбаунд получен: ID=%d, Protocol=%s, Port=%d",
			primaryInbound.ID, primaryInbound.Protocol, primaryInbound.Port)
	}

	// Получаем дополнительный инбаунд
	log.Printf("\n=== ДОПОЛНИТЕЛЬНЫЙ ИНБАУНД ===")
	log.Printf("SECONDARY_INBOUND_ID: %d", common.SECONDARY_INBOUND_ID)

	secondaryInbound, err := common.GetSecondaryInbound(sessionCookie)
	if err != nil {
		log.Printf("❌ Ошибка получения дополнительного инбаунда: %v", err)
	} else {
		log.Printf("✅ Дополнительный инбаунд получен: ID=%d, Protocol=%s, Port=%d",
			secondaryInbound.ID, secondaryInbound.Protocol, secondaryInbound.Port)
	}

	// Получаем все инбаунды
	log.Printf("\n=== ВСЕ ИНБАУНДЫ ===")
	allInbounds, err := getAllInbounds(sessionCookie)
	if err != nil {
		log.Printf("❌ Ошибка получения всех инбаундов: %v", err)
	} else {
		log.Printf("✅ Найдено инбаундов: %d", len(allInbounds))
		for _, inbound := range allInbounds {
			log.Printf("  ID=%d, Protocol=%s, Port=%d, Enable=%v, Remark=%s",
				inbound.ID, inbound.Protocol, inbound.Port, inbound.Enable, inbound.Remark)
		}
	}
}

// Копируем функцию из common/panel_api.go для тестирования
func getAllInbounds(sessionCookie string) ([]common.Inbound, error) {
	url := common.PANEL_URL + "/api/inbounds"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", sessionCookie)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Printf("GET_ALL_INBOUNDS: Ответ сервера: status=%d, body=%s", resp.StatusCode, string(body))

	var response struct {
		Success bool             `json:"success"`
		Msg     string           `json:"msg"`
		Obj     []common.Inbound `json:"obj"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if !response.Success {
		return nil, fmt.Errorf("получение списка инбаундов не удалось: %s", response.Msg)
	}

	return response.Obj, nil
}
