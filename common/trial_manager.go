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
		"У вас есть возможность получить пробный период!\n"+
		"На ваш баланс будет добавлено %d₽ для ознакомления с сервисом.\n\n"+
		"Нажмите кнопку ниже, чтобы активировать пробный период.",
		user.FirstName, TRIAL_BALANCE_AMOUNT)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки предложения пробного периода: %v", err)
	}
}

// CreateTrialConfig создает конфиг для пробного периода через добавление баланса
//
// НОВАЯ ЛОГИКА (вместо TRIAL_PERIOD_DAYS):
// Раньше создавался конфиг на фиксированное количество дней, но при пересчете баланса
// у пробных пользователей (0₽) конфиги затирались. Теперь добавляем реальные деньги на баланс,
// что решает проблему затирания и делает логику единообразной с обычными пользователями.
func (tm *TrialPeriodManager) CreateTrialConfig(bot *tgbotapi.BotAPI, user *User, chatID int64) error {
	log.Printf("TRIAL: Активация пробного периода для пользователя %d (добавление %d₽ на баланс)", user.TelegramID, TRIAL_BALANCE_AMOUNT)

	// Добавляем пробный баланс пользователю
	err := AddBalance(user.TelegramID, float64(TRIAL_BALANCE_AMOUNT))
	if err != nil {
		log.Printf("TRIAL: Ошибка добавления пробного баланса для пользователя %d: %v", user.TelegramID, err)
		return fmt.Errorf("ошибка добавления пробного баланса: %v", err)
	}

	// Обновляем данные пользователя
	user.Balance += float64(TRIAL_BALANCE_AMOUNT)
	user.HasUsedTrial = true

	// Сохраняем изменения в базу данных
	if err := UpdateUser(user); err != nil {
		log.Printf("TRIAL: Ошибка сохранения пользователя %d: %v", user.TelegramID, err)
		return fmt.Errorf("ошибка сохранения пользователя: %v", err)
	}

	log.Printf("TRIAL: Пробный баланс %d₽ успешно добавлен для пользователя %d, новый баланс: %.2f₽",
		TRIAL_BALANCE_AMOUNT, user.TelegramID, user.Balance)

	// КРИТИЧЕСКИ ВАЖНО: Принудительно запускаем пересчет баланса
	// Это мгновенно создаст конфиг на основе нового баланса
	log.Printf("TRIAL: Запуск принудительного пересчета баланса для создания конфига TelegramID=%d", user.TelegramID)
	ForceBalanceRecalculation(user.TelegramID)

	return nil
}

// GetTrialPeriodInfo возвращает информацию о пробных периодах
func (tm *TrialPeriodManager) GetTrialPeriodInfo() string {
	return fmt.Sprintf("📊 Информация о пробных периодах:\n\n"+
		"💰 Пробный баланс: %d₽\n"+
		"📝 Настройка: TRIAL_BALANCE_AMOUNT = %d в config.go\n\n"+
		"💡 При активации пробного периода пользователю добавляется указанная сумма на баланс",
		TRIAL_BALANCE_AMOUNT, TRIAL_BALANCE_AMOUNT)
}
