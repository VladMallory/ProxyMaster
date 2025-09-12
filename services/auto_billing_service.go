package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// AutoBillingService управляет автоматическим списанием средств
type AutoBillingService struct {
	bot                 *tgbotapi.BotAPI
	dailyBillingTicker  *time.Ticker
	balanceRecalcTicker *time.Ticker
}

// NewAutoBillingService создает новый сервис автосписания
func NewAutoBillingService(bot *tgbotapi.BotAPI) *AutoBillingService {
	return &AutoBillingService{
		bot: bot,
	}
}

// Start запускает сервис автосписания
func (abs *AutoBillingService) Start() {
	if !common.AUTO_BILLING_ENABLED {
		log.Printf("AUTO_BILLING: Автосписание отключено в конфигурации")
		return
	}

	if common.TARIFF_MODE_ENABLED {
		log.Printf("AUTO_BILLING: Включен тарифный режим, автосписание не запускается")
		return
	}

	log.Printf("AUTO_BILLING: Запуск сервиса автосписания")

	// Ежедневное списание в полночь
	abs.startDailyBilling()

	// Пересчет дней по балансу
	abs.startBalanceRecalculation()

	log.Printf("AUTO_BILLING: Сервис автосписания успешно запущен")
}

// Stop останавливает сервис автосписания
func (abs *AutoBillingService) Stop() {
	if abs.dailyBillingTicker != nil {
		abs.dailyBillingTicker.Stop()
		log.Printf("AUTO_BILLING: Ежедневное списание остановлено")
	}
	if abs.balanceRecalcTicker != nil {
		abs.balanceRecalcTicker.Stop()
		log.Printf("AUTO_BILLING: Пересчет баланса остановлен")
	}
}

// startDailyBilling запускает ежедневное списание
func (abs *AutoBillingService) startDailyBilling() {
	// Вычисляем время до следующей полуночи
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	timeToMidnight := nextMidnight.Sub(now)

	log.Printf("AUTO_BILLING: Ежедневное списание начнется через %v (в полночь)", timeToMidnight)

	// Запускаем первое списание через время до полуночи
	go func() {
		timer := time.NewTimer(timeToMidnight)
		<-timer.C

		// Выполняем первое списание
		abs.processDailyBilling()

		// Запускаем ежедневный ticker
		abs.dailyBillingTicker = time.NewTicker(24 * time.Hour)
		for range abs.dailyBillingTicker.C {
			abs.processDailyBilling()
		}
	}()
}

// startBalanceRecalculation запускает пересчет дней по балансу
func (abs *AutoBillingService) startBalanceRecalculation() {
	interval := time.Duration(common.BALANCE_RECALC_INTERVAL) * time.Minute
	log.Printf("AUTO_BILLING: Пересчет дней по балансу каждые %v", interval)

	abs.balanceRecalcTicker = time.NewTicker(interval)

	// Выполняем первый пересчет сразу
	go abs.processBalanceRecalculation()

	go func() {
		for range abs.balanceRecalcTicker.C {
			abs.processBalanceRecalculation()
		}
	}()
}

// processDailyBilling выполняет ежедневное списание
func (abs *AutoBillingService) processDailyBilling() {
	// Проверяем, что автосписание все еще включено
	if !common.AUTO_BILLING_ENABLED || common.TARIFF_MODE_ENABLED {
		log.Printf("AUTO_BILLING: Автосписание отключено или включен тарифный режим, пропускаем ежедневное списание")
		return
	}

	log.Printf("AUTO_BILLING: Начало ежедневного списания")

	// Получаем всех пользователей с активными конфигами
	users, err := common.GetUsersWithActiveConfigs()
	if err != nil {
		log.Printf("AUTO_BILLING: Ошибка получения пользователей: %v", err)
		return
	}

	billedCount := 0
	disabledCount := 0

	for _, user := range users {
		// Проверяем, что конфиг действительно активен
		if !common.IsConfigActive(&user) {
			continue
		}

		// Проверяем баланс
		if user.Balance >= float64(common.PRICE_PER_DAY) {
			// Списываем дневную плату
			err := abs.chargeDailyFee(&user)
			if err != nil {
				log.Printf("AUTO_BILLING: Ошибка списания для пользователя %d: %v", user.TelegramID, err)
				continue
			}
			billedCount++
			log.Printf("AUTO_BILLING: Списано %d₽ с пользователя %d, остаток: %.2f₽",
				common.PRICE_PER_DAY, user.TelegramID, user.Balance-float64(common.PRICE_PER_DAY))
		} else {
			// Недостаточно средств - отключаем конфиг
			err := abs.disableUserConfig(&user)
			if err != nil {
				log.Printf("AUTO_BILLING: Ошибка отключения конфига для пользователя %d: %v", user.TelegramID, err)
				continue
			}
			disabledCount++
			log.Printf("AUTO_BILLING: Конфиг отключен для пользователя %d (недостаточно средств: %.2f₽)",
				user.TelegramID, user.Balance)
		}
	}

	log.Printf("AUTO_BILLING: Ежедневное списание завершено. Списано: %d, отключено: %d", billedCount, disabledCount)
}

// chargeDailyFee списывает дневную плату
func (abs *AutoBillingService) chargeDailyFee(user *common.User) error {
	// Списываем средства
	user.Balance -= float64(common.PRICE_PER_DAY)

	// Обновляем пользователя в базе
	return common.UpdateUser(user)
}

// disableUserConfig отключает конфиг пользователя
func (abs *AutoBillingService) disableUserConfig(user *common.User) error {
	// Устанавливаем время истечения на текущее время
	user.ExpiryTime = time.Now().UnixMilli()
	user.HasActiveConfig = false

	// Обновляем пользователя в базе
	err := common.UpdateUser(user)
	if err != nil {
		return err
	}

	// Отправляем уведомление пользователю
	if abs.bot != nil {
		message := "⚠️ <b>Ваша подписка приостановлена!</b>\n\n" +
			"На вашем балансе недостаточно средств для автоматического продления.\n" +
			"Пополните баланс для возобновления доступа к VPN.\n\n" +
			"💰 Ваш текущий баланс: %.2f₽\n" +
			"💸 Стоимость дня: %d₽\n\n" +
			"Нажмите /start для пополнения баланса."

		msg := tgbotapi.NewMessage(user.TelegramID,
			fmt.Sprintf(message, user.Balance, common.PRICE_PER_DAY))
		msg.ParseMode = tgbotapi.ModeHTML

		_, err := abs.bot.Send(msg)
		if err != nil {
			log.Printf("AUTO_BILLING: Ошибка отправки уведомления пользователю %d: %v", user.TelegramID, err)
		}

		// Отправляем уведомление администратору о блокировке конфига
		common.SendConfigBlockingNotificationToAdmin(user)
	}

	return nil
}

// ProcessBalanceRecalculation экспортированный метод для принудительного пересчета баланса
func (abs *AutoBillingService) ProcessBalanceRecalculation() {
	abs.processBalanceRecalculation()
}

// processBalanceRecalculation выполняет пересчет дней по балансу
func (abs *AutoBillingService) processBalanceRecalculation() {
	// Проверяем, что автосписание все еще включено
	if !common.AUTO_BILLING_ENABLED || common.TARIFF_MODE_ENABLED {
		log.Printf("AUTO_BILLING: Автосписание отключено или включен тарифный режим, пропускаем пересчет баланса")
		return
	}

	log.Printf("AUTO_BILLING: Начало пересчета дней по балансу")

	// Получаем всех пользователей
	users, err := common.GetAllUsers()
	if err != nil {
		log.Printf("AUTO_BILLING: Ошибка получения пользователей для пересчета: %v", err)
		return
	}

	recalculatedCount := 0
	now := time.Now()

	for _, user := range users {
		// Пересчитываем только для пользователей с балансом больше 0
		if user.Balance <= 0 {
			continue
		}

		// Вычисляем количество дней по балансу
		availableDays := int(user.Balance / float64(common.PRICE_PER_DAY))

		if availableDays <= 0 {
			continue
		}

		// Если у пользователя нет активного конфига, создаем новый
		if !user.HasActiveConfig {
			err := abs.createConfigFromBalance(&user, availableDays)
			if err != nil {
				log.Printf("AUTO_BILLING: Ошибка создания конфига для пользователя %d: %v", user.TelegramID, err)
				continue
			}
			recalculatedCount++
			log.Printf("AUTO_BILLING: Создан конфиг на %d дней для пользователя %d", availableDays, user.TelegramID)
		} else {
			// Если конфиг есть, всегда синхронизируем время истечения с балансом
			currentExpiryTime := time.UnixMilli(user.ExpiryTime)

			// Вычисляем желаемое время истечения от текущего момента
			desiredExpiryTime := now.Add(time.Duration(availableDays) * 24 * time.Hour)

			// В режиме автосписания всегда синхронизируем время с балансом
			// Проверяем, отличается ли желаемое время от текущего больше чем на 1 час
			timeDiff := desiredExpiryTime.Sub(currentExpiryTime)
			absDiff := timeDiff
			if absDiff < 0 {
				absDiff = -absDiff
			}

			if absDiff > time.Hour {
				log.Printf("AUTO_BILLING: Принудительная синхронизация времени истечения для пользователя %d", user.TelegramID)
				log.Printf("AUTO_BILLING: Текущее время в базе: %s, желаемое время: %s, разница: %v",
					currentExpiryTime.Format("2006-01-02 15:04"),
					desiredExpiryTime.Format("2006-01-02 15:04"),
					timeDiff)

				// Принудительно обновляем время истечения
				err := abs.updateConfigExpiry(&user, availableDays)
				if err != nil {
					log.Printf("AUTO_BILLING: Ошибка принудительного обновления конфига для пользователя %d: %v", user.TelegramID, err)
					continue
				}
				recalculatedCount++
				log.Printf("AUTO_BILLING: Принудительно обновлен конфиг на %d дней для пользователя %d", availableDays, user.TelegramID)
			} else {
				log.Printf("AUTO_BILLING: Конфиг пользователя %d уже синхронизирован (до %s, доступно дней: %d)",
					user.TelegramID, currentExpiryTime.Format("2006-01-02 15:04"), availableDays)
			}
		}
	}

	log.Printf("AUTO_BILLING: Пересчет дней завершен. Обновлено: %d конфигов", recalculatedCount)
}

// createConfigFromBalance создает конфиг на основе баланса
func (abs *AutoBillingService) createConfigFromBalance(user *common.User, days int) error {
	// Используем существующую логику создания конфига
	_, err := common.ProcessPayment(user, days)
	return err
}

// updateConfigExpiry принудительно устанавливает время истечения конфига на основе баланса
func (abs *AutoBillingService) updateConfigExpiry(user *common.User, days int) error {
	// Авторизуемся в панели
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("AUTO_BILLING: Ошибка авторизации в панели для пользователя %d: %v", user.TelegramID, err)
		return err
	}

	// Принудительно обновляем время истечения в панели
	err = abs.forceUpdateExpiryTime(sessionCookie, user, days)
	if err != nil {
		log.Printf("AUTO_BILLING: Ошибка принудительного обновления времени в панели для пользователя %d: %v", user.TelegramID, err)
		return err
	}

	// Обновляем пользователя в базе данных
	return common.UpdateUser(user)
}

// forceUpdateExpiryTime принудительно устанавливает время истечения в панели
func (abs *AutoBillingService) forceUpdateExpiryTime(sessionCookie string, user *common.User, days int) error {
	// Получаем inbound из панели
	inbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		return fmt.Errorf("ошибка получения inbound: %v", err)
	}

	// Парсим settings
	var settings common.Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return fmt.Errorf("ошибка парсинга settings: %v", err)
	}

	// Находим клиента пользователя
	clientFound := false
	newExpiryTime := time.Now().Add(time.Duration(days) * 24 * time.Hour).UnixMilli()

	for i, client := range settings.Clients {
		telegramIDStr := fmt.Sprintf("%d", user.TelegramID)
		if strings.HasPrefix(client.Email, telegramIDStr+"_") ||
			strings.HasPrefix(client.Email, telegramIDStr+" ") ||
			client.Email == telegramIDStr {

			log.Printf("AUTO_BILLING: Принудительное обновление времени для клиента %s: %d -> %d",
				client.Email, client.ExpiryTime, newExpiryTime)

			// Обновляем время истечения
			settings.Clients[i].ExpiryTime = newExpiryTime
			settings.Clients[i].Enable = true
			settings.Clients[i].UpdatedAt = time.Now().UnixMilli()

			// Обновляем email с новой датой если нужно
			if common.SHOW_DATES_IN_CONFIGS {
				expiryDate := time.UnixMilli(newExpiryTime).Format("2006 02 01")
				settings.Clients[i].Email = fmt.Sprintf("%d до %s", user.TelegramID, expiryDate)
			}

			// Обновляем данные пользователя
			user.ExpiryTime = newExpiryTime
			user.HasActiveConfig = true
			user.Email = settings.Clients[i].Email
			user.UpdatedAt = time.Now()

			clientFound = true
			break
		}
	}

	if !clientFound {
		return fmt.Errorf("клиент пользователя %d не найден в панели", user.TelegramID)
	}

	// Сериализуем обратно
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("ошибка сериализации settings: %v", err)
	}

	// Обновляем inbound
	inbound.Settings = string(settingsJSON)
	err = common.UpdateInbound(sessionCookie, *inbound)
	if err != nil {
		return fmt.Errorf("ошибка обновления inbound: %v", err)
	}

	// ===== КРИТИЧЕСКИЙ FIX ДЛЯ СИНХРОНИЗАЦИИ КЛИЕНТОВ =====
	// ПРОБЛЕМА: После автосписания панель 3x-ui показывает правильное время (например, 23 часа),
	// но клиентские приложения (happ, v2rayTun, etc.) продолжают показывать старые данные (например, 18 дней).
	//
	// ПРИЧИНА: Панель обновляет время истечения, но клиенты кешируют конфигурацию и не получают
	// уведомление о необходимости обновления. Это происходит только при автосписании, но НЕ в тарифном режиме.
	//
	// РЕШЕНИЕ: Используем ту же проверенную логику, что работает в тарифном режиме (ProcessPayment).
	// ForceResetDepletedStatus выполняет двухфазовый сброс состояния клиента:
	// ФАЗА A: depleted/exhausted=TRUE + disable client (пауза 1000мс)
	// ФАЗА B: depleted/exhausted=FALSE + enable client с новыми данными
	// Это заставляет ВСЕ клиентские приложения "увидеть" изменения и обновить конфигурацию.
	//
	// РЕЗУЛЬТАТ: Синхронизация панели и клиентов восстановлена - все показывают одинаковое время!
	log.Printf("AUTO_BILLING: Принудительный сброс состояния 'исчерпано' для синхронизации клиентов TelegramID=%d", user.TelegramID)
	if err := common.ForceResetDepletedStatus(sessionCookie, user.TelegramID); err != nil {
		log.Printf("AUTO_BILLING: Предупреждение - не удалось сбросить состояние 'исчерпано' для TelegramID=%d: %v", user.TelegramID, err)
		// Не возвращаем ошибку, так как основная операция уже выполнена
	} else {
		log.Printf("AUTO_BILLING: Состояние 'исчерпано' успешно сброшено для TelegramID=%d - клиенты обновятся", user.TelegramID)
	}

	log.Printf("AUTO_BILLING: Конфигурация принудительно обновлена (время+синхронизация) для пользователя %d на %d дней (до %s) - FIX как в тарифах",
		user.TelegramID, days, time.UnixMilli(newExpiryTime).Format("2006-01-02 15:04"))

	return nil
}
