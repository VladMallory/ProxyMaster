package common

import "time"

// User представляет пользователя бота
type User struct {
	TelegramID      int64     `bson:"telegram_id" json:"telegram_id"`
	Username        string    `bson:"username" json:"username"`
	FirstName       string    `bson:"first_name" json:"first_name"`
	LastName        string    `bson:"last_name" json:"last_name"`
	Balance         float64   `bson:"balance" json:"balance"`
	TotalPaid       float64   `bson:"total_paid" json:"total_paid"`
	ConfigsCount    int       `bson:"configs_count" json:"configs_count"`
	HasActiveConfig bool      `bson:"has_active_config" json:"has_active_config"`
	ClientID        string    `bson:"client_id" json:"client_id"`
	SubID           string    `bson:"sub_id" json:"sub_id"`
	Email           string    `bson:"email" json:"email"`
	ConfigCreatedAt time.Time `bson:"config_created_at" json:"config_created_at"`
	ExpiryTime      int64     `bson:"expiry_time" json:"expiry_time"`
	HasUsedTrial    bool      `bson:"has_used_trial" json:"has_used_trial"`
	CreatedAt       time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time `bson:"updated_at" json:"updated_at"`
}

// TrafficConfig представляет конфигурацию трафика
type TrafficConfig struct {
	Enabled        bool `bson:"enabled" json:"enabled"`
	DailyLimitGB   int  `bson:"daily_limit_gb" json:"daily_limit_gb"`
	WeeklyLimitGB  int  `bson:"weekly_limit_gb" json:"weekly_limit_gb"`
	MonthlyLimitGB int  `bson:"monthly_limit_gb" json:"monthly_limit_gb"`
	LimitGB        int  `bson:"limit_gb" json:"limit_gb"`
	ResetDays      int  `bson:"reset_days" json:"reset_days"`
}

// Client структура для 3x-ui API
type Client struct {
	ID         string      `json:"id"`
	Flow       string      `json:"flow"`
	Email      string      `json:"email"`
	LimitIP    int         `json:"limitIp"`
	TotalGB    int         `json:"totalGB"`
	ExpiryTime int64       `json:"expiryTime"`
	Enable     bool        `json:"enable"`
	TgID       interface{} `json:"tgId"` // Может быть числом или строкой
	SubID      string      `json:"subId"`
	Reset      int         `json:"reset"`

	// Дополнительные поля, которые есть в реальном API
	CreatedAt int64 `json:"created_at,omitempty"`
	UpdatedAt int64 `json:"updated_at,omitempty"`

	// Попытка управлять состоянием "исчерпано"
	Depleted  *bool `json:"depleted,omitempty"`  // указатель, чтобы различать false и отсутствие поля
	Exhausted *bool `json:"exhausted,omitempty"` // на случай, если используется другое название
}

// Settings структура для поля settings
type Settings struct {
	Clients    []Client `json:"clients"`
	Decryption string   `json:"decryption"`
}

// Структуры для API запросов
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AddClientRequest struct {
	ID       int    `json:"id"`
	Settings string `json:"settings"`
}

type UpdateClientRequest struct {
	ID       int    `json:"id"`
	Settings string `json:"settings"`
}

// Структуры для API ответов
type LoginResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

type APIResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

type Inbound struct {
	ID             int         `json:"id"`
	Up             int64       `json:"up"`
	Down           int64       `json:"down"`
	Total          int64       `json:"total"`
	Remark         string      `json:"remark"`
	Enable         bool        `json:"enable"`
	ExpiryTime     int64       `json:"expiryTime"`
	Listen         string      `json:"listen"`
	Port           int         `json:"port"`
	Protocol       string      `json:"protocol"`
	Settings       string      `json:"settings"`
	StreamSettings string      `json:"streamSettings"`
	Tag            string      `json:"tag"`
	Sniffing       string      `json:"sniffing"`
	ClientStats    interface{} `json:"clientStats"`
}

type InboundInfo struct {
	Success bool    `json:"success"`
	Msg     string  `json:"msg"` // Поле Msg для совместимости с API
	Obj     Inbound `json:"obj"`
}

// TrafficStats структура для статистики трафика клиента
type TrafficStats struct {
	ID         int    `json:"id"`
	InboundID  int    `json:"inboundId"`
	Enable     bool   `json:"enable"`
	Email      string `json:"email"`
	Up         int64  `json:"up"`
	Down       int64  `json:"down"`
	ExpiryTime int64  `json:"expiryTime"`
	Total      int64  `json:"total"`
	Reset      int    `json:"reset"`
}

// UsersStatistics структура для статистики пользователей
type UsersStatistics struct {
	TotalUsers          int     `json:"total_users"`
	PayingUsers         int     `json:"paying_users"`
	TrialAvailableUsers int     `json:"trial_available_users"`
	TrialUsedUsers      int     `json:"trial_used_users"`
	InactiveUsers       int     `json:"inactive_users"`
	ActiveConfigs       int     `json:"active_configs"`
	TotalRevenue        float64 `json:"total_revenue"`
	NewThisWeek         int     `json:"new_this_week"`
	NewThisMonth        int     `json:"new_this_month"`
	ConversionRate      float64 `json:"conversion_rate"`
}
