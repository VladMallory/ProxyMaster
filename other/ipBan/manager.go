package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ConfigManager управляет конфигами через API x-ui
type ConfigManager struct {
	PanelURL      string
	PanelUser     string
	PanelPass     string
	InboundID     int
	Client        *http.Client
	SessionCookie string
}

// LoginRequest представляет запрос на авторизацию
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse представляет ответ на авторизацию
type LoginResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

// XUIConfig представляет конфигурацию пользователя в x-ui
type XUIConfig struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Enabled    bool   `json:"enable"`
	TotalGB    int    `json:"totalGB"`
	ExpiryTime int64  `json:"expiryTime"`
	TgID       string `json:"tgId"` // Изменено с int64 на string
}

// Inbound представляет inbound объект
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

// Settings представляет настройки inbound
type Settings struct {
	Clients    []XUIConfig `json:"clients"`
	Decryption string      `json:"decryption"`
}

// InboundInfo представляет ответ API для получения inbound
type InboundInfo struct {
	Success bool    `json:"success"`
	Msg     string  `json:"msg"`
	Obj     Inbound `json:"obj"`
}

// NewConfigManager создает новый менеджер конфигураций
func NewConfigManager(panelURL, panelUser, panelPass string, inboundID int) *ConfigManager {
	return &ConfigManager{
		PanelURL:  panelURL,
		PanelUser: panelUser,
		PanelPass: panelPass,
		InboundID: inboundID,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Login выполняет авторизацию и получает сессионную куку
func (cm *ConfigManager) Login() error {
	loginData := LoginRequest{
		Username: cm.PanelUser,
		Password: cm.PanelPass,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return fmt.Errorf("ошибка сериализации данных авторизации: %v", err)
	}

	req, err := http.NewRequest("POST", cm.PanelURL+"login", strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := cm.Client.Do(req)
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

	var response LoginResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	if !response.Success {
		return fmt.Errorf("ошибка авторизации: %s", response.Msg)
	}

	// Извлекаем сессионную куку
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "3x-ui" {
			cm.SessionCookie = cookie.String()
			return nil
		}
	}

	return fmt.Errorf("сессионная кука не найдена в ответе")
}

// GetInbound получает полный inbound объект
func (cm *ConfigManager) GetInbound() (*Inbound, error) {
	// Если нет сессионной куки, выполняем логин
	if cm.SessionCookie == "" {
		if err := cm.Login(); err != nil {
			return nil, fmt.Errorf("ошибка авторизации: %v", err)
		}
	}

	url := fmt.Sprintf("%spanel/api/inbounds/get/%d", cm.PanelURL, cm.InboundID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	// Добавляем сессионную куку
	req.Header.Set("Cookie", cm.SessionCookie)

	resp, err := cm.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	var response InboundInfo
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("ошибка API: %s", response.Msg)
	}

	return &response.Obj, nil
}

// GetConfigs получает список всех конфигураций
func (cm *ConfigManager) GetConfigs() ([]XUIConfig, error) {
	inbound, err := cm.GetInbound()
	if err != nil {
		return nil, err
	}

	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return nil, fmt.Errorf("ошибка десериализации settings: %v", err)
	}

	return settings.Clients, nil
}

// GetConfigByEmail находит конфигурацию по email
func (cm *ConfigManager) GetConfigByEmail(email string) (*XUIConfig, error) {
	configs, err := cm.GetConfigs()
	if err != nil {
		return nil, err
	}

	for _, config := range configs {
		if config.Email == email {
			return &config, nil
		}
	}

	return nil, fmt.Errorf("конфигурация с email %s не найдена", email)
}

// EnableConfig включает конфигурацию
func (cm *ConfigManager) EnableConfig(email string) error {
	config, err := cm.GetConfigByEmail(email)
	if err != nil {
		return err
	}

	if config.Enabled {
		fmt.Printf("Конфигурация %s уже включена\n", email)
		return nil
	}

	return cm.updateConfigStatus(email, true)
}

// DisableConfig отключает конфигурацию
func (cm *ConfigManager) DisableConfig(email string) error {
	config, err := cm.GetConfigByEmail(email)
	if err != nil {
		return err
	}

	if !config.Enabled {
		fmt.Printf("Конфигурация %s уже отключена\n", email)
		return nil
	}

	return cm.updateConfigStatus(email, false)
}

// updateConfigStatus обновляет статус конфигурации
func (cm *ConfigManager) updateConfigStatus(email string, enabled bool) error {
	// Получаем текущий inbound
	inbound, err := cm.GetInbound()
	if err != nil {
		return fmt.Errorf("ошибка получения inbound: %v", err)
	}

	// Парсим настройки
	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return fmt.Errorf("ошибка десериализации settings: %v", err)
	}

	// Находим и обновляем клиента
	found := false
	for i, client := range settings.Clients {
		if client.Email == email {
			settings.Clients[i].Enabled = enabled
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("клиент с email %s не найден", email)
	}

	// Сериализуем обновленные настройки
	updatedSettings, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("ошибка сериализации settings: %v", err)
	}

	// Обновляем inbound
	inbound.Settings = string(updatedSettings)

	// Отправляем обновление
	url := fmt.Sprintf("%spanel/api/inbounds/update/%d", cm.PanelURL, cm.InboundID)

	jsonData, err := json.Marshal(inbound)
	if err != nil {
		return fmt.Errorf("ошибка сериализации inbound: %v", err)
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Cookie", cm.SessionCookie)
	req.Header.Set("Content-Type", "application/json")

	resp, err := cm.Client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		msg := "неизвестная ошибка"
		if msgVal, ok := response["msg"].(string); ok {
			msg = msgVal
		}
		return fmt.Errorf("ошибка API при обновлении конфигурации: %s", msg)
	}

	status := "включена"
	if !enabled {
		status = "отключена"
	}
	fmt.Printf("Конфигурация %s успешно %s\n", email, status)

	return nil
}

// GetConfigStatus возвращает статус конфигурации
func (cm *ConfigManager) GetConfigStatus(email string) (bool, error) {
	config, err := cm.GetConfigByEmail(email)
	if err != nil {
		return false, err
	}

	return config.Enabled, nil
}

// ListAllConfigs выводит список всех конфигураций
func (cm *ConfigManager) ListAllConfigs() error {
	configs, err := cm.GetConfigs()
	if err != nil {
		return err
	}

	fmt.Println("=== Список всех конфигураций ===")
	for _, config := range configs {
		status := "отключена"
		if config.Enabled {
			status = "включена"
		}
		fmt.Printf("Email: %s, ID: %s, Статус: %s, TotalGB: %d\n",
			config.Email, config.ID, status, config.TotalGB)
	}

	return nil
}
