
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
	"strconv"

	"bot/common"
)

func main() {
	// Инициализация подключения к БД
	common.InitMongoDB()
	defer common.DisconnectMongoDB()

	telegramID := int64(6481415963)

	// 1. Получаем пользователя из БД
	user, err := common.GetUserByTelegramID(telegramID)
	if err != nil {
		log.Fatalf("Ошибка получения пользователя из БД: %v", err)
	}
	if user == nil {
		log.Fatalf("Пользователь с ID %d не найден в БД", telegramID)
	}

	fmt.Printf("--- ДАННЫЕ ИЗ БД ---\n")
	fmt.Printf("TelegramID: %d\n", user.TelegramID)
	fmt.Printf("HasActiveConfig: %v\n", user.HasActiveConfig)
	expiryDate := time.UnixMilli(user.ExpiryTime).Format("2006-01-02 15:04:05")
	fmt.Printf("ExpiryTime: %d (%s)\n", user.ExpiryTime, expiryDate)
	fmt.Printf("Balance: %.2f\n", user.Balance)
	fmt.Println("--------------------\n")

	// 2. Получаем данные из панели
	common.InitGlobals()
	sessionCookie, err := common.Login()
	if err != nil {
		log.Fatalf("Ошибка авторизации в панели: %v", err)
	}

	inbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Fatalf("Ошибка получения inbound: %v", err)
	}

	var settings common.Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Fatalf("Ошибка десериализации settings: %v", err)
	}

	telegramIDStr := strconv.FormatInt(telegramID, 10)
	var panelClient *common.Client
	for i, client := range settings.Clients {
		if strings.HasPrefix(client.Email, telegramIDStr) {
			panelClient = &settings.Clients[i]
			break
		}
	}

	if panelClient == nil {
		log.Fatalf("Клиент для пользователя %d не найден в панели", telegramID)
	}

	isDepleted := panelClient.Depleted != nil && *panelClient.Depleted
	isExhausted := panelClient.Exhausted != nil && *panelClient.Exhausted

	fmt.Printf("--- ДАННЫЕ ИЗ ПАНЕЛИ ---\n")
	fmt.Printf("Email: %s\n", panelClient.Email)
	fmt.Printf("Enable: %v\n", panelClient.Enable)
	fmt.Printf("Depleted: %v\n", isDepleted)
	fmt.Printf("Exhausted: %v\n", isExhausted)

	panelExpiryDate := time.UnixMilli(panelClient.ExpiryTime).Format("2006-01-02 15:04:05")
	fmt.Printf("ExpiryTime: %d (%s)\n", panelClient.ExpiryTime, panelExpiryDate)
	fmt.Println("----------------------\n")

	// 3. Проверяем логику shouldFixDepletedStatusOptimized
	now := time.Now()
	shouldFix := shouldFixDepletedStatusOptimized(user, now, &settings)
	fmt.Printf("--- РЕЗУЛЬТАТ ПРОВЕРКИ ---\n")
	fmt.Printf("Нужно ли исправлять статус? -> %v\n", shouldFix)
	fmt.Println("--------------------------\n")
}

// Копия логики из depleted_status_checker.go для отладки
func shouldFixDepletedStatusOptimized(user *common.User, now time.Time, settings *common.Settings) bool {
	fmt.Println("--- Логика shouldFixDepletedStatusOptimized ---")
	// Проверяем, что у пользователя есть активная подписка
	if !user.HasActiveConfig {
		fmt.Println(" -> Проверка не пройдена: user.HasActiveConfig == false")
		return false
	}
	fmt.Println(" -> Проверка пройдена: user.HasActiveConfig == true")

	// Проверяем, что подписка не истекла
	if user.ExpiryTime > 0 && user.ExpiryTime <= now.UnixMilli() {
		fmt.Println(" -> Проверка не пройдена: подписка истекла")
		return false
	}
	fmt.Println(" -> Проверка пройдена: подписка не истекла")

	// ОСНОВНАЯ ПРОВЕРКА: Если подписка активна и до истечения больше часа,
	// то состояние "исчерпано" должно быть сброшено
	if user.ExpiryTime > 0 {
		timeUntilExpiry := time.Duration(user.ExpiryTime-now.UnixMilli()) * time.Millisecond
		fmt.Printf(" -> Время до истечения: %v\n", timeUntilExpiry)
		if timeUntilExpiry > 1*time.Hour {
			fmt.Println(" -> Проверка пройдена: время до истечения > 1 часа")
			// Ищем клиента в уже загруженных settings
			telegramIDStr := fmt.Sprintf("%d", user.TelegramID)
			for _, client := range settings.Clients {
				if strings.HasPrefix(client.Email, telegramIDStr) {
					fmt.Printf(" -> Найден клиент в панели: Email=%s\n", client.Email)
					// Проверяем состояние "исчерпано" в загруженных данных
					isDepleted := client.Depleted != nil && *client.Depleted
					isExhausted := client.Exhausted != nil && *client.Exhausted
					fmt.Printf(" -> Статус в панели: Depleted=%v, Exhausted=%v, Enable=%v\n", isDepleted, isExhausted, client.Enable)

					// ВАЖНО: Если конфиг отключен в панели, НЕ сбрасываем статус "исчерпано"
					if !client.Enable {
						fmt.Println(" -> Проверка не пройдена: client.Enable == false")
						return false
					}
					fmt.Println(" -> Проверка пройдена: client.Enable == true")

					// Возвращаем true если конфиг в состоянии "исчерпано" И включен в панели
					result := isDepleted || isExhausted
					fmt.Printf(" -> Финальное решение: %v\n", result)
					return result
				}
			}
		} else {
			fmt.Println(" -> Проверка не пройдена: время до истечения <= 1 часа")
			return false // Не исправляем, подписка скоро истекает
		}
	}

	fmt.Println(" -> Не удалось выполнить основную проверку")
	return false
}
