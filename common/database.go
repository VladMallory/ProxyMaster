package common

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

// CheckAndDisableTrafficLimit проверяет трафик и отключает/включает клиентов
func CheckAndDisableTrafficLimit() error {
	log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Начало проверки трафика")

	// Если лимит трафика не установлен, пропускаем проверку
	if TRAFFIC_LIMIT_GB <= 0 {
		log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Лимит трафика не установлен (TRAFFIC_LIMIT_GB=%d), пропускаем проверку", TRAFFIC_LIMIT_GB)
		return nil
	}

	log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Функция проверки трафика временно отключена (GetClientTrafficStats не реализована)")
	disabledCount := 0

	log.Printf("CHECK_AND_DISABLE_TRAFFIC_LIMIT: Отключено клиентов по лимиту трафика: %d", disabledCount)
	return nil
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

// ResetAllTraffic сбрасывает трафик всех клиентов
func ResetAllTraffic() error {
	log.Printf("RESET_ALL_TRAFFIC: Начало сброса трафика для всех клиентов")

	// Авторизуемся в панели
	sessionCookie, err := Login()
	if err != nil {
		log.Printf("RESET_ALL_TRAFFIC: Ошибка авторизации: %v", err)
		return fmt.Errorf("ошибка авторизации в панели: %v", err)
	}

	// Получаем данные inbound
	inbound, err := GetInbound(sessionCookie)
	if err != nil {
		log.Printf("RESET_ALL_TRAFFIC: Ошибка получения данных inbound: %v", err)
		return fmt.Errorf("ошибка получения данных inbound: %v", err)
	}

	// Парсим settings
	var settings Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Printf("RESET_ALL_TRAFFIC: Ошибка парсинга settings: %v", err)
		return fmt.Errorf("ошибка парсинга settings: %v", err)
	}

	log.Printf("RESET_ALL_TRAFFIC: Найдено клиентов: %d", len(settings.Clients))

	resetCount := 0
	enabledCount := 0

	// Сбрасываем трафик для каждого клиента
	for i := range settings.Clients {
		client := &settings.Clients[i]

		// Сбрасываем трафик
		client.TotalGB = 0
		client.Reset = 0

		// Включаем клиента если он был отключен
		if !client.Enable {
			client.Enable = true
			enabledCount++
			log.Printf("RESET_ALL_TRAFFIC: Включаем клиента: %s", client.Email)
		}

		resetCount++
	}

	// Обновляем inbound
	updatedSettings, err := json.Marshal(settings)
	if err != nil {
		log.Printf("RESET_ALL_TRAFFIC: Ошибка сериализации настроек: %v", err)
		return fmt.Errorf("ошибка сериализации настроек: %v", err)
	}

	// Обновляем inbound с новыми настройками
	inbound.Settings = string(updatedSettings)

	err = UpdateInbound(sessionCookie, *inbound)
	if err != nil {
		log.Printf("RESET_ALL_TRAFFIC: Ошибка обновления inbound: %v", err)
		return fmt.Errorf("ошибка обновления inbound: %v", err)
	}

	// Обновляем статус пользователей в базе данных
	updateAllUsersActiveStatus(true)

	log.Printf("RESET_ALL_TRAFFIC: Успешно сброшен трафик для %d клиентов, включено %d клиентов", resetCount, enabledCount)
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

// copyLatestBackup копирует бэкап в папку latest
func copyLatestBackup(sourceDir string) error {
	latestDir := "./backups/latest"

	// Удаляем существующую папку latest, если она есть
	if err := os.RemoveAll(latestDir); err != nil {
		return fmt.Errorf("ошибка удаления старого latest: %v", err)
	}

	// Создаем папку latest
	if err := os.MkdirAll(latestDir, 0o755); err != nil {
		return fmt.Errorf("ошибка создания папки latest: %v", err)
	}

	// Копируем содержимое бэкапа
	cmd := exec.Command("cp", "-r", filepath.Join(sourceDir, MONGO_DB_NAME), latestDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка копирования бэкапа: %v", err)
	}

	log.Printf("COPY_LATEST_BACKUP: Последний бэкап скопирован в %s", latestDir)
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
