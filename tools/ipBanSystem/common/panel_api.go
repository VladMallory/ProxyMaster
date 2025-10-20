package common

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
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
func (cm *ConfigManager) GetConfigs() ([]Client, error) {
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
func (cm *ConfigManager) GetConfigByEmail(email string) (*Client, error) {
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
		LogIPBanError("Ошибка получения конфига %s для включения: %v", email, err)
		return err
	}

	if config.Enable {
		fmt.Printf("Конфигурация %s уже включена\n", email)
		LogIPBanInfo("Конфиг %s уже включен", email)
		return nil
	}

	LogIPBanInfo("Включение конфига %s через API панели", email)

	err = cm.updateConfigStatus(email, true)
	if err != nil {
		LogIPBanError("Ошибка включения конфига %s через API: %v", email, err)
	} else {
		LogIPBanAction("ВКЛЮЧЕН_ЧЕРЕЗ_API", email, 0, []string{})
	}

	return err
}

// DisableConfig отключает конфигурацию
func (cm *ConfigManager) DisableConfig(email string) error {
	config, err := cm.GetConfigByEmail(email)
	if err != nil {
		LogIPBanError("Ошибка получения конфига %s для отключения: %v", email, err)
		return err
	}

	if !config.Enable {
		fmt.Printf("Конфигурация %s уже отключена\n", email)
		LogIPBanInfo("Конфиг %s уже отключен", email)
		return nil
	}

	LogIPBanInfo("Отключение конфига %s через API панели", email)

	err = cm.updateConfigStatus(email, false)
	if err != nil {
		LogIPBanError("Ошибка отключения конфига %s через API: %v", email, err)
	} else {
		LogIPBanAction("ОТКЛЮЧЕН_ЧЕРЕЗ_API", email, 0, []string{})
	}

	return err
}

// updateConfigStatus обновляет статус конфигурации
func (cm *ConfigManager) updateConfigStatus(email string, enabled bool) error {
	inbound, err := cm.GetInbound()
	if err != nil {
		return fmt.Errorf("ошибка получения inbound: %v", err)
	}

	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return fmt.Errorf("ошибка десериализации settings: %v", err)
	}

	found := false
	for i, client := range settings.Clients {
		if client.Email == email {
			settings.Clients[i].Enable = enabled
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("клиент с email %s не найден", email)
	}

	updatedSettings, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("ошибка сериализации settings: %v", err)
	}

	inbound.Settings = string(updatedSettings)

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

// DisableAndRotateConfig отключает конфиг и меняет его UUID (ID) для немедленного обрыва активных сессий
func (cm *ConfigManager) DisableAndRotateConfig(email string) (string, error) {
	inbound, err := cm.GetInbound()
	if err != nil {
		return "", fmt.Errorf("ошибка получения inbound: %v", err)
	}

	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return "", fmt.Errorf("ошибка десериализации settings: %v", err)
	}

	found := false
	newID := ""
	for i, client := range settings.Clients {
		if client.Email == email {
			settings.Clients[i].Enable = false
			newID = uuid.New().String()
			settings.Clients[i].ID = newID
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("клиент с email %s не найден", email)
	}

	updatedSettings, err := json.Marshal(settings)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации settings: %v", err)
	}

	inbound.Settings = string(updatedSettings)

	url := fmt.Sprintf("%spanel/api/inbounds/update/%d", cm.PanelURL, cm.InboundID)

	jsonData, err := json.Marshal(inbound)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации inbound: %v", err)
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Cookie", cm.SessionCookie)
	req.Header.Set("Content-Type", "application/json")

	resp, err := cm.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		msg := "неизвестная ошибка"
		if msgVal, ok := response["msg"].(string); ok {
			msg = msgVal
		}
		return "", fmt.Errorf("ошибка API при обновлении конфигурации: %s", msg)
	}

	fmt.Printf("Конфигурация %s успешно отключена и UUID обновлён\n", email)
	return newID, nil
}

// GetConfigStatus возвращает статус конфигурации
func (cm *ConfigManager) GetConfigStatus(email string) (bool, error) {
	config, err := cm.GetConfigByEmail(email)
	if err != nil {
		return false, err
	}

	return config.Enable, nil
}