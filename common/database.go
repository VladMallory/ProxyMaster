package common

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

// Константы для совместимости (теперь используется PostgreSQL)
const (
	MONGO_URI     = "mongodb://localhost:27017" // Для обратной совместимости
	MONGO_DB_NAME = "vpn_bot"                   // Для обратной совместимости
)

// InitMongoDB инициализирует подключение к базе данных (теперь PostgreSQL)
func InitMongoDB() error {
	// Переадресация к PostgreSQL
	return InitPostgreSQL()
}

// logUsersAfterConnection выводит информацию о пользователях после подключения к базе данных
func logUsersAfterConnection() {
	// Переадресация к PostgreSQL
	logUsersAfterConnectionPG()
}

// DisconnectMongoDB отключается от базы данных (теперь PostgreSQL)
func DisconnectMongoDB() {
	// Переадресация к PostgreSQL
	DisconnectPostgreSQL()
}

// GetDatabase возвращает объект базы данных (для совместимости)
func GetDatabase() interface{} {
	// Возвращаем PostgreSQL соединение
	return GetDatabasePG()
}

// GetOrCreateUser получает или создает пользователя
func GetOrCreateUser(telegramID int64, username, firstName, lastName string) (*User, error) {
	// Переадресация к PostgreSQL
	return GetOrCreateUserPG(telegramID, username, firstName, lastName)
}

// GetUserByTelegramID получает пользователя по Telegram ID
func GetUserByTelegramID(telegramID int64) (*User, error) {
	// Переадресация к PostgreSQL
	return GetUserByTelegramIDPG(telegramID)
}

// GetAllUsers получает всех пользователей
func GetAllUsers() ([]User, error) {
	// Переадресация к PostgreSQL
	return GetAllUsersPG()
}

// GetUsersWithActiveConfigs получает всех пользователей с активными конфигами
func GetUsersWithActiveConfigs() ([]User, error) {
	// Переадресация к PostgreSQL
	return GetUsersWithActiveConfigsPG()
}

// AddBalance добавляет баланс пользователю
func AddBalance(telegramID int64, amount float64) error {
	// Переадресация к PostgreSQL
	return AddBalancePG(telegramID, amount)
}

// UpdateTrialFlag обновляет флаг использования пробного периода
func UpdateTrialFlag(telegramID int64) error {
	// Переадресация к PostgreSQL
	return UpdateTrialFlagPG(telegramID)
}

// ResetTrialFlag сбрасывает флаг использования пробного периода
func ResetTrialFlag(telegramID int64) error {
	// Переадресация к PostgreSQL
	return ResetTrialFlagPG(telegramID)
}

// ClearAllUsers удаляет всех пользователей
func ClearAllUsers() error {
	// Переадресация к PostgreSQL
	return ClearAllUsersPG()
}

// UpdateUser обновляет данные пользователя
func UpdateUser(user *User) error {
	// Переадресация к PostgreSQL
	return UpdateUserPG(user)
}

// ClearDatabase очищает всю базу данных
func ClearDatabase() error {
	// Переадресация к PostgreSQL
	return ClearDatabasePG()
}

// BackupMongoDB создает бэкап базы данных (теперь PostgreSQL)
func BackupMongoDB() error {
	// Переадресация к PostgreSQL
	return BackupPostgreSQLPG()
}

// RestoreMongoDB восстанавливает базу данных из бэкапа (теперь PostgreSQL)
func RestoreMongoDB() error {
	// Переадресация к PostgreSQL
	return RestorePostgreSQLPG()
}

// ProcessPayment обрабатывает платеж
func ProcessPayment(user *User, days int) (string, error) {
	log.Printf("PROCESS_PAYMENT: Начало обработки платежа для TelegramID=%d, days=%d", user.TelegramID, days)

	cost := float64(days * PRICE_PER_DAY)
	log.Printf("PROCESS_PAYMENT: Расчёт стоимости: TelegramID=%d, days=%d, balance=%.2f, cost=%.2f", user.TelegramID, days, user.Balance, cost)

	// Проверяем баланс
	if user.Balance < cost {
		log.Printf("PROCESS_PAYMENT: Недостаточно средств для TelegramID=%d, Balance=%.2f, Cost=%.2f", user.TelegramID, user.Balance, cost)
		return "", fmt.Errorf("недостаточно средств на балансе. Нужно: %.2f₽, доступно: %.2f₽", cost, user.Balance)
	}

	// Создаем конфиг через панель 3x-ui
	sessionCookie, err := Login()
	if err != nil {
		log.Printf("PROCESS_PAYMENT: Ошибка авторизации в панели для TelegramID=%d: %v", user.TelegramID, err)
		return "", fmt.Errorf("ошибка авторизации в панели: %v", err)
	}

	err = AddClient(sessionCookie, user, days)
	if err != nil {
		log.Printf("PROCESS_PAYMENT: Ошибка создания конфига для TelegramID=%d: %v", user.TelegramID, err)
		return "", fmt.Errorf("ошибка создания конфига: %v", err)
	}

	// Принудительно сбрасываем состояние "исчерпано" после создания/продления
	log.Printf("PROCESS_PAYMENT: Принудительный сброс состояния 'исчерпано' для TelegramID=%d", user.TelegramID)
	if err := ForceResetDepletedStatus(sessionCookie, user.TelegramID); err != nil {
		log.Printf("PROCESS_PAYMENT: Предупреждение - не удалось сбросить состояние 'исчерпано' для TelegramID=%d: %v", user.TelegramID, err)
		// Не возвращаем ошибку, так как основная операция уже выполнена
	} else {
		log.Printf("PROCESS_PAYMENT: Состояние 'исчерпано' успешно сброшено для TelegramID=%d", user.TelegramID)
	}

	// Списываем деньги с баланса
	user.Balance -= cost
	log.Printf("PROCESS_PAYMENT: Деньги списаны с баланса: TelegramID=%d, списано=%.2f, остаток=%.2f", user.TelegramID, cost, user.Balance)

	// Обновляем данные пользователя в базе
	if err := UpdateUser(user); err != nil {
		log.Printf("PROCESS_PAYMENT: Ошибка обновления пользователя: %v", err)
		return "", fmt.Errorf("ошибка обновления пользователя: %v", err)
	}

	configURL := fmt.Sprintf("%s%s", CONFIG_BASE_URL, user.SubID)
	log.Printf("PROCESS_PAYMENT: Конфиг успешно создан для TelegramID=%d, ConfigURL=%s", user.TelegramID, configURL)

	// Проверяем, нужно ли отправить уведомление о подписке
	if NOTIFICATION_ENABLED && GlobalBot != nil {
		go checkUserSubscriptionNotification(user)
	}

	return configURL, nil
}

// checkUserSubscriptionNotification проверяет подписку пользователя и отправляет уведомление при необходимости
func checkUserSubscriptionNotification(user *User) {
	if !NOTIFICATION_ENABLED || GlobalBot == nil {
		return
	}

	now := time.Now()

	// Проверяем, что у пользователя есть активная подписка
	if !user.HasActiveConfig || user.ExpiryTime <= 0 {
		return
	}

	// Проверяем, что подписка еще не истекла
	if user.ExpiryTime <= now.UnixMilli() {
		return
	}

	// Вычисляем количество дней до истечения
	expiry := time.UnixMilli(user.ExpiryTime)
	diff := expiry.Sub(now)
	daysLeft := int(diff.Hours() / 24)

	// Если осталось меньше дня, но больше 0, считаем как 1 день
	if daysLeft == 0 && diff > 0 {
		daysLeft = 1
	}

	// Проверяем, есть ли этот день в списке дней для уведомлений
	shouldNotify := false
	for _, day := range NOTIFICATION_DAYS_BEFORE {
		if daysLeft == day {
			shouldNotify = true
			break
		}
	}

	if !shouldNotify {
		return
	}

	// Получаем сообщение для уведомления
	var message string
	switch daysLeft {
	case 1:
		message = NOTIFICATION_MESSAGE_1_DAY
	case 3:
		message = NOTIFICATION_MESSAGE_3_DAYS
	case 7:
		message = NOTIFICATION_MESSAGE_7_DAYS
	default:
		return
	}

	// Отправляем уведомление
	msg := tgbotapi.NewMessage(user.TelegramID, message)
	msg.ParseMode = tgbotapi.ModeHTML

	_, err := GlobalBot.Send(msg)
	if err != nil {
		log.Printf("NOTIFICATION: Ошибка отправки уведомления пользователю %d: %v", user.TelegramID, err)
	} else {
		log.Printf("NOTIFICATION: Уведомление отправлено пользователю %d (осталось %d дней)", user.TelegramID, daysLeft)
	}
}

// CheckUserSubscriptionNotification проверяет подписку пользователя и отправляет уведомление при необходимости
// Эта функция экспортируется для использования в других пакетах
func CheckUserSubscriptionNotification(user *User) {
	checkUserSubscriptionNotification(user)
}

// ResetAllTrialFlags сбрасывает флаги пробных периодов для всех пользователей
func ResetAllTrialFlags() error {
	// Переадресация к PostgreSQL
	return ResetAllTrialFlagsPG()
}

// GetTrafficConfig получает конфигурацию трафика
func GetTrafficConfig() *TrafficConfig {
	// Переадресация к PostgreSQL
	return GetTrafficConfigPG()
}

// SetTrafficConfig сохраняет конфигурацию трафика
func SetTrafficConfig(config *TrafficConfig) error {
	// Переадресация к PostgreSQL
	return SetTrafficConfigPG(config)
}

// GetClientTrafficStats получает статистику трафика для всех клиентов
func GetClientTrafficStats(sessionCookie string) ([]TrafficStats, error) {
	log.Printf("GET_CLIENT_TRAFFIC_STATS: Получение статистики трафика")

	req, err := http.NewRequest("GET", fmt.Sprintf("%spanel/api/inbounds/list", PANEL_URL), nil)
	if err != nil {
		log.Printf("GET_CLIENT_TRAFFIC_STATS: Ошибка создания запроса: %v", err)
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}
	req.Header.Set("Cookie", sessionCookie)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("GET_CLIENT_TRAFFIC_STATS: Ошибка выполнения запроса: %v", err)
		return nil, fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("GET_CLIENT_TRAFFIC_STATS: Ошибка чтения ответа: %v", err)
		return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("GET_CLIENT_TRAFFIC_STATS: Некорректный статус ответа: %d", resp.StatusCode)
		return nil, fmt.Errorf("некорректный статус ответа: %d", resp.StatusCode)
	}

	var inboundListResponse struct {
		Success bool      `json:"success"`
		Msg     string    `json:"msg"`
		Obj     []Inbound `json:"obj"`
	}

	if err := json.Unmarshal(body, &inboundListResponse); err != nil {
		log.Printf("GET_CLIENT_TRAFFIC_STATS: Ошибка десериализации ответа: %v", err)
		return nil, fmt.Errorf("ошибка десериализации ответа: %v", err)
	}

	if !inboundListResponse.Success {
		log.Printf("GET_CLIENT_TRAFFIC_STATS: Запрос не успешен: %s", inboundListResponse.Msg)
		return nil, fmt.Errorf("запрос не успешен: %s", inboundListResponse.Msg)
	}

	// Ищем нужный inbound
	var targetInbound *Inbound
	for _, inbound := range inboundListResponse.Obj {
		if inbound.ID == INBOUND_ID {
			targetInbound = &inbound
			break
		}
	}

	if targetInbound == nil {
		log.Printf("GET_CLIENT_TRAFFIC_STATS: Inbound с ID=%d не найден", INBOUND_ID)
		return nil, fmt.Errorf("inbound с ID=%d не найден", INBOUND_ID)
	}

	// Извлекаем статистику клиентов из clientStats
	var clientStats []TrafficStats
	if targetInbound.ClientStats != nil {
		statsData, err := json.Marshal(targetInbound.ClientStats)
		if err != nil {
			log.Printf("GET_CLIENT_TRAFFIC_STATS: Ошибка сериализации clientStats: %v", err)
			return nil, fmt.Errorf("ошибка сериализации clientStats: %v", err)
		}

		if err := json.Unmarshal(statsData, &clientStats); err != nil {
			log.Printf("GET_CLIENT_TRAFFIC_STATS: Ошибка десериализации clientStats: %v", err)
			return nil, fmt.Errorf("ошибка десериализации clientStats: %v", err)
		}
	}

	log.Printf("GET_CLIENT_TRAFFIC_STATS: Получено %d записей статистики", len(clientStats))
	return clientStats, nil
}

// CheckAndDisableTrafficLimit проверяет трафик и отключает/включает клиентов
func CheckAndDisableTrafficLimit() error {
	log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Начало проверки трафика")

	// Если лимит трафика не установлен, пропускаем проверку
	if TRAFFIC_LIMIT_GB <= 0 {
		log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Лимит трафика не установлен (TRAFFIC_LIMIT_GB=%d), пропускаем проверку", TRAFFIC_LIMIT_GB)
		return nil
	}

	// Авторизуемся в панели
	sessionCookie, err := Login()
	if err != nil {
		log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Ошибка авторизации: %v", err)
		return fmt.Errorf("ошибка авторизации в панели: %v", err)
	}

	// Получаем статистику трафика
	trafficStats, err := GetClientTrafficStats(sessionCookie)
	if err != nil {
		log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Ошибка получения статистики трафика: %v", err)
		return fmt.Errorf("ошибка получения статистики трафика: %v", err)
	}

	// Получаем inbound для обновления
	inbound, err := GetInbound(sessionCookie)
	if err != nil {
		log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Ошибка получения inbound: %v", err)
		return fmt.Errorf("ошибка получения inbound: %v", err)
	}

	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Ошибка десериализации settings: %v", err)
		return fmt.Errorf("ошибка десериализации settings: %v", err)
	}

	// Создаем карту статистики по email клиента
	statsMap := make(map[string]TrafficStats)
	for _, stat := range trafficStats {
		statsMap[stat.Email] = stat
	}

	// Проверяем каждого клиента
	modified := false
	trafficLimitBytes := int64(TRAFFIC_LIMIT_GB) * 1024 * 1024 * 1024 // конвертируем ГБ в байты
	disabledCount := 0
	enabledCount := 0

	for i, client := range settings.Clients {
		if !client.Enable {
			continue // пропускаем уже отключенных клиентов
		}

		// Ищем статистику для этого клиента
		stat, exists := statsMap[client.Email]
		if !exists {
			log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Статистика не найдена для клиента ID=%s, Email=%s", client.ID, client.Email)
			continue
		}

		// Вычисляем общий трафик (up + down)
		totalTraffic := stat.Up + stat.Down

		// Проверяем лимит трафика
		if totalTraffic > trafficLimitBytes {
			log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Превышен лимит трафика для клиента ID=%s, Email=%s, Up=%d bytes (%.2f GB), Down=%d bytes (%.2f GB), Total=%d bytes (%.2f GB), Limit=%d bytes (%.2f GB)",
				client.ID, client.Email, stat.Up, float64(stat.Up)/1024/1024/1024, stat.Down, float64(stat.Down)/1024/1024/1024, totalTraffic, float64(totalTraffic)/1024/1024/1024, trafficLimitBytes, float64(trafficLimitBytes)/1024/1024/1024)

			// Отключаем клиента и ротируем UUID для немедленного обрыва активных сессий
			settings.Clients[i].Enable = false
			newUUID := uuid.New().String()
			settings.Clients[i].ID = newUUID
			modified = true
			disabledCount++

			log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Клиент отключен и UUID обновлен: Email=%s, OldUUID=%s, NewUUID=%s",
				client.Email, client.ID, newUUID)

			// Обновляем статус в базе данных
			updateUserTrafficStatus(client.Email, false)
		} else {
			// Проверяем, нужно ли включить клиента обратно
			if !client.Enable && totalTraffic <= trafficLimitBytes {
				log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Включаем клиента обратно (трафик в норме) ID=%s, Email=%s, Up=%d bytes (%.2f GB), Down=%d bytes (%.2f GB), Total=%d bytes (%.2f GB), Limit=%d bytes (%.2f GB)",
					client.ID, client.Email, stat.Up, float64(stat.Up)/1024/1024/1024, stat.Down, float64(stat.Down)/1024/1024/1024, totalTraffic, float64(totalTraffic)/1024/1024/1024, trafficLimitBytes, float64(trafficLimitBytes)/1024/1024/1024)

				// Включаем клиента обратно и генерируем новый UUID
				settings.Clients[i].Enable = true
				newUUID := uuid.New().String()
				settings.Clients[i].ID = newUUID
				modified = true
				enabledCount++

				log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Клиент включен обратно с новым UUID: Email=%s, NewUUID=%s",
					client.Email, newUUID)

				// Обновляем статус в базе данных
				updateUserTrafficStatus(client.Email, true)
			} else {
				log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Трафик в норме для клиента ID=%s, Email=%s, Up=%d bytes (%.2f GB), Down=%d bytes (%.2f GB), Total=%d bytes (%.2f GB), Limit=%d bytes (%.2f GB)",
					client.ID, client.Email, stat.Up, float64(stat.Up)/1024/1024/1024, stat.Down, float64(stat.Down)/1024/1024/1024, totalTraffic, float64(totalTraffic)/1024/1024/1024, trafficLimitBytes, float64(trafficLimitBytes)/1024/1024/1024)
			}
		}
	}

	if modified {
		log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Обновление inbound с отключенными клиентами")
		settingsJSON, err := json.Marshal(settings)
		if err != nil {
			log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Ошибка сериализации settings: %v", err)
			return fmt.Errorf("ошибка сериализации settings: %v", err)
		}
		inbound.Settings = string(settingsJSON)

		if err := UpdateInbound(sessionCookie, *inbound); err != nil {
			log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Ошибка обновления inbound: %v", err)
			return fmt.Errorf("ошибка обновления inbound: %v", err)
		}
		log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Inbound успешно обновлен")
	} else {
		log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Нет клиентов для отключения")
	}

	log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Отключено клиентов: %d, включено клиентов: %d", disabledCount, enabledCount)
	return nil
}

// StartTrafficMonitoring запускает периодический мониторинг трафика
func StartTrafficMonitoring() {
	log.Printf("START_TRAFFIC_MONITORING: Запуск мониторинга трафика с интервалом %d минут", TRAFFIC_CHECK_INTERVAL)

	interval := time.Duration(TRAFFIC_CHECK_INTERVAL) * time.Minute
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		for range ticker.C {
			log.Printf("START_TRAFFIC_MONITORING: Выполнение проверки трафика")
			if err := CheckAndDisableTrafficLimit(); err != nil {
				log.Printf("START_TRAFFIC_MONITORING: Ошибка проверки трафика: %v", err)
			} else {
				log.Printf("START_TRAFFIC_MONITORING: Проверка трафика успешно выполнена")
			}
		}
	}()

	// Запускаем периодический сброс трафика
	if TRAFFIC_RESET_ENABLED && TRAFFIC_RESET_INTERVAL > 0 {
		go startPeriodicTrafficReset()
	}

	log.Printf("START_TRAFFIC_MONITORING: Запущен мониторинг трафика (каждые %d минут)", TRAFFIC_CHECK_INTERVAL)
}

// startPeriodicTrafficReset запускает периодический сброс трафика
func startPeriodicTrafficReset() {
	log.Printf("START_PERIODIC_TRAFFIC_RESET: Запуск периодического сброса трафика каждые %d минут", TRAFFIC_RESET_INTERVAL)
	LogServiceStart("PERIODIC_TRAFFIC_RESET", TRAFFIC_RESET_INTERVAL)

	interval := time.Duration(TRAFFIC_RESET_INTERVAL) * time.Minute

	// Вычисляем время до следующего сброса (чтобы не сбрасывать при перезапуске)
	now := time.Now()
	// Находим следующий час, кратный интервалу
	nextReset := now.Truncate(interval).Add(interval)
	if nextReset.Before(now) {
		nextReset = nextReset.Add(interval)
	}
	timeToNextReset := nextReset.Sub(now)

	log.Printf("START_PERIODIC_TRAFFIC_RESET: Первый сброс трафика через %v (в %s)",
		timeToNextReset, nextReset.Format("2006-01-02 15:04:05"))
	LogTraffic("PERIODIC_TRAFFIC_RESET", "Первый сброс трафика через %v (в %s)",
		timeToNextReset, nextReset.Format("2006-01-02 15:04:05"))

	go func() {
		// Ждем до времени следующего сброса
		timer := time.NewTimer(timeToNextReset)
		<-timer.C

		// Выполняем первый сброс
		log.Printf("START_PERIODIC_TRAFFIC_RESET: Выполнение первого сброса трафика по расписанию")
		LogTraffic("PERIODIC_TRAFFIC_RESET", "Выполнение первого сброса трафика по расписанию")

		if err := ResetAllTraffic(); err != nil {
			log.Printf("START_PERIODIC_TRAFFIC_RESET: Ошибка сброса трафика: %v", err)
			LogTraffic("PERIODIC_TRAFFIC_RESET", "ОШИБКА сброса трафика: %v", err)
		} else {
			log.Printf("START_PERIODIC_TRAFFIC_RESET: Сброс трафика успешно выполнен")
			LogTraffic("PERIODIC_TRAFFIC_RESET", "Сброс трафика успешно выполнен")
		}

		// Запускаем периодический ticker
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			log.Printf("START_PERIODIC_TRAFFIC_RESET: Выполнение периодического сброса трафика")
			LogTraffic("PERIODIC_TRAFFIC_RESET", "Выполнение периодического сброса трафика")

			if err := ResetAllTraffic(); err != nil {
				log.Printf("START_PERIODIC_TRAFFIC_RESET: Ошибка сброса трафика: %v", err)
				LogTraffic("PERIODIC_TRAFFIC_RESET", "ОШИБКА сброса трафика: %v", err)
			} else {
				log.Printf("START_PERIODIC_TRAFFIC_RESET: Сброс трафика успешно выполнен")
				LogTraffic("PERIODIC_TRAFFIC_RESET", "Сброс трафика успешно выполнен")
			}
		}
	}()

	log.Printf("START_PERIODIC_TRAFFIC_RESET: Запущен периодический сброс трафика (каждые %d минут)", TRAFFIC_RESET_INTERVAL)
}

// updateUserTrafficStatus обновляет статус пользователя в БД при изменении статуса трафика
func updateUserTrafficStatus(email string, isEnabled bool) {
	// Извлекаем telegram_id из email
	if !strings.Contains(email, "@") {
		log.Printf("UPDATE_USER_TRAFFIC_STATUS: Некорректный email формат: %s", email)
		return
	}

	parts := strings.Split(email, "@")
	telegramIDStr := parts[0]

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		log.Printf("UPDATE_USER_TRAFFIC_STATUS: Ошибка парсинга telegram_id из email %s: %v", email, err)
		return
	}

	// Получаем пользователя
	user, err := GetUserByTelegramID(telegramID)
	if err != nil {
		log.Printf("UPDATE_USER_TRAFFIC_STATUS: Ошибка получения пользователя TelegramID=%d: %v", telegramID, err)
		return
	}

	if user == nil {
		log.Printf("UPDATE_USER_TRAFFIC_STATUS: Пользователь не найден TelegramID=%d", telegramID)
		return
	}

	// Обновляем статус активного конфига (если он изменился)
	if user.HasActiveConfig != isEnabled {
		user.HasActiveConfig = isEnabled
		user.UpdatedAt = time.Now()

		err = UpdateUser(user)
		if err != nil {
			log.Printf("UPDATE_USER_TRAFFIC_STATUS: Ошибка обновления пользователя TelegramID=%d: %v", telegramID, err)
		} else {
			log.Printf("UPDATE_USER_TRAFFIC_STATUS: Обновлен статус пользователя TelegramID=%d, HasActiveConfig=%t", telegramID, isEnabled)
		}
	}
}

// ResetAllTraffic сбрасывает трафик всех клиентов и включает отключенных
func ResetAllTraffic() error {
	log.Printf("RESET_ALL_TRAFFIC: Начало сброса трафика для всех клиентов")
	LogTraffic("RESET_ALL_TRAFFIC", "Начало сброса трафика для всех клиентов")

	// Авторизуемся в панели
	sessionCookie, err := Login()
	if err != nil {
		log.Printf("RESET_ALL_TRAFFIC: Ошибка авторизации: %v", err)
		LogTraffic("RESET_ALL_TRAFFIC", "ОШИБКА авторизации: %v", err)
		return fmt.Errorf("ошибка авторизации в панели: %v", err)
	}

	// Используем API панели для сброса трафика всех клиентов в inbound
	err = resetAllClientTrafficsAPI(sessionCookie, INBOUND_ID)
	if err != nil {
		log.Printf("RESET_ALL_TRAFFIC: Ошибка сброса трафика через API: %v", err)
		LogTraffic("RESET_ALL_TRAFFIC", "ОШИБКА сброса трафика через API: %v", err)
		return fmt.Errorf("ошибка сброса трафика через API: %v", err)
	}

	// Получаем данные inbound для включения отключенных клиентов
	inbound, err := GetInbound(sessionCookie)
	if err != nil {
		log.Printf("RESET_ALL_TRAFFIC: Ошибка получения данных inbound: %v", err)
		LogTraffic("RESET_ALL_TRAFFIC", "ОШИБКА получения данных inbound: %v", err)
		return fmt.Errorf("ошибка получения данных inbound: %v", err)
	}

	// Парсим settings
	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Printf("RESET_ALL_TRAFFIC: Ошибка парсинга settings: %v", err)
		LogTraffic("RESET_ALL_TRAFFIC", "ОШИБКА парсинга settings: %v", err)
		return fmt.Errorf("ошибка парсинга settings: %v", err)
	}

	log.Printf("RESET_ALL_TRAFFIC: Найдено клиентов: %d", len(settings.Clients))
	LogTraffic("RESET_ALL_TRAFFIC", "Найдено клиентов в панели: %d", len(settings.Clients))

	// Включаем всех отключенных клиентов
	modified := false
	enabledCount := 0

	for i, client := range settings.Clients {
		if !client.Enable {
			log.Printf("RESET_ALL_TRAFFIC: Включаем клиента ID=%s, Email=%s", client.ID, client.Email)
			LogClientOperation("RESET_ALL_TRAFFIC", 0, client.Email, "Включение отключенного клиента")
			settings.Clients[i].Enable = true
			modified = true
			enabledCount++
			updateUserTrafficStatus(client.Email, true)
		}
	}

	// Обновляем inbound если есть изменения
	if modified {
		log.Printf("RESET_ALL_TRAFFIC: Обновление inbound с включенными клиентами")
		LogTraffic("RESET_ALL_TRAFFIC", "Обновление inbound с %d включенными клиентами", enabledCount)
		settingsJSON, err := json.Marshal(settings)
		if err != nil {
			log.Printf("RESET_ALL_TRAFFIC: Ошибка сериализации settings: %v", err)
			LogTraffic("RESET_ALL_TRAFFIC", "ОШИБКА сериализации settings: %v", err)
			return fmt.Errorf("ошибка сериализации settings: %v", err)
		}
		inbound.Settings = string(settingsJSON)
		if err := UpdateInbound(sessionCookie, *inbound); err != nil {
			log.Printf("RESET_ALL_TRAFFIC: Ошибка обновления inbound: %v", err)
			LogTraffic("RESET_ALL_TRAFFIC", "ОШИБКА обновления inbound: %v", err)
			return fmt.Errorf("ошибка обновления inbound: %v", err)
		}
	}

	log.Printf("RESET_ALL_TRAFFIC: Успешно сброшен трафик для %d клиентов, включено %d клиентов", len(settings.Clients), enabledCount)
	LogTrafficReset("RESET_ALL_TRAFFIC", len(settings.Clients), fmt.Sprintf("включено %d клиентов", enabledCount))
	return nil
}

// resetAllClientTrafficsAPI сбрасывает трафик всех клиентов через API панели
func resetAllClientTrafficsAPI(sessionCookie string, inboundID int) error {
	log.Printf("RESET_ALL_CLIENT_TRAFFICS_API: Сброс трафика через API для inbound ID=%d", inboundID)

	url := fmt.Sprintf("%spanel/api/inbound/resetAllClientTraffics/%d", PANEL_URL, inboundID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Printf("RESET_ALL_CLIENT_TRAFFICS_API: Ошибка создания запроса: %v", err)
		return fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("RESET_ALL_CLIENT_TRAFFICS_API: Ошибка выполнения запроса: %v", err)
		return fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("RESET_ALL_CLIENT_TRAFFICS_API: Ошибка чтения ответа: %v", err)
		return fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	log.Printf("RESET_ALL_CLIENT_TRAFFICS_API: Ответ сервера: status=%d, body=%s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("неверный статус код: %d, ответ: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("RESET_ALL_CLIENT_TRAFFICS_API: Ошибка парсинга JSON: %v", err)
		return fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	if !response.Success {
		return fmt.Errorf("ошибка API: %s", response.Msg)
	}

	log.Printf("RESET_ALL_CLIENT_TRAFFICS_API: Трафик успешно сброшен через API")
	return nil
}

// updateAllUsersActiveStatus обновляет статус has_active_config для всех пользователей
func updateAllUsersActiveStatus(status bool) {
	users, err := GetAllUsers()
	if err != nil {
		log.Printf("UPDATE_ALL_USERS_ACTIVE_STATUS: Ошибка получения пользователей: %v", err)
		return
	}

	updatedCount := 0
	for _, user := range users {
		if user.HasActiveConfig != status {
			user.HasActiveConfig = status
			user.UpdatedAt = time.Now()

			err = UpdateUser(&user)
			if err != nil {
				log.Printf("UPDATE_ALL_USERS_ACTIVE_STATUS: Ошибка обновления пользователя TelegramID=%d: %v", user.TelegramID, err)
			} else {
				updatedCount++
			}
		}
	}

	log.Printf("UPDATE_ALL_USERS_ACTIVE_STATUS: Обновлен статус для %d пользователей, HasActiveConfig=%t", updatedCount, status)
}

// restoreFromBackup восстанавливает данные из указанной папки бэкапа
func restoreFromBackup(backupPath string) error {
	log.Printf("RESTORE_FROM_BACKUP: Начало восстановления из %s", backupPath)

	// Проверяем, существует ли путь к бэкапу
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("путь к бэкапу не существует: %s", backupPath)
	}

	// Ищем папку с данными MongoDB
	mongoDbPath := filepath.Join(backupPath, MONGO_DB_NAME)
	if _, err := os.Stat(mongoDbPath); os.IsNotExist(err) {
		return fmt.Errorf("папка с данными БД не найдена: %s", mongoDbPath)
	}

	// Выполняем mongorestore
	cmd := exec.Command("mongorestore", "--uri", MONGO_URI, "--db", MONGO_DB_NAME, "--drop", mongoDbPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка восстановления: %v, вывод: %s", err, string(output))
	}

	log.Printf("RESTORE_FROM_BACKUP: ✅ Данные успешно восстановлены из %s", backupPath)
	log.Printf("RESTORE_FROM_BACKUP: ========================================")
	log.Printf("RESTORE_FROM_BACKUP: ВОССТАНОВЛЕНИЕ ЗАВЕРШЕНО УСПЕШНО")
	log.Printf("RESTORE_FROM_BACKUP: ========================================")
	return nil
}

// GetUsersStatistics получает статистику пользователей
func GetUsersStatistics() (*UsersStatistics, error) {
	log.Printf("GET_USERS_STATISTICS: Получение статистики пользователей")

	// Переадресация к PostgreSQL
	return GetUsersStatisticsPG()
}

// GetUsersSorted получает отсортированных пользователей с лимитом
func GetUsersSorted(limit int) ([]User, error) {
	log.Printf("GET_USERS_SORTED: Получение отсортированных пользователей, лимит: %d", limit)

	users, err := GetAllUsers()
	if err != nil {
		log.Printf("GET_USERS_SORTED: Ошибка получения пользователей: %v", err)
		return nil, err
	}

	// Сортируем по дате создания (новые сначала)
	sort.Slice(users, func(i, j int) bool {
		return users[i].CreatedAt.After(users[j].CreatedAt)
	})

	// Применяем лимит
	if limit > 0 && len(users) > limit {
		users = users[:limit]
	}

	log.Printf("GET_USERS_SORTED: Возвращено пользователей: %d", len(users))
	return users, nil
}

// SendConfigBlockingNotificationToAdmin отправляет уведомление администратору о блокировке конфига (автосписание)
func SendConfigBlockingNotificationToAdmin(user *User) {
	if !ADMIN_NOTIFICATIONS_ENABLED || !ADMIN_CONFIG_BLOCKING_ENABLED || GlobalBot == nil {
		return
	}

	displayName := getUserDisplayName(user)
	message := fmt.Sprintf(
		"🚫 <b>Конфиг заблокирован</b>\n\n"+
			"👤 Пользователь: %s (ID: %d)\n"+
			"💰 Баланс: %.2f₽\n"+
			"📧 Email: %s\n"+
			"🕐 Время блокировки: %s\n\n"+
			"Причина: недостаточно средств для автосписания",
		displayName, user.TelegramID, user.Balance, user.Email, time.Now().Format("2006-01-02 15:04:05"))

	msg := tgbotapi.NewMessage(ADMIN_ID, message)
	msg.ParseMode = tgbotapi.ModeHTML

	_, err := GlobalBot.Send(msg)
	if err != nil {
		log.Printf("NOTIFICATION: Ошибка отправки уведомления администратору о блокировке конфига пользователя %d: %v", user.TelegramID, err)
	} else {
		log.Printf("NOTIFICATION: Уведомление о блокировке конфига пользователя %d отправлено администратору", user.TelegramID)
	}
}

// SendIPBanNotificationToAdmin отправляет уведомление администратору о срабатывании IP ban
func SendIPBanNotificationToAdmin(email string, ipAddresses []string, ipCount int) {
	if !ADMIN_NOTIFICATIONS_ENABLED || !ADMIN_IP_BAN_ENABLED || GlobalBot == nil {
		return
	}

	// Пытаемся найти пользователя по email
	var displayName string
	var telegramID int64

	// Email в системе обычно имеет формат "123456789" (telegram_id) или "123456789 до 2025 03 09"
	var emailParts []string
	if strings.Contains(email, " ") {
		emailParts = strings.Split(email, " ")
	} else {
		emailParts = []string{email}
	}

	if len(emailParts) > 0 {
		if id, err := strconv.ParseInt(emailParts[0], 10, 64); err == nil {
			telegramID = id
			if user, err := GetUserByTelegramID(telegramID); err == nil && user != nil {
				displayName = getUserDisplayName(user)
			}
		}
	}

	if displayName == "" {
		displayName = email
	}

	// Формируем список IP адресов для сообщения
	ipList := strings.Join(ipAddresses, ", ")
	if len(ipList) > 200 {
		ipList = ipList[:200] + "..."
	}

	message := fmt.Sprintf(
		"🚨 <b>IP Ban - конфиг заблокирован</b>\n\n"+
			"👤 Пользователь: %s\n"+
			"📧 Email: %s\n"+
			"🌐 Количество IP: %d (лимит: %d)\n"+
			"📍 IP адреса: %s\n"+
			"🕐 Время блокировки: %s\n\n"+
			"Причина: превышен лимит IP адресов",
		displayName, email, ipCount, MAX_IPS_PER_CONFIG, ipList, time.Now().Format("2006-01-02 15:04:05"))

	msg := tgbotapi.NewMessage(ADMIN_ID, message)
	msg.ParseMode = tgbotapi.ModeHTML

	_, err := GlobalBot.Send(msg)
	if err != nil {
		log.Printf("NOTIFICATION: Ошибка отправки уведомления администратору о IP ban для %s: %v", email, err)
	} else {
		log.Printf("NOTIFICATION: Уведомление о IP ban для %s отправлено администратору", email)
	}
}

// SendBalanceTopupNotificationToAdmin отправляет уведомление администратору о пополнении баланса
func SendBalanceTopupNotificationToAdmin(user *User, amount float64) {
	if !ADMIN_NOTIFICATIONS_ENABLED || !ADMIN_BALANCE_TOPUP_ENABLED || GlobalBot == nil {
		return
	}

	displayName := getUserDisplayName(user)
	message := fmt.Sprintf(
		"💳 <b>Пополнение баланса</b>\n\n"+
			"👤 Пользователь: %s (ID: %d)\n"+
			"💰 Сумма пополнения: %.2f₽\n"+
			"💳 Новый баланс: %.2f₽\n"+
			"📊 Всего заплачено: %.2f₽\n"+
			"🕐 Время пополнения: %s",
		displayName, user.TelegramID, amount, user.Balance, user.TotalPaid, time.Now().Format("2006-01-02 15:04:05"))

	msg := tgbotapi.NewMessage(ADMIN_ID, message)
	msg.ParseMode = tgbotapi.ModeHTML

	_, err := GlobalBot.Send(msg)
	if err != nil {
		log.Printf("NOTIFICATION: Ошибка отправки уведомления администратору о пополнении баланса пользователя %d: %v", user.TelegramID, err)
	} else {
		log.Printf("NOTIFICATION: Уведомление о пополнении баланса пользователя %d отправлено администратору", user.TelegramID)
	}
}

// SendReminderNotificationToAdmin отправляет уведомление администратору о отправленном напоминании
func SendReminderNotificationToAdmin(user *User, daysLeft, hoursLeft int) {
	if !ADMIN_NOTIFICATIONS_ENABLED || !ADMIN_REMINDER_ENABLED || GlobalBot == nil {
		return
	}

	displayName := getUserDisplayName(user)
	message := fmt.Sprintf(
		"⏰ <b>Напоминание о подписке отправлено</b>\n\n"+
			"👤 Пользователь: %s (ID: %d)\n"+
			"💰 Баланс: %.2f₽\n"+
			"📧 Email: %s\n"+
			"⏳ Осталось: %d дней %d часов\n"+
			"🕐 Время отправки: %s",
		displayName, user.TelegramID, user.Balance, user.Email, daysLeft, hoursLeft, time.Now().Format("2006-01-02 15:04:05"))

	msg := tgbotapi.NewMessage(ADMIN_ID, message)
	msg.ParseMode = tgbotapi.ModeHTML

	_, err := GlobalBot.Send(msg)
	if err != nil {
		log.Printf("NOTIFICATION: Ошибка отправки уведомления администратору о напоминании пользователя %d: %v", user.TelegramID, err)
	} else {
		log.Printf("NOTIFICATION: Уведомление о напоминании пользователя %d отправлено администратору", user.TelegramID)
	}
}

// getUserDisplayName возвращает читаемое имя пользователя
func getUserDisplayName(user *User) string {
	if user.FirstName != "" {
		displayName := user.FirstName
		if user.LastName != "" {
			displayName += " " + user.LastName
		}
		if user.Username != "" {
			displayName += " (@" + user.Username + ")"
		}
		return displayName
	}
	if user.Username != "" {
		return "@" + user.Username
	}
	return fmt.Sprintf("ID: %d", user.TelegramID)
}

// GetUsersByCategory получает пользователей по категории
func GetUsersByCategory(category string, limit int) ([]User, error) {
	log.Printf("GET_USERS_BY_CATEGORY: Получение пользователей категории '%s', лимит: %d", category, limit)

	users, err := GetAllUsers()
	if err != nil {
		log.Printf("GET_USERS_BY_CATEGORY: Ошибка получения пользователей: %v", err)
		return nil, err
	}

	var filteredUsers []User

	for _, user := range users {
		switch category {
		case "paying":
			// Платящие пользователи (баланс > 0 или уже платили)
			if user.TotalPaid > 0 {
				filteredUsers = append(filteredUsers, user)
			}
		case "trial_available":
			// Могут использовать пробный период
			if !user.HasUsedTrial && user.TotalPaid <= 0 {
				filteredUsers = append(filteredUsers, user)
			}
		case "trial_used":
			// Использовали пробный период, но не платили
			if user.HasUsedTrial && user.TotalPaid <= 0 {
				filteredUsers = append(filteredUsers, user)
			}
		case "inactive":
			// Неактивные пользователи
			if !user.HasActiveConfig {
				filteredUsers = append(filteredUsers, user)
			}
		case "active":
			// Активные пользователи
			if user.HasActiveConfig {
				filteredUsers = append(filteredUsers, user)
			}
		default:
			// Если категория не распознана, возвращаем всех
			filteredUsers = users
		}
	}

	// Сортируем по дате создания (новые сначала)
	sort.Slice(filteredUsers, func(i, j int) bool {
		return filteredUsers[i].CreatedAt.After(filteredUsers[j].CreatedAt)
	})

	// Применяем лимит
	if limit > 0 && len(filteredUsers) > limit {
		filteredUsers = filteredUsers[:limit]
	}

	log.Printf("GET_USERS_BY_CATEGORY: Категория '%s': найдено %d пользователей", category, len(filteredUsers))
	return filteredUsers, nil
}

// logUsersList выводит список пользователей в лог
func logUsersList(users []User) {
	displayCount := len(users)
	if displayCount > 50 {
		displayCount = 50
	}

	for i := 0; i < displayCount; i++ {
		user := users[i]
		status := "неактивен"
		if user.HasActiveConfig {
			status = "активен"
		}

		trialStatus := "доступен"
		if user.HasUsedTrial {
			trialStatus = "использован"
		}

		log.Printf("INIT_MONGODB: %d) @%s (%s %s) - Баланс: %.2f₽, Статус: %s, Пробный: %s",
			i+1, user.Username, user.FirstName, user.LastName,
			user.Balance, status, trialStatus)
	}

	if len(users) > 50 {
		log.Printf("INIT_MONGODB: ... и еще %d пользователей", len(users)-50)
	}
}
