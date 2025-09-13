package telegram_bot

import (
	"fmt"
	"log"
	"time"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// NotificationManager управляет уведомлениями о подписке
type NotificationManager struct {
	bot *tgbotapi.BotAPI
}

// NewNotificationManager создает новый менеджер уведомлений
func NewNotificationManager(bot *tgbotapi.BotAPI) *NotificationManager {
	return &NotificationManager{
		bot: bot,
	}
}

// StartNotificationScheduler запускает планировщик уведомлений
func (nm *NotificationManager) StartNotificationScheduler() {
	if !common.NOTIFICATION_ENABLED {
		log.Printf("NOTIFICATION: Уведомления отключены в конфигурации")
		return
	}

	log.Printf("NOTIFICATION: Запуск планировщика уведомлений с интервалом %d минут", common.NOTIFICATION_CHECK_INTERVAL)

	ticker := time.NewTicker(time.Duration(common.NOTIFICATION_CHECK_INTERVAL) * time.Minute)
	defer ticker.Stop()

	// Запускаем проверку сразу при старте
	go nm.checkAndSendNotifications()

	// Затем проверяем по расписанию
	for range ticker.C {
		go nm.checkAndSendNotifications()
	}
}

// checkAndSendNotifications проверяет подписки и отправляет уведомления
func (nm *NotificationManager) checkAndSendNotifications() {
	log.Printf("NOTIFICATION: Начало проверки подписок для уведомлений")

	// Получаем всех пользователей с активными конфигами
	users, err := common.GetUsersWithActiveConfigs()
	if err != nil {
		log.Printf("NOTIFICATION: Ошибка получения пользователей: %v", err)
		return
	}

	now := time.Now()
	notificationsSent := 0

	for _, user := range users {
		// Проверяем, нужно ли отправить уведомление
		if shouldSendNotification(&user, now) {
			daysLeft := calculateDaysLeft(user.ExpiryTime, now)
			message := getNotificationMessage(daysLeft)

			if message != "" {
				err := nm.sendNotification(user.TelegramID, message)
				if err != nil {
					log.Printf("NOTIFICATION: Ошибка отправки уведомления пользователю %d: %v", user.TelegramID, err)
				} else {
					notificationsSent++
					log.Printf("NOTIFICATION: Уведомление отправлено пользователю %d (осталось %d дней)", user.TelegramID, daysLeft)
				}
			}
		}
	}

	log.Printf("NOTIFICATION: Проверка завершена, отправлено %d уведомлений", notificationsSent)
}

// shouldSendNotification проверяет, нужно ли отправить уведомление пользователю
func shouldSendNotification(user *common.User, now time.Time) bool {
	// Проверяем, что у пользователя есть активная подписка
	if !user.HasActiveConfig || user.ExpiryTime <= 0 {
		return false
	}

	// Проверяем, что подписка еще не истекла
	if user.ExpiryTime <= now.UnixMilli() {
		return false
	}

	// Проверяем, что подписка истекает в ближайшие дни
	daysLeft := calculateDaysLeft(user.ExpiryTime, now)

	// Проверяем, есть ли этот день в списке дней для уведомлений
	for _, day := range common.NOTIFICATION_DAYS_BEFORE {
		if daysLeft == day {
			return true
		}
	}

	return false
}

// calculateDaysLeft вычисляет количество дней до истечения подписки
func calculateDaysLeft(expiryTime int64, now time.Time) int {
	expiry := time.UnixMilli(expiryTime)
	diff := expiry.Sub(now)
	days := int(diff.Hours() / 24)

	// Если осталось меньше дня, но больше 0, считаем как 1 день
	if days == 0 && diff > 0 {
		days = 1
	}

	return days
}

// getNotificationMessage возвращает сообщение для уведомления в зависимости от количества дней
func getNotificationMessage(daysLeft int) string {
	switch daysLeft {
	case 1:
		return common.NOTIFICATION_MESSAGE_1_DAY
	case 3:
		return common.NOTIFICATION_MESSAGE_3_DAYS
	case 7:
		return common.NOTIFICATION_MESSAGE_7_DAYS
	default:
		return ""
	}
}

// sendNotification отправляет уведомление пользователю
func (nm *NotificationManager) sendNotification(telegramID int64, message string) error {
	msg := tgbotapi.NewMessage(telegramID, message)
	msg.ParseMode = tgbotapi.ModeHTML

	_, err := nm.bot.Send(msg)
	return err
}

// SendImmediateNotification отправляет немедленное уведомление пользователю
func (nm *NotificationManager) SendImmediateNotification(telegramID int64, message string) error {
	log.Printf("NOTIFICATION: Отправка немедленного уведомления пользователю %d", telegramID)
	return nm.sendNotification(telegramID, message)
}

// CheckUserSubscription проверяет подписку конкретного пользователя и отправляет уведомление при необходимости
func (nm *NotificationManager) CheckUserSubscription(user *common.User) {
	if !common.NOTIFICATION_ENABLED {
		return
	}

	now := time.Now()
	if shouldSendNotification(user, now) {
		daysLeft := calculateDaysLeft(user.ExpiryTime, now)
		message := getNotificationMessage(daysLeft)

		if message != "" {
			err := nm.sendNotification(user.TelegramID, message)
			if err != nil {
				log.Printf("NOTIFICATION: Ошибка отправки уведомления пользователю %d: %v", user.TelegramID, err)
			} else {
				log.Printf("NOTIFICATION: Уведомление отправлено пользователю %d (осталось %d дней)", user.TelegramID, daysLeft)
			}
		}
	}
}

// SendConfigBlockingNotification отправляет уведомление администратору о блокировке конфига
func (nm *NotificationManager) SendConfigBlockingNotification(user *common.User) {
	if !common.ADMIN_NOTIFICATIONS_ENABLED || !common.ADMIN_CONFIG_BLOCKING_ENABLED {
		return
	}

	message := fmt.Sprintf(
		"🚫 <b>Конфиг заблокирован</b>\n\n"+
			"👤 Пользователь: %s (ID: %d)\n"+
			"💰 Баланс: %.2f₽\n"+
			"📧 Email: %s\n"+
			"🕐 Время блокировки: %s\n\n"+
			"Причина: недостаточно средств для автосписания",
		getUserDisplayName(user), user.TelegramID, user.Balance, user.Email, time.Now().Format("2006-01-02 15:04:05"))

	err := nm.sendNotification(common.ADMIN_ID, message)
	if err != nil {
		log.Printf("NOTIFICATION: Ошибка отправки уведомления администратору о блокировке конфига пользователя %d: %v", user.TelegramID, err)
	} else {
		log.Printf("NOTIFICATION: Уведомление о блокировке конфига пользователя %d отправлено администратору", user.TelegramID)
	}
}

// SendBalanceTopupNotification отправляет уведомление администратору о пополнении баланса
func (nm *NotificationManager) SendBalanceTopupNotification(user *common.User, amount float64) {
	if !common.ADMIN_NOTIFICATIONS_ENABLED || !common.ADMIN_BALANCE_TOPUP_ENABLED {
		return
	}

	message := fmt.Sprintf(
		"💳 <b>Пополнение баланса</b>\n\n"+
			"👤 Пользователь: %s (ID: %d)\n"+
			"💰 Сумма пополнения: %.2f₽\n"+
			"💳 Новый баланс: %.2f₽\n"+
			"📊 Всего заплачено: %.2f₽\n"+
			"🕐 Время пополнения: %s",
		getUserDisplayName(user), user.TelegramID, amount, user.Balance, user.TotalPaid, time.Now().Format("2006-01-02 15:04:05"))

	err := nm.sendNotification(common.ADMIN_ID, message)
	if err != nil {
		log.Printf("NOTIFICATION: Ошибка отправки уведомления администратору о пополнении баланса пользователя %d: %v", user.TelegramID, err)
	} else {
		log.Printf("NOTIFICATION: Уведомление о пополнении баланса пользователя %d отправлено администратору", user.TelegramID)
	}
}

// getUserDisplayName возвращает читаемое имя пользователя
func getUserDisplayName(user *common.User) string {
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
