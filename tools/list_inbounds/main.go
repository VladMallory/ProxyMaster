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
	log.Printf("=== СПИСОК ВСЕХ ИНБАУНДОВ ===")

	// Авторизация
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("❌ Ошибка авторизации: %v", err)
		return
	}
	log.Printf("✅ Авторизация успешна")

	// Получаем список инбаундов
	url := fmt.Sprintf("%s/api/inbounds", common.PANEL_URL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("❌ Ошибка создания запроса: %v", err)
		return
	}
	req.Header.Set("Cookie", sessionCookie)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Ошибка выполнения запроса: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Ошибка чтения ответа: %v", err)
		return
	}

	log.Printf("Ответ сервера: status=%d, body=%s", resp.StatusCode, string(body))

	var response struct {
		Success bool             `json:"success"`
		Msg     string           `json:"msg"`
		Obj     []common.Inbound `json:"obj"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("❌ Ошибка десериализации: %v", err)
		return
	}

	if !response.Success {
		log.Printf("❌ Получение списка инбаундов не удалось: %s", response.Msg)
		return
	}

	log.Printf("✅ Найдено инбаундов: %d", len(response.Obj))

	for _, inbound := range response.Obj {
		log.Printf("  ID=%d, Protocol=%s, Port=%d, Enable=%v, Remark=%s",
			inbound.ID, inbound.Protocol, inbound.Port, inbound.Enable, inbound.Remark)
	}
}
