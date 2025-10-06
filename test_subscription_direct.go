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

	"github.com/google/uuid"
)

// Структуры для работы с панелью 3x-ui
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

type InboundInfo struct {
	Success bool    `json:"success"`
	Msg     string  `json:"msg"`
	Obj     Inbound `json:"obj"`
}

type Inbound struct {
	ID             int         `json:"id"`
	Up             int64       `json:"up"`
	Down           int64       `json:"down"`
	Total          int64       `json:"total"`
	Remark         string      `json:"remark"`
	Enable         bool        `json:"enable"`
	ExpiryTime     int64       `json:"expiryTime"`
	Listen         string      `json:"listen"`
	Port           int         `json:"port"`
	Protocol       string      `json:"protocol"`
	Settings       string      `json:"settings"`
	StreamSettings string      `json:"streamSettings"`
	Tag            string      `json:"tag"`
	Sniffing       string      `json:"sniffing"`
	ClientStats    interface{} `json:"clientStats"`
}

type Client struct {
	ID         string `json:"id"`
	Flow       string `json:"flow"`
	Email      string `json:"email"`
	TotalGB    int    `json:"totalGB"`
	ExpiryTime int64  `json:"expiryTime"`
	Enable     bool   `json:"enable"`
	TgID       int64  `json:"tgId"`
	SubID      string `json:"subId"`
	Reset      int    `json:"reset"`
	Depleted   *bool  `json:"depleted"`
	Exhausted  *bool  `json:"exhausted"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type Settings struct {
	Clients    []Client `json:"clients"`
	Decryption string   `json:"decryption"`
	Encryption string   `json:"encryption"`
}

type APIResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

// HTTP клиент для работы с панелью 3x-ui
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

// Конфигурация для тестового скрипта
const (
	PANEL_URL  = "https://shadowfade.ru:24413/YMNUhU6HfF9PVVol2s/"
	PANEL_USER = "PKkxfWQGatttjacjVcFg7A6dKAHowxNmyCtE7PRafarnHtFanN"
	PANEL_PASS = "toJFL4atmG7xwuvXkXepjVgyMHJMK9znbNWmoM7337jCN84PVE"
	INBOUND_ID = 17 // Инбаунд 17 (обновлено)
)

func main() {
	log.Println("=== ТЕСТОВЫЙ СКРИПТ ПРЯМОГО СОЗДАНИЯ ПОДПИСКИ ===")

	// Инициализируем базу данных
	if err := common.InitPostgreSQL(); err != nil {
		log.Fatalf("Ошибка инициализации базы данных: %v", err)
	}
	defer common.DisconnectPostgreSQL()

	// Тестовый Telegram ID
	testTelegramID := int64(123456789)

	log.Printf("🧪 Тестируем создание подписки для Telegram ID: %d", testTelegramID)

	// Получаем или создаем пользователя
	user, err := common.GetOrCreateUser(
		testTelegramID,
		"test_user",
		"Test",
		"User",
	)
	if err != nil {
		log.Fatalf("Ошибка работы с пользователем: %v", err)
	}

	log.Printf("👤 Пользователь: %s (ID: %d), Баланс: %.2f₽", user.FirstName, user.TelegramID, user.Balance)

	// Создаем подписку с префиксом _1
	if err := createSubscriptionWithPrefix(user); err != nil {
		log.Fatalf("Ошибка создания подписки: %v", err)
	}

	log.Println("✅ Тест завершен успешно!")
}

// createSubscriptionWithPrefix создает подписку с префиксом _1
func createSubscriptionWithPrefix(user *common.User) error {
	log.Printf("🔧 Создание подписки с префиксом _1 для пользователя %d", user.TelegramID)

	// Авторизуемся в панели
	sessionCookie, err := loginToPanel()
	if err != nil {
		return fmt.Errorf("ошибка авторизации в панели: %v", err)
	}

	// Сначала проверим доступные инбаунды
	log.Printf("🔍 Проверяем доступные инбаунды...")
	availableInbounds, err := getAllInbounds(sessionCookie)
	if err != nil {
		log.Printf("⚠️ Не удалось получить список инбаундов: %v", err)
	} else {
		log.Printf("📋 Найдено инбаундов: %d", len(availableInbounds))
		for _, inbound := range availableInbounds {
			log.Printf("   🆔 Inbound ID: %d", inbound.ID)
		}
	}

	// Получаем inbound
	inbound, err := getInbound(sessionCookie)
	if err != nil {
		return fmt.Errorf("ошибка получения inbound: %v", err)
	}

	// Парсим settings
	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return fmt.Errorf("ошибка парсинга settings: %v", err)
	}

	log.Printf("📋 Анализ текущего inbound:")
	log.Printf("   🆔 Inbound ID: %d", inbound.ID)
	log.Printf("   📡 Порт: %d", inbound.Port)
	log.Printf("   🔧 Протокол: %s", inbound.Protocol)
	log.Printf("   🏷️ Tag: %s", inbound.Tag)
	log.Printf("   🌐 StreamSettings: %s", inbound.StreamSettings)
	log.Printf("   📊 Количество клиентов: %d", len(settings.Clients))
	log.Printf("   ⚙️ Settings JSON размер: %d байт", len(inbound.Settings))
	log.Printf("   🔧 Полные настройки inbound:")
	log.Printf("      📄 Settings: %s", inbound.Settings)

	// Анализируем структуру settings более детально
	log.Printf("   🔍 Детальный анализ settings:")
	log.Printf("      🔐 decryption: %s", settings.Decryption)
	log.Printf("      🔐 encryption: %s", settings.Encryption)
	log.Printf("      📊 Количество клиентов в settings: %d", len(settings.Clients))

	// Показываем первые несколько клиентов для анализа
	for i, client := range settings.Clients {
		if i < 3 { // Показываем только первых 3 клиентов
			log.Printf("   👤 Клиент %d: Email=%s, Flow='%s', Enable=%v, ExpiryTime=%d",
				i+1, client.Email, client.Flow, client.Enable, client.ExpiryTime)
		}
	}

	// Генерируем email с префиксом _1
	email := fmt.Sprintf("%d_1", user.TelegramID)
	clientUUID := uuid.New().String()
	subID := generateSubID()

	// Устанавливаем срок действия на 30 дней
	expiryTime := time.Now().AddDate(0, 0, 30).UnixMilli()
	falseValue := false

	// Создаем нового клиента с правильными настройками для VLESS
	newClient := Client{
		ID:         clientUUID,
		Flow:       "", // Убираем flow как указано
		Email:      email,
		TotalGB:    0, // Безлимитный трафик
		ExpiryTime: expiryTime,
		Enable:     true,
		TgID:       0,
		SubID:      subID,
		Reset:      0,
		Depleted:   &falseValue,
		Exhausted:  &falseValue,
		CreatedAt:  time.Now().UnixMilli(),
		UpdatedAt:  time.Now().UnixMilli(),
	}

	log.Printf("🔧 Создаем клиента с настройками:")
	log.Printf("   📧 Email: %s", email)
	log.Printf("   🔑 SubID: %s", subID)
	log.Printf("   🆔 ClientID: %s", clientUUID)
	log.Printf("   🌊 Flow: '%s' (пустой как указано)", newClient.Flow)
	log.Printf("   ⏰ ExpiryTime: %d (%s)", expiryTime, time.UnixMilli(expiryTime).Format("2006-01-02 15:04:05"))
	log.Printf("   ✅ Enable: %v", newClient.Enable)
	log.Printf("   📊 TotalGB: %d (0 = безлимит)", newClient.TotalGB)

	// Добавляем клиента в список
	settings.Clients = append(settings.Clients, newClient)

	// Обновляем данные пользователя
	user.HasActiveConfig = true
	user.ClientID = clientUUID
	user.Email = email
	user.SubID = subID
	user.ConfigCreatedAt = time.Now()
	user.ExpiryTime = expiryTime

	// Сериализуем обновленные settings
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("ошибка сериализации settings: %v", err)
	}
	inbound.Settings = string(settingsJSON)

	log.Printf("📤 Отправляем обновленные настройки в панель:")
	log.Printf("   📋 Количество клиентов: %d", len(settings.Clients))
	log.Printf("   🔧 JSON размер: %d байт", len(settingsJSON))
	log.Printf("   📡 Inbound ID: %d", inbound.ID)

	// Обновляем inbound в панели
	if err := updateInbound(sessionCookie, inbound); err != nil {
		log.Printf("❌ Ошибка обновления inbound: %v", err)
		return fmt.Errorf("ошибка обновления inbound: %v", err)
	}

	log.Printf("✅ Inbound успешно обновлен в панели")

	// Анализируем ответ от панели после обновления
	log.Printf("🔍 Анализ ответа панели после обновления:")
	log.Printf("   📊 Статус: success")
	log.Printf("   💬 Сообщение: Inbound has been successfully updated.")
	log.Printf("   ⚠️ ВАЖНО: Проверяем, что не потерялись настройки транспорта!")

	// Проверяем, что клиент действительно добавился
	log.Printf("🔍 Проверяем созданного клиента...")

	// Получаем обновленный inbound для проверки
	updatedInbound, err := getInbound(sessionCookie)
	if err != nil {
		log.Printf("⚠️ Не удалось получить обновленный inbound для проверки: %v", err)
	} else {
		log.Printf("🔍 Анализ обновленного inbound:")
		log.Printf("   🆔 ID: %d", updatedInbound.ID)
		log.Printf("   📡 Порт: %d", updatedInbound.Port)
		log.Printf("   🔧 Протокол: %s", updatedInbound.Protocol)
		log.Printf("   🏷️ Tag: %s", updatedInbound.Tag)
		log.Printf("   🌐 StreamSettings: %s", updatedInbound.StreamSettings)
		log.Printf("   📄 Settings размер: %d байт", len(updatedInbound.Settings))
		log.Printf("   📄 Полные settings: %s", updatedInbound.Settings)

		var updatedSettings Settings
		if err := json.Unmarshal([]byte(updatedInbound.Settings), &updatedSettings); err != nil {
			log.Printf("⚠️ Ошибка парсинга обновленных settings: %v", err)
		} else {
			log.Printf("   🔍 Детали обновленных settings:")
			log.Printf("      🔐 decryption: %s", updatedSettings.Decryption)
			log.Printf("      🔐 encryption: %s", updatedSettings.Encryption)
			log.Printf("      📊 Количество клиентов: %d", len(updatedSettings.Clients))

			// Ищем нашего клиента
			found := false
			for i, client := range updatedSettings.Clients {
				log.Printf("      👤 Клиент %d: Email=%s, Flow='%s', Enable=%v",
					i+1, client.Email, client.Flow, client.Enable)
				if client.Email == email {
					log.Printf("✅ НАШ КЛИЕНТ найден в панели:")
					log.Printf("   📧 Email: %s", client.Email)
					log.Printf("   🔑 SubID: %s", client.SubID)
					log.Printf("   🌊 Flow: '%s'", client.Flow)
					log.Printf("   ✅ Enable: %v", client.Enable)
					log.Printf("   ⏰ ExpiryTime: %d (%s)", client.ExpiryTime, time.UnixMilli(client.ExpiryTime).Format("2006-01-02 15:04:05"))
					found = true
					break
				}
			}
			if !found {
				log.Printf("❌ Клиент НЕ найден в обновленном inbound!")
			}
		}
	}

	// Сохраняем пользователя в базу данных
	if err := common.UpdateUser(user); err != nil {
		log.Printf("⚠️ Ошибка сохранения пользователя в БД: %v", err)
		// Не возвращаем ошибку, так как подписка уже создана в панели
	} else {
		log.Printf("✅ Пользователь сохранен в БД")
	}

	log.Printf("✅ Подписка создана: Email=%s, SubID=%s, ExpiryTime=%d",
		email, subID, expiryTime)

	return nil
}

// loginToPanel выполняет авторизацию в панели 3x-ui
func loginToPanel() (string, error) {
	log.Printf("🔐 Авторизация в панели:")
	log.Printf("   🔗 URL: %s", PANEL_URL)
	log.Printf("   👤 Username: %s", PANEL_USER)
	log.Printf("   🔑 Password: %s", PANEL_PASS[:10]+"...")

	loginData := LoginRequest{
		Username: PANEL_USER,
		Password: PANEL_PASS,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		log.Printf("❌ Ошибка сериализации данных авторизации: %v", err)
		return "", fmt.Errorf("ошибка сериализации данных авторизации: %v", err)
	}

	log.Printf("📤 Отправляем данные авторизации (%d байт)", len(jsonData))

	req, err := http.NewRequest("POST", PANEL_URL+"login", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ Ошибка создания HTTP запроса: %v", err)
		return "", fmt.Errorf("ошибка создания запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	log.Printf("📡 Выполняем запрос авторизации...")
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("❌ Ошибка выполнения HTTP запроса: %v", err)
		return "", fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Ошибка чтения ответа: %v", err)
		return "", fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	log.Printf("📥 Получен ответ авторизации:")
	log.Printf("   📊 Статус код: %d", resp.StatusCode)
	log.Printf("   📄 Размер ответа: %d байт", len(body))
	log.Printf("   📝 Тело ответа: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Некорректный статус ответа: %d", resp.StatusCode)
		return "", fmt.Errorf("некорректный статус ответа: %d, body=%s", resp.StatusCode, string(body))
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		log.Printf("❌ Ошибка парсинга JSON ответа: %v", err)
		return "", fmt.Errorf("ошибка десериализации ответа: %v", err)
	}

	log.Printf("📋 Результат авторизации:")
	log.Printf("   ✅ Success: %v", loginResp.Success)
	log.Printf("   💬 Message: %s", loginResp.Msg)

	if !loginResp.Success {
		log.Printf("❌ Авторизация не удалась: %s", loginResp.Msg)
		return "", fmt.Errorf("авторизация не удалась: %s", loginResp.Msg)
	}

	// Извлекаем куку сессии
	log.Printf("🍪 Ищем куку сессии в заголовках...")
	for i, cookie := range resp.Header.Values("Set-Cookie") {
		log.Printf("   Cookie %d: %s", i+1, cookie)
		if contains(cookie, "3x-ui=") {
			sessionCookie := split(cookie, ";")[0]
			log.Printf("✅ Найдена кука сессии: %s", sessionCookie)
			return sessionCookie, nil
		}
	}

	log.Printf("❌ Кука сессии не найдена в заголовках")
	return "", fmt.Errorf("кука сессии не найдена")
}

// getAllInbounds получает список всех инбаундов
func getAllInbounds(sessionCookie string) ([]Inbound, error) {
	log.Printf("📡 Получение списка всех инбаундов")
	url := PANEL_URL + "panel/api/inbounds"
	log.Printf("   🔗 URL: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("❌ Ошибка создания HTTP запроса: %v", err)
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Cookie", sessionCookie)
	log.Printf("   🍪 Cookie: %s", sessionCookie[:50]+"...")

	log.Printf("📡 Выполняем запрос получения списка инбаундов...")
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("❌ Ошибка выполнения HTTP запроса: %v", err)
		return nil, fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Ошибка чтения ответа: %v", err)
		return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	log.Printf("📥 Получен ответ от панели:")
	log.Printf("   📊 Статус код: %d", resp.StatusCode)
	log.Printf("   📄 Размер ответа: %d байт", len(body))
	log.Printf("   📝 Тело ответа: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Некорректный статус ответа: %d", resp.StatusCode)
		return nil, fmt.Errorf("некорректный статус ответа: %d, body=%s", resp.StatusCode, string(body))
	}

	var response struct {
		Success bool      `json:"success"`
		Msg     string    `json:"msg"`
		Obj     []Inbound `json:"obj"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("❌ Ошибка парсинга JSON ответа: %v", err)
		return nil, fmt.Errorf("ошибка парсинга ответа: %v", err)
	}

	log.Printf("📋 Результат получения списка инбаундов:")
	log.Printf("   ✅ Success: %v", response.Success)
	log.Printf("   💬 Message: %s", response.Msg)

	if !response.Success {
		log.Printf("❌ Получение списка инбаундов не удалось: %s", response.Msg)
		return nil, fmt.Errorf("неудачное получение списка инбаундов: %s", response.Msg)
	}

	log.Printf("✅ Получено %d инбаундов", len(response.Obj))
	return response.Obj, nil
}

// getInbound получает inbound по ID
func getInbound(sessionCookie string) (*Inbound, error) {
	log.Printf("📡 Получение inbound ID=%d", INBOUND_ID)
	url := fmt.Sprintf("%spanel/api/inbounds/get/%d", PANEL_URL, INBOUND_ID)
	log.Printf("   🔗 URL: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("❌ Ошибка создания HTTP запроса: %v", err)
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Cookie", sessionCookie)
	log.Printf("   🍪 Cookie: %s", sessionCookie[:50]+"...")

	log.Printf("📡 Выполняем запрос получения inbound...")
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("❌ Ошибка выполнения HTTP запроса: %v", err)
		return nil, fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Ошибка чтения ответа: %v", err)
		return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	log.Printf("📥 Получен ответ от панели:")
	log.Printf("   📊 Статус код: %d", resp.StatusCode)
	log.Printf("   📄 Размер ответа: %d байт", len(body))
	log.Printf("   📝 Тело ответа: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Некорректный статус ответа: %d", resp.StatusCode)
		return nil, fmt.Errorf("некорректный статус ответа: %d, body=%s", resp.StatusCode, string(body))
	}

	var inboundInfo InboundInfo
	if err := json.Unmarshal(body, &inboundInfo); err != nil {
		log.Printf("❌ Ошибка парсинга JSON ответа: %v", err)
		return nil, fmt.Errorf("ошибка десериализации ответа: %v", err)
	}

	log.Printf("📋 Результат получения inbound:")
	log.Printf("   ✅ Success: %v", inboundInfo.Success)
	log.Printf("   💬 Message: %s", inboundInfo.Msg)

	if !inboundInfo.Success {
		log.Printf("❌ Получение inbound не удалось: %s", inboundInfo.Msg)
		return nil, fmt.Errorf("получение inbound не удалось: %s", inboundInfo.Msg)
	}

	log.Printf("✅ Успешно получен inbound: ID=%d", inboundInfo.Obj.ID)
	return &inboundInfo.Obj, nil
}

// updateInbound обновляет inbound в панели
func updateInbound(sessionCookie string, inbound *Inbound) error {
	log.Printf("🔄 Обновление inbound ID=%d", inbound.ID)

	jsonData, err := json.Marshal(inbound)
	if err != nil {
		log.Printf("❌ Ошибка сериализации inbound данных: %v", err)
		return fmt.Errorf("ошибка сериализации данных: %v", err)
	}

	log.Printf("📤 Отправляем JSON данные (%d байт):", len(jsonData))
	log.Printf("   🔗 URL: %s", PANEL_URL+"panel/api/inbounds/update/"+fmt.Sprintf("%d", inbound.ID))
	log.Printf("   🍪 Cookie: %s", sessionCookie[:50]+"...")

	req, err := http.NewRequest("POST", PANEL_URL+"panel/api/inbounds/update/"+fmt.Sprintf("%d", inbound.ID), bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ Ошибка создания HTTP запроса: %v", err)
		return fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", sessionCookie)

	log.Printf("📡 Выполняем HTTP запрос...")
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("❌ Ошибка выполнения HTTP запроса: %v", err)
		return fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Ошибка чтения ответа: %v", err)
		return fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	log.Printf("📥 Получен ответ от панели:")
	log.Printf("   📊 Статус код: %d", resp.StatusCode)
	log.Printf("   📄 Размер ответа: %d байт", len(body))
	log.Printf("   📝 Тело ответа: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Некорректный статус ответа: %d", resp.StatusCode)
		return fmt.Errorf("некорректный статус ответа: %d, body=%s", resp.StatusCode, string(body))
	}

	var updateResp APIResponse
	if err := json.Unmarshal(body, &updateResp); err != nil {
		log.Printf("❌ Ошибка парсинга JSON ответа: %v", err)
		return fmt.Errorf("ошибка десериализации ответа: %v", err)
	}

	log.Printf("📋 Ответ панели:")
	log.Printf("   ✅ Success: %v", updateResp.Success)
	log.Printf("   💬 Message: %s", updateResp.Msg)

	if !updateResp.Success {
		log.Printf("❌ Обновление inbound не удалось: %s", updateResp.Msg)
		return fmt.Errorf("обновление inbound не удалось: %s", updateResp.Msg)
	}

	log.Printf("✅ Inbound успешно обновлен: ID=%d", inbound.ID)
	return nil
}

// generateSubID генерирует уникальный SubID
func generateSubID() string {
	return uuid.New().String()
}

// Вспомогательные функции
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

func split(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}
