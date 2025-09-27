package services

import (
	"fmt"
	"log"
	"time"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ExpiredSubscriptionService управляет автоматическим отключением истекших подписок
type ExpiredSubscriptionService struct {
	bot           *tgbotapi.BotAPI
	configManager *common.ConfigManager
	checkTicker   *time.Ticker
}

// NewExpiredSubscriptionService создает новый сервис проверки истекших подписок
func NewExpiredSubscriptionService(bot *tgbotapi.BotAPI, configManager *common.ConfigManager) *ExpiredSubscriptionService {
	return &ExpiredSubscriptionService{
		bot:           bot,
		configManager: configManager,
	}
}

// Start запускает сервис проверки истекших подписок
func (ess *ExpiredSubscriptionService) Start() {
	if !common.EXPIRED_SUBSCRIPTION_CHECK_ENABLED {
		log.Printf("EXPIRED_SUBSCRIPTION: Проверка истекших подписок отключена в конфигурации")
		return
	}

	log.Printf("EXPIRED_SUBSCRIPTION: Запуск сервиса проверки истекших подписок")
	log.Printf("EXPIRED_SUBSCRIPTION: Интервал проверки: %d минут", common.EXPIRED_SUBSCRIPTION_CHECK_INTERVAL)

	// Создаем тикер для периодической проверки
	ess.checkTicker = time.NewTicker(time.Duration(common.EXPIRED_SUBSCRIPTION_CHECK_INTERVAL) * time.Minute)

	// Запускаем горутину для периодических проверок
	go func() {
		// Первая проверка сразу при запуске
		ess.checkExpiredSubscriptions()

		// Затем проверяем по расписанию
		for range ess.checkTicker.C {
			ess.checkExpiredSubscriptions()
		}
	}()

	log.Printf("EXPIRED_SUBSCRIPTION: Сервис проверки истекших подписок успешно запущен")
}

// Stop останавливает сервис
func (ess *ExpiredSubscriptionService) Stop() {
	if ess.checkTicker != nil {
		ess.checkTicker.Stop()
		log.Printf("EXPIRED_SUBSCRIPTION: Сервис проверки истекших подписок остановлен")
	}
}

// checkExpiredSubscriptions проверяет всех пользователей на истекшие подписки
func (ess *ExpiredSubscriptionService) checkExpiredSubscriptions() {
	log.Printf("EXPIRED_SUBSCRIPTION: Начинаем проверку истекших подписок...")

	// Получаем всех пользователей с активными конфигами
	users, err := common.GetUsersWithActiveConfigs()
	if err != nil {
		log.Printf("EXPIRED_SUBSCRIPTION: Ошибка получения пользователей: %v", err)
		return
	}

	now := time.Now()
	expiredCount := 0
	disabledCount := 0

	for _, user := range users {
		// Проверяем, истекла ли подписка
		if ess.isSubscriptionExpired(&user, now) {
			expiredCount++
			log.Printf("EXPIRED_SUBSCRIPTION: Найдена истекшая подписка для пользователя %d (email: %s)",
				user.TelegramID, user.Email)

			// Отключаем конфиг пользователя
			if err := ess.disableExpiredSubscription(&user); err != nil {
				log.Printf("EXPIRED_SUBSCRIPTION: Ошибка отключения конфига для пользователя %d: %v",
					user.TelegramID, err)
			} else {
				disabledCount++
				log.Printf("EXPIRED_SUBSCRIPTION: Конфиг успешно отключен для пользователя %d", user.TelegramID)
			}
		}
	}

	if expiredCount > 0 {
		log.Printf("EXPIRED_SUBSCRIPTION: Проверка завершена. Найдено истекших подписок: %d, отключено конфигов: %d",
			expiredCount, disabledCount)
	} else {
		log.Printf("EXPIRED_SUBSCRIPTION: Проверка завершена. Истекших подписок не найдено")
	}
}

// isSubscriptionExpired проверяет, истекла ли подписка пользователя
func (ess *ExpiredSubscriptionService) isSubscriptionExpired(user *common.User, now time.Time) bool {
	// Проверяем, что у пользователя есть активная подписка
	if !user.HasActiveConfig {
		return false
	}

	// Проверяем, установлено ли время истечения
	if user.ExpiryTime <= 0 {
		return false
	}

	// Проверяем, истекла ли подписка
	return user.ExpiryTime <= now.UnixMilli()
}

// disableExpiredSubscription отключает истекшую подписку пользователя
func (ess *ExpiredSubscriptionService) disableExpiredSubscription(user *common.User) error {
	log.Printf("EXPIRED_SUBSCRIPTION: Отключение истекшей подписки для пользователя %d", user.TelegramID)

	// Отключаем конфиг в панели управления
	if user.Email != "" {
		err := ess.configManager.DisableConfig(user.Email)
		if err != nil {
			log.Printf("EXPIRED_SUBSCRIPTION: Ошибка отключения конфига в панели для пользователя %d: %v",
				user.TelegramID, err)
			// Не возвращаем ошибку, продолжаем обновление в базе данных
		}
	}

	// Обновляем статус пользователя в базе данных
	user.HasActiveConfig = false
	// Оставляем ExpiryTime как есть, чтобы сохранить информацию о том, когда подписка истекла

	err := common.UpdateUser(user)
	if err != nil {
		return fmt.Errorf("ошибка обновления пользователя в базе данных: %v", err)
	}

	// ВАЖНО: Проверяем, можно ли сразу включить конфиг обратно, если у пользователя есть баланс
	if user.Balance > 0 && common.AUTO_BILLING_ENABLED && !common.TARIFF_MODE_ENABLED {
		canAffordDays := int(user.Balance / float64(common.PRICE_PER_DAY))
		if canAffordDays > 0 {
			log.Printf("EXPIRED_SUBSCRIPTION: У пользователя %d есть баланс %.2f₽ (доступно %d дней), запускаем принудительный пересчет для включения конфига",
				user.TelegramID, user.Balance, canAffordDays)

			// Запускаем принудительный пересчет баланса для включения конфига
			go func() {
				time.Sleep(1 * time.Second) // Небольшая задержка
				common.ForceBalanceRecalculation(user.TelegramID)
			}()
		}
	}

	// Отправляем уведомление пользователю
	if ess.bot != nil {
		ess.sendExpiredSubscriptionNotification(user)
	}

	// Отправляем уведомление администратору
	if common.ADMIN_NOTIFICATIONS_ENABLED && common.ADMIN_CONFIG_BLOCKING_ENABLED {
		ess.sendAdminNotification(user)
	}

	return nil
}

// sendExpiredSubscriptionNotification отправляет уведомление пользователю об истечении подписки
func (ess *ExpiredSubscriptionService) sendExpiredSubscriptionNotification(user *common.User) {
	message := "⚠️ <b>Ваша подписка истекла!</b>\n\n" +
		"Время действия вашей подписки закончилось, и доступ к VPN был приостановлен.\n\n" +
		"Для возобновления доступа пополните баланс и продлите подписку.\n\n" +
		"💰 Ваш текущий баланс: %.2f₽\n" +
		"💸 Стоимость дня: %d₽\n\n" +
		"Нажмите /start для пополнения баланса и продления подписки."

	msg := tgbotapi.NewMessage(user.TelegramID,
		fmt.Sprintf(message, user.Balance, common.PRICE_PER_DAY))
	msg.ParseMode = tgbotapi.ModeHTML

	_, err := ess.bot.Send(msg)
	if err != nil {
		log.Printf("EXPIRED_SUBSCRIPTION: Ошибка отправки уведомления пользователю %d: %v",
			user.TelegramID, err)
	} else {
		log.Printf("EXPIRED_SUBSCRIPTION: Уведомление об истечении подписки отправлено пользователю %d",
			user.TelegramID)
	}
}

// sendAdminNotification отправляет уведомление администратору об отключении конфига
func (ess *ExpiredSubscriptionService) sendAdminNotification(user *common.User) {
	expiryTime := time.UnixMilli(user.ExpiryTime).Format("2006-01-02 15:04:05")

	message := fmt.Sprintf("🔔 <b>Уведомление об отключении конфига</b>\n\n"+
		"📧 Email: %s\n"+
		"👤 Пользователь: %s (ID: %d)\n"+
		"⏰ Время истечения: %s\n"+
		"💰 Баланс: %.2f₽\n\n"+
		"Конфиг был автоматически отключен из-за истечения подписки.",
		user.Email, user.FirstName, user.TelegramID, expiryTime, user.Balance)

	msg := tgbotapi.NewMessage(common.ADMIN_ID, message)
	msg.ParseMode = tgbotapi.ModeHTML

	_, err := ess.bot.Send(msg)
	if err != nil {
		log.Printf("EXPIRED_SUBSCRIPTION: Ошибка отправки уведомления администратору: %v", err)
	} else {
		log.Printf("EXPIRED_SUBSCRIPTION: Уведомление об отключении конфига отправлено администратору")
	}
}

// ForceCheckExpiredSubscriptions экспортированный метод для принудительной проверки истекших подписок
func (ess *ExpiredSubscriptionService) ForceCheckExpiredSubscriptions() {
	log.Printf("EXPIRED_SUBSCRIPTION: Принудительная проверка истекших подписок")
	ess.checkExpiredSubscriptions()
}
