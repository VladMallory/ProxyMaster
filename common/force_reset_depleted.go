package common

import (
    "encoding/json"
    "fmt"
    "log"
    "strings"
    "time"
)

// ForceResetDepletedStatus принудительно сбрасывает состояние "исчерпано" для клиента
// Безопасный однофазный сброс: просто выставляет depleted/exhausted=false без промежуточного включения true
func ForceResetDepletedStatus(sessionCookie string, telegramID int64) error {
    log.Printf("FORCE_RESET: Начало принудительного однофазного сброса состояния 'исчерпано' для TelegramID=%d", telegramID)
    // ЛОГИРОВАНИЕ exhausted: фиксируем кто инициировал сброс и для кого.
    LogExhausted("FORCE_RESET_DEPLETED", "Старт однофазного сброса exhausted/depleted: TelegramID=%d", telegramID)
    LogClientOperation("FORCE_RESET_DEPLETED", telegramID, "", "Начало однофазного сброса состояния 'исчерпано'")

	// Получаем текущий inbound
	inbound, err := GetInbound(sessionCookie)
	if err != nil {
		log.Printf("FORCE_RESET: Ошибка получения inbound: %v", err)
		return fmt.Errorf("ошибка получения inbound: %v", err)
	}

	var settings Settings
	if err = json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
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

    // Однофазный: просто ставим флаги в false и обновляем UpdatedAt
    falseValue := false
    // ЛОГИРОВАНИЕ exhausted: явные записи в момент присвоения статуса false.
    // Это ключевой момент, когда статус "исчерпано" меняется на false.
    settings.Clients[clientIndex].Depleted = &falseValue
    LogExhausted("FORCE_RESET_DEPLETED", "Присвоение depleted=false для Email=%s, TelegramID=%d", targetClient.Email, telegramID)
    settings.Clients[clientIndex].Exhausted = &falseValue
    LogExhausted("FORCE_RESET_DEPLETED", "Присвоение exhausted=false для Email=%s, TelegramID=%d", targetClient.Email, telegramID)
    settings.Clients[clientIndex].UpdatedAt = time.Now().UnixMilli()

    // Обновляем inbound
    settingsJSON, err := json.Marshal(settings)
    if err != nil {
        log.Printf("FORCE_RESET: Ошибка сериализации settings: %v", err)
        return fmt.Errorf("ошибка сериализации settings: %v", err)
    }
    inbound.Settings = string(settingsJSON)

    if err := updateInbound(sessionCookie, *inbound); err != nil {
        log.Printf("FORCE_RESET: Ошибка обновления inbound: %v", err)
        return fmt.Errorf("ошибка обновления inbound: %v", err)
    }

    log.Printf("FORCE_RESET: ✅ Однофазный сброс состояния 'исчерпано' завершён для TelegramID=%d", telegramID)
    log.Printf("FORCE_RESET: Финальное состояние: Email=%s, Enable=%t, Depleted=false, Exhausted=false",
        targetClient.Email, targetClient.Enable)
    // Успешное завершение операции сброса.
    LogExhausted("FORCE_RESET_DEPLETED", "Завершён сброс exhausted/depleted: Email=%s, Enable=%t", targetClient.Email, targetClient.Enable)

    LogClientOperation("FORCE_RESET_DEPLETED", telegramID, targetClient.Email, "Однофазный сброс состояния завершён успешно")
    return nil
}
