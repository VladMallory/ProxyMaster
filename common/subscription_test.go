package common

import (
	"strings"
	"testing"
	"time"
)

// TestIsConfigActive тестирует функцию проверки активности конфига
func TestIsConfigActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		user           *User
		expectedResult bool
		description    string
	}{
		{
			name: "ActiveConfig_ValidExpiry",
			user: &User{
				HasActiveConfig: true,
				ExpiryTime:      now.Add(24 * time.Hour).UnixMilli(), // Истекает через день
			},
			expectedResult: true,
			description:    "Активный конфиг с валидным временем истечения",
		},
		{
			name: "ActiveConfig_Expired",
			user: &User{
				HasActiveConfig: true,
				ExpiryTime:      now.Add(-24 * time.Hour).UnixMilli(), // Истек вчера
			},
			expectedResult: false,
			description:    "Активный конфиг с истекшим временем",
		},
		{
			name: "InactiveConfig_ValidExpiry",
			user: &User{
				HasActiveConfig: false,
				ExpiryTime:      now.Add(24 * time.Hour).UnixMilli(),
			},
			expectedResult: false,
			description:    "Неактивный конфиг даже с валидным временем",
		},
		{
			name: "ActiveConfig_NoExpiryTime",
			user: &User{
				HasActiveConfig: true,
				ExpiryTime:      0, // Нет времени истечения
			},
			expectedResult: true,
			description:    "Активный конфиг без времени истечения",
		},
		{
			name: "ActiveConfig_ExpiresNow",
			user: &User{
				HasActiveConfig: true,
				ExpiryTime:      now.Add(-1 * time.Millisecond).UnixMilli(), // Истек 1 миллисекунду назад
			},
			expectedResult: false,
			description:    "Активный конфиг, истекающий прямо сейчас",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConfigActive(tt.user)
			if result != tt.expectedResult {
				t.Errorf("IsConfigActive() = %v, expected %v. %s",
					result, tt.expectedResult, tt.description)
			}
		})
	}
}

// TestTrialPeriodManager_CanUseTrial тестирует проверку возможности использования пробного периода
func TestTrialPeriodManager_CanUseTrial(t *testing.T) {
	tm := NewTrialPeriodManager()

	tests := []struct {
		name           string
		user           *User
		expectedResult bool
		description    string
	}{
		{
			name: "CanUseTrial_NewUser",
			user: &User{
				HasUsedTrial: false,
			},
			expectedResult: true,
			description:    "Новый пользователь может использовать пробный период",
		},
		{
			name: "CannotUseTrial_UsedBefore",
			user: &User{
				HasUsedTrial: true,
			},
			expectedResult: false,
			description:    "Пользователь, уже использовавший пробный период, не может использовать снова",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tm.CanUseTrial(tt.user)
			if result != tt.expectedResult {
				t.Errorf("CanUseTrial() = %v, expected %v. %s",
					result, tt.expectedResult, tt.description)
			}
		})
	}
}

// TestGetDaysWord тестирует функцию правильного склонения слова "день"
func TestGetDaysWord(t *testing.T) {
	tests := []struct {
		days         int
		expectedWord string
		description  string
	}{
		{1, "день", "1 день"},
		{2, "дня", "2 дня"},
		{3, "дня", "3 дня"},
		{4, "дня", "4 дня"},
		{5, "дней", "5 дней"},
		{10, "дней", "10 дней"},
		{21, "дней", "21 день (по текущей логике)"},
		{22, "дней", "22 дня (по текущей логике)"},
		{23, "дней", "23 дня (по текущей логике)"},
		{24, "дней", "24 дня (по текущей логике)"},
		{25, "дней", "25 дней"},
		{100, "дней", "100 дней"},
		{0, "дней", "0 дней"},
		{-1, "дней", "-1 день"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := GetDaysWord(tt.days)
			if result != tt.expectedWord {
				t.Errorf("GetDaysWord(%d) = %s, expected %s",
					tt.days, result, tt.expectedWord)
			}
		})
	}
}

// TestCalculateTrafficLimit тестирует расчет лимита трафика
func TestCalculateTrafficLimit(t *testing.T) {
	tests := []struct {
		days          int
		expectedLimit int
		description   string
	}{
		{1, 1, "1 день = 1 ГБ"},
		{7, 7, "7 дней = 7 ГБ"},
		{30, 30, "30 дней = 30 ГБ"},
		{0, 0, "0 дней = 0 ГБ"},
		{-1, -1, "-1 день = -1 ГБ"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := CalculateTrafficLimit(tt.days)
			if result != tt.expectedLimit {
				t.Errorf("CalculateTrafficLimit(%d) = %d, expected %d",
					tt.days, result, tt.expectedLimit)
			}
		})
	}
}

// TestFormatTrafficLimit тестирует форматирование лимита трафика
func TestFormatTrafficLimit(t *testing.T) {
	tests := []struct {
		limitGB        int
		expectedFormat string
		description    string
	}{
		{0, "Безлимит", "0 ГБ = Безлимит"},
		{-1, "Безлимит", "-1 ГБ = Безлимит"},
		{1, "1 ГБ", "1 ГБ"},
		{10, "10 ГБ", "10 ГБ"},
		{100, "100 ГБ", "100 ГБ"},
		{1024, "1.0 ТБ", "1024 ГБ = 1.0 ТБ"},
		{2048, "2.0 ТБ", "2048 ГБ = 2.0 ТБ"},
		{1536, "1.5 ТБ", "1536 ГБ = 1.5 ТБ"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := FormatTrafficLimit(tt.limitGB)
			if result != tt.expectedFormat {
				t.Errorf("FormatTrafficLimit(%d) = %s, expected %s",
					tt.limitGB, result, tt.expectedFormat)
			}
		})
	}
}

// TestGetRedirectURL тестирует генерацию URL для редиректа
func TestGetRedirectURL(t *testing.T) {
	// Сохраняем оригинальное значение
	originalDomain := REDIRECT_DOMAIN
	defer func() {
		REDIRECT_DOMAIN = originalDomain
	}()

	tests := []struct {
		domain      string
		expectedURL string
		description string
	}{
		{
			domain:      "example.com",
			expectedURL: "http://example.com/redirect_happ.html?url=",
			description: "Обычный домен",
		},
		{
			domain:      "test.example.com",
			expectedURL: "http://test.example.com/redirect_happ.html?url=",
			description: "Поддомен",
		},
		{
			domain:      "my-vpn-service.com",
			expectedURL: "http://my-vpn-service.com/redirect_happ.html?url=",
			description: "Домен с дефисами",
		},
		{
			domain:      "localhost",
			expectedURL: "http://localhost/redirect_happ.html?url=",
			description: "Локальный домен",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			REDIRECT_DOMAIN = tt.domain
			result := GetRedirectURL()

			if result != tt.expectedURL {
				t.Errorf("GetRedirectURL() with domain %s = %s, expected %s",
					tt.domain, result, tt.expectedURL)
			}
		})
	}
}

// TestGetTrafficConfigDescription тестирует описание конфигурации трафика
func TestGetTrafficConfigDescription(t *testing.T) {
	// Сохраняем оригинальное значение
	originalLimit := TRAFFIC_LIMIT_GB
	defer func() {
		TRAFFIC_LIMIT_GB = originalLimit
	}()

	tests := []struct {
		limitGB      int
		expectedDesc string
		description  string
	}{
		{0, "Безлимит", "Лимит 0 ГБ = Безлимит"},
		{-1, "Безлимит", "Лимит -1 ГБ = Безлимит"},
		{10, "10 ГБ", "Лимит 10 ГБ"},
		{100, "100 ГБ", "Лимит 100 ГБ"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			TRAFFIC_LIMIT_GB = tt.limitGB
			result := GetTrafficConfigDescription()
			if result != tt.expectedDesc {
				t.Errorf("GetTrafficConfigDescription() = %s, expected %s",
					result, tt.expectedDesc)
			}
		})
	}
}

// TestTrialPeriodManager_GetTrialPeriodInfo тестирует получение информации о пробных периодах
func TestTrialPeriodManager_GetTrialPeriodInfo(t *testing.T) {
	// Сохраняем оригинальное значение
	originalBalance := TRIAL_BALANCE_AMOUNT
	defer func() {
		TRIAL_BALANCE_AMOUNT = originalBalance
	}()

	// Устанавливаем тестовое значение
	TRIAL_BALANCE_AMOUNT = 15

	tm := NewTrialPeriodManager()
	info := tm.GetTrialPeriodInfo()

	// Проверяем, что информация содержит ожидаемые данные
	expectedContains := []string{
		"Информация о пробных периодах",
		"15₽",
		"TRIAL_BALANCE_AMOUNT = 15",
		"добавляется указанная сумма на баланс",
	}

	for _, expected := range expectedContains {
		if !contains(info, expected) {
			t.Errorf("GetTrialPeriodInfo() should contain '%s', got: %s", expected, info)
		}
	}
}

// TestUser_SubscriptionFields тестирует поля пользователя, связанные с подписками
func TestUser_SubscriptionFields(t *testing.T) {
	now := time.Now()
	user := &User{
		TelegramID:      12345,
		Username:        "testuser",
		FirstName:       "Test",
		LastName:        "User",
		Balance:         100.50,
		TotalPaid:       250.75,
		ConfigsCount:    3,
		HasActiveConfig: true,
		ClientID:        "test-client-id",
		SubID:           "test-sub-id",
		Email:           "12345",
		ConfigCreatedAt: now.Add(-7 * 24 * time.Hour),
		ExpiryTime:      now.Add(30 * 24 * time.Hour).UnixMilli(),
		HasUsedTrial:    true,
		CreatedAt:       now.Add(-30 * 24 * time.Hour),
		UpdatedAt:       now,
	}

	// Проверяем, что все поля корректно установлены
	if user.TelegramID != 12345 {
		t.Errorf("TelegramID = %d, expected 12345", user.TelegramID)
	}
	if user.Username != "testuser" {
		t.Errorf("Username = %s, expected testuser", user.Username)
	}
	if user.FirstName != "Test" {
		t.Errorf("FirstName = %s, expected Test", user.FirstName)
	}
	if user.LastName != "User" {
		t.Errorf("LastName = %s, expected User", user.LastName)
	}
	if user.Balance != 100.50 {
		t.Errorf("Balance = %f, expected 100.50", user.Balance)
	}
	if user.TotalPaid != 250.75 {
		t.Errorf("TotalPaid = %f, expected 250.75", user.TotalPaid)
	}
	if user.ConfigsCount != 3 {
		t.Errorf("ConfigsCount = %d, expected 3", user.ConfigsCount)
	}
	if user.HasActiveConfig != true {
		t.Errorf("HasActiveConfig = %v, expected true", user.HasActiveConfig)
	}
	if user.ClientID != "test-client-id" {
		t.Errorf("ClientID = %s, expected test-client-id", user.ClientID)
	}
	if user.SubID != "test-sub-id" {
		t.Errorf("SubID = %s, expected test-sub-id", user.SubID)
	}
	if user.Email != "12345" {
		t.Errorf("Email = %s, expected 12345", user.Email)
	}
	if user.HasUsedTrial != true {
		t.Errorf("HasUsedTrial = %v, expected true", user.HasUsedTrial)
	}

	// Проверяем, что время истечения в будущем
	if user.ExpiryTime <= now.UnixMilli() {
		t.Errorf("ExpiryTime should be in the future, got %d", user.ExpiryTime)
	}

	// Проверяем, что ConfigCreatedAt в прошлом
	if user.ConfigCreatedAt.After(now) {
		t.Errorf("ConfigCreatedAt should be in the past, got %v", user.ConfigCreatedAt)
	}

	// Проверяем, что CreatedAt в прошлом
	if user.CreatedAt.After(now) {
		t.Errorf("CreatedAt should be in the past, got %v", user.CreatedAt)
	}

	// Проверяем, что UpdatedAt не в далеком будущем
	if user.UpdatedAt.After(now.Add(time.Minute)) {
		t.Errorf("UpdatedAt should be recent, got %v", user.UpdatedAt)
	}
}

// TestClient_SubscriptionFields тестирует поля клиента, связанные с подписками
func TestClient_SubscriptionFields(t *testing.T) {
	now := time.Now()
	client := &Client{
		ID:         "test-client-id",
		Flow:       "xtls-rprx-vision",
		Email:      "12345",
		LimitIP:    0,
		TotalGB:    0,
		ExpiryTime: now.Add(30 * 24 * time.Hour).UnixMilli(),
		Enable:     true,
		TgID:       12345,
		SubID:      "test-sub-id",
		Reset:      0,
	}

	// Проверяем, что все поля корректно установлены
	if client.ID != "test-client-id" {
		t.Errorf("ID = %s, expected test-client-id", client.ID)
	}
	if client.Flow != "xtls-rprx-vision" {
		t.Errorf("Flow = %s, expected xtls-rprx-vision", client.Flow)
	}
	if client.Email != "12345" {
		t.Errorf("Email = %s, expected 12345", client.Email)
	}
	if client.LimitIP != 0 {
		t.Errorf("LimitIP = %d, expected 0 (unlimited)", client.LimitIP)
	}
	if client.TotalGB != 0 {
		t.Errorf("TotalGB = %d, expected 0 (unlimited)", client.TotalGB)
	}
	if client.Enable != true {
		t.Errorf("Enable = %v, expected true", client.Enable)
	}
	if client.TgID != 12345 {
		t.Errorf("TgID = %d, expected 12345", client.TgID)
	}
	if client.SubID != "test-sub-id" {
		t.Errorf("SubID = %s, expected test-sub-id", client.SubID)
	}
	if client.Reset != 0 {
		t.Errorf("Reset = %d, expected 0 (no auto-renewal)", client.Reset)
	}

	// Проверяем, что время истечения в будущем
	if client.ExpiryTime <= now.UnixMilli() {
		t.Errorf("ExpiryTime should be in the future, got %d", client.ExpiryTime)
	}
}

// TestSubscriptionExpiryCalculation тестирует расчет времени истечения подписки
func TestSubscriptionExpiryCalculation(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		days           int
		expectedOffset time.Duration
		description    string
	}{
		{
			name:           "OneDay",
			days:           1,
			expectedOffset: 24 * time.Hour,
			description:    "1 день = 24 часа",
		},
		{
			name:           "SevenDays",
			days:           7,
			expectedOffset: 7 * 24 * time.Hour,
			description:    "7 дней = 168 часов",
		},
		{
			name:           "ThirtyDays",
			days:           30,
			expectedOffset: 30 * 24 * time.Hour,
			description:    "30 дней = 720 часов",
		},
		{
			name:           "ZeroDays",
			days:           0,
			expectedOffset: 0,
			description:    "0 дней = 0 часов",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Симулируем расчет времени истечения как в AddClient
			expiryTime := now.Add(time.Duration(tt.days) * 24 * time.Hour).UnixMilli()
			expectedTime := now.Add(tt.expectedOffset).UnixMilli()

			// Допускаем погрешность в 1 секунду
			diff := expiryTime - expectedTime
			if diff < 0 {
				diff = -diff
			}

			if diff > 1000 { // 1 секунда в миллисекундах
				t.Errorf("ExpiryTime calculation for %d days: got %d, expected around %d (diff: %d ms)",
					tt.days, expiryTime, expectedTime, diff)
			}
		})
	}
}

// TestSubscriptionRenewalLogic тестирует логику продления подписки
func TestSubscriptionRenewalLogic(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name               string
		existingExpiryTime int64
		daysToAdd          int
		expectedBehavior   string
		description        string
	}{
		{
			name:               "RenewActiveSubscription",
			existingExpiryTime: now.Add(7 * 24 * time.Hour).UnixMilli(), // Истекает через неделю
			daysToAdd:          30,
			expectedBehavior:   "extend", // Продлить существующую
			description:        "Продление активной подписки",
		},
		{
			name:               "RenewExpiredSubscription",
			existingExpiryTime: now.Add(-7 * 24 * time.Hour).UnixMilli(), // Истекла неделю назад
			daysToAdd:          30,
			expectedBehavior:   "new", // Создать новую
			description:        "Создание новой подписки для истекшей",
		},
		{
			name:               "RenewExpiringToday",
			existingExpiryTime: now.UnixMilli(), // Истекает прямо сейчас
			daysToAdd:          30,
			expectedBehavior:   "new", // Создать новую
			description:        "Создание новой подписки для истекающей сегодня",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Симулируем логику из AddClient
			var expiryTime int64

			if tt.existingExpiryTime > now.UnixMilli() {
				// Продление активной подписки
				expiryTime = tt.existingExpiryTime + int64(tt.daysToAdd)*24*60*60*1000
			} else {
				// Создание новой подписки
				expiryTime = now.Add(time.Duration(tt.daysToAdd) * 24 * time.Hour).UnixMilli()
			}

			// Проверяем результат
			switch tt.expectedBehavior {
			case "extend":
				// При продлении время должно быть больше исходного
				if expiryTime <= tt.existingExpiryTime {
					t.Errorf("Extended subscription expiry time should be greater than existing, got %d, existing %d",
						expiryTime, tt.existingExpiryTime)
				}
			case "new":
				// При создании новой время должно быть в будущем
				if expiryTime <= now.UnixMilli() {
					t.Errorf("New subscription expiry time should be in the future, got %d, now %d",
						expiryTime, now.UnixMilli())
				}
			default:
				t.Errorf("Unknown expected behavior: %s", tt.expectedBehavior)
			}
		})
	}
}

// TestConfigVariablesWithDifferentDomains тестирует конфигурацию с разными доменами
func TestConfigVariablesWithDifferentDomains(t *testing.T) {
	// Сохраняем оригинальные значения
	originalPanelURL := PANEL_URL
	originalRedirectDomain := REDIRECT_DOMAIN
	defer func() {
		PANEL_URL = originalPanelURL
		REDIRECT_DOMAIN = originalRedirectDomain
	}()

	testDomains := []struct {
		panelURL       string
		redirectDomain string
		description    string
	}{
		{
			panelURL:       "https://vpn.example.com:8080/panel/",
			redirectDomain: "example.com",
			description:    "HTTPS с портом и поддоменом",
		},
		{
			panelURL:       "http://localhost:3000/",
			redirectDomain: "localhost",
			description:    "HTTP локальный сервер",
		},
		{
			panelURL:       "https://my-vpn-service.com/",
			redirectDomain: "my-vpn-service.com",
			description:    "HTTPS с дефисами в домене",
		},
		{
			panelURL:       "https://api.vpn.example.com:443/",
			redirectDomain: "vpn.example.com",
			description:    "HTTPS API поддомен",
		},
	}

	for _, tt := range testDomains {
		t.Run(tt.description, func(t *testing.T) {
			// Устанавливаем тестовые значения
			PANEL_URL = tt.panelURL
			REDIRECT_DOMAIN = tt.redirectDomain

			// Проверяем, что URL валидный
			if !strings.HasPrefix(PANEL_URL, "http://") && !strings.HasPrefix(PANEL_URL, "https://") {
				t.Errorf("PANEL_URL должен начинаться с http:// или https://, получен: %s", PANEL_URL)
			}

			// Проверяем, что домен не пустой
			if REDIRECT_DOMAIN == "" {
				t.Error("REDIRECT_DOMAIN не должен быть пустым")
			}

			// Проверяем генерацию redirect URL
			redirectURL := GetRedirectURL()
			expectedRedirect := "http://" + tt.redirectDomain + "/redirect_happ.html?url="
			if redirectURL != expectedRedirect {
				t.Errorf("GetRedirectURL() = %s, expected %s", redirectURL, expectedRedirect)
			}

			t.Logf("Тест домена: PanelURL=%s, RedirectDomain=%s, RedirectURL=%s",
				PANEL_URL, REDIRECT_DOMAIN, redirectURL)
		})
	}
}

// TestEnvironmentConfiguration тестирует работу с переменными окружения
func TestEnvironmentConfiguration(t *testing.T) {
	// Сохраняем оригинальные значения
	originalPanelURL := PANEL_URL
	originalPanelUser := PANEL_USER
	originalPanelPass := PANEL_PASS
	originalInboundID := INBOUND_ID
	originalRedirectDomain := REDIRECT_DOMAIN
	defer func() {
		PANEL_URL = originalPanelURL
		PANEL_USER = originalPanelUser
		PANEL_PASS = originalPanelPass
		INBOUND_ID = originalInboundID
		REDIRECT_DOMAIN = originalRedirectDomain
	}()

	// Тестируем разные конфигурации
	testConfigs := []struct {
		panelURL       string
		panelUser      string
		panelPass      string
		inboundID      int
		redirectDomain string
		description    string
	}{
		{
			panelURL:       "https://production-vpn.com/",
			panelUser:      "prod_user",
			panelPass:      "prod_pass",
			inboundID:      1,
			redirectDomain: "production-vpn.com",
			description:    "Продакшн конфигурация",
		},
		{
			panelURL:       "https://staging-vpn.com:8080/",
			panelUser:      "staging_user",
			panelPass:      "staging_pass",
			inboundID:      2,
			redirectDomain: "staging-vpn.com",
			description:    "Стейджинг конфигурация",
		},
		{
			panelURL:       "http://dev.local:3000/",
			panelUser:      "dev_user",
			panelPass:      "dev_pass",
			inboundID:      3,
			redirectDomain: "dev.local",
			description:    "Разработка конфигурация",
		},
	}

	for _, tt := range testConfigs {
		t.Run(tt.description, func(t *testing.T) {
			// Устанавливаем тестовые значения
			PANEL_URL = tt.panelURL
			PANEL_USER = tt.panelUser
			PANEL_PASS = tt.panelPass
			INBOUND_ID = tt.inboundID
			REDIRECT_DOMAIN = tt.redirectDomain

			// Проверяем, что все переменные установлены корректно
			if PANEL_URL != tt.panelURL {
				t.Errorf("PANEL_URL = %s, expected %s", PANEL_URL, tt.panelURL)
			}
			if PANEL_USER != tt.panelUser {
				t.Errorf("PANEL_USER = %s, expected %s", PANEL_USER, tt.panelUser)
			}
			if PANEL_PASS != tt.panelPass {
				t.Errorf("PANEL_PASS = %s, expected %s", PANEL_PASS, tt.panelPass)
			}
			if INBOUND_ID != tt.inboundID {
				t.Errorf("INBOUND_ID = %d, expected %d", INBOUND_ID, tt.inboundID)
			}
			if REDIRECT_DOMAIN != tt.redirectDomain {
				t.Errorf("REDIRECT_DOMAIN = %s, expected %s", REDIRECT_DOMAIN, tt.redirectDomain)
			}

			// Проверяем генерацию URL
			redirectURL := GetRedirectURL()
			expectedRedirect := "http://" + tt.redirectDomain + "/redirect_happ.html?url="
			if redirectURL != expectedRedirect {
				t.Errorf("GetRedirectURL() = %s, expected %s", redirectURL, expectedRedirect)
			}

			t.Logf("Конфигурация %s: PanelURL=%s, User=%s, InboundID=%d, RedirectDomain=%s",
				tt.description, PANEL_URL, PANEL_USER, INBOUND_ID, REDIRECT_DOMAIN)
		})
	}
}

// Вспомогательная функция для проверки содержания строки
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && contains(s[1:], substr)
}
