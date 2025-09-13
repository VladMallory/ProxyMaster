package payments

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	PaymentLogFile = "/root/bot/payments/pay.log"
)

// PaymentLogEntry представляет запись о платеже в логе
type PaymentLogEntry struct {
	PaymentID string    `json:"payment_id"`
	UserID    int64     `json:"user_id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Processed bool      `json:"processed"`
}

// PaymentLogger управляет логированием платежей
type PaymentLogger struct {
	logFile string
}

// NewPaymentLogger создает новый логгер платежей
func NewPaymentLogger() *PaymentLogger {
	return &PaymentLogger{
		logFile: PaymentLogFile,
	}
}

// LogPayment записывает информацию о платеже в лог
func (pl *PaymentLogger) LogPayment(paymentID string, userID int64, amount float64, status string) error {
	// Создаем директорию если не существует
	dir := filepath.Dir(pl.logFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории: %v", err)
	}

	entry := PaymentLogEntry{
		PaymentID: paymentID,
		UserID:    userID,
		Amount:    amount,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Processed: false,
	}

	// Открываем файл для добавления
	file, err := os.OpenFile(pl.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла лога: %v", err)
	}
	defer file.Close()

	// Записываем JSON строку
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %v", err)
	}

	if _, err := file.WriteString(string(jsonData) + "\n"); err != nil {
		return fmt.Errorf("ошибка записи в файл: %v", err)
	}

	return nil
}

// UpdatePaymentStatus обновляет статус платежа в логе
func (pl *PaymentLogger) UpdatePaymentStatus(paymentID string, status string, processed bool) error {
	entries, err := pl.readAllEntries()
	if err != nil {
		return err
	}

	// Обновляем запись
	for i, entry := range entries {
		if entry.PaymentID == paymentID {
			entries[i].Status = status
			entries[i].UpdatedAt = time.Now()
			entries[i].Processed = processed
			break
		}
	}

	return pl.writeAllEntries(entries)
}

// GetPendingPayments возвращает список необработанных платежей
func (pl *PaymentLogger) GetPendingPayments() ([]PaymentLogEntry, error) {
	entries, err := pl.readAllEntries()
	if err != nil {
		return nil, err
	}

	var pending []PaymentLogEntry
	for _, entry := range entries {
		if !entry.Processed && entry.Status == "pending" {
			// Проверяем, не слишком ли старый платеж (больше 1 часа)
			if time.Since(entry.CreatedAt) < time.Hour {
				pending = append(pending, entry)
			}
		}
	}

	return pending, nil
}

// readAllEntries читает все записи из файла лога
func (pl *PaymentLogger) readAllEntries() ([]PaymentLogEntry, error) {
	if _, err := os.Stat(pl.logFile); os.IsNotExist(err) {
		return []PaymentLogEntry{}, nil
	}

	content, err := os.ReadFile(pl.logFile)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла лога: %v", err)
	}

	var entries []PaymentLogEntry
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var entry PaymentLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Пропускаем некорректные строки
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// writeAllEntries записывает все записи в файл лога
func (pl *PaymentLogger) writeAllEntries(entries []PaymentLogEntry) error {
	file, err := os.Create(pl.logFile)
	if err != nil {
		return fmt.Errorf("ошибка создания файла лога: %v", err)
	}
	defer file.Close()

	for _, entry := range entries {
		jsonData, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		file.WriteString(string(jsonData) + "\n")
	}

	return nil
}
