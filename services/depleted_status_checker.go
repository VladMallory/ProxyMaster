package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
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
// Теперь использует оптимизированную версию с батчингом по умолчанию
func (dsc *DepletedStatusChecker) CheckAndFixDepletedStatus() {
	// Используем оптимизированную версию с батчингом
	dsc.CheckAndFixDepletedStatusOptimized()
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

	// ОСНОВНАЯ ПРОВЕРКА: Если подписка активна и до истечения больше часа,
	// то состояние "исчерпано" должно быть сброшено
	if user.ExpiryTime > 0 {
		timeUntilExpiry := time.Duration(user.ExpiryTime-now.UnixMilli()) * time.Millisecond
		if timeUntilExpiry > 1*time.Hour {
			log.Printf("DEPLETED_STATUS_CHECKER: У пользователя %d есть активная подписка (осталось %v), сбрасываем состояние 'исчерпано'",
				user.TelegramID, timeUntilExpiry.Round(time.Hour))
			return true // Исправляем состояние "исчерпано"
		} else {
			log.Printf("DEPLETED_STATUS_CHECKER: У пользователя %d подписка истекает через %v, пропускаем исправление",
				user.TelegramID, timeUntilExpiry.Round(time.Minute))
			return false // Не исправляем, подписка скоро истекает
		}
	}

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

			// ВАЖНО: Если конфиг отключен в панели (Enable=false), НЕ сбрасываем статус "исчерпано"
			// Это может быть из-за системы бана по трафику или других причин
			if !client.Enable {
				log.Printf("DEPLETED_STATUS_CHECKER: Конфиг пользователя %d отключен в панели (Enable=false), не сбрасываем статус 'исчерпано'",
					user.TelegramID)
				return false
			}

			// Возвращаем true если конфиг в состоянии "исчерпано" И включен в панели
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

// fixMultipleDepletedStatus исправляет состояния "исчерпано" для нескольких пользователей за один раз
func (dsc *DepletedStatusChecker) fixMultipleDepletedStatus(sessionCookie string, settings *common.Settings, usersToFix []*common.User) int {
	if len(usersToFix) == 0 {
		return 0
	}

	log.Printf("DEPLETED_STATUS_CHECKER: Начинаем групповое исправление для %d пользователей", len(usersToFix))

	modified := false
	fixedCount := 0

	// Создаем карту для быстрого поиска пользователей
	userMap := make(map[string]*common.User)
	for _, user := range usersToFix {
		userMap[fmt.Sprintf("%d", user.TelegramID)] = user
	}

	// Проходим по всем клиентам и исправляем тех, кто в списке
	for i, client := range settings.Clients {
		// Ищем пользователя по email (который содержит TelegramID)
		var foundUser *common.User
		for _, user := range usersToFix {
			telegramIDStr := fmt.Sprintf("%d", user.TelegramID)
			if strings.HasPrefix(client.Email, telegramIDStr+"_") ||
				strings.HasPrefix(client.Email, telegramIDStr+" ") ||
				client.Email == telegramIDStr {
				foundUser = user
				break
			}
		}

		// Если нашли пользователя для исправления
		if foundUser != nil {
			// Проверяем, что клиент включен и имеет активную подписку
			if client.Enable && foundUser.ExpiryTime > time.Now().UnixMilli() {
				// Сбрасываем флаги depleted и exhausted
				falseValue := false
				settings.Clients[i].Depleted = &falseValue
				settings.Clients[i].Exhausted = &falseValue
				settings.Clients[i].UpdatedAt = time.Now().UnixMilli()

				log.Printf("DEPLETED_STATUS_CHECKER: Сброшены флаги depleted/exhausted для пользователя %d", foundUser.TelegramID)
				common.LogClientOperation("DEPLETED_STATUS_CHECKER", foundUser.TelegramID, client.Email, "Групповой сброс флагов depleted/exhausted")

				modified = true
				fixedCount++
			}
		}
	}

	if modified {
		log.Printf("DEPLETED_STATUS_CHECKER: Групповое исправление завершено. Исправлено пользователей: %d", fixedCount)
	}

	return fixedCount
}

// CheckAndFixDepletedStatusOptimized оптимизированная версия с батчингом
func (dsc *DepletedStatusChecker) CheckAndFixDepletedStatusOptimized() {
	log.Printf("DEPLETED_STATUS_CHECKER: Начинаем оптимизированную проверку ложных состояний 'исчерпано'...")

	// Получаем всех пользователей с активными конфигами
	users, err := common.GetUsersWithActiveConfigs()
	if err != nil {
		log.Printf("DEPLETED_STATUS_CHECKER: Ошибка получения пользователей: %v", err)
		return
	}

	if len(users) == 0 {
		log.Printf("DEPLETED_STATUS_CHECKER: Нет пользователей для проверки")
		return
	}

	// Получаем сессию ОДИН РАЗ
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("DEPLETED_STATUS_CHECKER: Ошибка авторизации в панели: %v", err)
		return
	}

	// Получаем inbound ОДИН РАЗ
	inbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Printf("DEPLETED_STATUS_CHECKER: Ошибка получения inbound: %v", err)
		return
	}

	// Парсим settings ОДИН РАЗ
	var settings common.Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Printf("DEPLETED_STATUS_CHECKER: Ошибка парсинга settings: %v", err)
		return
	}

	now := time.Now()
	var usersToFix []*common.User

	// Находим всех пользователей для исправления (без API вызовов)
	for _, user := range users {
		if dsc.shouldFixDepletedStatusOptimized(&user, now, &settings) {
			usersToFix = append(usersToFix, &user)
		}
	}

	if len(usersToFix) == 0 {
		log.Printf("DEPLETED_STATUS_CHECKER: Ложных состояний 'исчерпано' не найдено")
		return
	}

	// Исправляем ВСЕХ пользователей в группе
	fixedCount := dsc.fixMultipleDepletedStatus(sessionCookie, &settings, usersToFix)

	// Обновляем inbound ОДИН РАЗ, если были изменения
	if fixedCount > 0 {
		log.Printf("DEPLETED_STATUS_CHECKER: Обновление inbound с исправленными пользователями...")
		settingsJSON, err := json.Marshal(settings)
		if err != nil {
			log.Printf("DEPLETED_STATUS_CHECKER: Ошибка сериализации settings: %v", err)
			return
		}
		inbound.Settings = string(settingsJSON)

		if err := common.UpdateInbound(sessionCookie, *inbound); err != nil {
			log.Printf("DEPLETED_STATUS_CHECKER: Ошибка обновления inbound: %v", err)
			return
		}

		log.Printf("DEPLETED_STATUS_CHECKER: Inbound успешно обновлен")
	}

	log.Printf("DEPLETED_STATUS_CHECKER: Оптимизированная проверка завершена. Исправлено ложных состояний 'исчерпано': %d", fixedCount)
}

// shouldFixDepletedStatusOptimized оптимизированная версия без API вызовов
func (dsc *DepletedStatusChecker) shouldFixDepletedStatusOptimized(user *common.User, now time.Time, settings *common.Settings) bool {
	// Проверяем, что у пользователя есть активная подписка
	if !user.HasActiveConfig {
		return false
	}

	// Проверяем, что подписка не истекла
	if user.ExpiryTime > 0 && user.ExpiryTime <= now.UnixMilli() {
		return false
	}

	// ОСНОВНАЯ ПРОВЕРКА: Если подписка активна и до истечения больше часа,
	// то состояние "исчерпано" должно быть сброшено
	if user.ExpiryTime > 0 {
		timeUntilExpiry := time.Duration(user.ExpiryTime-now.UnixMilli()) * time.Millisecond
		if timeUntilExpiry > 1*time.Hour {
			// Ищем клиента в уже загруженных settings
			telegramIDStr := fmt.Sprintf("%d", user.TelegramID)
			for _, client := range settings.Clients {
				if strings.HasPrefix(client.Email, telegramIDStr+"_") ||
					strings.HasPrefix(client.Email, telegramIDStr+" ") ||
					client.Email == telegramIDStr {

					// Проверяем состояние "исчерпано" в загруженных данных
					isDepleted := client.Depleted != nil && *client.Depleted
					isExhausted := client.Exhausted != nil && *client.Exhausted

					// ВАЖНО: Если конфиг отключен в панели, НЕ сбрасываем статус "исчерпано"
					if !client.Enable {
						return false
					}

					// Возвращаем true если конфиг в состоянии "исчерпано" И включен в панели
					return isDepleted || isExhausted
				}
			}
		} else {
			return false // Не исправляем, подписка скоро истекает
		}
	}

	return false
}

// CheckAndFixDepletedStatusParallel параллельная обработка с горутинами
func (dsc *DepletedStatusChecker) CheckAndFixDepletedStatusParallel() {
	log.Printf("DEPLETED_STATUS_CHECKER: Начинаем параллельную проверку ложных состояний 'исчерпано'...")

	// Получаем всех пользователей с активными конфигами
	users, err := common.GetUsersWithActiveConfigs()
	if err != nil {
		log.Printf("DEPLETED_STATUS_CHECKER: Ошибка получения пользователей: %v", err)
		return
	}

	if len(users) == 0 {
		log.Printf("DEPLETED_STATUS_CHECKER: Нет пользователей для проверки")
		return
	}

	// Создаем каналы для параллельной обработки
	const maxConcurrency = 10 // Максимум 10 одновременных горутин
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mutex sync.Mutex
	fixedCount := 0

	now := time.Now()

	log.Printf("DEPLETED_STATUS_CHECKER: Запускаем параллельную обработку %d пользователей с максимальной параллельностью %d", len(users), maxConcurrency)

	for _, user := range users {
		wg.Add(1)
		go func(u common.User) {
			defer wg.Done()

			// Захватываем слот семафора
			semaphore <- struct{}{}
			defer func() { <-semaphore }() // Освобождаем слот

			// Проверяем, нужно ли исправить состояние "исчерпано" для этого пользователя
			if dsc.shouldFixDepletedStatus(&u, now) {
				log.Printf("DEPLETED_STATUS_CHECKER: Найдено ложное состояние 'исчерпано' для пользователя %d (email: %s)",
					u.TelegramID, u.Email)

				// Исправляем состояние "исчерпано"
				if err := dsc.fixDepletedStatus(&u); err != nil {
					log.Printf("DEPLETED_STATUS_CHECKER: Ошибка исправления состояния 'исчерпано' для пользователя %d: %v",
						u.TelegramID, err)
				} else {
					mutex.Lock()
					fixedCount++
					mutex.Unlock()
					log.Printf("DEPLETED_STATUS_CHECKER: Состояние 'исчерпано' успешно исправлено для пользователя %d", u.TelegramID)
				}
			}
		}(user)
	}

	// Ждем завершения всех горутин
	wg.Wait()

	if fixedCount > 0 {
		log.Printf("DEPLETED_STATUS_CHECKER: Параллельная проверка завершена. Исправлено ложных состояний 'исчерпано': %d", fixedCount)
	} else {
		log.Printf("DEPLETED_STATUS_CHECKER: Параллельная проверка завершена. Ложных состояний 'исчерпано' не найдено")
	}
}

// ForceCheckDepletedStatus принудительно запускает проверку ложных состояний "исчерпано"
func ForceCheckDepletedStatus() {
	log.Printf("DEPLETED_STATUS_CHECKER: Принудительный запуск проверки ложных состояний 'исчерпано'")

	checker := NewDepletedStatusChecker()
	checker.CheckAndFixDepletedStatusOptimized() // Используем оптимизированную версию
}
