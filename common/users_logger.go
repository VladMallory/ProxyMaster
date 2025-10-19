package common

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// UsersLogger представляет логгер для записи действий пользователей
type UsersLogger struct {
	file    *os.File
	mu      sync.Mutex
	enabled bool
}

var (
	usersLogger *UsersLogger
	usersOnce   sync.Once
)

// InitUsersLogger инициализирует глобальный логгер действий пользователей
func InitUsersLogger() error {
	var err error
	usersOnce.Do(func() {
		usersLogger = &UsersLogger{
			enabled: USERS_LOG_ENABLED,
		}

		if !usersLogger.enabled {
			return
		}

		err = usersLogger.init()
	})
	return err
}

// init инициализирует файл логов действий пользователей
func (ul *UsersLogger) init() error {
	// Создаем директорию для логов, если она не существует
	logDir := filepath.Dir(USERS_LOG_PATH)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории логов пользователей: %v", err)
	}

	// Открываем файл для записи
	file, err := os.OpenFile(USERS_LOG_PATH, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла логов пользователей: %v", err)
	}

	ul.file = file

	// Записываем заголовок в лог
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	ul.file.WriteString(fmt.Sprintf("\n=== USERS LOG STARTED: %s ===\n", timestamp))
	ul.file.Sync()

	log.Printf("USERS_LOGGER: Логгер действий пользователей успешно инициализирован, запись в файл: %s", USERS_LOG_PATH)

	return nil
}

// Close закрывает файл логов действий пользователей
func (ul *UsersLogger) Close() error {
	ul.mu.Lock()
	defer ul.mu.Unlock()

	if !ul.enabled || ul.file == nil {
		return nil
	}

	// Записываем завершающий заголовок
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	ul.file.WriteString(fmt.Sprintf("\n=== USERS LOG ENDED: %s ===\n", timestamp))

	// Закрываем файл
	err := ul.file.Close()
	ul.file = nil

	return err
}

// LogUserAction записывает действие пользователя в лог
func LogUserAction(telegramID int64, username, firstName, lastName, action, details string) {
	if !USERS_LOG_ENABLED || usersLogger == nil || usersLogger.file == nil {
		return
	}

	usersLogger.mu.Lock()
	defer usersLogger.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] USER_ACTION: ID=%d, Username=%s, Name=%s %s, Action=%s, Details=%s\n",
		timestamp, telegramID, username, firstName, lastName, action, details)

	usersLogger.file.WriteString(logEntry)
	usersLogger.file.Sync()
}

// LogUserMessage записывает сообщение пользователя в лог
func LogUserMessage(message *tgbotapi.Message) {
	if !USERS_LOG_ENABLED || usersLogger == nil || usersLogger.file == nil {
		return
	}

	usersLogger.mu.Lock()
	defer usersLogger.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	username := ""
	if message.From.UserName != "" {
		username = "@" + message.From.UserName
	}

	logEntry := fmt.Sprintf("[%s] USER_MESSAGE: ID=%d, Username=%s, Name=%s %s, Text=%s\n",
		timestamp, message.From.ID, username, message.From.FirstName, message.From.LastName, message.Text)

	usersLogger.file.WriteString(logEntry)
	usersLogger.file.Sync()
}

// LogUserCallback записывает callback от пользователя в лог
func LogUserCallback(callback *tgbotapi.CallbackQuery) {
	if !USERS_LOG_ENABLED || usersLogger == nil || usersLogger.file == nil {
		return
	}

	usersLogger.mu.Lock()
	defer usersLogger.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	username := ""
	if callback.From.UserName != "" {
		username = "@" + callback.From.UserName
	}

	logEntry := fmt.Sprintf("[%s] USER_CALLBACK: ID=%d, Username=%s, Name=%s %s, Data=%s\n",
		timestamp, callback.From.ID, username, callback.From.FirstName, callback.From.LastName, callback.Data)

	usersLogger.file.WriteString(logEntry)
	usersLogger.file.Sync()
}

// LogUserCommand записывает команду пользователя в лог
func LogUserCommand(telegramID int64, username, firstName, lastName, command string) {
	LogUserAction(telegramID, username, firstName, lastName, "COMMAND", command)
}

// LogUserButtonClick записывает нажатие кнопки пользователем в лог
func LogUserButtonClick(telegramID int64, username, firstName, lastName, buttonText, callbackData string) {
	LogUserAction(telegramID, username, firstName, lastName, "BUTTON_CLICK", fmt.Sprintf("Button=%s, Data=%s", buttonText, callbackData))
}

// LogUserPayment записывает платеж пользователя в лог
func LogUserPayment(telegramID int64, username, firstName, lastName, paymentMethod string, amount float64, details string) {
	LogUserAction(telegramID, username, firstName, lastName, "PAYMENT", fmt.Sprintf("Method=%s, Amount=%.2f₽, Details=%s", paymentMethod, amount, details))
}

// LogUserConfigOperation записывает операции с конфигом пользователя в лог
func LogUserConfigOperation(telegramID int64, username, firstName, lastName, operation, details string) {
	LogUserAction(telegramID, username, firstName, lastName, "CONFIG_OPERATION", fmt.Sprintf("Operation=%s, Details=%s", operation, details))
}

// LogUserBalanceOperation записывает операции с балансом пользователя в лог
func LogUserBalanceOperation(telegramID int64, username, firstName, lastName, operation string, oldBalance, newBalance float64, details string) {
	LogUserAction(telegramID, username, firstName, lastName, "BALANCE_OPERATION", fmt.Sprintf("Operation=%s, OldBalance=%.2f₽, NewBalance=%.2f₽, Details=%s", operation, oldBalance, newBalance, details))
}

// GetUsersLogger возвращает экземпляр логгера действий пользователей
func GetUsersLogger() *UsersLogger {
	return usersLogger
}

// IsEnabled возвращает статус логгера действий пользователей
func (ul *UsersLogger) IsEnabled() bool {
	return ul.enabled
}

// GetLogPath возвращает путь к файлу логов действий пользователей
func (ul *UsersLogger) GetLogPath() string {
	return USERS_LOG_PATH
}
