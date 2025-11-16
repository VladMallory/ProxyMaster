package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"bot/common"
)

// ResetStatusChecker сервис для проверки и исправления состояния reset
type ResetStatusChecker struct {
	ticker *time.Ticker
	done   chan bool
}

// NewResetStatusChecker создает новый экземпляр сервиса проверки состояния reset
func NewResetStatusChecker() *ResetStatusChecker {
	return &ResetStatusChecker{
		done: make(chan bool),
	}
}

// Start запускает сервис проверки состояния reset
func (rsc *ResetStatusChecker) Start() {
	if !common.RESET_STATUS_CHECK_ENABLED {
		log.Printf("RESET_STATUS_CHECKER: Проверка состояния reset отключена")
		return
	}

	interval := time.Duration(common.RESET_STATUS_CHECK_INTERVAL) * time.Minute
	log.Printf("RESET_STATUS_CHECKER: Запуск сервиса проверки состояния reset с интервалом %v", interval)
	common.LogServiceStart("RESET_STATUS_CHECKER", common.RESET_STATUS_CHECK_INTERVAL)

	rsc.ticker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-rsc.ticker.C:
				rsc.CheckAndFixResetStatus()
			case <-rsc.done:
				log.Printf("RESET_STATUS_CHECKER: Сервис остановлен")
				common.LogServiceStop("RESET_STATUS_CHECKER")
				return
			}
		}
	}()
}

// Stop останавливает сервис проверки состояния reset
func (rsc *ResetStatusChecker) Stop() {
	if rsc.ticker != nil {
		rsc.ticker.Stop()
	}
	close(rsc.done)
}

// CheckAndFixResetStatus проверяет и исправляет состояние reset для всех пользователей
func (rsc *ResetStatusChecker) CheckAndFixResetStatus() {
	log.Printf("RESET_STATUS_CHECKER: ===== НАЧАЛО ПРОВЕРКИ СОСТОЯНИЯ RESET =====")

	// Получаем всех пользователей с активными конфигами
	users, err := common.GetAllUsers()
	if err != nil {
		log.Printf("RESET_STATUS_CHECKER: Ошибка получения пользователей: %v", err)
		return
	}

	// Фильтруем только пользователей с активными конфигами
	var activeUsers []*common.User
	for _, user := range users {
		if user.HasActiveConfig && user.Balance > 0 {
			activeUsers = append(activeUsers, &user)
		}
	}

	log.Printf("RESET_STATUS_CHECKER: Найдено %d пользователей с активными конфигами", len(activeUsers))

	if len(activeUsers) == 0 {
		log.Printf("RESET_STATUS_CHECKER: Нет пользователей для проверки")
		return
	}

	// Авторизуемся в панели
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("RESET_STATUS_CHECKER: Ошибка авторизации в панели: %v", err)
		return
	}

	// Получаем данные inbound
	inbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Printf("RESET_STATUS_CHECKER: Ошибка получения inbound: %v", err)
		return
	}

	// Парсим settings
	var settings common.Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Printf("RESET_STATUS_CHECKER: Ошибка парсинга settings: %v", err)
		return
	}

	log.Printf("RESET_STATUS_CHECKER: Получено %d клиентов из панели", len(settings.Clients))

	fixedCount := 0
	checkedCount := 0

	// Проверяем каждого пользователя
	for _, user := range activeUsers {
		checkedCount++
		log.Printf("RESET_STATUS_CHECKER: Проверка пользователя %d (%s)", user.TelegramID, user.FirstName)

		// Ищем клиента в панели
		clientIndex := rsc.findClientInPanel(settings.Clients, user)
		if clientIndex == -1 {
			log.Printf("RESET_STATUS_CHECKER: Клиент пользователя %d не найден в панели", user.TelegramID)
			continue
		}

		client := &settings.Clients[clientIndex]

		// Проверяем, есть ли проблема с reset
		hasResetIssue := rsc.checkResetIssues(client, user)
		if !hasResetIssue {
			log.Printf("RESET_STATUS_CHECKER: Пользователь %d в нормальном состоянии", user.TelegramID)
			continue
		}

		// Исправляем проблемы
		log.Printf("RESET_STATUS_CHECKER: Исправление проблем для пользователя %d", user.TelegramID)
		rsc.fixResetIssues(client, user)
		fixedCount++
	}

	// Обновляем inbound если были изменения
	if fixedCount > 0 {
		log.Printf("RESET_STATUS_CHECKER: Обновление inbound в панели...")
		settingsJSON, err := json.Marshal(settings)
		if err != nil {
			log.Printf("RESET_STATUS_CHECKER: Ошибка сериализации settings: %v", err)
			return
		}
		inbound.Settings = string(settingsJSON)

		err = common.UpdateInbound(sessionCookie, *inbound)
		if err != nil {
			log.Printf("RESET_STATUS_CHECKER: Ошибка обновления inbound: %v", err)
			return
		}

		log.Printf("RESET_STATUS_CHECKER: Inbound успешно обновлен")
	}

	log.Printf("RESET_STATUS_CHECKER: ===== ПРОВЕРКА ЗАВЕРШЕНА =====")
	log.Printf("RESET_STATUS_CHECKER: Проверено пользователей: %d, исправлено: %d", checkedCount, fixedCount)
}

// findClientInPanel находит клиента в панели по данным пользователя
func (rsc *ResetStatusChecker) findClientInPanel(clients []common.Client, user *common.User) int {
	for i, client := range clients {
		// Ищем по ClientID
		if client.ID == user.ClientID {
			return i
		}
		// Ищем по SubID
		if client.SubID == user.SubID {
			return i
		}
		// Ищем по email (включая варианты с -reset)
		if client.Email == user.Email || client.Email == user.Email+"-reset" {
			return i
		}
	}
	return -1
}

// checkResetIssues проверяет, есть ли проблемы с состоянием reset
func (rsc *ResetStatusChecker) checkResetIssues(client *common.Client, user *common.User) bool {
	hasIssues := false

	// Проверяем email с суффиксом -reset
	if strings.HasSuffix(client.Email, "-reset") {
		log.Printf("RESET_STATUS_CHECKER: Пользователь %d: Email содержит суффикс '-reset'", user.TelegramID)
		hasIssues = true
	}

	// ВАЖНО: НЕ проверяем флаги Depleted и Exhausted как проблемы!
	// Эти флаги могут быть true по законным причинам (превышение лимита трафика)
	// и должны сбрасываться только при периодическом сбросе трафика (раз в 7 дней).

	// Проверяем, что клиент отключен (это может быть ошибка)
	if !client.Enable {
		// Только если безопасно включать, считаем это проблемой
		if shouldEnable, _ := rsc.shouldEnableClient(user, client); shouldEnable {
			log.Printf("RESET_STATUS_CHECKER: Пользователь %d: Enable = false и безопасно включать", user.TelegramID)
			hasIssues = true
		}
	}

	// Проверяем несоответствие времени истечения
	if client.ExpiryTime != user.ExpiryTime {
		log.Printf("RESET_STATUS_CHECKER: Пользователь %d: Несоответствие времени истечения (панель: %d, база: %d)",
			user.TelegramID, client.ExpiryTime, user.ExpiryTime)
		hasIssues = true
	}

	return hasIssues
}

// fixResetIssues исправляет проблемы с состоянием reset
func (rsc *ResetStatusChecker) fixResetIssues(client *common.Client, user *common.User) {
	// Восстанавливаем правильный email
	originalEmail := fmt.Sprintf("%d", user.TelegramID)
	if client.Email != originalEmail {
		log.Printf("RESET_STATUS_CHECKER: Восстанавливаем email: %s -> %s", client.Email, originalEmail)
		common.LogClientOperation("RESET_STATUS_CHECKER", user.TelegramID, client.Email, "Восстановление email")
		client.Email = originalEmail
	}

	// ВАЖНО: НЕ сбрасываем флаги Depleted и Exhausted - это может влиять на трафик!
	// Эти флаги должны сбрасываться только при периодическом сбросе трафика (раз в 7 дней).
	// Здесь мы только включаем клиента, если он был отключен по ошибке.
	if !client.Enable {
		// Защитные проверки, чтобы избежать циклов включения/отключения
		shouldEnable, reason := rsc.shouldEnableClient(user, client)
		if !shouldEnable {
			log.Printf("RESET_STATUS_CHECKER: Пропускаем включение клиента %s: %s", client.Email, reason)
			common.LogClientOperation("RESET_STATUS_CHECKER", user.TelegramID, client.Email, "Пропуск включения: "+reason)
		} else {
			log.Printf("RESET_STATUS_CHECKER: Включаем клиента: %s", client.Email)
			common.LogClientOperation("RESET_STATUS_CHECKER", user.TelegramID, client.Email, "Включение отключенного клиента")
			client.Enable = true
		}
	}

	// Восстанавливаем время истечения из базы данных
	if client.ExpiryTime != user.ExpiryTime {
		log.Printf("RESET_STATUS_CHECKER: Восстанавливаем время истечения: %d -> %d", client.ExpiryTime, user.ExpiryTime)
		common.LogConfigChange("RESET_STATUS_CHECKER", user.TelegramID,
			fmt.Sprintf("ExpiryTime=%d", client.ExpiryTime),
			fmt.Sprintf("ExpiryTime=%d", user.ExpiryTime))
		client.ExpiryTime = user.ExpiryTime
	}

	// Обновляем время последнего изменения
	client.UpdatedAt = time.Now().UnixMilli()

	log.Printf("RESET_STATUS_CHECKER: Проблемы исправлены для пользователя %d (БЕЗ сброса флагов трафика)", user.TelegramID)
	common.LogClientOperation("RESET_STATUS_CHECKER", user.TelegramID, client.Email, "Проблемы исправлены БЕЗ сброса флагов трафика")
}

// shouldEnableClient определяет, безопасно ли включать отключённого клиента
func (rsc *ResetStatusChecker) shouldEnableClient(user *common.User, client *common.Client) (bool, string) {
	// Если в БД конфиг помечен как неактивный, не включаем
	if !user.HasActiveConfig {
		return false, "в БД HasActiveConfig=false"
	}

	// Проверяем достаточность баланса (минимум на 1 день)
	if user.Balance < float64(common.PRICE_PER_DAY) {
		return false, fmt.Sprintf("недостаточный баланс %.2f₽ < %d₽", user.Balance, common.PRICE_PER_DAY)
	}

	// Не включаем истёкшие подписки
	nowMs := time.Now().UnixMilli()
	if user.ExpiryTime > 0 && user.ExpiryTime <= nowMs {
		return false, "подписка истекла"
	}

	// Если клиент в состоянии исчерпан/выжат — не считаем это проблемой ResetStatusChecker
	if client.Depleted != nil && *client.Depleted {
		return false, "клиент в состоянии 'depleted'"
	}
	if client.Exhausted != nil && *client.Exhausted {
		return false, "клиент в состоянии 'exhausted'"
	}

	return true, ""
}
