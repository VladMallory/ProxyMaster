package common

import (
	"crypto/md5"
	"fmt"
	"log"
	"math/rand"
	"time"
)

// GeneratePaymentID генерирует уникальный ID для платежа
func GeneratePaymentID() string {
	timestamp := time.Now().UnixNano()
	random := rand.Int63()

	source := fmt.Sprintf("%d_%d", timestamp, random)
	hash := md5.Sum([]byte(source))

	return fmt.Sprintf("payment_%x", hash[:8])
}

// ValidateAmount проверяет корректность суммы платежа
func ValidateAmount(amount float64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	// Минимальная сумма - 1 рубль
	if amount < 1.0 {
		return fmt.Errorf("минимальная сумма платежа: 1₽, указано: %.2f₽", amount)
	}

	// Максимальная сумма - 100000 рублей
	if amount > 100000.0 {
		return fmt.Errorf("максимальная сумма платежа: 100000₽, указано: %.2f₽", amount)
	}

	return nil
}

// FormatAmount форматирует сумму для отображения
func FormatAmount(amount float64) string {
	return fmt.Sprintf("%.2f₽", amount)
}

// LogPaymentEvent логирует события платежной системы
func LogPaymentEvent(level string, method PaymentMethod, message string, args ...interface{}) {
	prefix := fmt.Sprintf("PAYMENT_%s_%s", string(method), level)
	fullMessage := fmt.Sprintf(message, args...)
	log.Printf("%s: %s", prefix, fullMessage)
}

// GetCurrentTimestamp возвращает текущее время в формате ISO 8601
func GetCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

// ParseTimestamp парсит строку времени в формате ISO 8601
func ParseTimestamp(timestamp string) (time.Time, error) {
	return time.Parse(time.RFC3339, timestamp)
}

// ConvertRublesToKopecks конвертирует рубли в копейки
func ConvertRublesToKopecks(rubles float64) int {
	return int(rubles * 100)
}

// ConvertKopecksToRubles конвертирует копейки в рубли
func ConvertKopecksToRubles(kopecks int) float64 {
	return float64(kopecks) / 100.0
}

// SanitizeDescription очищает описание платежа от недопустимых символов
func SanitizeDescription(description string) string {
	if len(description) > 250 {
		description = description[:250]
	}

	// Базовая очистка - можно расширить по необходимости
	return description
}

// CreatePaymentMetadata создает метаданные для платежа
func CreatePaymentMetadata(userID int64, additionalData map[string]interface{}) map[string]interface{} {
	metadata := map[string]interface{}{
		"user_id":    userID,
		"created_by": "vpn_bot",
		"timestamp":  GetCurrentTimestamp(),
	}

	// Добавляем дополнительные данные
	for key, value := range additionalData {
		metadata[key] = value
	}

	return metadata
}

// IsPaymentStatusFinal проверяет, является ли статус платежа финальным
func IsPaymentStatusFinal(status PaymentStatus) bool {
	return status == PaymentStatusSucceeded ||
		status == PaymentStatusCanceled ||
		status == PaymentStatusFailed
}

// GetPaymentStatusDescription возвращает описание статуса платежа на русском
func GetPaymentStatusDescription(status PaymentStatus) string {
	switch status {
	case PaymentStatusPending:
		return "Ожидает оплаты"
	case PaymentStatusSucceeded:
		return "Оплачен"
	case PaymentStatusCanceled:
		return "Отменен"
	case PaymentStatusFailed:
		return "Ошибка оплаты"
	default:
		return "Неизвестный статус"
	}
}

// GetMethodDescription возвращает описание метода оплаты на русском
func GetMethodDescription(method PaymentMethod) string {
	switch method {
	case PaymentMethodTelegram:
		return "Telegram Bot API"
	case PaymentMethodAPI:
		return "ЮKassa API"
	default:
		return "Неизвестный метод"
	}
}
