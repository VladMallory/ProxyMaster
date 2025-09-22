package common

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
)

// GenerateClientID генерирует уникальный ID клиента
func GenerateClientID() string {
	return uuid.New().String()
}

// GenerateSubID генерирует случайный subId
func GenerateSubID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// GenerateEmail генерирует email для клиента на основе Telegram ID и времени истечения
func GenerateEmail(telegramID int64, expiryTime int64) string {
	if SHOW_DATES_IN_CONFIGS {
		expiryDate := time.UnixMilli(expiryTime).Format("2006 02 01")
		return fmt.Sprintf("%d до %s", telegramID, expiryDate)
	}
	return fmt.Sprintf("%d", telegramID)
}
