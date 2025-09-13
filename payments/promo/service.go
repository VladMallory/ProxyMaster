package promo

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"bot/common"

	_ "github.com/lib/pq"
)

// PromoService сервис для управления промокодами
type PromoService struct {
	db *sql.DB
}

// NewPromoService создает новый экземпляр сервиса промокодов
func NewPromoService() (*PromoService, error) {
	db := common.GetDatabasePG()
	if db == nil {
		return nil, fmt.Errorf("база данных не инициализирована")
	}

	// Создаем таблицы если их нет
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("ошибка создания таблиц: %v", err)
	}

	return &PromoService{db: db}, nil
}

// createTables создает необходимые таблицы для промокодов
func createTables(db *sql.DB) error {
	// Таблица промокодов
	promoTableSQL := `
	CREATE TABLE IF NOT EXISTS promo_codes (
		id VARCHAR(255) PRIMARY KEY,
		code VARCHAR(255) UNIQUE NOT NULL,
		amount DECIMAL(10,2) NOT NULL,
		created_by BIGINT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		is_active BOOLEAN NOT NULL DEFAULT true,
		used_by BIGINT,
		used_at TIMESTAMP WITH TIME ZONE,
		usage_count INTEGER NOT NULL DEFAULT 0,
		max_uses INTEGER NOT NULL DEFAULT 1
	);`

	// Таблица использования промокодов
	usageTableSQL := `
	CREATE TABLE IF NOT EXISTS promo_usage (
		id SERIAL PRIMARY KEY,
		promo_id VARCHAR(255) NOT NULL,
		user_id BIGINT NOT NULL,
		amount DECIMAL(10,2) NOT NULL,
		used_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		FOREIGN KEY (promo_id) REFERENCES promo_codes(id) ON DELETE CASCADE
	);`

	// Индексы для оптимизации
	indexSQL := `
	CREATE INDEX IF NOT EXISTS idx_promo_codes_code ON promo_codes(code);
	CREATE INDEX IF NOT EXISTS idx_promo_codes_expires_at ON promo_codes(expires_at);
	CREATE INDEX IF NOT EXISTS idx_promo_codes_created_by ON promo_codes(created_by);
	CREATE INDEX IF NOT EXISTS idx_promo_usage_user_id ON promo_usage(user_id);
	CREATE INDEX IF NOT EXISTS idx_promo_usage_promo_id ON promo_usage(promo_id);
	CREATE INDEX IF NOT EXISTS idx_promo_usage_used_at ON promo_usage(used_at);`

	if _, err := db.Exec(promoTableSQL); err != nil {
		return fmt.Errorf("ошибка создания таблицы promo_codes: %v", err)
	}

	if _, err := db.Exec(usageTableSQL); err != nil {
		return fmt.Errorf("ошибка создания таблицы promo_usage: %v", err)
	}

	if _, err := db.Exec(indexSQL); err != nil {
		return fmt.Errorf("ошибка создания индексов: %v", err)
	}

	return nil
}

// generatePromoCode генерирует уникальный промокод
func (ps *PromoService) generatePromoCode() (string, error) {
	const maxAttempts = 10

	for i := 0; i < maxAttempts; i++ {
		// Генерируем случайный код
		bytes := make([]byte, 4) // 8 символов в hex
		if _, err := rand.Read(bytes); err != nil {
			return "", fmt.Errorf("ошибка генерации случайных байт: %v", err)
		}

		code := hex.EncodeToString(bytes)
		code = strings.ToLower(code)

		// Проверяем уникальность
		var exists bool
		query := "SELECT EXISTS(SELECT 1 FROM promo_codes WHERE code = $1)"
		err := ps.db.QueryRow(query, code).Scan(&exists)
		if err != nil {
			return "", fmt.Errorf("ошибка проверки уникальности кода: %v", err)
		}

		if !exists {
			return code, nil
		}
	}

	return "", fmt.Errorf("не удалось сгенерировать уникальный промокод за %d попыток", maxAttempts)
}

// CreatePromoCode создает новый промокод
func (ps *PromoService) CreatePromoCode(amount float64, createdBy int64) (*PromoCode, error) {
	// Генерируем уникальный код
	code, err := ps.generatePromoCode()
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации промокода: %v", err)
	}

	// Создаем ID для промокода
	promoID := fmt.Sprintf("promo_%d_%s", time.Now().Unix(), code)

	// Вычисляем время истечения
	expiresAt := time.Now().AddDate(0, 0, PromoCodeExpirationDays)

	// Создаем промокод
	promo := &PromoCode{
		ID:        promoID,
		Code:      code,
		Amount:    amount,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		IsActive:  true,
		MaxUses:   1,
	}

	// Сохраняем в базу данных
	query := `
		INSERT INTO promo_codes (id, code, amount, created_by, created_at, expires_at, is_active, max_uses)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = ps.db.Exec(query, promo.ID, promo.Code, promo.Amount, promo.CreatedBy,
		promo.CreatedAt, promo.ExpiresAt, promo.IsActive, promo.MaxUses)
	if err != nil {
		return nil, fmt.Errorf("ошибка сохранения промокода: %v", err)
	}

	log.Printf("PROMO: Создан промокод %s на сумму %.2f₽ (создатель: %d)",
		promo.Code, promo.Amount, promo.CreatedBy)

	return promo, nil
}

// ValidatePromoCode проверяет валидность промокода для пользователя
func (ps *PromoService) ValidatePromoCode(code string, userID int64) (*PromoCode, PromoCodeStatus, error) {
	// Ищем промокод
	query := `
		SELECT id, code, amount, created_by, created_at, expires_at, is_active, 
		       used_by, used_at, usage_count, max_uses
		FROM promo_codes 
		WHERE code = $1 AND is_active = true`

	var promo PromoCode
	err := ps.db.QueryRow(query, code).Scan(
		&promo.ID, &promo.Code, &promo.Amount, &promo.CreatedBy,
		&promo.CreatedAt, &promo.ExpiresAt, &promo.IsActive,
		&promo.UsedBy, &promo.UsedAt, &promo.UsageCount, &promo.MaxUses)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, PromoCodeNotFound, nil
		}
		return nil, PromoCodeNotFound, fmt.Errorf("ошибка поиска промокода: %v", err)
	}

	// Проверяем срок действия
	if time.Now().After(promo.ExpiresAt) {
		return &promo, PromoCodeExpired, nil
	}

	// Проверяем лимит использований
	if promo.UsageCount >= promo.MaxUses {
		return &promo, PromoCodeMaxUsesReached, nil
	}

	// Проверяем, использовал ли уже этот пользователь промокод
	if promo.UsedBy.Valid && promo.UsedBy.Int64 == userID {
		return &promo, PromoCodeAlreadyUsedByUser, nil
	}

	// Проверяем кулдаун пользователя (24 часа между использованиями)
	cooldownQuery := `
		SELECT EXISTS(
			SELECT 1 FROM promo_usage 
			WHERE user_id = $1 AND used_at > NOW() - INTERVAL '%d hours'
		)`

	var hasCooldown bool
	cooldownErr := ps.db.QueryRow(fmt.Sprintf(cooldownQuery, UserPromoCooldownHours), userID).Scan(&hasCooldown)
	if cooldownErr != nil {
		log.Printf("PROMO: Ошибка проверки кулдауна для пользователя %d: %v", userID, cooldownErr)
		// Продолжаем выполнение, не блокируем из-за ошибки кулдауна
	} else if hasCooldown {
		return &promo, PromoCodeAlreadyUsedByUser, nil
	}

	return &promo, PromoCodeActive, nil
}

// UsePromoCode активирует промокод для пользователя
func (ps *PromoService) UsePromoCode(code string, userID int64) (*PromoCode, error) {
	// Проверяем валидность промокода
	promo, status, err := ps.ValidatePromoCode(code, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка валидации промокода: %v", err)
	}

	if status != PromoCodeActive {
		return nil, fmt.Errorf("промокод не может быть использован: %s", status.String())
	}

	// Начинаем транзакцию
	tx, err := ps.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("ошибка начала транзакции: %v", err)
	}
	defer tx.Rollback()

	// Обновляем промокод
	updateQuery := `
		UPDATE promo_codes 
		SET used_by = $1, used_at = NOW(), usage_count = usage_count + 1
		WHERE id = $2 AND is_active = true AND usage_count < max_uses`

	result, err := tx.Exec(updateQuery, userID, promo.ID)
	if err != nil {
		return nil, fmt.Errorf("ошибка обновления промокода: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения количества обновленных строк: %v", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("промокод уже использован или неактивен")
	}

	// Записываем использование
	usageQuery := `
		INSERT INTO promo_usage (promo_id, user_id, amount, used_at)
		VALUES ($1, $2, $3, NOW())`

	_, err = tx.Exec(usageQuery, promo.ID, userID, promo.Amount)
	if err != nil {
		return nil, fmt.Errorf("ошибка записи использования промокода: %v", err)
	}

	// Пополняем баланс пользователя
	balanceQuery := "UPDATE users SET balance = balance + $1, updated_at = NOW() WHERE telegram_id = $2"
	_, err = tx.Exec(balanceQuery, promo.Amount, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка пополнения баланса: %v", err)
	}

	// Коммитим транзакцию
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("ошибка коммита транзакции: %v", err)
	}

	// Обновляем данные промокода
	promo.UsedBy = sql.NullInt64{Int64: userID, Valid: true}
	promo.UsedAt = sql.NullTime{Time: time.Now(), Valid: true}
	promo.UsageCount++

	log.Printf("PROMO: Промокод %s использован пользователем %d на сумму %.2f₽",
		promo.Code, userID, promo.Amount)

	return promo, nil
}

// GetUserPromoHistory возвращает историю использования промокодов пользователем
func (ps *PromoService) GetUserPromoHistory(userID int64, limit int) ([]PromoUsage, error) {
	query := `
		SELECT pu.id, pu.promo_id, pu.user_id, pu.amount, pu.used_at
		FROM promo_usage pu
		JOIN promo_codes pc ON pu.promo_id = pc.id
		WHERE pu.user_id = $1
		ORDER BY pu.used_at DESC
		LIMIT $2`

	rows, err := ps.db.Query(query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения истории промокодов: %v", err)
	}
	defer rows.Close()

	var history []PromoUsage
	for rows.Next() {
		var usage PromoUsage
		err := rows.Scan(&usage.ID, &usage.PromoID, &usage.UserID, &usage.Amount, &usage.UsedAt)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования истории: %v", err)
		}
		history = append(history, usage)
	}

	return history, nil
}

// GetPromoStats возвращает статистику промокодов
func (ps *PromoService) GetPromoStats(createdBy int64) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Общее количество созданных промокодов
	var totalCreated int
	query := "SELECT COUNT(*) FROM promo_codes WHERE created_by = $1"
	err := ps.db.QueryRow(query, createdBy).Scan(&totalCreated)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения общего количества промокодов: %v", err)
	}

	// Количество использованных промокодов
	var totalUsed int
	query = `
		SELECT COUNT(*) FROM promo_codes 
		WHERE created_by = $1 AND usage_count > 0`
	err = ps.db.QueryRow(query, createdBy).Scan(&totalUsed)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения количества использованных промокодов: %v", err)
	}

	// Общая сумма выданных промокодов
	var totalAmount float64
	query = `
		SELECT COALESCE(SUM(pu.amount), 0) FROM promo_usage pu
		JOIN promo_codes pc ON pu.promo_id = pc.id
		WHERE pc.created_by = $1`
	err = ps.db.QueryRow(query, createdBy).Scan(&totalAmount)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения общей суммы: %v", err)
	}

	stats["total_created"] = totalCreated
	stats["total_used"] = totalUsed
	stats["total_amount"] = totalAmount
	stats["usage_rate"] = float64(totalUsed) / float64(totalCreated) * 100

	return stats, nil
}

// CleanupExpiredPromos очищает истекшие промокоды (можно вызывать периодически)
func (ps *PromoService) CleanupExpiredPromos() error {
	query := "UPDATE promo_codes SET is_active = false WHERE expires_at < NOW() AND is_active = true"
	result, err := ps.db.Exec(query)
	if err != nil {
		return fmt.Errorf("ошибка очистки истекших промокодов: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества обновленных строк: %v", err)
	}

	if rowsAffected > 0 {
		log.Printf("PROMO: Деактивировано %d истекших промокодов", rowsAffected)
	}

	return nil
}
