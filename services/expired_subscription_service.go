package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
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
// ТЕПЕРЬ ИСПОЛЬЗУЕТ ОПТИМИЗИРОВАННУЮ ВЕРСИЮ: БАТЧИНГ + ГОРУТИНЫ для максимальной производительности
func (ess *ExpiredSubscriptionService) checkExpiredSubscriptions() {
	// ОПТИМИЗАЦИЯ: Используем оптимизированную версию (батчинг + горутины) для максимальной производительности
	// Результат: 3 API вызова вместо N×2 + параллельная обработка пользователей
	ess.checkExpiredSubscriptionsOptimized()
}

// checkExpiredSubscriptionsOriginal оригинальная версия проверки истекших подписок (для тестирования)
func (ess *ExpiredSubscriptionService) checkExpiredSubscriptionsOriginal() {
	log.Printf("EXPIRED_SUBSCRIPTION: Начинаем ОРИГИНАЛЬНУЮ проверку истекших подписок...")

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
		log.Printf("EXPIRED_SUBSCRIPTION: ОРИГИНАЛЬНАЯ проверка завершена. Найдено истекших подписок: %d, отключено конфигов: %d (включая рассинхронизацию с панелью)",
			expiredCount, disabledCount)
	} else {
		log.Printf("EXPIRED_SUBSCRIPTION: ОРИГИНАЛЬНАЯ проверка завершена. Истекших подписок не найдено")
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
	ess.checkExpiredSubscriptionsOptimized()
}

// ForceCheckExpiredSubscriptionsOriginal экспортированный метод для тестирования оригинальной версии
func (ess *ExpiredSubscriptionService) ForceCheckExpiredSubscriptionsOriginal() {
	log.Printf("EXPIRED_SUBSCRIPTION: Принудительная проверка истекших подписок (оригинальная версия)")
	ess.checkExpiredSubscriptionsOriginal()
}

// ForceCheckExpiredSubscriptionsOptimized экспортированный метод для тестирования оптимизированной версии
func (ess *ExpiredSubscriptionService) ForceCheckExpiredSubscriptionsOptimized() {
	log.Printf("EXPIRED_SUBSCRIPTION: Принудительная проверка истекших подписок (оптимизированная версия)")
	ess.checkExpiredSubscriptionsOptimized()
}

// ============================================================================
// ОПТИМИЗИРОВАННЫЕ ФУНКЦИИ EXPIRED_SUBSCRIPTION (БАТЧИНГ + ПАРАЛЛЕЛИЗМ)
// ============================================================================

// checkExpiredSubscriptionsOptimized оптимизированная версия проверки истекших подписок с батчингом и горутинами
// ОСНОВНАЯ ОПТИМИЗАЦИЯ: Минимальные API вызовы + максимальная скорость обработки
//
// ПРОБЛЕМА ИЗНАЧАЛЬНОЙ ВЕРСИИ:
// - Для каждого пользователя: disableExpiredSubscription() → configManager.DisableAndRotateConfig()
// - Каждый DisableAndRotateConfig() делает свой Login()
// - Дополнительная проверка: checkPanelSyncAndDisableExpired() с собственным Login()
// - Результат: N×2 API вызовов для N пользователей + последовательная обработка
//
// РЕШЕНИЕ ОПТИМИЗИРОВАННОЙ ВЕРСИИ:
// - БАТЧИНГ: Один Login() + один GetInbound() + один UpdateInbound() для всех изменений
// - ГОРУТИНЫ: Параллельная обработка пользователей в памяти
// - ИНТЕГРАЦИЯ: checkPanelSyncAndDisableExpired() встроена в основной поток
// - Результат: 3 API вызова + максимальная скорость обработки
func (ess *ExpiredSubscriptionService) checkExpiredSubscriptionsOptimized() {
	log.Printf("EXPIRED_SUBSCRIPTION: Начинаем ОПТИМИЗИРОВАННУЮ проверку истекших подписок (батчинг + горутины)...")

	// ===== ЭТАП 1: БАТЧИНГ - ПОЛУЧАЕМ ДАННЫЕ ОДИН РАЗ =====
	// Получаем всех пользователей с активными конфигами
	users, err := common.GetUsersWithActiveConfigs()
	if err != nil {
		log.Printf("EXPIRED_SUBSCRIPTION: Ошибка получения пользователей: %v", err)
		return
	}

	if len(users) == 0 {
		log.Printf("EXPIRED_SUBSCRIPTION: Нет пользователей для проверки")
		return
	}

	// БАТЧИНГ: Получаем сессию ОДИН РАЗ для всех операций
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("EXPIRED_SUBSCRIPTION: Ошибка авторизации в панели: %v", err)
		return
	}

	// БАТЧИНГ: Получаем данные inbound ОДИН РАЗ для всех пользователей
	inbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Printf("EXPIRED_SUBSCRIPTION: Ошибка получения inbound: %v", err)
		return
	}

	// БАТЧИНГ: Парсим settings ОДИН РАЗ и работаем с данными в памяти
	var settings common.Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		log.Printf("EXPIRED_SUBSCRIPTION: Ошибка парсинга settings: %v", err)
		return
	}

	// ===== ЭТАП 2: ГОРУТИНЫ - ПАРАЛЛЕЛЬНАЯ ПРОВЕРКА =====
	now := time.Now()
	var usersToDisable []*common.User
	var mu sync.Mutex
	var wg sync.WaitGroup

	// ГОРУТИНЫ: Обрабатываем пользователей параллельно с ограничением
	const maxConcurrency = 5 // Максимум 5 одновременных горутин
	semaphore := make(chan struct{}, maxConcurrency)

	log.Printf("EXPIRED_SUBSCRIPTION: Запускаем параллельную проверку %d пользователей с максимальной параллельностью %d", len(users), maxConcurrency)

	for _, user := range users {
		wg.Add(1)
		go func(u common.User) {
			defer wg.Done()

			// Получаем семафор для ограничения параллельности
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Проверяем, истекла ли подписка (без API вызовов)
			if ess.isSubscriptionExpired(&u, now) {
				mu.Lock()
				usersToDisable = append(usersToDisable, &u)
				mu.Unlock()

				log.Printf("EXPIRED_SUBSCRIPTION: Найдена истекшая подписка для пользователя %d (email: %s)",
					u.TelegramID, u.Email)
			}
		}(user)
	}

	wg.Wait()
	log.Printf("EXPIRED_SUBSCRIPTION: Параллельная проверка завершена. Найдено истекших подписок: %d", len(usersToDisable))

	// ===== ЭТАП 3: ГОРУТИНЫ - ПАРАЛЛЕЛЬНОЕ ОТКЛЮЧЕНИЕ =====
	disabledCount := 0
	var disabledMu sync.Mutex

	if len(usersToDisable) > 0 {
		log.Printf("EXPIRED_SUBSCRIPTION: Запускаем параллельное отключение %d истекших подписок", len(usersToDisable))

		for _, user := range usersToDisable {
			wg.Add(1)
			go func(u *common.User) {
				defer wg.Done()

				// Получаем семафор для ограничения параллельности
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// ГОРУТИНЫ: Отключаем конфиг в памяти и обновляем базу данных
				if ess.processExpiredUserInMemory(u, &settings, now) {
					disabledMu.Lock()
					disabledCount++
					disabledMu.Unlock()
				}
			}(user)
		}

		wg.Wait()
	}

	// ===== ЭТАП 4: ИНТЕГРИРОВАННАЯ ПРОВЕРКА РАССИНХРОНИЗАЦИИ =====
	// Встроенная проверка рассинхронизации с панелью (без дополнительных API вызовов)
	panelDisabledCount := ess.checkPanelSyncAndDisableExpiredInMemory(&settings, now)
	disabledCount += panelDisabledCount

	// ===== ЭТАП 5: БАТЧИНГ - ОБНОВЛЯЕМ ПАНЕЛЬ ОДИН РАЗ =====
	if disabledCount > 0 {
		// БАТЧИНГ: Обновляем inbound ОДИН РАЗ для всех изменений
		settingsJSON, err := json.Marshal(settings)
		if err != nil {
			log.Printf("EXPIRED_SUBSCRIPTION: Ошибка сериализации settings: %v", err)
			return
		}

		inbound.Settings = string(settingsJSON)
		if err := common.UpdateInbound(sessionCookie, *inbound); err != nil {
			log.Printf("EXPIRED_SUBSCRIPTION: Ошибка обновления inbound: %v", err)
		} else {
			log.Printf("EXPIRED_SUBSCRIPTION: Inbound успешно обновлен с %d изменениями", disabledCount)
		}
	}

	if disabledCount > 0 {
		log.Printf("EXPIRED_SUBSCRIPTION: ОПТИМИЗИРОВАННАЯ проверка завершена. Отключено конфигов: %d (включая рассинхронизацию с панелью)", disabledCount)
	} else {
		log.Printf("EXPIRED_SUBSCRIPTION: ОПТИМИЗИРОВАННАЯ проверка завершена. Истекших подписок не найдено")
	}
}

// processExpiredUserInMemory обрабатывает истекшего пользователя в памяти (для оптимизированной версии)
// ГОРУТИНЫ: Работает с данными в памяти, безопасна для параллельного выполнения
func (ess *ExpiredSubscriptionService) processExpiredUserInMemory(user *common.User, settings *common.Settings, now time.Time) bool {
	// ГОРУТИНЫ: Отключаем конфиг в settings (данные в памяти)
	telegramIDStr := fmt.Sprintf("%d", user.TelegramID)
	for i, client := range settings.Clients {
		if client.Email == user.Email ||
			client.Email == telegramIDStr ||
			strings.HasPrefix(client.Email, telegramIDStr+"_") ||
			strings.HasPrefix(client.Email, telegramIDStr+" ") {

			// Отключаем конфиг в памяти
			settings.Clients[i].Enable = false

			log.Printf("EXPIRED_SUBSCRIPTION: Конфиг отключен для пользователя %d (email: %s)", user.TelegramID, user.Email)
			break
		}
	}

	// Обновляем статус пользователя в базе данных
	user.HasActiveConfig = false

	// Обновляем пользователя в базе данных
	if err := common.UpdateUser(user); err != nil {
		log.Printf("EXPIRED_SUBSCRIPTION: Ошибка обновления пользователя %d: %v", user.TelegramID, err)
		return false
	}

	// ВАЖНО: Проверяем, можно ли сразу включить конфиг обратно, если у пользователя есть баланс
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

	// ГОРУТИНЫ: Отправляем уведомления асинхронно
	if ess.bot != nil {
		go ess.sendExpiredSubscriptionNotification(user)
	}
	if common.ADMIN_NOTIFICATIONS_ENABLED && common.ADMIN_CONFIG_BLOCKING_ENABLED {
		go ess.sendAdminNotification(user)
	}

	return true
}

// checkPanelSyncAndDisableExpiredInMemory проверяет рассинхронизацию с панелью управления в памяти
// ИНТЕГРАЦИЯ: Работает с данными в памяти, без дополнительных API вызовов
func (ess *ExpiredSubscriptionService) checkPanelSyncAndDisableExpiredInMemory(settings *common.Settings, now time.Time) int {
	log.Printf("EXPIRED_SUBSCRIPTION: Проверка рассинхронизации с панелью управления в памяти...")

	disabledCount := 0

	// Проверяем всех клиентов в панели (данные уже в памяти)
	for i, client := range settings.Clients {
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

				// ВАЖНО: Если баланс >= PRICE_PER_DAY, пропускаем отключение
				if user.Balance >= float64(common.PRICE_PER_DAY) && common.AUTO_BILLING_ENABLED && !common.TARIFF_MODE_ENABLED {
					log.Printf("EXPIRED_SUBSCRIPTION: У пользователя %d есть баланс %.2f₽ (>= %d₽), пропускаем отключение - пересчет баланса включит конфиг автоматически",
						user.TelegramID, user.Balance, common.PRICE_PER_DAY)
					continue
				}

				// Отключаем конфиг в памяти
				settings.Clients[i].Enable = false
				disabledCount++

				log.Printf("EXPIRED_SUBSCRIPTION: Конфиг в панели отключен для пользователя %d (исправлена рассинхронизация, баланс недостаточен: %.2f₽)",
					user.TelegramID, user.Balance)

				// Отправляем уведомление администратору о исправлении рассинхронизации
				if common.ADMIN_NOTIFICATIONS_ENABLED && common.ADMIN_CONFIG_BLOCKING_ENABLED {
					go ess.sendPanelSyncNotification(user, client.ExpiryTime)
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
