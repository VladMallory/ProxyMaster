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

	// КРИТИЧЕСКИ ВАЖНО: Синхронно создаем конфиг на основе добавленного баланса
	//
	// ЛОГИКА:
	// 1. Добавили TRIAL_BALANCE_AMOUNT на баланс
	// 2. Рассчитываем доступные дни: баланс / PRICE_PER_DAY
	// 3. ProcessPayment создает конфиг и списывает стоимость
	// 4. Остаток: TRIAL_BALANCE_AMOUNT - (дни * PRICE_PER_DAY)
	//
	// Пример: TRIAL_BALANCE_AMOUNT=50₽, PRICE_PER_DAY=50₽ → 1 день, остаток 0₽
	availableDays := int(user.Balance) / PRICE_PER_DAY
	log.Printf("TRIAL: Расчет доступных дней: %.2f₽ / %d₽ = %d дней", user.Balance, PRICE_PER_DAY, availableDays)

	if availableDays > 0 {
		log.Printf("TRIAL: Синхронное создание конфига на %d дней для пользователя %d", availableDays, user.TelegramID)

		// Используем ProcessPayment для создания конфига (включает ForceResetDepletedStatus для синхронизации)
		configURL, err := ProcessPayment(user, availableDays)
		if err != nil {
			log.Printf("TRIAL: Ошибка создания конфига для пользователя %d: %v", user.TelegramID, err)
			return fmt.Errorf("ошибка создания конфига: %v", err)
		}

		log.Printf("TRIAL: ✅ Конфиг успешно создан для пользователя %d на %d дней, URL: %s, остаток баланса: %.2f₽",
			user.TelegramID, availableDays, configURL, user.Balance)
	} else {
		log.Printf("TRIAL: ❌ Недостаточно баланса для создания конфига: %.2f₽ < %d₽", user.Balance, PRICE_PER_DAY)
		log.Printf("TRIAL: ВНИМАНИЕ: Увеличьте TRIAL_BALANCE_AMOUNT до минимум %d₽ в config.go", PRICE_PER_DAY)
		return fmt.Errorf("недостаточно пробного баланса для создания конфига (нужно минимум %d₽)", PRICE_PER_DAY)
	}

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
