package common

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// BanInfo содержит информацию о бане пользователя
type BanInfo struct {
	Email       string    `json:"email"`
	BannedAt    time.Time `json:"banned_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Reason      string    `json:"reason"`
	IPAddresses []string  `json:"ip_addresses"`
}

// BanManager управляет банами пользователей
type BanManager struct {
	BansFile string
	Bans     map[string]*BanInfo
}

// NewBanManager создает новый менеджер банов
func NewBanManager(bansFile string) *BanManager {
	bm := &BanManager{
		BansFile: bansFile,
		Bans:     make(map[string]*BanInfo),
	}
	bm.loadBans()
	return bm
}

// loadBans загружает баны из файла
func (bm *BanManager) loadBans() {
	data, err := os.ReadFile(bm.BansFile)
	if err != nil {
		// Файл не существует или пуст - это нормально
		return
	}

	if err := json.Unmarshal(data, &bm.Bans); err != nil {
		fmt.Printf("BAN_MANAGER: Ошибка загрузки банов: %v\n", err)
		bm.Bans = make(map[string]*BanInfo)
	}
}

// saveBans сохраняет баны в файл
func (bm *BanManager) saveBans() error {
	data, err := json.MarshalIndent(bm.Bans, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации банов: %v", err)
	}

	return os.WriteFile(bm.BansFile, data, 0644)
}

// IsBanned проверяет, забанен ли пользователь
func (bm *BanManager) IsBanned(email string) bool {
	ban, exists := bm.Bans[email]
	if !exists {
		return false
	}

	// Проверяем, не истек ли бан
	if time.Now().After(ban.ExpiresAt) {
		// Бан истек, удаляем его
		delete(bm.Bans, email)
		bm.saveBans()
		return false
	}

	return true
}

// BanUser банит пользователя
func (bm *BanManager) BanUser(email string, reason string, ipAddresses []string) error {
	banDuration := time.Duration(IP_BAN_DURATION) * time.Minute
	if IP_BAN_DURATION <= 0 {
		banDuration = 0 // Бесконечный бан
	}

	now := time.Now()
	ban := &BanInfo{
		Email:       email,
		BannedAt:    now,
		ExpiresAt:   now.Add(banDuration),
		Reason:      reason,
		IPAddresses: ipAddresses,
	}

	bm.Bans[email] = ban
	return bm.saveBans()
}

// UnbanUser разбанивает пользователя
func (bm *BanManager) UnbanUser(email string) error {
	delete(bm.Bans, email)
	return bm.saveBans()
}

// GetBanInfo возвращает информацию о бане пользователя
func (bm *BanManager) GetBanInfo(email string) *BanInfo {
	ban, exists := bm.Bans[email]
	if !exists {
		return nil
	}

	// Проверяем, не истек ли бан
	if time.Now().After(ban.ExpiresAt) {
		delete(bm.Bans, email)
		bm.saveBans()
		return nil
	}

	return ban
}

// CleanupExpiredBans удаляет истекшие баны
func (bm *BanManager) CleanupExpiredBans() {
	now := time.Now()
	expiredCount := 0

	for email, ban := range bm.Bans {
		if now.After(ban.ExpiresAt) {
			delete(bm.Bans, email)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		bm.saveBans()
		fmt.Printf("BAN_MANAGER: Удалено %d истекших банов\n", expiredCount)
	}
}

// CleanupOldBans удаляет баны, которые истекли дольше заданного времени назад
func (bm *BanManager) CleanupOldBans(retentionMinutes int) {
	if retentionMinutes <= 0 {
		return // Если время хранения = 0, данные хранятся бесконечно
	}

	now := time.Now()
	cutoffTime := now.Add(-time.Duration(retentionMinutes) * time.Minute)
	oldBansCount := 0

	fmt.Printf("BAN_MANAGER: Очистка старых банов: удаляются баны, истекшие дольше %d минут назад\n", retentionMinutes)

	for email, ban := range bm.Bans {
		// Удаляем баны, которые истекли дольше retentionMinutes назад
		if ban.ExpiresAt.Before(cutoffTime) {
			delete(bm.Bans, email)
			oldBansCount++
			fmt.Printf("BAN_MANAGER: Удален старый бан для %s (истёк: %s)\n",
				email, ban.ExpiresAt.Format("15:04:05 02.01.2006"))
		}
	}

	if oldBansCount > 0 {
		bm.saveBans()
		fmt.Printf("BAN_MANAGER: Удалено %d старых банов из файла\n", oldBansCount)
	}
}

// GetActiveBans возвращает список активных банов
func (bm *BanManager) GetActiveBans() map[string]*BanInfo {
	bm.CleanupExpiredBans() // Очищаем истекшие баны
	return bm.Bans
}

// GetBanStats возвращает статистику банов
func (bm *BanManager) GetBanStats() map[string]interface{} {
	bm.CleanupExpiredBans()

	totalBans := len(bm.Bans)
	expiredSoon := 0
	now := time.Now()

	for _, ban := range bm.Bans {
		if ban.ExpiresAt.Sub(now) < time.Hour {
			expiredSoon++
		}
	}

	return map[string]interface{}{
		"total_bans":     totalBans,
		"expired_soon":   expiredSoon,
		"ban_duration":   IP_BAN_DURATION,
		"unlimited_bans": IP_BAN_DURATION <= 0,
	}
}
