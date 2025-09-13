package common

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

var IPBanLogger *log.Logger

// InitIPBanLogger инициализирует логгер для IP ban в отдельный файл
func InitIPBanLogger() error {
	// Создаем директорию для логов, если она не существует
	logDir := filepath.Dir(IP_BAN_LOG_PATH)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return err
	}

	// Открываем файл для записи логов IP ban
	logFile, err := os.OpenFile(IP_BAN_LOG_PATH, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		return err
	}

	// Создаем логгер, который пишет и в файл, и в stdout
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	IPBanLogger = log.New(multiWriter, "IP_BAN: ", log.LstdFlags|log.Lshortfile)

	return nil
}

// LogIPBanInfo логирует информационное сообщение о IP ban
func LogIPBanInfo(format string, v ...interface{}) {
	if IPBanLogger != nil {
		IPBanLogger.Printf("[INFO] "+format, v...)
	} else {
		log.Printf("IP_BAN [INFO]: "+format, v...)
	}
}

// LogIPBanWarning логирует предупреждение о IP ban
func LogIPBanWarning(format string, v ...interface{}) {
	if IPBanLogger != nil {
		IPBanLogger.Printf("[WARNING] "+format, v...)
	} else {
		log.Printf("IP_BAN [WARNING]: "+format, v...)
	}
}

// LogIPBanError логирует ошибку IP ban
func LogIPBanError(format string, v ...interface{}) {
	if IPBanLogger != nil {
		IPBanLogger.Printf("[ERROR] "+format, v...)
	} else {
		log.Printf("IP_BAN [ERROR]: "+format, v...)
	}
}

// LogIPBanAction логирует действие IP ban (включение/отключение конфига)
func LogIPBanAction(action, email string, ipCount int, ips []string) {
	if IPBanLogger != nil {
		IPBanLogger.Printf("[ACTION] %s конфиг %s (IP адресов: %d, IP: %v)", action, email, ipCount, ips)
	} else {
		log.Printf("IP_BAN [ACTION]: %s конфиг %s (IP адресов: %d, IP: %v)", action, email, ipCount, ips)
	}
}

// LogIPBanStats логирует статистику IP ban
func LogIPBanStats(totalEmails, suspiciousCount, normalCount int) {
	if IPBanLogger != nil {
		IPBanLogger.Printf("[STATS] Всего email: %d, Подозрительных: %d, Нормальных: %d", totalEmails, suspiciousCount, normalCount)
	} else {
		log.Printf("IP_BAN [STATS]: Всего email: %d, Подозрительных: %d, Нормальных: %d", totalEmails, suspiciousCount, normalCount)
	}
}
