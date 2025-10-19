package common

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var trafficLogFile *os.File

// InitTrafficLogger инициализирует логгер для трафика
func InitTrafficLogger() error {
	logDir := "/root/bot/logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории логов: %v", err)
	}

	logFile := filepath.Join(logDir, "traffic.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла логов трафика: %v", err)
	}

	trafficLogFile = file
	return nil
}

// CloseTrafficLogger закрывает файл логов трафика
func CloseTrafficLogger() {
	if trafficLogFile != nil {
		trafficLogFile.Close()
	}
}

// LogTraffic записывает сообщение в лог трафика
func LogTraffic(operation string, message string, args ...interface{}) {
	if trafficLogFile == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formattedMessage := fmt.Sprintf(message, args...)
	logEntry := fmt.Sprintf("[%s] [%s] %s\n", timestamp, operation, formattedMessage)

	trafficLogFile.WriteString(logEntry)
	trafficLogFile.Sync() // Принудительно записываем на диск

	// Также выводим в основной лог
	log.Printf("TRAFFIC_LOG [%s]: %s", operation, formattedMessage)
}

// LogTrafficReset записывает лог сброса трафика
func LogTrafficReset(operation string, clientCount int, details string) {
	LogTraffic(operation, "Сброс трафика: клиентов=%d, детали=%s", clientCount, details)
}

// LogClientOperation записывает лог операции с клиентом
func LogClientOperation(operation string, telegramID int64, email string, details string) {
	LogTraffic(operation, "Клиент ID=%d, Email=%s: %s", telegramID, email, details)
}

// LogConfigChange записывает лог изменения конфигурации
func LogConfigChange(operation string, telegramID int64, oldValue, newValue string) {
	LogTraffic(operation, "Изменение конфига ID=%d: %s -> %s", telegramID, oldValue, newValue)
}

// LogServiceStart записывает лог запуска сервиса
func LogServiceStart(serviceName string, interval int) {
	LogTraffic("SERVICE_START", "Запуск сервиса %s с интервалом %d минут", serviceName, interval)
}

// LogServiceStop записывает лог остановки сервиса
func LogServiceStop(serviceName string) {
	LogTraffic("SERVICE_STOP", "Остановка сервиса %s", serviceName)
}
