package common

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TrialPeriodManager управляет пробными периодами
type TrialPeriodManager struct {
	// Здесь могут быть дополнительные поля для управления пробными периодами
}

// NewTrialPeriodManager создает новый менеджер пробных периодов
func NewTrialPeriodManager() *TrialPeriodManager {
	return &TrialPeriodManager{}
}

// CanUseTrial проверяет, может ли пользователь использовать пробный период
func (tm *TrialPeriodManager) CanUseTrial(user *User) bool {
	return !user.HasUsedTrial
}

// HandleTrialPeriod обрабатывает предложение пробного периода
func (tm *TrialPeriodManager) HandleTrialPeriod(bot *tgbotapi.BotAPI, user *User, chatID int64) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎁 Активировать пробный период", "activate_trial"),
		),
	)

	text := fmt.Sprintf("🎁 Добро пожаловать, %s!\n\n"+
		"У вас есть возможность получить пробный период на %d дней!\n\n"+
		"Нажмите кнопку ниже, чтобы активировать пробный период.",
		user.FirstName, TRIAL_PERIOD_DAYS)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки предложения пробного периода: %v", err)
	}
}

// CreateTrialConfig создает конфиг для пробного периода
func (tm *TrialPeriodManager) CreateTrialConfig(bot *tgbotapi.BotAPI, user *User, chatID int64) error {
	log.Printf("Создание пробного конфига для пользователя %d", user.TelegramID)

	// Создаем конфиг через панель 3x-ui
	sessionCookie, err := Login()
	if err != nil {
		log.Printf("Ошибка авторизации в панели для пользователя %d: %v", user.TelegramID, err)
		return fmt.Errorf("ошибка авторизации в панели: %v", err)
	}

	err = AddClient(sessionCookie, user, TRIAL_PERIOD_DAYS)
	if err != nil {
		log.Printf("Ошибка создания конфига для пользователя %d: %v", user.TelegramID, err)
		return fmt.Errorf("ошибка создания конфига: %v", err)
	}

	// Обновляем флаг использования пробного периода
	user.HasUsedTrial = true

	// Сохраняем изменения в базу данных
	if err := UpdateUser(user); err != nil {
		log.Printf("Ошибка сохранения пользователя %d: %v", user.TelegramID, err)
		return fmt.Errorf("ошибка сохранения пользователя: %v", err)
	}

	log.Printf("Пробный период успешно активирован для пользователя %d: ClientID=%s, SubID=%s, Email=%s, ExpiryTime=%d",
		user.TelegramID, user.ClientID, user.SubID, user.Email, user.ExpiryTime)
	return nil
}

// GetTrialPeriodInfo возвращает информацию о пробных периодах
func (tm *TrialPeriodManager) GetTrialPeriodInfo() string {
	return fmt.Sprintf("📊 Информация о пробных периодах:\n\n"+
		"🎁 Длительность пробного периода: %d дней\n"+
		"👥 Всего пользователей с пробными периодами: 0\n"+
		"✅ Активных пробных периодов: 0",
		TRIAL_PERIOD_DAYS)
}
