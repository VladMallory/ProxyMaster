package common

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ConsoleLogger представляет логгер для записи консольного вывода в файл
type ConsoleLogger struct {
	file              *os.File
	mu                sync.Mutex
	enabled           bool
	originalStdout    *os.File
	originalStderr    *os.File
	originalLogWriter io.Writer
}

var (
	consoleLogger *ConsoleLogger
	once          sync.Once
)

// InitConsoleLogger инициализирует глобальный логгер консоли
func InitConsoleLogger() error {
	var err error
	once.Do(func() {
		consoleLogger = &ConsoleLogger{
			enabled: CONSOLE_LOG_ENABLED,
		}

		if !consoleLogger.enabled {
			return
		}

		err = consoleLogger.init()
	})
	return err
}

// init инициализирует файл логов и настраивает перехват вывода
func (cl *ConsoleLogger) init() error {
	// Создаем директорию для логов, если она не существует
	logDir := filepath.Dir(CONSOLE_LOG_PATH)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории логов: %v", err)
	}

	// Открываем файл для записи
	file, err := os.OpenFile(CONSOLE_LOG_PATH, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла консольных логов: %v", err)
	}

	cl.file = file

	// Сохраняем оригинальные потоки
	cl.originalStdout = os.Stdout
	cl.originalStderr = os.Stderr
	cl.originalLogWriter = log.Writer()

	// Настраиваем стандартный логгер для записи в файл и консоль
	log.SetOutput(io.MultiWriter(cl.originalLogWriter, cl.file))

	// Записываем заголовок в лог
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	cl.file.WriteString(fmt.Sprintf("\n=== CONSOLE LOG STARTED: %s ===\n", timestamp))
	cl.file.Sync()

	log.Printf("CONSOLE_LOGGER: Логгер консоли успешно инициализирован, запись в файл: %s", CONSOLE_LOG_PATH)

	return nil
}

// Close закрывает файл логов и восстанавливает оригинальные потоки
func (cl *ConsoleLogger) Close() error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if !cl.enabled || cl.file == nil {
		return nil
	}

	// Восстанавливаем оригинальные потоки
	log.SetOutput(cl.originalLogWriter)

	// Записываем завершающий заголовок
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	cl.file.WriteString(fmt.Sprintf("\n=== CONSOLE LOG ENDED: %s ===\n", timestamp))

	// Закрываем файл
	err := cl.file.Close()
	cl.file = nil

	return err
}

// WriteConsoleLog записывает произвольное сообщение в консольный лог
func WriteConsoleLog(message string, args ...interface{}) {
	if !CONSOLE_LOG_ENABLED || consoleLogger == nil || consoleLogger.file == nil {
		return
	}

	consoleLogger.mu.Lock()
	defer consoleLogger.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formattedMessage := fmt.Sprintf(message, args...)
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, formattedMessage)

	consoleLogger.file.WriteString(logEntry)
	consoleLogger.file.Sync()
}

// GetConsoleLogger возвращает экземпляр логгера консоли
func GetConsoleLogger() *ConsoleLogger {
	return consoleLogger
}

// IsEnabled возвращает статус логгера
func (cl *ConsoleLogger) IsEnabled() bool {
	return cl.enabled
}

// GetLogPath возвращает путь к файлу логов
func (cl *ConsoleLogger) GetLogPath() string {
	return CONSOLE_LOG_PATH
}
