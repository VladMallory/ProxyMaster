package services

import (
	"testing"
	"time"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TestExpiredSubscriptionService_isSubscriptionExpired тестирует функцию проверки истечения подписки
func TestExpiredSubscriptionService_isSubscriptionExpired(t *testing.T) {
	// Создаем мок-объекты
	bot := &tgbotapi.BotAPI{}                // Мок бота
	configManager := &common.ConfigManager{} // Мок менеджера конфигов

	service := NewExpiredSubscriptionService(bot, configManager)
	now := time.Now()

	tests := []struct {
		name           string
		user           *common.User
		expectedResult bool
		description    string
	}{
		{
			name: "ActiveSubscription_NotExpired",
			user: &common.User{
				HasActiveConfig: true,
				ExpiryTime:      now.Add(24 * time.Hour).UnixMilli(), // Истекает через день
			},
			expectedResult: false,
			description:    "Активная подписка, не истекшая",
		},
		{
			name: "ActiveSubscription_Expired",
			user: &common.User{
				HasActiveConfig: true,
				ExpiryTime:      now.Add(-24 * time.Hour).UnixMilli(), // Истекла вчера
			},
			expectedResult: true,
			description:    "Активная подписка, истекшая",
		},
		{
			name: "InactiveSubscription_Expired",
			user: &common.User{
				HasActiveConfig: false,
				ExpiryTime:      now.Add(-24 * time.Hour).UnixMilli(),
			},
			expectedResult: false,
			description:    "Неактивная подписка, даже если время истекло",
		},
		{
			name: "ActiveSubscription_NoExpiryTime",
			user: &common.User{
				HasActiveConfig: true,
				ExpiryTime:      0, // Нет времени истечения
			},
			expectedResult: false,
			description:    "Активная подписка без времени истечения",
		},
		{
			name: "ActiveSubscription_ExpiresNow",
			user: &common.User{
				HasActiveConfig: true,
				ExpiryTime:      now.Add(-1 * time.Millisecond).UnixMilli(), // Истекла 1 миллисекунду назад
			},
			expectedResult: true,
			description:    "Активная подписка, истекающая прямо сейчас",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isSubscriptionExpired(tt.user, now)
			if result != tt.expectedResult {
				t.Errorf("isSubscriptionExpired() = %v, expected %v. %s",
					result, tt.expectedResult, tt.description)
			}
		})
	}
}
