package common

import (
	"fmt"
	"time"
)

// IsConfigActive проверяет, активен ли конфиг пользователя
func IsConfigActive(user *User) bool {
	if !user.HasActiveConfig {
		return false
	}

	// Проверяем, не истек ли конфиг
	if user.ExpiryTime > 0 && time.Now().UnixMilli() > user.ExpiryTime {
		return false
	}

	return true
}

// GetDaysWord возвращает правильную форму слова "день"
func GetDaysWord(days int) string {
	if days == 1 {
		return "день"
	} else if days >= 2 && days <= 4 {
		return "дня"
	} else {
		return "дней"
	}
}

// GetAppName возвращает название приложения на основе REDIRECT_IMPORT
func GetAppName() string {
	switch REDIRECT_IMPORT {
	case "v2raytun":
		return "v2raytun"
	case "happ":
		return "Happ"
	default:
		return "Happ" // по умолчанию
	}
}

// GetRedirectURL возвращает URL для редиректа в зависимости от типа импорта
func GetRedirectURL() string {
	var redirectFile string
	switch REDIRECT_IMPORT {
	case "v2raytun":
		redirectFile = "redirect_v2raytun.html"
	case "happ":
		redirectFile = "redirect_happ.html"
	default:
		redirectFile = "redirect_happ.html" // по умолчанию используем happ
	}
	return "http://" + REDIRECT_DOMAIN + "/" + redirectFile + "?url="
}

// CalculateTrafficLimit рассчитывает лимит трафика для указанного количества дней
func CalculateTrafficLimit(days int) int {
	// Простая логика: 1 ГБ на день
	return days
}

// FormatTrafficLimit форматирует лимит трафика для отображения
func FormatTrafficLimit(limitGB int) string {
	if limitGB <= 0 {
		return "Безлимит"
	}

	if limitGB >= 1024 {
		return fmt.Sprintf("%.1f ТБ", float64(limitGB)/1024)
	}

	return fmt.Sprintf("%d ГБ", limitGB)
}

// GetTrafficConfigDescription возвращает описание конфигурации трафика
func GetTrafficConfigDescription() string {
	// Проверяем глобальный лимит из config.go
	if TRAFFIC_LIMIT_GB <= 0 {
		return "Безлимит"
	}

	// Показываем глобальный лимит из config.go в коротком формате
	return fmt.Sprintf("%d ГБ", TRAFFIC_LIMIT_GB)
}
