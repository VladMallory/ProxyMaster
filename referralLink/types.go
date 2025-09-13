package referralLink

import "time"

// ReferralTransition представляет переход по реферальной ссылке
type ReferralTransition struct {
	ID                 int       `db:"id" json:"id"`
	ReferrerTelegramID int64     `db:"referrer_telegram_id" json:"referrer_telegram_id"`
	ReferredTelegramID int64     `db:"referred_telegram_id" json:"referred_telegram_id"`
	ReferralCode       string    `db:"referral_code" json:"referral_code"`
	TransitionDate     time.Time `db:"transition_date" json:"transition_date"`
	BonusPaid          bool      `db:"bonus_paid" json:"bonus_paid"`
	BonusAmount        float64   `db:"bonus_amount" json:"bonus_amount"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
}

// ReferralBonus представляет реферальный бонус
type ReferralBonus struct {
	ID             int       `db:"id" json:"id"`
	UserTelegramID int64     `db:"user_telegram_id" json:"user_telegram_id"`
	BonusType      string    `db:"bonus_type" json:"bonus_type"` // "referrer" или "referred"
	Amount         float64   `db:"amount" json:"amount"`
	ReferralCode   string    `db:"referral_code" json:"referral_code"`
	RelatedUserID  int64     `db:"related_user_id" json:"related_user_id"`
	Description    string    `db:"description" json:"description"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

// ReferralStats представляет статистику рефералов
type ReferralStats struct {
	TotalReferrals      int     `json:"total_referrals"`
	TotalEarnings       float64 `json:"total_earnings"`
	SuccessfulReferrals int     `json:"successful_referrals"`
	PendingReferrals    int     `json:"pending_referrals"`
}

// ReferralLinkInfo представляет информацию о реферальной ссылке
type ReferralLinkInfo struct {
	ReferralCode  string  `json:"referral_code"`
	ReferralLink  string  `json:"referral_link"`
	UserID        int64   `json:"user_id"`
	Username      string  `json:"username"`
	FirstName     string  `json:"first_name"`
	Earnings      float64 `json:"earnings"`
	ReferralCount int     `json:"referral_count"`
}
