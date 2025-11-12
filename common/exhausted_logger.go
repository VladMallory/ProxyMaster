package common

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
)

var exhaustedLogFile *os.File

// InitExhaustedLogger инициализирует логгер для изменений статуса "исчерпано"
func InitExhaustedLogger() error {
    // Если логирование exhausted отключено, ничего не инициализируем
    if !EXHAUSTED_LOG_ENABLED {
        return nil
    }

    // Создаем директорию для файла логов согласно пути из конфигурации
    logDir := filepath.Dir(EXHAUSTED_LOG_PATH)
    if err := os.MkdirAll(logDir, 0755); err != nil {
        return fmt.Errorf("ошибка создания директории логов: %v", err)
    }

    // Открываем файл по указанному пути
    file, err := os.OpenFile(EXHAUSTED_LOG_PATH, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        return fmt.Errorf("ошибка открытия файла exhausted.log: %v", err)
    }

    exhaustedLogFile = file
    return nil
}

// CloseExhaustedLogger закрывает файл логов exhausted
func CloseExhaustedLogger() {
	if exhaustedLogFile != nil {
		exhaustedLogFile.Close()
	}
}

// LogExhausted записывает сообщение в exhausted.log
func LogExhausted(operation string, message string, args ...interface{}) {
    if !EXHAUSTED_LOG_ENABLED || exhaustedLogFile == nil {
        return
    }

    timestamp := time.Now().Format("2006-01-02 15:04:05")
    formattedMessage := fmt.Sprintf(message, args...)
    logEntry := fmt.Sprintf("[%s] [%s] %s\n", timestamp, operation, formattedMessage)

    exhaustedLogFile.WriteString(logEntry)
    exhaustedLogFile.Sync() // Принудительная запись на диск
}
