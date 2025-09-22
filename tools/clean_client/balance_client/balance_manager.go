package balance_client

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// BalanceManager управляет балансом клиентов
type BalanceManager struct {
	db *sql.DB
}

// NewBalanceManager создает новый экземпляр BalanceManager
func NewBalanceManager(db *sql.DB) *BalanceManager {
	return &BalanceManager{db: db}
}

// GetUserBalance получает текущий баланс пользователя
func (bm *BalanceManager) GetUserBalance(telegramID int64) (float64, error) {
	query := `SELECT balance FROM users WHERE telegram_id = $1`
	var balance float64
	
	err := bm.db.QueryRow(query, telegramID).Scan(&balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("пользователь с TelegramID %d не найден", telegramID)
		}
		return 0, fmt.Errorf("ошибка получения баланса: %v", err)
	}
	
	return balance, nil
}

// AddBalance добавляет средства на баланс пользователя
func (bm *BalanceManager) AddBalance(telegramID int64, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("сумма должна быть положительной")
	}

	query := `
		UPDATE users SET 
			balance = balance + $2,
			total_paid = total_paid + $2,
			updated_at = $3
		WHERE telegram_id = $1`

	result, err := bm.db.Exec(query, telegramID, amount, time.Now())
	if err != nil {
		return fmt.Errorf("ошибка добавления баланса: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества обновленных строк: %v", err)
	}

	if affected == 0 {
		return fmt.Errorf("пользователь с TelegramID %d не найден", telegramID)
	}

	log.Printf("✅ Добавлено %.2f₽ на баланс пользователя %d", amount, telegramID)
	return nil
}

// SubtractBalance списывает средства с баланса пользователя
func (bm *BalanceManager) SubtractBalance(telegramID int64, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("сумма должна быть положительной")
	}

	// Сначала проверяем текущий баланс
	currentBalance, err := bm.GetUserBalance(telegramID)
	if err != nil {
		return err
	}

	if currentBalance < amount {
		return fmt.Errorf("недостаточно средств. Текущий баланс: %.2f₽, требуется: %.2f₽", currentBalance, amount)
	}

	query := `
		UPDATE users SET 
			balance = balance - $2,
			updated_at = $3
		WHERE telegram_id = $1`

	result, err := bm.db.Exec(query, telegramID, amount, time.Now())
	if err != nil {
		return fmt.Errorf("ошибка списания баланса: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества обновленных строк: %v", err)
	}

	if affected == 0 {
		return fmt.Errorf("пользователь с TelegramID %d не найден", telegramID)
	}

	log.Printf("✅ Списано %.2f₽ с баланса пользователя %d", amount, telegramID)
	return nil
}

// SetBalance устанавливает точный баланс пользователя
func (bm *BalanceManager) SetBalance(telegramID int64, amount float64) error {
	if amount < 0 {
		return fmt.Errorf("баланс не может быть отрицательным")
	}

	query := `
		UPDATE users SET 
			balance = $2,
			updated_at = $3
		WHERE telegram_id = $1`

	result, err := bm.db.Exec(query, telegramID, amount, time.Now())
	if err != nil {
		return fmt.Errorf("ошибка установки баланса: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества обновленных строк: %v", err)
	}

	if affected == 0 {
		return fmt.Errorf("пользователь с TelegramID %d не найден", telegramID)
	}

	log.Printf("✅ Установлен баланс %.2f₽ для пользователя %d", amount, telegramID)
	return nil
}

// GetUserInfo получает полную информацию о пользователе
func (bm *BalanceManager) GetUserInfo(telegramID int64) (*UserInfo, error) {
	query := `
		SELECT telegram_id, username, first_name, last_name, balance, total_paid, 
		       has_active_config, client_id, sub_id, email, expiry_time, created_at
		FROM users 
		WHERE telegram_id = $1`

	var userInfo UserInfo
	var username, firstName, lastName sql.NullString
	var clientID, subID, email sql.NullString
	var expiryTime sql.NullInt64
	var createdAt time.Time

	err := bm.db.QueryRow(query, telegramID).Scan(
		&userInfo.TelegramID, &username, &firstName, &lastName,
		&userInfo.Balance, &userInfo.TotalPaid, &userInfo.HasActiveConfig,
		&clientID, &subID, &email, &expiryTime, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("пользователь с TelegramID %d не найден", telegramID)
		}
		return nil, fmt.Errorf("ошибка получения информации о пользователе: %v", err)
	}

	// Обрабатываем NULL значения
	if username.Valid {
		userInfo.Username = username.String
	}
	if firstName.Valid {
		userInfo.FirstName = firstName.String
	}
	if lastName.Valid {
		userInfo.LastName = lastName.String
	}
	if clientID.Valid {
		userInfo.ClientID = clientID.String
	}
	if subID.Valid {
		userInfo.SubID = subID.String
	}
	if email.Valid {
		userInfo.Email = email.String
	}
	if expiryTime.Valid {
		userInfo.ExpiryTime = expiryTime.Int64
	}

	userInfo.CreatedAt = createdAt
	return &userInfo, nil
}

// UserInfo содержит информацию о пользователе
type UserInfo struct {
	TelegramID      int64     `json:"telegram_id"`
	Username        string    `json:"username"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	Balance         float64   `json:"balance"`
	TotalPaid       float64   `json:"total_paid"`
	HasActiveConfig bool      `json:"has_active_config"`
	ClientID        string    `json:"client_id"`
	SubID           string    `json:"sub_id"`
	Email           string    `json:"email"`
	ExpiryTime      int64     `json:"expiry_time"`
	CreatedAt       time.Time `json:"created_at"`
}

// GetBalanceHistory получает историю изменений баланса (если есть таблица логов)
func (bm *BalanceManager) GetBalanceHistory(telegramID int64, limit int) ([]BalanceHistory, error) {
	// Проверяем, существует ли таблица balance_history
	checkQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'balance_history'
		)`
	
	var tableExists bool
	err := bm.db.QueryRow(checkQuery).Scan(&tableExists)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки существования таблицы истории: %v", err)
	}

	if !tableExists {
		return []BalanceHistory{}, nil // Возвращаем пустой массив, если таблицы нет
	}

	query := `
		SELECT telegram_id, amount, operation_type, description, created_at
		FROM balance_history 
		WHERE telegram_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2`

	rows, err := bm.db.Query(query, telegramID, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения истории баланса: %v", err)
	}
	defer rows.Close()

	var history []BalanceHistory
	for rows.Next() {
		var h BalanceHistory
		err := rows.Scan(&h.TelegramID, &h.Amount, &h.OperationType, &h.Description, &h.CreatedAt)
		if err != nil {
			log.Printf("Ошибка сканирования истории: %v", err)
			continue
		}
		history = append(history, h)
	}

	return history, nil
}

// BalanceHistory содержит запись об изменении баланса
type BalanceHistory struct {
	TelegramID    int64     `json:"telegram_id"`
	Amount        float64   `json:"amount"`
	OperationType string    `json:"operation_type"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
}

// LogBalanceChange записывает изменение баланса в историю (если таблица существует)
func (bm *BalanceManager) LogBalanceChange(telegramID int64, amount float64, operationType, description string) error {
	// Проверяем, существует ли таблица balance_history
	checkQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'balance_history'
		)`
	
	var tableExists bool
	err := bm.db.QueryRow(checkQuery).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("ошибка проверки существования таблицы истории: %v", err)
	}

	if !tableExists {
		return nil // Если таблицы нет, просто пропускаем логирование
	}

	query := `
		INSERT INTO balance_history (telegram_id, amount, operation_type, description, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err = bm.db.Exec(query, telegramID, amount, operationType, description, time.Now())
	if err != nil {
		return fmt.Errorf("ошибка записи в историю баланса: %v", err)
	}

	return nil
}
