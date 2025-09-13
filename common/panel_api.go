package common

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

	"github.com/google/uuid"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

// Login выполняет авторизацию в панели 3x-ui
func Login() (string, error) {
	log.Printf("LOGIN: Начало авторизации в панели, URL=%s, Username=%s", PANEL_URL, PANEL_USER)
	loginData := LoginRequest{
		Username: PANEL_USER,
		Password: PANEL_PASS,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		log.Printf("LOGIN: Ошибка сериализации данных авторизации: %v", err)
		return "", fmt.Errorf("ошибка сериализации данных авторизации: %v", err)
	}
	log.Printf("LOGIN: Данные авторизации: %s", string(jsonData))

	req, err := http.NewRequest("POST", PANEL_URL+"login", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("LOGIN: Ошибка создания запроса: %v", err)
		return "", fmt.Errorf("ошибка создания запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	log.Printf("LOGIN: Запрос создан, заголовки: %+v", req.Header)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("LOGIN: Ошибка выполнения запроса: %v", err)
		return "", fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("LOGIN: Ошибка чтения ответа: %v", err)
		return "", fmt.Errorf("ошибка чтения ответа: %v", err)
	}
	log.Printf("LOGIN: Ответ сервера: status=%d, body=%s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		log.Printf("LOGIN: Некорректный статус ответа: %d", resp.StatusCode)
		return "", fmt.Errorf("некорректный статус ответа: %d, body=%s", resp.StatusCode, string(body))
	}

	if len(body) == 0 {
		log.Printf("LOGIN: Пустой ответ от сервера")
		return "", fmt.Errorf("пустой ответ от сервера")
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		log.Printf("LOGIN: Ошибка десериализации ответа: %v, body=%s", err, string(body))
		return "", fmt.Errorf("ошибка десериализации ответа: %v, body=%s", err, string(body))
	}

	if !loginResp.Success {
		log.Printf("LOGIN: Авторизация не удалась: msg=%s", loginResp.Msg)
		return "", fmt.Errorf("авторизация не удалась: %s", loginResp.Msg)
	}

	// Извлекаем куку
	for _, cookie := range resp.Header.Values("Set-Cookie") {
		log.Printf("LOGIN: Найдена кука: %s", cookie)
		if strings.Contains(cookie, "3x-ui=") {
			sessionCookie := strings.Split(cookie, ";")[0]
			log.Printf("LOGIN: Успешная авторизация, кука: %s", sessionCookie)
			return sessionCookie, nil
		}
	}

	log.Printf("LOGIN: Куки сессии не найдены в заголовках: %+v", resp.Header)
	return "", fmt.Errorf("кука сессии не найдена")
}

// GetInbound получает полный inbound object
func GetInbound(sessionCookie string) (*Inbound, error) {
	log.Printf("GET_INBOUND: Получение inbound, ID=%d", INBOUND_ID)
	req, err := http.NewRequest("GET", fmt.Sprintf("%spanel/api/inbounds/get/%d", PANEL_URL, INBOUND_ID), nil)
	if err != nil {
		log.Printf("GET_INBOUND: Ошибка создания запроса: %v", err)
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Cookie", sessionCookie)
	log.Printf("GET_INBOUND: Запрос создан, заголовки: %+v", req.Header)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("GET_INBOUND: Ошибка выполнения запроса: %v", err)
		return nil, fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("GET_INBOUND: Ошибка чтения ответа: %v", err)
		return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}
	log.Printf("GET_INBOUND: Ответ сервера: status=%d, body=%s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		log.Printf("GET_INBOUND: Некорректный статус ответа: %d", resp.StatusCode)
		return nil, fmt.Errorf("некорректный статус ответа: %d, body=%s", resp.StatusCode, string(body))
	}

	var inboundInfo InboundInfo
	if err := json.Unmarshal(body, &inboundInfo); err != nil {
		log.Printf("GET_INBOUND: Ошибка десериализации ответа: %v, body=%s", err, string(body))
		return nil, fmt.Errorf("ошибка десериализации ответа: %v, body=%s", err, string(body))
	}

	if !inboundInfo.Success {
		log.Printf("GET_INBOUND: Получение inbound не удалось: msg=%s", inboundInfo.Msg)
		return nil, fmt.Errorf("получение inbound не удалось: %s", inboundInfo.Msg)
	}

	log.Printf("GET_INBOUND: Успешно получен inbound: ID=%d", inboundInfo.Obj.ID)
	return &inboundInfo.Obj, nil
}

// AddClient добавляет или обновляет клиента в панели
func AddClient(sessionCookie string, user *User, days int) error {
	log.Printf("ADD_CLIENT: Начало добавления/обновления клиента для TelegramID=%d, days=%d", user.TelegramID, days)
	inbound, err := GetInbound(sessionCookie)
	if err != nil {
		log.Printf("ADD_CLIENT: Ошибка получения inbound: %v", err)
		return fmt.Errorf("ошибка получения inbound: %v", err)
	}

	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Printf("ADD_CLIENT: Ошибка десериализации settings: %v", err)
		return fmt.Errorf("ошибка десериализации settings: %v", err)
	}

	clientUUID := uuid.New().String()

	// ИСПРАВЛЕНИЕ: Правильный расчёт времени истечения
	var expiryTime int64
	now := time.Now()

	// Ищем существующего клиента в АКТУАЛЬНОМ списке из панели
	existingClient := FindClientByTelegramID(settings.Clients, user.TelegramID)

	if existingClient != nil && existingClient.ExpiryTime > now.UnixMilli() {
		// Если у клиента есть активная подписка, добавляем дни к существующему времени
		expiryTime = existingClient.ExpiryTime + int64(days)*24*60*60*1000
		log.Printf("ADD_CLIENT: Продление активной подписки: TelegramID=%d, старое время=%d, новое время=%d",
			user.TelegramID, existingClient.ExpiryTime, expiryTime)
	} else {
		// Если подписка истекла или клиента НЕТ В ПАНЕЛИ (3x-ui удалила его), считаем от текущего времени
		expiryTime = now.Add(time.Duration(days) * 24 * time.Hour).UnixMilli()
		if existingClient != nil {
			log.Printf("ADD_CLIENT: Клиент найден в панели, но истёк. Продление истёкшего: TelegramID=%d, время=%d",
				user.TelegramID, expiryTime)
		} else {
			log.Printf("ADD_CLIENT: Клиент НЕ найден в панели (вероятно удален 3x-ui). Создание нового: TelegramID=%d, время=%d",
				user.TelegramID, expiryTime)
		}
	}

	// Формируем email с датой окончания подписки
	var email string
	if SHOW_DATES_IN_CONFIGS {
		expiryDate := time.UnixMilli(expiryTime).Format("2006 02 01")
		email = fmt.Sprintf("%d до %s", user.TelegramID, expiryDate)
	} else {
		email = fmt.Sprintf("%d", user.TelegramID)
	}

	log.Printf("ADD_CLIENT: Подготовка клиента: TelegramID=%d, ClientUUID=%s, Email=%s, ExpiryTime=%d",
		user.TelegramID, clientUUID, email, expiryTime)

	// Проверяем, существует ли клиент В АКТУАЛЬНОМ СПИСКЕ ПАНЕЛИ
	actualExistingClient := FindClientByTelegramID(settings.Clients, user.TelegramID)

	if actualExistingClient != nil {
		now := time.Now()
		isExpired := actualExistingClient.ExpiryTime <= now.UnixMilli()

		if isExpired {
			log.Printf("ADD_CLIENT: Конфиг истёк (состояние 'исчерпано'), пробуем СБРОСИТЬ флаг depleted для TelegramID=%d", user.TelegramID)

			// АГРЕССИВНЫЙ ПОДХОД: Двухфазовый сброс состояния "исчерпано" как в тестовом скрипте
			telegramIDStr := fmt.Sprintf("%d", user.TelegramID)
			resetSuccess := false

			for i, client := range settings.Clients {
				if strings.HasPrefix(client.Email, telegramIDStr+"_") || strings.HasPrefix(client.Email, telegramIDStr+" ") || client.Email == telegramIDStr {
					log.Printf("ADD_CLIENT: Агрессивный двухфазовый сброс состояния 'исчерпано' для клиента: Email=%s", client.Email)

					// ФАЗА A: Сначала устанавливаем depleted=true, exhausted=true, enable=false
					trueValue := true
					toggleEmail := fmt.Sprintf("%s-reset", client.Email)
					settings.Clients[i].Depleted = &trueValue
					settings.Clients[i].Exhausted = &trueValue
					settings.Clients[i].Enable = false
					settings.Clients[i].Email = toggleEmail
					settings.Clients[i].UpdatedAt = time.Now().UnixMilli()

					// Сериализуем и обновляем inbound (ФАЗА A)
					settingsJSON, err := json.Marshal(settings)
					if err != nil {
						log.Printf("ADD_CLIENT: Ошибка сериализации settings (ФАЗА A): %v", err)
						continue
					}
					inbound.Settings = string(settingsJSON)

					log.Printf("ADD_CLIENT: ФАЗА A - устанавливаем depleted=true, exhausted=true для TelegramID=%d", user.TelegramID)
					if err := updateInbound(sessionCookie, *inbound); err != nil {
						log.Printf("ADD_CLIENT: Ошибка обновления inbound (ФАЗА A): %v", err)
						continue
					}

					// Пауза между фазами
					time.Sleep(500 * time.Millisecond)

					// ФАЗА B: Теперь сбрасываем в false и восстанавливаем нормальное состояние
					falseValue := false
					settings.Clients[i].Depleted = &falseValue
					settings.Clients[i].Exhausted = &falseValue
					settings.Clients[i].Enable = true
					settings.Clients[i].Email = email
					settings.Clients[i].ExpiryTime = expiryTime
					settings.Clients[i].Flow = "xtls-rprx-vision"
					settings.Clients[i].TotalGB = 0
					settings.Clients[i].Reset = 0
					settings.Clients[i].UpdatedAt = time.Now().UnixMilli()

					user.ClientID = client.ID
					user.SubID = client.SubID
					user.ExpiryTime = expiryTime
					user.HasActiveConfig = true

					log.Printf("ADD_CLIENT: ФАЗА B - устанавливаем depleted=false, exhausted=false для TelegramID=%d, Email=%s, SubID=%s, ExpiryTime=%d",
						user.TelegramID, email, client.SubID, expiryTime)
					resetSuccess = true
					break
				}
			}

			// Если не удалось сбросить состояние, откатываемся к старому методу
			if !resetSuccess {
				log.Printf("ADD_CLIENT: Сброс флагов не удался, используем метод полного пересоздания для TelegramID=%d", user.TelegramID)
				// Для истекших конфигов УДАЛЯЕМ старый клиент и создаём совершенно новый

				telegramIDStr = fmt.Sprintf("%d", user.TelegramID)
				var newClients []Client

				// Удаляем старого клиента из списка
				for _, client := range settings.Clients {
					// Если это НЕ наш клиент, добавляем в новый список
					if !(strings.HasPrefix(client.Email, telegramIDStr+"_") || strings.HasPrefix(client.Email, telegramIDStr+" ") || client.Email == telegramIDStr) {
						newClients = append(newClients, client)
					} else {
						log.Printf("ADD_CLIENT: Удаляем истекший клиент: Email=%s, UUID=%s, SubID=%s", client.Email, client.ID, client.SubID)
					}
				}

				// Создаём СОВЕРШЕННО НОВОГО клиента
				newClientUUID := uuid.New().String()
				newSubID := GenerateSubID()

				newClient := Client{
					ID:         newClientUUID,
					Flow:       "xtls-rprx-vision",
					Email:      email,
					TotalGB:    0, // Убираем лимит трафика (0 = безлимит)
					ExpiryTime: expiryTime,
					Enable:     true,
					TgID:       0,
					SubID:      newSubID,
					Reset:      0, // Убираем автопродление
				}

				// Добавляем нового клиента в список
				newClients = append(newClients, newClient)
				settings.Clients = newClients

				// Обновляем данные пользователя
				user.HasActiveConfig = true
				user.ClientID = newClientUUID
				user.Email = email
				user.SubID = newSubID
				user.ExpiryTime = expiryTime

				log.Printf("ADD_CLIENT: Истекший клиент УДАЛЁН и создан новый: TelegramID=%d, Email=%s, NewSubID=%s, NewUUID=%s, ExpiryTime=%d",
					user.TelegramID, email, newSubID, newClientUUID, expiryTime)
			}
		} else {
			log.Printf("ADD_CLIENT: Клиент с префиксом %d активен, просто продлеваем", user.TelegramID)

			telegramIDStr := fmt.Sprintf("%d", user.TelegramID)
			for i, client := range settings.Clients {
				// Ищем клиентов, которые начинаются с TelegramID (с подчеркиванием, пробелом или без)
				if strings.HasPrefix(client.Email, telegramIDStr+"_") || strings.HasPrefix(client.Email, telegramIDStr+" ") || client.Email == telegramIDStr {
					settings.Clients[i].ExpiryTime = expiryTime
					settings.Clients[i].Enable = true
					settings.Clients[i].Email = email             // Обновляем email с новой датой окончания
					settings.Clients[i].Flow = "xtls-rprx-vision" // Устанавливаем правильный flow
					settings.Clients[i].TotalGB = 0               // Убираем лимит трафика (0 = безлимит)
					settings.Clients[i].Reset = 0                 // Убираем автопродление
					// ЯВНО сбрасываем возможные флаги состояния "исчерпано"
					falseValue := false
					settings.Clients[i].Depleted = &falseValue
					settings.Clients[i].Exhausted = &falseValue
					settings.Clients[i].UpdatedAt = time.Now().UnixMilli()
					user.ClientID = client.ID
					user.SubID = client.SubID // Используем SubID из панели
					user.ExpiryTime = expiryTime
					user.HasActiveConfig = true
					log.Printf("ADD_CLIENT: Активный клиент продлён: TelegramID=%d, Email=%s, SubID=%s, ExpiryTime=%d",
						user.TelegramID, email, client.SubID, expiryTime)
					break
				}
			}
		}
	} else {
		// Клиент НЕ найден в панели - значит 3x-ui уже удалила его или он никогда не существовал
		log.Printf("ADD_CLIENT: Клиент НЕ найден в актуальном списке панели. Создание НОВОГО клиента для TelegramID=%d", user.TelegramID)
		subID := GenerateSubID()
		falseValue := false

		newClient := Client{
			ID:         clientUUID,
			Flow:       "xtls-rprx-vision",
			Email:      email,
			TotalGB:    0, // Убираем лимит трафика (0 = безлимит)
			ExpiryTime: expiryTime,
			Enable:     true,
			TgID:       0,
			SubID:      subID, // Добавляем отдельное поле SubID
			Reset:      0,     // Убираем автопродление
			Depleted:   &falseValue,
			Exhausted:  &falseValue,
			CreatedAt:  time.Now().UnixMilli(),
			UpdatedAt:  time.Now().UnixMilli(),
		}

		// Добавляем нового клиента
		settings.Clients = append(settings.Clients, newClient)

		// Обновляем данные пользователя
		user.HasActiveConfig = true
		user.ClientID = clientUUID
		user.Email = email
		user.SubID = subID
		user.ConfigCreatedAt = time.Now()
		user.ExpiryTime = expiryTime

		log.Printf("ADD_CLIENT: НОВЫЙ клиент создан (старый был удален 3x-ui): TelegramID=%d, Email=%s, SubID=%s, ExpiryTime=%d",
			user.TelegramID, email, subID, expiryTime)
	}

	// Сериализуем обновлённые settings
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		log.Printf("ADD_CLIENT: Ошибка сериализации settings: %v", err)
		return fmt.Errorf("ошибка сериализации settings: %v", err)
	}
	inbound.Settings = string(settingsJSON)

	// Обновляем inbound
	log.Printf("ADD_CLIENT: Обновление inbound для TelegramID=%d", user.TelegramID)
	err = updateInbound(sessionCookie, *inbound)
	if err != nil {
		log.Printf("ADD_CLIENT: Ошибка обновления inbound: %v", err)
		return fmt.Errorf("ошибка обновления inbound: %v", err)
	}

	// Для клиентов, которые были удалены и пересозданы, дополнительно проверяем обновление
	if actualExistingClient == nil {
		log.Printf("ADD_CLIENT: Клиент был пересоздан вместо удаленного 3x-ui. Дополнительная проверка для TelegramID=%d", user.TelegramID)
		if err := restartInbound(sessionCookie, INBOUND_ID); err != nil {
			log.Printf("ADD_CLIENT: Предупреждение - не удалось выполнить дополнительную проверку: %v", err)
			// Не возвращаем ошибку, так как основная операция уже выполнена
		}
	}

	user.ConfigsCount++
	log.Printf("ADD_CLIENT: Успешно завершено, ConfigsCount=%d", user.ConfigsCount)
	return nil
}

// UpdateInbound обновляет inbound полностью (экспортированная версия для тестирования)
func UpdateInbound(sessionCookie string, inbound Inbound) error {
	return updateInbound(sessionCookie, inbound)
}

// updateInbound обновляет inbound полностью
func updateInbound(sessionCookie string, inbound Inbound) error {
	log.Printf("UPDATE_INBOUND: Обновление inbound, ID=%d", inbound.ID)

	// Передаем полную структуру inbound, а не только ID и Settings
	jsonData, err := json.Marshal(inbound)
	if err != nil {
		log.Printf("UPDATE_INBOUND: Ошибка сериализации данных: %v", err)
		return fmt.Errorf("ошибка сериализации данных: %v", err)
	}

	req, err := http.NewRequest("POST", PANEL_URL+"panel/api/inbounds/update/"+fmt.Sprintf("%d", inbound.ID), bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("UPDATE_INBOUND: Ошибка создания запроса: %v", err)
		return fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", sessionCookie)
	log.Printf("UPDATE_INBOUND: Запрос создан, заголовки: %+v", req.Header)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("UPDATE_INBOUND: Ошибка выполнения запроса: %v", err)
		return fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("UPDATE_INBOUND: Ошибка чтения ответа: %v", err)
		return fmt.Errorf("ошибка чтения ответа: %v", err)
	}
	log.Printf("UPDATE_INBOUND: Ответ сервера: status=%d, body=%s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		log.Printf("UPDATE_INBOUND: Некорректный статус ответа: %d", resp.StatusCode)
		return fmt.Errorf("некорректный статус ответа: %d, body=%s", resp.StatusCode, string(body))
	}

	var updateResp APIResponse
	if err := json.Unmarshal(body, &updateResp); err != nil {
		log.Printf("UPDATE_INBOUND: Ошибка десериализации ответа: %v, body=%s", err, string(body))
		return fmt.Errorf("ошибка десериализации ответа: %v, body=%s", err, string(body))
	}

	if !updateResp.Success {
		log.Printf("UPDATE_INBOUND: Обновление не успешно: msg=%s", updateResp.Msg)
		return fmt.Errorf("обновление inbound не удалось: %s", updateResp.Msg)
	}

	log.Printf("UPDATE_INBOUND: Inbound успешно обновлён: ID=%d", inbound.ID)
	return nil
}

// restartInbound перезапускает inbound для сброса кэша состояний клиентов
func restartInbound(sessionCookie string, inboundID int) error {
	log.Printf("RESTART_INBOUND: Попытка перезапуска inbound ID=%d для сброса кэша", inboundID)

	// Пытаемся использовать API для перезапуска Xray или просто делаем паузу
	// так как основная логика уже обновила inbound
	time.Sleep(500 * time.Millisecond)

	// Попробуем получить inbound снова, чтобы убедиться что изменения применились
	_, err := GetInbound(sessionCookie)
	if err != nil {
		return fmt.Errorf("ошибка проверки inbound после обновления: %v", err)
	}

	log.Printf("RESTART_INBOUND: Inbound ID=%d проверен после обновления", inboundID)
	return nil
}

// FindClientByTelegramID находит клиента по префиксу TelegramID
func FindClientByTelegramID(clients []Client, telegramID int64) *Client {
	telegramIDStr := fmt.Sprintf("%d", telegramID)
	for _, client := range clients {
		if strings.HasPrefix(client.Email, telegramIDStr+"_") || strings.HasPrefix(client.Email, telegramIDStr+" ") || client.Email == telegramIDStr {
			return &client
		}
	}
	return nil
}

// AddTrialClient создает конфиг для пробного периода БЕЗ установки статуса "исчерпано"
func AddTrialClient(sessionCookie string, user *User, days int) error {
	log.Printf("ADD_TRIAL_CLIENT: Создание конфига для пробного периода TelegramID=%d, days=%d", user.TelegramID, days)

	inbound, err := GetInbound(sessionCookie)
	if err != nil {
		log.Printf("ADD_TRIAL_CLIENT: Ошибка получения inbound: %v", err)
		return fmt.Errorf("ошибка получения inbound: %v", err)
	}

	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Printf("ADD_TRIAL_CLIENT: Ошибка десериализации settings: %v", err)
		return fmt.Errorf("ошибка десериализации settings: %v", err)
	}

	email := fmt.Sprintf("%d", user.TelegramID)

	// Рассчитываем время истечения
	now := time.Now()
	expiryTime := now.Add(time.Duration(days) * 24 * time.Hour).UnixMilli()

	// Проверяем, существует ли уже клиент с таким TelegramID
	existingClient := FindClientByTelegramID(settings.Clients, user.TelegramID)

	if existingClient != nil {
		log.Printf("ADD_TRIAL_CLIENT: Клиент уже существует, обновляем для пробного периода TelegramID=%d", user.TelegramID)

		// Обновляем существующего клиента
		for i, client := range settings.Clients {
			if strings.HasPrefix(client.Email, email+"_") || strings.HasPrefix(client.Email, email+" ") || client.Email == email {
				// Обновляем данные клиента
				settings.Clients[i].ExpiryTime = expiryTime
				settings.Clients[i].Enable = true
				settings.Clients[i].TotalGB = 0 // Убираем лимит трафика
				settings.Clients[i].Reset = 0   // Убираем автопродление
				settings.Clients[i].UpdatedAt = time.Now().UnixMilli()

				// Сбрасываем статус "исчерпано"
				falseValue := false
				settings.Clients[i].Depleted = &falseValue
				settings.Clients[i].Exhausted = &falseValue

				// Обновляем данные пользователя
				user.HasActiveConfig = true
				user.ClientID = client.ID
				user.Email = email
				user.SubID = client.SubID
				user.ConfigCreatedAt = time.Now()
				user.ExpiryTime = expiryTime

				log.Printf("ADD_TRIAL_CLIENT: Существующий клиент обновлен для пробного периода: TelegramID=%d, Email=%s, SubID=%s, ExpiryTime=%d",
					user.TelegramID, email, client.SubID, expiryTime)
				break
			}
		}
	} else {
		log.Printf("ADD_TRIAL_CLIENT: Создание нового клиента для пробного периода TelegramID=%d, ExpiryTime=%d", user.TelegramID, expiryTime)

		clientUUID := uuid.New().String()
		subID := GenerateSubID()
		falseValue := false

		newClient := Client{
			ID:         clientUUID,
			Flow:       "xtls-rprx-vision",
			Email:      email,
			TotalGB:    0, // Убираем лимит трафика (0 = безлимит)
			ExpiryTime: expiryTime,
			Enable:     true,
			TgID:       0,
			SubID:      subID,
			Reset:      0,           // Убираем автопродление
			Depleted:   &falseValue, // НЕ устанавливаем статус "исчерпано"
			Exhausted:  &falseValue, // НЕ устанавливаем статус "исчерпано"
			CreatedAt:  time.Now().UnixMilli(),
			UpdatedAt:  time.Now().UnixMilli(),
		}

		// Добавляем нового клиента
		settings.Clients = append(settings.Clients, newClient)

		// Обновляем данные пользователя
		user.HasActiveConfig = true
		user.ClientID = clientUUID
		user.Email = email
		user.SubID = subID
		user.ConfigCreatedAt = time.Now()
		user.ExpiryTime = expiryTime

		log.Printf("ADD_TRIAL_CLIENT: Новый клиент для пробного периода создан: TelegramID=%d, Email=%s, SubID=%s, ExpiryTime=%d",
			user.TelegramID, email, subID, expiryTime)
	}

	// Сериализуем обновлённые settings
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		log.Printf("ADD_TRIAL_CLIENT: Ошибка сериализации settings: %v", err)
		return fmt.Errorf("ошибка сериализации settings: %v", err)
	}
	inbound.Settings = string(settingsJSON)

	// Обновляем inbound
	log.Printf("ADD_TRIAL_CLIENT: Обновление inbound для TelegramID=%d", user.TelegramID)
	err = updateInbound(sessionCookie, *inbound)
	if err != nil {
		log.Printf("ADD_TRIAL_CLIENT: Ошибка обновления inbound: %v", err)
		return fmt.Errorf("ошибка обновления inbound: %v", err)
	}

	user.ConfigsCount++
	log.Printf("ADD_TRIAL_CLIENT: Конфиг для пробного периода успешно создан, ConfigsCount=%d", user.ConfigsCount)
	return nil
}

// RemoveDuplicateClients удаляет дубликаты клиентов в панели 3x-ui
func RemoveDuplicateClients() error {
	log.Printf("REMOVE_DUPLICATES: Начало удаления дубликатов клиентов")

	// Авторизуемся в панели
	sessionCookie, err := Login()
	if err != nil {
		log.Printf("REMOVE_DUPLICATES: Ошибка авторизации: %v", err)
		return fmt.Errorf("ошибка авторизации в панели: %v", err)
	}

	// Получаем данные inbound
	inbound, err := GetInbound(sessionCookie)
	if err != nil {
		log.Printf("REMOVE_DUPLICATES: Ошибка получения данных inbound: %v", err)
		return fmt.Errorf("ошибка получения данных inbound: %v", err)
	}

	// Парсим settings
	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Printf("REMOVE_DUPLICATES: Ошибка парсинга settings: %v", err)
		return fmt.Errorf("ошибка парсинга settings: %v", err)
	}

	log.Printf("REMOVE_DUPLICATES: Найдено клиентов до очистки: %d", len(settings.Clients))

	// Создаем карту для отслеживания уникальных клиентов
	uniqueClients := make(map[string]Client)
	duplicateCount := 0

	// Проходим по всем клиентам и оставляем только уникальные
	for _, client := range settings.Clients {
		email := client.Email

		// Если клиент с таким email уже есть, проверяем какой оставить
		if existingClient, exists := uniqueClients[email]; exists {
			duplicateCount++
			log.Printf("REMOVE_DUPLICATES: Найден дубликат для email %s", email)

			// Оставляем клиента с более поздним временем создания или обновления
			// Приоритет: UpdatedAt > CreatedAt > более новый ID
			keepExisting := true

			if client.UpdatedAt > existingClient.UpdatedAt {
				keepExisting = false
			} else if client.UpdatedAt == existingClient.UpdatedAt && client.CreatedAt > existingClient.CreatedAt {
				keepExisting = false
			} else if client.UpdatedAt == existingClient.UpdatedAt && client.CreatedAt == existingClient.CreatedAt {
				// Если времена одинаковые, оставляем с более новым ID (лексикографически)
				keepExisting = client.ID < existingClient.ID
			}

			if !keepExisting {
				log.Printf("REMOVE_DUPLICATES: Заменяем клиента %s (старый: %s, новый: %s)",
					email, existingClient.ID, client.ID)
				uniqueClients[email] = client
			} else {
				log.Printf("REMOVE_DUPLICATES: Оставляем существующего клиента %s (ID: %s)",
					email, existingClient.ID)
			}
		} else {
			// Первый клиент с таким email
			uniqueClients[email] = client
		}
	}

	// Создаем новый список клиентов без дубликатов
	var cleanedClients []Client
	for _, client := range uniqueClients {
		cleanedClients = append(cleanedClients, client)
	}

	log.Printf("REMOVE_DUPLICATES: Удалено дубликатов: %d", duplicateCount)
	log.Printf("REMOVE_DUPLICATES: Клиентов после очистки: %d", len(cleanedClients))

	// Обновляем список клиентов
	settings.Clients = cleanedClients

	// Сериализуем обновлённые settings
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		log.Printf("REMOVE_DUPLICATES: Ошибка сериализации settings: %v", err)
		return fmt.Errorf("ошибка сериализации settings: %v", err)
	}
	inbound.Settings = string(settingsJSON)

	// Обновляем inbound
	log.Printf("REMOVE_DUPLICATES: Обновление inbound после удаления дубликатов")
	err = updateInbound(sessionCookie, *inbound)
	if err != nil {
		log.Printf("REMOVE_DUPLICATES: Ошибка обновления inbound: %v", err)
		return fmt.Errorf("ошибка обновления inbound: %v", err)
	}

	log.Printf("REMOVE_DUPLICATES: Дубликаты успешно удалены")
	return nil
}
