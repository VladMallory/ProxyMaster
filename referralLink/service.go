package referralLink

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
)

// ReferralService сервис для работы с реферальной системой
type ReferralService struct {
	db *sql.DB
}

// NewReferralService создает новый экземпляр сервиса рефералов
func NewReferralService(db *sql.DB) *ReferralService {
	return &ReferralService{db: db}
}

// GenerateReferralCode генерирует уникальный реферальный код для пользователя
func (rs *ReferralService) GenerateReferralCode(telegramID int64) (string, error) {
	// Проверяем, есть ли уже код у пользователя
	var existingCode sql.NullString
	query := "SELECT referral_code FROM users WHERE telegram_id = $1"
	err := rs.db.QueryRow(query, telegramID).Scan(&existingCode)

	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("ошибка проверки существующего кода: %v", err)
	}

	// Если код уже есть, возвращаем его
	if existingCode.Valid && existingCode.String != "" {
		return existingCode.String, nil
	}

	// Генерируем новый код
	code := rs.generateUniqueCode(telegramID)

	// Сохраняем код в БД
	updateQuery := "UPDATE users SET referral_code = $1 WHERE telegram_id = $2"
	_, err = rs.db.Exec(updateQuery, code, telegramID)
	if err != nil {
		return "", fmt.Errorf("ошибка сохранения реферального кода: %v", err)
	}

	log.Printf("REFERRAL_SERVICE: Сгенерирован реферальный код %s для пользователя %d", code, telegramID)
	return code, nil
}

// generateUniqueCode генерирует уникальный реферальный код
func (rs *ReferralService) generateUniqueCode(telegramID int64) string {
	// Генерируем код вручную без префиксов
	code := fmt.Sprintf("%d%03d", telegramID, int(telegramID%1000))

	// Проверяем уникальность кода
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE referral_code = $1)"
	var exists bool
	err := rs.db.QueryRow(query, code).Scan(&exists)
	if err != nil {
		log.Printf("REFERRAL_SERVICE: Ошибка проверки уникальности кода: %v", err)
		return code
	}

	// Если код уже существует, добавляем случайное число
	if exists {
		code = fmt.Sprintf("%d%03d%d", telegramID, int(telegramID%1000), int(telegramID%100))
	}

	return code
}

// GetReferralLinkInfo получает информацию о реферальной ссылке пользователя
func (rs *ReferralService) GetReferralLinkInfo(telegramID int64) (*ReferralLinkInfo, error) {
	query := `
		SELECT u.telegram_id, u.username, u.first_name, u.referral_code, 
		       u.referral_earnings, u.referral_count
		FROM users u 
		WHERE u.telegram_id = $1`

	var info ReferralLinkInfo
	var username, firstName, referralCode sql.NullString

	err := rs.db.QueryRow(query, telegramID).Scan(
		&info.UserID, &username, &firstName, &referralCode,
		&info.Earnings, &info.ReferralCount,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("пользователь не найден")
		}
		return nil, fmt.Errorf("ошибка получения информации о реферальной ссылке: %v", err)
	}

	// Обработка NULL значений
	if username.Valid {
		info.Username = username.String
	}
	if firstName.Valid {
		info.FirstName = firstName.String
	}
	if referralCode.Valid {
		info.ReferralCode = referralCode.String
		// Убираем префикс "ref_" из кода для ссылки, так как он уже есть в REFERRAL_LINK_BASE_URL
		codeWithoutPrefix := strings.TrimPrefix(referralCode.String, "ref_")
		info.ReferralLink = common.REFERRAL_LINK_BASE_URL + codeWithoutPrefix
	} else {
		// Генерируем код, если его нет
		code, err := rs.GenerateReferralCode(telegramID)
		if err != nil {
			return nil, fmt.Errorf("ошибка генерации реферального кода: %v", err)
		}
		info.ReferralCode = code
		// Убираем префикс "ref_" из кода для ссылки
		codeWithoutPrefix := strings.TrimPrefix(code, "ref_")
		info.ReferralLink = common.REFERRAL_LINK_BASE_URL + codeWithoutPrefix
	}

	return &info, nil
}

// ProcessReferralTransition обрабатывает переход по реферальной ссылке
func (rs *ReferralService) ProcessReferralTransition(referrerID, referredID int64, referralCode string) error {
	log.Printf("REFERRAL_SERVICE: ===== ОБРАБОТКА РЕФЕРАЛЬНОГО ПЕРЕХОДА =====")
	log.Printf("REFERRAL_SERVICE: ReferrerID=%d, ReferredID=%d, Code='%s'", referrerID, referredID, referralCode)

	// Проверяем, что реферальная система включена
	if !common.REFERRAL_SYSTEM_ENABLED {
		log.Printf("REFERRAL_SERVICE: ❌ Реферальная система отключена в конфигурации")
		return fmt.Errorf("реферальная система отключена")
	}
	log.Printf("REFERRAL_SERVICE: ✅ Реферальная система включена")

	// Проверяем минимальный баланс для получения реферальной ссылки
	log.Printf("REFERRAL_SERVICE: Получение информации о пригласившем %d", referrerID)
	referrer, err := common.GetUserByTelegramID(referrerID)
	if err != nil {
		log.Printf("REFERRAL_SERVICE: ❌ Ошибка получения информации о пригласившем %d: %v", referrerID, err)
		return fmt.Errorf("ошибка получения информации о пригласившем: %v", err)
	}
	log.Printf("REFERRAL_SERVICE: ✅ Информация о пригласившем получена: Balance=%.2f, MinRequired=%.2f", referrer.Balance, common.REFERRAL_MIN_BALANCE_FOR_REF)

	if referrer.Balance < common.REFERRAL_MIN_BALANCE_FOR_REF {
		log.Printf("REFERRAL_SERVICE: ❌ Недостаточный баланс: %.2f < %.2f", referrer.Balance, common.REFERRAL_MIN_BALANCE_FOR_REF)
		return fmt.Errorf("недостаточный баланс для получения реферальной ссылки")
	}
	log.Printf("REFERRAL_SERVICE: ✅ Баланс достаточен")

	// Используем функцию из БД для обработки перехода
	log.Printf("REFERRAL_SERVICE: Вызов функции БД process_referral_transition")
	query := "SELECT process_referral_transition($1, $2, $3)"
	var success bool
	err = rs.db.QueryRow(query, referrerID, referredID, referralCode).Scan(&success)
	if err != nil {
		log.Printf("REFERRAL_SERVICE: ❌ Ошибка выполнения запроса process_referral_transition: %v", err)
		return fmt.Errorf("ошибка обработки реферального перехода: %v", err)
	}

	if !success {
		log.Printf("REFERRAL_SERVICE: ❌ Функция process_referral_transition вернула false")
		return fmt.Errorf("не удалось обработать реферальный переход")
	}

	log.Printf("REFERRAL_SERVICE: ✅ Реферальный переход успешно обработан: %d -> %d (код: %s)", referrerID, referredID, referralCode)
	return nil
}

// AwardReferralBonuses начисляет реферальные бонусы
func (rs *ReferralService) AwardReferralBonuses(referrerID, referredID int64, referralCode string) error {
	log.Printf("REFERRAL_SERVICE: ===== НАЧИСЛЕНИЕ РЕФЕРАЛЬНЫХ БОНУСОВ =====")
	log.Printf("REFERRAL_SERVICE: ReferrerID=%d, ReferredID=%d, Code='%s'", referrerID, referredID, referralCode)
	log.Printf("REFERRAL_SERVICE: ReferrerBonus=%.2f, WelcomeBonus=%.2f", common.REFERRAL_BONUS_AMOUNT, common.REFERRAL_WELCOME_BONUS)

	// Начисляем бонус пригласившему
	if common.REFERRAL_BONUS_AMOUNT > 0 {
		log.Printf("REFERRAL_SERVICE: Начисление бонуса пригласившему %d: %.2f", referrerID, common.REFERRAL_BONUS_AMOUNT)
		err := rs.awardBonus(referrerID, "referrer", common.REFERRAL_BONUS_AMOUNT, referralCode, referredID, "Реферальный бонус за приглашение друга")
		if err != nil {
			log.Printf("REFERRAL_SERVICE: ❌ Ошибка начисления бонуса пригласившему %d: %v", referrerID, err)
			return err
		}
		log.Printf("REFERRAL_SERVICE: ✅ Начислен бонус %.2f пригласившему %d", common.REFERRAL_BONUS_AMOUNT, referrerID)
	} else {
		log.Printf("REFERRAL_SERVICE: ⏭️ Бонус пригласившему отключен (%.2f)", common.REFERRAL_BONUS_AMOUNT)
	}

	// Начисляем бонус приглашенному
	if common.REFERRAL_WELCOME_BONUS > 0 {
		log.Printf("REFERRAL_SERVICE: Начисление приветственного бонуса приглашенному %d: %.2f", referredID, common.REFERRAL_WELCOME_BONUS)
		err := rs.awardBonus(referredID, "referred", common.REFERRAL_WELCOME_BONUS, referralCode, referrerID, "Приветственный бонус за регистрацию по реферальной ссылке")
		if err != nil {
			log.Printf("REFERRAL_SERVICE: ❌ Ошибка начисления приветственного бонуса %d: %v", referredID, err)
			return err
		}
		log.Printf("REFERRAL_SERVICE: ✅ Начислен приветственный бонус %.2f приглашенному %d", common.REFERRAL_WELCOME_BONUS, referredID)
	} else {
		log.Printf("REFERRAL_SERVICE: ⏭️ Приветственный бонус отключен (%.2f)", common.REFERRAL_WELCOME_BONUS)
	}

	// Отправляем уведомление администратору
	log.Printf("REFERRAL_SERVICE: Отправка уведомления администратору")
	rs.sendAdminNotification(referrerID, referredID, referralCode)

	log.Printf("REFERRAL_SERVICE: ✅ Все бонусы успешно начислены")
	return nil
}

// awardBonus начисляет бонус пользователю
func (rs *ReferralService) awardBonus(userID int64, bonusType string, amount float64, referralCode string, relatedUserID int64, description string) error {
	log.Printf("REFERRAL_SERVICE: ===== НАЧИСЛЕНИЕ БОНУСА =====")
	log.Printf("REFERRAL_SERVICE: UserID=%d, Type='%s', Amount=%.2f, Code='%s', RelatedUserID=%d", userID, bonusType, amount, referralCode, relatedUserID)

	// Используем AddBalance для начисления бонуса
	log.Printf("REFERRAL_SERVICE: Начисление баланса через AddBalance")
	err := common.AddBalance(userID, amount)
	if err != nil {
		log.Printf("REFERRAL_SERVICE: ❌ Ошибка начисления бонуса через AddBalance: %v", err)
		return fmt.Errorf("ошибка начисления бонуса через AddBalance: %v", err)
	}
	log.Printf("REFERRAL_SERVICE: ✅ Баланс успешно начислен")

	// Записываем в историю бонусов
	log.Printf("REFERRAL_SERVICE: Запись в историю бонусов")
	query := `
		INSERT INTO referral_bonuses (user_telegram_id, bonus_type, amount, referral_code, related_user_id, description)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err = rs.db.Exec(query, userID, bonusType, amount, referralCode, relatedUserID, description)
	if err != nil {
		log.Printf("REFERRAL_SERVICE: ❌ Ошибка записи в историю бонусов для пользователя %d: %v", userID, err)
		// Не возвращаем ошибку, так как бонус уже начислен
	} else {
		log.Printf("REFERRAL_SERVICE: ✅ Запись в историю бонусов успешна")
	}

	// Если это бонус пригласившему, обновляем общую сумму реферальных заработков
	if bonusType == "referrer" {
		log.Printf("REFERRAL_SERVICE: Обновление реферальной статистики для пригласившего")
		updateQuery := `
			UPDATE users 
			SET referral_earnings = referral_earnings + $2, referral_count = referral_count + 1
			WHERE telegram_id = $1`

		_, err = rs.db.Exec(updateQuery, userID, amount)
		if err != nil {
			log.Printf("REFERRAL_SERVICE: ❌ Ошибка обновления реферальной статистики для пользователя %d: %v", userID, err)
			// Не возвращаем ошибку, так как бонус уже начислен
		} else {
			log.Printf("REFERRAL_SERVICE: ✅ Реферальная статистика обновлена")
		}
	}

	log.Printf("REFERRAL_SERVICE: ✅ Бонус успешно начислен пользователю %d", userID)
	return nil
}

// GetReferralStats получает статистику рефералов пользователя
func (rs *ReferralService) GetReferralStats(telegramID int64) (*ReferralStats, error) {
	query := `
		SELECT 
			COALESCE(u.referral_count, 0) as total_referrals,
			COALESCE(u.referral_earnings, 0) as total_earnings,
			COUNT(CASE WHEN rt.bonus_paid = true THEN 1 END) as successful_referrals,
			COUNT(CASE WHEN rt.bonus_paid = false THEN 1 END) as pending_referrals
		FROM users u
		LEFT JOIN referral_transitions rt ON u.telegram_id = rt.referrer_telegram_id
		WHERE u.telegram_id = $1
		GROUP BY u.telegram_id, u.referral_count, u.referral_earnings`

	var stats ReferralStats
	err := rs.db.QueryRow(query, telegramID).Scan(
		&stats.TotalReferrals,
		&stats.TotalEarnings,
		&stats.SuccessfulReferrals,
		&stats.PendingReferrals,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return &ReferralStats{}, nil
		}
		return nil, fmt.Errorf("ошибка получения статистики рефералов: %v", err)
	}

	return &stats, nil
}

// GetReferralHistory получает историю реферальных бонусов пользователя
func (rs *ReferralService) GetReferralHistory(telegramID int64, limit int) ([]ReferralBonus, error) {
	query := `
		SELECT id, user_telegram_id, bonus_type, amount, referral_code, 
		       related_user_id, description, created_at
		FROM referral_bonuses 
		WHERE user_telegram_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2`

	rows, err := rs.db.Query(query, telegramID, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения истории реферальных бонусов: %v", err)
	}
	defer rows.Close()

	var bonuses []ReferralBonus
	for rows.Next() {
		var bonus ReferralBonus
		var referralCode, description sql.NullString
		var relatedUserID sql.NullInt64

		err := rows.Scan(
			&bonus.ID, &bonus.UserTelegramID, &bonus.BonusType, &bonus.Amount,
			&referralCode, &relatedUserID, &description, &bonus.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования истории бонусов: %v", err)
		}

		// Обработка NULL значений
		if referralCode.Valid {
			bonus.ReferralCode = referralCode.String
		}
		if relatedUserID.Valid {
			bonus.RelatedUserID = relatedUserID.Int64
		}
		if description.Valid {
			bonus.Description = description.String
		}

		bonuses = append(bonuses, bonus)
	}

	return bonuses, nil
}

// IsValidReferralCode проверяет, является ли код валидным реферальным кодом
func (rs *ReferralService) IsValidReferralCode(code string) bool {
	// Убираем префикс "ref_" если он есть
	referralCode := strings.TrimPrefix(code, "ref_")

	log.Printf("REFERRAL_SERVICE: Проверка валидности кода: '%s' -> '%s'", code, referralCode)

	query := "SELECT EXISTS(SELECT 1 FROM users WHERE referral_code = $1)"
	var exists bool
	err := rs.db.QueryRow(query, referralCode).Scan(&exists)
	if err != nil {
		log.Printf("REFERRAL_SERVICE: Ошибка проверки реферального кода %s: %v", referralCode, err)
		return false
	}

	log.Printf("REFERRAL_SERVICE: Код '%s' валиден: %v", referralCode, exists)
	return exists
}

// GetReferrerByCode получает информацию о пригласившем по реферальному коду
func (rs *ReferralService) GetReferrerByCode(referralCode string) (*common.User, error) {
	query := `
		SELECT telegram_id, username, first_name, last_name, balance, total_paid,
		       configs_count, has_active_config, client_id, sub_id, email,
		       config_created_at, expiry_time, has_used_trial, created_at, updated_at
		FROM users 
		WHERE referral_code = $1`

	var user common.User
	var username, firstName, lastName sql.NullString
	var clientID, subID, email sql.NullString
	var configCreatedAt sql.NullTime
	var expiryTime sql.NullInt64

	err := rs.db.QueryRow(query, referralCode).Scan(
		&user.TelegramID, &username, &firstName, &lastName,
		&user.Balance, &user.TotalPaid, &user.ConfigsCount, &user.HasActiveConfig,
		&clientID, &subID, &email, &configCreatedAt,
		&expiryTime, &user.HasUsedTrial, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("реферальный код не найден")
		}
		return nil, fmt.Errorf("ошибка получения информации о пригласившем: %v", err)
	}

	// Обработка NULL значений
	if username.Valid {
		user.Username = username.String
	}
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}
	if clientID.Valid {
		user.ClientID = clientID.String
	}
	if subID.Valid {
		user.SubID = subID.String
	}
	if email.Valid {
		user.Email = email.String
	}
	if configCreatedAt.Valid {
		user.ConfigCreatedAt = configCreatedAt.Time
	}
	if expiryTime.Valid {
		user.ExpiryTime = expiryTime.Int64
	}

	return &user, nil
}

// sendAdminNotification отправляет уведомление администратору о новом реферале
func (rs *ReferralService) sendAdminNotification(referrerID, referredID int64, referralCode string) {
	// Проверяем, включены ли уведомления для администратора
	if !common.ADMIN_NOTIFICATIONS_ENABLED || !common.ADMIN_REFERRAL_ENABLED {
		return
	}

	// Получаем информацию о пригласившем
	referrer, err := common.GetUserByTelegramID(referrerID)
	if err != nil {
		log.Printf("REFERRAL_SERVICE: Ошибка получения информации о пригласившем %d: %v", referrerID, err)
		return
	}

	// Получаем информацию о приглашенном
	referred, err := common.GetUserByTelegramID(referredID)
	if err != nil {
		log.Printf("REFERRAL_SERVICE: Ошибка получения информации о приглашенном %d: %v", referredID, err)
		return
	}

	// Формируем сообщение для администратора
	text := "🎯 <b>Новый реферал!</b>\n\n"
	text += fmt.Sprintf("👤 <b>Пригласивший:</b> %s (ID: %d)\n", referrer.FirstName, referrerID)
	if referrer.Username != "" {
		text += fmt.Sprintf("🔗 Username: @%s\n", referrer.Username)
	}
	text += fmt.Sprintf("📊 Всего рефералов: %d\n", referrer.ReferralCount)

	text += fmt.Sprintf("👤 <b>Приглашенный:</b> %s (ID: %d)\n", referred.FirstName, referredID)
	if referred.Username != "" {
		text += fmt.Sprintf("🔗 Username: @%s\n", referred.Username)
	}
	text += fmt.Sprintf("📅 Дата регистрации: %s\n\n", referred.CreatedAt.Format("02.01.2006 15:04"))

	text += fmt.Sprintf("🔗 <b>Реферальный код:</b> %s\n", referralCode)
	text += fmt.Sprintf("💰 <b>Бонусы:</b>\n")
	text += fmt.Sprintf("• Пригласившему: %.0f₽\n", common.REFERRAL_BONUS_AMOUNT)
	text += fmt.Sprintf("• Приглашенному: %.0f₽\n", common.REFERRAL_WELCOME_BONUS)

	// Отправляем уведомление администратору
	if common.GlobalBot != nil {
		msg := tgbotapi.NewMessage(common.ADMIN_ID, text)
		msg.ParseMode = "HTML"
		if _, err := common.GlobalBot.Send(msg); err != nil {
			log.Printf("REFERRAL_SERVICE: Ошибка отправки уведомления администратору: %v", err)
		} else {
			log.Printf("REFERRAL_SERVICE: Уведомление о реферале отправлено администратору")
		}
	} else {
		log.Printf("REFERRAL_SERVICE: Глобальный бот не инициализирован, уведомление не отправлено")
	}
}
