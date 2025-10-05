package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
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

	// ДОПОЛНИТЕЛЬНАЯ ПРОВЕРКА: Ищем рассинхронизацию с панелью управления
	panelDisabledCount := ess.checkPanelSyncAndDisableExpired()
	disabledCount += panelDisabledCount

	if expiredCount > 0 || panelDisabledCount > 0 {
		log.Printf("EXPIRED_SUBSCRIPTION: Проверка завершена. Найдено истекших подписок: %d, отключено конфигов: %d (включая рассинхронизацию с панелью)",
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

	// Отключаем конфиг в панели управления и ротируем UUID для немедленного обрыва активных сессий
	if user.Email != "" {
		newUUID, err := ess.configManager.DisableAndRotateConfig(user.Email)
		if err != nil {
			log.Printf("EXPIRED_SUBSCRIPTION: Ошибка отключения конфига и ротации UUID в панели для пользователя %d: %v",
				user.TelegramID, err)
			// Не возвращаем ошибку, продолжаем обновление в базе данных
		} else {
			log.Printf("EXPIRED_SUBSCRIPTION: Конфиг успешно отключен и UUID обновлен для пользователя %d (новый UUID: %s)",
				user.TelegramID, newUUID)
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
	// Включаем конфиг только если баланса хватает хотя бы на 1 день (>= PRICE_PER_DAY)
	if user.Balance >= float64(common.PRICE_PER_DAY) && common.AUTO_BILLING_ENABLED && !common.TARIFF_MODE_ENABLED {
		canAffordDays := int(user.Balance / float64(common.PRICE_PER_DAY))
		log.Printf("EXPIRED_SUBSCRIPTION: У пользователя %d есть баланс %.2f₽ (доступно %d дней), запускаем принудительный пересчет для включения конфига",
			user.TelegramID, user.Balance, canAffordDays)

		// Запускаем принудительный пересчет баланса для включения конфига
		go func() {
			time.Sleep(1 * time.Second) // Небольшая задержка
			common.ForceBalanceRecalculation(user.TelegramID)
		}()
	} else if user.Balance > 0 && user.Balance < float64(common.PRICE_PER_DAY) {
		log.Printf("EXPIRED_SUBSCRIPTION: У пользователя %d недостаточный баланс %.2f₽ (требуется минимум %d₽), конфиг остается отключенным",
			user.TelegramID, user.Balance, common.PRICE_PER_DAY)
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

// checkPanelSyncAndDisableExpired проверяет рассинхронизацию с панелью управления и отключает истекшие конфиги
func (ess *ExpiredSubscriptionService) checkPanelSyncAndDisableExpired() int {
	log.Printf("EXPIRED_SUBSCRIPTION: Проверка рассинхронизации с панелью управления...")

	// Авторизуемся в панели
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("EXPIRED_SUBSCRIPTION: Ошибка авторизации в панели для проверки синхронизации: %v", err)
		return 0
	}

	// Получаем inbound
	targetInbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Printf("EXPIRED_SUBSCRIPTION: Ошибка получения inbound для проверки синхронизации: %v", err)
		return 0
	}

	if targetInbound == nil {
		log.Printf("EXPIRED_SUBSCRIPTION: Inbound с ID %d не найден для проверки синхронизации", common.INBOUND_ID)
		return 0
	}

	// Парсим settings
	var settings common.Settings
	if err := json.Unmarshal([]byte(targetInbound.Settings), &settings); err != nil {
		log.Printf("EXPIRED_SUBSCRIPTION: Ошибка парсинга settings для проверки синхронизации: %v", err)
		return 0
	}

	now := time.Now()
	disabledCount := 0

	// Проверяем всех клиентов в панели
	for _, client := range settings.Clients {
		// Ищем пользователя в базе данных по email (который равен telegram_id)
		telegramID, err := strconv.ParseInt(client.Email, 10, 64)
		if err != nil {
			continue // Пропускаем клиентов с нечисловыми email
		}

		user, err := common.GetUserByTelegramID(telegramID)
		if err != nil || user == nil {
			continue // Пользователь не найден в базе данных
		}

		// Проверяем рассинхронизацию: конфиг активен в панели, но отключен в БД
		if client.Enable && !user.HasActiveConfig {
			// Проверяем, истекла ли подписка по времени
			if client.ExpiryTime > 0 && client.ExpiryTime <= now.UnixMilli() {
				log.Printf("EXPIRED_SUBSCRIPTION: Найдена рассинхронизация для пользователя %d (email: %s) - конфиг активен в панели, но подписка истекла (баланс: %.2f₽)",
					user.TelegramID, client.Email, user.Balance)

				// ВАЖНО: Если баланс <= 0, отключаем конфиг и блокируем повторное включение
				// Если баланс > 0 и >= PRICE_PER_DAY, запускаем пересчет вместо отключения
				if user.Balance >= float64(common.PRICE_PER_DAY) && common.AUTO_BILLING_ENABLED && !common.TARIFF_MODE_ENABLED {
					log.Printf("EXPIRED_SUBSCRIPTION: У пользователя %d есть баланс %.2f₽ (>= %d₽), пропускаем отключение - пересчет баланса включит конфиг автоматически",
						user.TelegramID, user.Balance, common.PRICE_PER_DAY)
					continue
				}

				// Отключаем конфиг в панели и ротируем UUID только если баланса недостаточно
				newUUID, err := ess.configManager.DisableAndRotateConfig(client.Email)
				if err != nil {
					log.Printf("EXPIRED_SUBSCRIPTION: Ошибка отключения конфига и ротации UUID в панели для пользователя %d: %v",
						user.TelegramID, err)
				} else {
					disabledCount++
					log.Printf("EXPIRED_SUBSCRIPTION: Конфиг в панели успешно отключен и UUID обновлен для пользователя %d (исправлена рассинхронизация, баланс недостаточен: %.2f₽, новый UUID: %s)",
						user.TelegramID, user.Balance, newUUID)

					// Отправляем уведомление администратору о исправлении рассинхронизации
					if common.ADMIN_NOTIFICATIONS_ENABLED && common.ADMIN_CONFIG_BLOCKING_ENABLED {
						ess.sendPanelSyncNotification(user, client.ExpiryTime)
					}
				}
			}
		}
	}

	if disabledCount > 0 {
		log.Printf("EXPIRED_SUBSCRIPTION: Исправлено рассинхронизаций с панелью управления: %d", disabledCount)
	} else {
		log.Printf("EXPIRED_SUBSCRIPTION: Рассинхронизаций с панелью управления не найдено")
	}

	return disabledCount
}

// sendPanelSyncNotification отправляет уведомление администратору о исправлении рассинхронизации
func (ess *ExpiredSubscriptionService) sendPanelSyncNotification(user *common.User, expiryTime int64) {
	expiryTimeStr := time.UnixMilli(expiryTime).Format("2006-01-02 15:04:05")

	message := fmt.Sprintf("🔧 <b>Исправлена рассинхронизация с панелью</b>\n\n"+
		"📧 Email: %s\n"+
		"👤 Пользователь: %s (ID: %d)\n"+
		"⏰ Время истечения: %s\n"+
		"💰 Баланс: %.2f₽\n\n"+
		"Конфиг был активен в панели управления, но подписка истекла. Конфиг отключен в панели.",
		user.Email, user.FirstName, user.TelegramID, expiryTimeStr, user.Balance)

	msg := tgbotapi.NewMessage(common.ADMIN_ID, message)
	msg.ParseMode = tgbotapi.ModeHTML

	_, err := ess.bot.Send(msg)
	if err != nil {
		log.Printf("EXPIRED_SUBSCRIPTION: Ошибка отправки уведомления администратору о рассинхронизации: %v", err)
	} else {
		log.Printf("EXPIRED_SUBSCRIPTION: Уведомление об исправлении рассинхронизации отправлено администратору")
	}
}

// ForceCheckExpiredSubscriptions экспортированный метод для принудительной проверки истекших подписок
func (ess *ExpiredSubscriptionService) ForceCheckExpiredSubscriptions() {
	log.Printf("EXPIRED_SUBSCRIPTION: Принудительная проверка истекших подписок")
	ess.checkExpiredSubscriptions()
}
