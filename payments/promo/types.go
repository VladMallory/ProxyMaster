package promo

import (
	"database/sql"
	"time"
)

// PromoCode представляет промокод
type PromoCode struct {
	ID         string        `json:"id" db:"id"`
	Code       string        `json:"code" db:"code"`
	Amount     float64       `json:"amount" db:"amount"`
	CreatedBy  int64         `json:"created_by" db:"created_by"` // Telegram ID создателя
	CreatedAt  time.Time     `json:"created_at" db:"created_at"`
	ExpiresAt  time.Time     `json:"expires_at" db:"expires_at"`
	IsActive   bool          `json:"is_active" db:"is_active"`
	UsedBy     sql.NullInt64 `json:"used_by,omitempty" db:"used_by"` // Telegram ID пользователя, который использовал
	UsedAt     sql.NullTime  `json:"used_at,omitempty" db:"used_at"`
	UsageCount int           `json:"usage_count" db:"usage_count"`
	MaxUses    int           `json:"max_uses" db:"max_uses"` // Максимальное количество использований (по умолчанию 1)
}

// PromoUsage представляет использование промокода пользователем
type PromoUsage struct {
	ID      int64     `json:"id" db:"id"`
	PromoID string    `json:"promo_id" db:"promo_id"`
	UserID  int64     `json:"user_id" db:"user_id"`
	Amount  float64   `json:"amount" db:"amount"`
	UsedAt  time.Time `json:"used_at" db:"used_at"`
}

// PromoCodeStatus статус промокода
type PromoCodeStatus int

const (
	PromoCodeActive PromoCodeStatus = iota
	PromoCodeExpired
	PromoCodeUsed
	PromoCodeNotFound
	PromoCodeAlreadyUsedByUser
	PromoCodeMaxUsesReached
)

// String возвращает текстовое представление статуса
func (s PromoCodeStatus) String() string {
	switch s {
	case PromoCodeActive:
		return "Активен"
	case PromoCodeExpired:
		return "Истек"
	case PromoCodeUsed:
		return "Использован"
	case PromoCodeNotFound:
		return "Не найден"
	case PromoCodeAlreadyUsedByUser:
		return "Уже использован этим пользователем"
	case PromoCodeMaxUsesReached:
		return "Достигнут лимит использований"
	default:
		return "Неизвестный статус"
	}
}

// PredefinedAmounts предопределенные суммы для промокодов
var PredefinedAmounts = []float64{100, 500, 1000, 2000, 5000}

// PromoCodeExpirationDays количество дней действия промокода
const PromoCodeExpirationDays = 14

// UserPromoCooldownHours количество часов между использованием промокодов одним пользователем
const UserPromoCooldownHours = 24
