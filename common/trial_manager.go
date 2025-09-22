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
	return tm.CreateTrialConfigWithReferral(bot, user, chatID, "")
}

// CreateTrialConfigWithReferral создает конфиг для пробного периода с возможным реферальным кодом
func (tm *TrialPeriodManager) CreateTrialConfigWithReferral(bot *tgbotapi.BotAPI, user *User, chatID int64, referralCode string) error {
	log.Printf("TRIAL: Активация пробного периода для пользователя %d (добавление %d₽ на баланс)", user.TelegramID, TRIAL_BALANCE_AMOUNT)

	// Дополнительная проверка на возможность использования пробного периода
	if !tm.CanUseTrial(user) {
		log.Printf("TRIAL: ❌ Пользователь %d уже использовал пробный период, отменяем активацию", user.TelegramID)
		return fmt.Errorf("пробный период уже был использован")
	}

	// Добавляем пробный баланс пользователю
	err := AddBalance(user.TelegramID, float64(TRIAL_BALANCE_AMOUNT))
	if err != nil {
		log.Printf("TRIAL: Ошибка добавления пробного баланса для пользователя %d: %v", user.TelegramID, err)
		return fmt.Errorf("ошибка добавления пробного баланса: %v", err)
	}

	// Получаем актуальные данные пользователя из базы данных
	updatedUser, err := GetUserByTelegramID(user.TelegramID)
	if err != nil {
		log.Printf("TRIAL: Ошибка получения обновленных данных пользователя %d: %v", user.TelegramID, err)
		return fmt.Errorf("ошибка получения данных пользователя: %v", err)
	}
	if updatedUser == nil {
		log.Printf("TRIAL: Пользователь %d не найден после добавления баланса", user.TelegramID)
		return fmt.Errorf("пользователь не найден")
	}

	// Обновляем данные пользователя в памяти актуальными данными из базы
	*user = *updatedUser
	user.HasUsedTrial = true

	log.Printf("TRIAL: Пробный баланс %d₽ успешно добавлен для пользователя %d, новый баланс: %.2f₽",
		TRIAL_BALANCE_AMOUNT, user.TelegramID, user.Balance)

	// Обрабатываем реферальный код, если он есть
	if referralCode != "" {
		log.Printf("TRIAL: Обработка реферального кода %s для пользователя %d", referralCode, user.TelegramID)
		tm.processReferralCode(bot, user, chatID, referralCode)
	}

	// Обновляем только флаг использования пробного периода в базе данных
	// Баланс уже обновлен через AddBalance
	err = UpdateTrialFlag(user.TelegramID)
	if err != nil {
		log.Printf("TRIAL: Ошибка обновления флага пробного периода для пользователя %d: %v", user.TelegramID, err)
		// Не возвращаем ошибку, так как баланс уже добавлен
	} else {
		log.Printf("TRIAL: Флаг пробного периода успешно обновлен для пользователя %d", user.TelegramID)
	}

	// КРИТИЧЕСКИ ВАЖНО: Создаем конфиг БЕЗ списания денег для пробного периода
	//
	// ЛОГИКА ПРОБНОГО ПЕРИОДА:
	// 1. Добавили TRIAL_BALANCE_AMOUNT на баланс (50₽)
	// 2. Создаем конфиг БЕЗ списания денег
	// 3. Автосписание будет списывать по PRICE_PER_DAY (1₽) в день
	// 4. Пользователь получит 50 дней пробного периода
	//
	// При автосписании (TARIFF_MODE_ENABLED = false) деньги списываются постепенно
	log.Printf("TRIAL: Создание бесплатного конфига для пробного периода пользователя %d", user.TelegramID)

	// Создаем конфиг через панель 3x-ui БЕЗ списания денег
	sessionCookie, err := Login()
	if err != nil {
		log.Printf("TRIAL: Ошибка авторизации в панели для пользователя %d: %v", user.TelegramID, err)
		return fmt.Errorf("ошибка авторизации в панели: %v", err)
	}

	// Создаем конфиг для пробного периода БЕЗ установки статуса "исчерпано"
	// Рассчитываем дни на основе пробного баланса
	trialDays := TRIAL_BALANCE_AMOUNT / PRICE_PER_DAY
	log.Printf("TRIAL: Создание конфига на %d дней для пробного периода пользователя %d", trialDays, user.TelegramID)
	err = AddTrialClient(sessionCookie, user, trialDays)
	if err != nil {
		log.Printf("TRIAL: Ошибка создания конфига для пользователя %d: %v", user.TelegramID, err)
		return fmt.Errorf("ошибка создания конфига: %v", err)
	}

	// НЕ списываем деньги - они остаются на балансе для автосписания
	// Обновляем только данные пользователя в базе (без изменения баланса)
	if err := UpdateUser(user); err != nil {
		log.Printf("TRIAL: Ошибка обновления пользователя: %v", err)
		return fmt.Errorf("ошибка обновления пользователя: %v", err)
	}

	configURL := fmt.Sprintf("%s%s", CONFIG_BASE_URL, user.SubID)
	log.Printf("TRIAL: ✅ Бесплатный конфиг успешно создан для пользователя %d, URL: %s, баланс остался: %.2f₽",
		user.TelegramID, configURL, user.Balance)

	return nil
}

// GetTrialPeriodInfo возвращает информацию о пробных периодах
func (tm *TrialPeriodManager) GetTrialPeriodInfo() string {
	days := TRIAL_BALANCE_AMOUNT / PRICE_PER_DAY
	return fmt.Sprintf("📊 Информация о пробных периодах:\n\n"+
		"💰 Пробный баланс: %d₽\n"+
		"📅 Дней пробного периода: %d дней\n"+
		"💸 Стоимость в день: %d₽\n"+
		"📝 Настройка: TRIAL_BALANCE_AMOUNT = %d в config.go\n\n"+
		"💡 При активации пробного периода:\n"+
		"• Пользователю добавляется %d₽ на баланс\n"+
		"• Создается бесплатный конфиг\n"+
		"• Автосписание списывает по %d₽ в день\n"+
		"• Пользователь получает %d дней доступа",
		TRIAL_BALANCE_AMOUNT, days, PRICE_PER_DAY, TRIAL_BALANCE_AMOUNT,
		TRIAL_BALANCE_AMOUNT, PRICE_PER_DAY, days)
}

// processReferralCode обрабатывает реферальный код при активации пробного периода
func (tm *TrialPeriodManager) processReferralCode(bot *tgbotapi.BotAPI, user *User, chatID int64, referralCode string) {
	log.Printf("TRIAL: ===== НАЧАЛО ОБРАБОТКИ РЕФЕРАЛЬНОГО КОДА =====")
	log.Printf("TRIAL: Пользователь: %d, Реферальный код: '%s'", user.TelegramID, referralCode)

	// Проверяем, включена ли реферальная система
	if !REFERRAL_SYSTEM_ENABLED {
		log.Printf("TRIAL: ❌ Реферальная система отключена в конфигурации, пропускаем обработку кода %s", referralCode)
		return
	}
	log.Printf("TRIAL: ✅ Реферальная система включена")

	// Проверяем глобальный менеджер
	if GlobalReferralManager == nil {
		log.Printf("TRIAL: ❌ GlobalReferralManager не инициализирован, реферальный код %s не обработан", referralCode)
		return
	}
	log.Printf("TRIAL: ✅ GlobalReferralManager инициализирован")

	// ИСПРАВЛЕНИЕ: Сначала находим пригласившего по реферальному коду
	log.Printf("TRIAL: 🔍 Поиск пригласившего по коду '%s'...", referralCode)
	referrer, err := GlobalReferralManager.GetReferrerByCode(referralCode)
	if err != nil {
		log.Printf("TRIAL: ❌ Ошибка поиска пригласившего по коду '%s': %v", referralCode, err)
		return
	}
	log.Printf("TRIAL: ✅ Пригласивший найден: ID=%d, Name=%s", referrer.TelegramID, referrer.FirstName)

	// Проверяем, что пользователь не приглашает сам себя
	if referrer.TelegramID == user.TelegramID {
		log.Printf("TRIAL: ❌ Пользователь %d пытается пригласить сам себя", user.TelegramID)
		return
	}
	log.Printf("TRIAL: ✅ Проверка самоприглашения пройдена")

	// Обрабатываем реферальный переход с правильным referrerID
	log.Printf("TRIAL: 🔄 Обработка реферального перехода...")
	err = GlobalReferralManager.ProcessReferralTransition(referrer.TelegramID, user.TelegramID, referralCode)
	if err != nil {
		log.Printf("TRIAL: ❌ Ошибка обработки реферального перехода: %v", err)
		return
	}
	log.Printf("TRIAL: ✅ Реферальный переход успешно обработан")

	// Начисляем бонусы с правильным referrerID
	log.Printf("TRIAL: 💰 Начисление реферальных бонусов...")
	err = GlobalReferralManager.AwardReferralBonuses(referrer.TelegramID, user.TelegramID, referralCode)
	if err != nil {
		log.Printf("TRIAL: ❌ Ошибка начисления реферальных бонусов: %v", err)
	} else {
		log.Printf("TRIAL: ✅ Реферальные бонусы успешно начислены")
	}

	log.Printf("TRIAL: ===== КОНЕЦ ОБРАБОТКИ РЕФЕРАЛЬНОГО КОДА =====")
}
