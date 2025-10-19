package services

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ReminderLogEntry представляет запись в логе напоминаний
type ReminderLogEntry struct {
	UserID     int64     `json:"user_id"`
	SentAt     time.Time `json:"sent_at"`
	DaysLeft   int       `json:"days_left"`
	HoursLeft  int       `json:"hours_left"`
	ExpiryTime int64     `json:"expiry_time"`
}

// UniversalReminderService управляет универсальными напоминаниями о подписке
type UniversalReminderService struct {
	bot            *tgbotapi.BotAPI
	reminderTicker *time.Ticker
	logFilePath    string
}

// NewUniversalReminderService создает новый сервис универсальных напоминаний
func NewUniversalReminderService(bot *tgbotapi.BotAPI) *UniversalReminderService {
	return &UniversalReminderService{
		bot:         bot,
		logFilePath: common.UNIVERSAL_REMINDER_LOG_PATH,
	}
}

// Start запускает сервис универсальных напоминаний
func (urs *UniversalReminderService) Start() {
	if !common.UNIVERSAL_REMINDER_ENABLED {
		log.Printf("UNIVERSAL_REMINDER: Универсальные напоминания отключены в конфигурации")
		return
	}

	// Создаем директорию для логов если её нет
	if err := os.MkdirAll(filepath.Dir(urs.logFilePath), 0755); err != nil {
		log.Printf("UNIVERSAL_REMINDER: Ошибка создания директории для логов: %v", err)
		return
	}

	interval := time.Duration(common.UNIVERSAL_REMINDER_INTERVAL) * time.Minute
	log.Printf("UNIVERSAL_REMINDER: Запуск сервиса универсальных напоминаний с интервалом %v", interval)

	urs.reminderTicker = time.NewTicker(interval)

	// Запускаем проверку сразу при старте
	go urs.checkAndSendReminders()

	// Затем проверяем по расписанию
	go func() {
		for range urs.reminderTicker.C {
			urs.checkAndSendReminders()
		}
	}()

	log.Printf("UNIVERSAL_REMINDER: Сервис универсальных напоминаний успешно запущен")
}

// Stop останавливает сервис универсальных напоминаний
func (urs *UniversalReminderService) Stop() {
	if urs.reminderTicker != nil {
		urs.reminderTicker.Stop()
		log.Printf("UNIVERSAL_REMINDER: Сервис универсальных напоминаний остановлен")
	}
}

// checkAndSendReminders проверяет подписки и отправляет напоминания
func (urs *UniversalReminderService) checkAndSendReminders() {
	log.Printf("UNIVERSAL_REMINDER: Начало проверки подписок для напоминаний")

	// Получаем всех пользователей с активными конфигами
	users, err := common.GetUsersWithActiveConfigs()
	if err != nil {
		log.Printf("UNIVERSAL_REMINDER: Ошибка получения пользователей: %v", err)
		return
	}

	now := time.Now()
	remindersSent := 0

	for _, user := range users {
		// Проверяем, нужно ли отправить напоминание
		if urs.shouldSendReminder(&user, now) {
			daysLeft, hoursLeft := urs.calculateTimeLeft(user.ExpiryTime, now)
			message := urs.buildReminderMessage(daysLeft, hoursLeft)

			err := urs.sendReminder(user.TelegramID, message)
			if err != nil {
				log.Printf("UNIVERSAL_REMINDER: Ошибка отправки напоминания пользователю %d: %v", user.TelegramID, err)
			} else {
				// Записываем в лог
				urs.logReminderSent(&user, daysLeft, hoursLeft)

				// Отправляем уведомление администратору
				common.SendReminderNotificationToAdmin(&user, daysLeft, hoursLeft)

				remindersSent++
				log.Printf("UNIVERSAL_REMINDER: Напоминание отправлено пользователю %d (осталось %d дней %d часов)",
					user.TelegramID, daysLeft, hoursLeft)
			}
		}
	}

	log.Printf("UNIVERSAL_REMINDER: Проверка завершена, отправлено %d напоминаний", remindersSent)
}

// shouldSendReminder проверяет, нужно ли отправить напоминание пользователю
func (urs *UniversalReminderService) shouldSendReminder(user *common.User, now time.Time) bool {
	// Проверяем, что у пользователя есть активная подписка
	if !user.HasActiveConfig || user.ExpiryTime <= 0 {
		return false
	}

	// Проверяем, что подписка еще не истекла
	if user.ExpiryTime <= now.UnixMilli() {
		return false
	}

	// Проверяем, что подписка истекает в ближайшие дни
	daysLeft, _ := urs.calculateTimeLeft(user.ExpiryTime, now)

	// Отправляем напоминания только если осталось меньше или равно настроенному количеству дней
	if daysLeft > common.UNIVERSAL_REMINDER_DAYS_BEFORE {
		return false
	}

	// Проверяем, не отправляли ли мы уже напоминание сегодня для этой подписки
	return !urs.wasReminderSentToday(user.TelegramID, user.ExpiryTime, now)
}

// calculateTimeLeft вычисляет количество дней и часов до истечения подписки
func (urs *UniversalReminderService) calculateTimeLeft(expiryTime int64, now time.Time) (int, int) {
	expiry := time.UnixMilli(expiryTime)
	diff := expiry.Sub(now)

	days := int(diff.Hours() / 24)
	hours := int(diff.Hours()) % 24

	// Если осталось меньше дня, но больше 0, считаем как 0 дней и показываем часы
	if days < 0 {
		days = 0
		hours = 0
	}

	return days, hours
}

// buildReminderMessage создает сообщение напоминания с подстановкой времени
func (urs *UniversalReminderService) buildReminderMessage(daysLeft, hoursLeft int) string {
	message := common.UNIVERSAL_REMINDER_MESSAGE

	// Заменяем плейсхолдеры
	message = strings.ReplaceAll(message, "{DAYS}", strconv.Itoa(daysLeft))
	message = strings.ReplaceAll(message, "{HOURS}", strconv.Itoa(hoursLeft))

	return message
}

// sendReminder отправляет напоминание пользователю
func (urs *UniversalReminderService) sendReminder(telegramID int64, message string) error {
	msg := tgbotapi.NewMessage(telegramID, message)
	msg.ParseMode = tgbotapi.ModeHTML

	_, err := urs.bot.Send(msg)
	return err
}

// SendReminder публичный метод для отправки напоминания (для тестирования)
func (urs *UniversalReminderService) SendReminder(telegramID int64, message string) error {
	return urs.sendReminder(telegramID, message)
}

// LogReminderSent публичный метод для записи отправленного напоминания в лог
func (urs *UniversalReminderService) LogReminderSent(user *common.User, daysLeft, hoursLeft int) {
	urs.logReminderSent(user, daysLeft, hoursLeft)
}

// wasReminderSentToday проверяет, отправлялось ли напоминание пользователю сегодня для текущей подписки
func (urs *UniversalReminderService) wasReminderSentToday(userID int64, currentExpiryTime int64, now time.Time) bool {
	// Читаем лог файл
	file, err := os.Open(urs.logFilePath)
	if err != nil {
		// Если файл не существует, значит напоминаний еще не отправляли
		return false
	}
	defer file.Close()

	today := now.Format("2006-01-02")
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var entry ReminderLogEntry
		if err := json.Unmarshal([]byte(scanner.Text()), &entry); err != nil {
			continue
		}

		// Проверяем, что это тот же пользователь, напоминание отправлено сегодня
		// И время истечения подписки совпадает (значит это та же подписка)
		if entry.UserID == userID &&
			entry.SentAt.Format("2006-01-02") == today &&
			entry.ExpiryTime == currentExpiryTime {
			return true
		}
	}

	return false
}

// logReminderSent записывает отправленное напоминание в лог
func (urs *UniversalReminderService) logReminderSent(user *common.User, daysLeft, hoursLeft int) {
	entry := ReminderLogEntry{
		UserID:     user.TelegramID,
		SentAt:     time.Now(),
		DaysLeft:   daysLeft,
		HoursLeft:  hoursLeft,
		ExpiryTime: user.ExpiryTime,
	}

	// Сериализуем в JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Printf("UNIVERSAL_REMINDER: Ошибка сериализации записи лога: %v", err)
		return
	}

	// Записываем в файл
	file, err := os.OpenFile(urs.logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("UNIVERSAL_REMINDER: Ошибка открытия файла лога: %v", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(string(jsonData) + "\n"); err != nil {
		log.Printf("UNIVERSAL_REMINDER: Ошибка записи в файл лога: %v", err)
	}
}

// GetReminderStats возвращает статистику отправленных напоминаний
func (urs *UniversalReminderService) GetReminderStats() (int, error) {
	file, err := os.Open(urs.logFilePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var entry ReminderLogEntry
		if err := json.Unmarshal([]byte(scanner.Text()), &entry); err != nil {
			continue
		}
		count++
	}

	return count, nil
}

// CleanOldLogs очищает старые записи из лога (старше 30 дней)
func (urs *UniversalReminderService) CleanOldLogs() error {
	file, err := os.Open(urs.logFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var validEntries []ReminderLogEntry
	cutoffDate := time.Now().AddDate(0, 0, -30) // 30 дней назад

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry ReminderLogEntry
		if err := json.Unmarshal([]byte(scanner.Text()), &entry); err != nil {
			continue
		}

		// Оставляем только записи младше 30 дней
		if entry.SentAt.After(cutoffDate) {
			validEntries = append(validEntries, entry)
		}
	}

	// Перезаписываем файл только валидными записями
	file, err = os.Create(urs.logFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, entry := range validEntries {
		jsonData, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		file.WriteString(string(jsonData) + "\n")
	}

	log.Printf("UNIVERSAL_REMINDER: Очищено %d старых записей из лога", len(validEntries))
	return nil
}
