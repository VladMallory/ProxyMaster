package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"bot/common"
)

// DepletedStatusChecker сервис для проверки и исправления ложных состояний "исчерпано"
type DepletedStatusChecker struct {
	ticker *time.Ticker
	done   chan bool
}

// NewDepletedStatusChecker создает новый экземпляр сервиса проверки ложных состояний "исчерпано"
func NewDepletedStatusChecker() *DepletedStatusChecker {
	return &DepletedStatusChecker{
		done: make(chan bool),
	}
}

// Start запускает сервис проверки ложных состояний "исчерпано"
func (dsc *DepletedStatusChecker) Start() {
	if !common.DEPLETED_STATUS_CHECK_ENABLED {
		log.Printf("DEPLETED_STATUS_CHECKER: Проверка ложных состояний 'исчерпано' отключена")
		return
	}

	interval := time.Duration(common.DEPLETED_STATUS_CHECK_INTERVAL) * time.Minute
	log.Printf("DEPLETED_STATUS_CHECKER: Запуск сервиса проверки ложных состояний 'исчерпано' с интервалом %v", interval)
	common.LogServiceStart("DEPLETED_STATUS_CHECKER", common.DEPLETED_STATUS_CHECK_INTERVAL)

	dsc.ticker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-dsc.ticker.C:
				dsc.CheckAndFixDepletedStatus()
			case <-dsc.done:
				log.Printf("DEPLETED_STATUS_CHECKER: Сервис остановлен")
				common.LogServiceStop("DEPLETED_STATUS_CHECKER")
				return
			}
		}
	}()
}

// Stop останавливает сервис
func (dsc *DepletedStatusChecker) Stop() {
	if dsc.ticker != nil {
		dsc.ticker.Stop()
	}
	close(dsc.done)
}

// CheckAndFixDepletedStatus проверяет и исправляет ложные состояния "исчерпано"
func (dsc *DepletedStatusChecker) CheckAndFixDepletedStatus() {
	log.Printf("DEPLETED_STATUS_CHECKER: Начинаем проверку ложных состояний 'исчерпано'...")

	// Получаем всех пользователей с активными конфигами
	users, err := common.GetUsersWithActiveConfigs()
	if err != nil {
		log.Printf("DEPLETED_STATUS_CHECKER: Ошибка получения пользователей: %v", err)
		return
	}

	now := time.Now()
	fixedCount := 0

	for _, user := range users {
		// Проверяем, нужно ли исправить состояние "исчерпано" для этого пользователя
		if dsc.shouldFixDepletedStatus(&user, now) {
			log.Printf("DEPLETED_STATUS_CHECKER: Найдено ложное состояние 'исчерпано' для пользователя %d (email: %s)",
				user.TelegramID, user.Email)

			// Исправляем состояние "исчерпано"
			if err := dsc.fixDepletedStatus(&user); err != nil {
				log.Printf("DEPLETED_STATUS_CHECKER: Ошибка исправления состояния 'исчерпано' для пользователя %d: %v",
					user.TelegramID, err)
			} else {
				fixedCount++
				log.Printf("DEPLETED_STATUS_CHECKER: Состояние 'исчерпано' успешно исправлено для пользователя %d", user.TelegramID)
			}
		}
	}

	if fixedCount > 0 {
		log.Printf("DEPLETED_STATUS_CHECKER: Проверка завершена. Исправлено ложных состояний 'исчерпано': %d", fixedCount)
	} else {
		log.Printf("DEPLETED_STATUS_CHECKER: Проверка завершена. Ложных состояний 'исчерпано' не найдено")
	}
}

// shouldFixDepletedStatus проверяет, нужно ли исправить состояние "исчерпано" для пользователя
func (dsc *DepletedStatusChecker) shouldFixDepletedStatus(user *common.User, now time.Time) bool {
	// Проверяем, что у пользователя есть активная подписка
	if !user.HasActiveConfig {
		return false
	}

	// Проверяем, что подписка не истекла
	if user.ExpiryTime > 0 && user.ExpiryTime <= now.UnixMilli() {
		return false
	}

	// Проверяем, что конфиг активен достаточно долго (больше DEPLETED_ACTIVE_THRESHOLD часов)
	// Используем время создания конфига из базы данных
	if user.ConfigCreatedAt.IsZero() {
		// Если время создания не установлено, считаем что конфиг достаточно старый
		return true
	}

	configAge := now.Sub(user.ConfigCreatedAt)
	threshold := time.Duration(common.DEPLETED_ACTIVE_THRESHOLD) * time.Hour

	if configAge < threshold {
		log.Printf("DEPLETED_STATUS_CHECKER: Конфиг пользователя %d слишком новый (возраст: %v, порог: %v), пропускаем",
			user.TelegramID, configAge, threshold)
		return false
	}

	// Проверяем состояние в панели управления
	return dsc.isDepletedInPanel(user)
}

// isDepletedInPanel проверяет, находится ли конфиг в состоянии "исчерпано" в панели управления
func (dsc *DepletedStatusChecker) isDepletedInPanel(user *common.User) bool {
	// Получаем сессию для работы с панелью
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("DEPLETED_STATUS_CHECKER: Ошибка авторизации в панели: %v", err)
		return false
	}

	// Получаем текущий inbound
	inbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Printf("DEPLETED_STATUS_CHECKER: Ошибка получения inbound: %v", err)
		return false
	}

	var settings common.Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Printf("DEPLETED_STATUS_CHECKER: Ошибка десериализации settings: %v", err)
		return false
	}

	// Ищем клиента по TelegramID
	telegramIDStr := fmt.Sprintf("%d", user.TelegramID)
	for _, client := range settings.Clients {
		if strings.HasPrefix(client.Email, telegramIDStr+"_") ||
			strings.HasPrefix(client.Email, telegramIDStr+" ") ||
			client.Email == telegramIDStr {

			// Проверяем состояние "исчерпано"
			isDepleted := client.Depleted != nil && *client.Depleted
			isExhausted := client.Exhausted != nil && *client.Exhausted

			log.Printf("DEPLETED_STATUS_CHECKER: Клиент %d в панели: Depleted=%v, Exhausted=%v, Enable=%v",
				user.TelegramID, isDepleted, isExhausted, client.Enable)

			// Возвращаем true если конфиг в состоянии "исчерпано" (даже если включен)
			// Это исправляет проблему с кэшированием в веб-интерфейсе
			return isDepleted || isExhausted
		}
	}

	return false
}

// fixDepletedStatus исправляет ложное состояние "исчерпано" для пользователя
func (dsc *DepletedStatusChecker) fixDepletedStatus(user *common.User) error {
	log.Printf("DEPLETED_STATUS_CHECKER: Исправление ложного состояния 'исчерпано' для пользователя %d", user.TelegramID)

	// Получаем сессию для работы с панелью
	sessionCookie, err := common.Login()
	if err != nil {
		return fmt.Errorf("ошибка авторизации в панели: %v", err)
	}

	// Используем существующую функцию принудительного сброса состояния "исчерпано"
	if err := common.ForceResetDepletedStatus(sessionCookie, user.TelegramID); err != nil {
		return fmt.Errorf("ошибка сброса состояния 'исчерпано': %v", err)
	}

	log.Printf("DEPLETED_STATUS_CHECKER: Состояние 'исчерпано' успешно сброшено для пользователя %d", user.TelegramID)
	return nil
}

// ForceCheckDepletedStatus принудительно запускает проверку ложных состояний "исчерпано"
func ForceCheckDepletedStatus() {
	log.Printf("DEPLETED_STATUS_CHECKER: Принудительный запуск проверки ложных состояний 'исчерпано'")

	checker := NewDepletedStatusChecker()
	checker.CheckAndFixDepletedStatus()
}
