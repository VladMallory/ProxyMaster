package common

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// ForceResetDepletedStatus принудительно сбрасывает состояние "исчерпано" для клиента
// Использует тот же двухфазовый подход, что и в тестовом скрипте
func ForceResetDepletedStatus(sessionCookie string, telegramID int64) error {
	log.Printf("FORCE_RESET: Начало принудительного сброса состояния 'исчерпано' для TelegramID=%d", telegramID)

	// Получаем текущий inbound
	inbound, err := GetInbound(sessionCookie)
	if err != nil {
		log.Printf("FORCE_RESET: Ошибка получения inbound: %v", err)
		return fmt.Errorf("ошибка получения inbound: %v", err)
	}

	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Printf("FORCE_RESET: Ошибка десериализации settings: %v", err)
		return fmt.Errorf("ошибка десериализации settings: %v", err)
	}

	// Ищем клиента по TelegramID
	telegramIDStr := fmt.Sprintf("%d", telegramID)
	clientIndex := -1
	var targetClient *Client

	for i, client := range settings.Clients {
		if strings.HasPrefix(client.Email, telegramIDStr+"_") ||
			strings.HasPrefix(client.Email, telegramIDStr+" ") ||
			client.Email == telegramIDStr {
			clientIndex = i
			targetClient = &settings.Clients[i]
			break
		}
	}

	if clientIndex == -1 {
		log.Printf("FORCE_RESET: Клиент с TelegramID=%d не найден", telegramID)
		return fmt.Errorf("клиент с TelegramID=%d не найден", telegramID)
	}

	log.Printf("FORCE_RESET: Найден клиент: Email=%s, UUID=%s, Enable=%t",
		targetClient.Email, targetClient.ID, targetClient.Enable)

	originalEmail := targetClient.Email
	originalExpiry := targetClient.ExpiryTime
	originalEnable := targetClient.Enable

	// ==================== ФАЗА A ====================
	log.Printf("FORCE_RESET: 🅰️  ФАЗА A - Установка depleted/exhausted=TRUE и выключение")

	trueValue := true
	toggleEmail := originalEmail + "-reset"

	settings.Clients[clientIndex].Depleted = &trueValue
	settings.Clients[clientIndex].Exhausted = &trueValue
	settings.Clients[clientIndex].Enable = false
	settings.Clients[clientIndex].Email = toggleEmail
	settings.Clients[clientIndex].UpdatedAt = time.Now().UnixMilli()

	// Обновляем inbound (ФАЗА A)
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		log.Printf("FORCE_RESET: Ошибка сериализации settings (ФАЗА A): %v", err)
		return fmt.Errorf("ошибка сериализации settings (ФАЗА A): %v", err)
	}
	inbound.Settings = string(settingsJSON)

	if err := updateInbound(sessionCookie, *inbound); err != nil {
		log.Printf("FORCE_RESET: Ошибка обновления inbound (ФАЗА A): %v", err)
		return fmt.Errorf("ошибка обновления inbound (ФАЗА A): %v", err)
	}

	log.Printf("FORCE_RESET: ФАЗА A завершена, пауза 1000мс...")
	time.Sleep(1000 * time.Millisecond)

	// ==================== ФАЗА B ====================
	log.Printf("FORCE_RESET: 🅱️  ФАЗА B - Установка depleted/exhausted=FALSE и включение")

	falseValue := false

	settings.Clients[clientIndex].Depleted = &falseValue
	settings.Clients[clientIndex].Exhausted = &falseValue
	settings.Clients[clientIndex].Enable = originalEnable
	settings.Clients[clientIndex].Email = originalEmail
	settings.Clients[clientIndex].ExpiryTime = originalExpiry
	settings.Clients[clientIndex].UpdatedAt = time.Now().UnixMilli()

	// Обновляем inbound (ФАЗА B)
	settingsJSON, err = json.Marshal(settings)
	if err != nil {
		log.Printf("FORCE_RESET: Ошибка сериализации settings (ФАЗА B): %v", err)
		return fmt.Errorf("ошибка сериализации settings (ФАЗА B): %v", err)
	}
	inbound.Settings = string(settingsJSON)

	if err := updateInbound(sessionCookie, *inbound); err != nil {
		log.Printf("FORCE_RESET: Ошибка обновления inbound (ФАЗА B): %v", err)
		return fmt.Errorf("ошибка обновления inbound (ФАЗА B): %v", err)
	}

	log.Printf("FORCE_RESET: ✅ Принудительный сброс состояния 'исчерпано' завершён для TelegramID=%d", telegramID)
	log.Printf("FORCE_RESET: Финальное состояние: Email=%s, Enable=%t, Depleted=false, Exhausted=false",
		originalEmail, originalEnable)

	return nil
}
