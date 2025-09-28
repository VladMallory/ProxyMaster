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

// ProcessBalanceRecalculationForUser экспортированный метод для пересчета баланса конкретного пользователя
func (abs *AutoBillingService) ProcessBalanceRecalculationForUser(telegramID int64) {
	abs.processBalanceRecalculationForUser(telegramID)
}

// processBalanceRecalculation выполняет пересчет дней по балансу
func (abs *AutoBillingService) processBalanceRecalculation() {
	// Проверяем, что автосписание все еще включено
	if !common.AUTO_BILLING_ENABLED || common.TARIFF_MODE_ENABLED {
		log.Printf("AUTO_BILLING: Автосписание отключено или включен тарифный режим, пропускаем пересчет баланса")
		return
	}

	log.Printf("AUTO_BILLING: Начало пересчета дней по балансу")

	// Сначала выполняем диагностику рассинхронизации
	abs.diagnoseAndFixSyncIssues()

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

		// Проверяем наличие активного конфига в любом из инбаундов
		hasAnyActiveConfig := user.HasActiveConfig || user.HasActiveSecondaryConfig

		// Если у пользователя нет активного конфига, создаем новый
		if !hasAnyActiveConfig {
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

// processBalanceRecalculationForUser выполняет пересчет дней по балансу для конкретного пользователя
func (abs *AutoBillingService) processBalanceRecalculationForUser(telegramID int64) {
	// Проверяем, что автосписание все еще включено
	if !common.AUTO_BILLING_ENABLED || common.TARIFF_MODE_ENABLED {
		log.Printf("AUTO_BILLING: Автосписание отключено или включен тарифный режим, пропускаем пересчет баланса для пользователя %d", telegramID)
		return
	}

	log.Printf("AUTO_BILLING: Начало пересчета дней по балансу для пользователя %d", telegramID)

	// Получаем конкретного пользователя
	user, err := common.GetUserByTelegramID(telegramID)
	if err != nil {
		log.Printf("AUTO_BILLING: Ошибка получения пользователя %d для пересчета: %v", telegramID, err)
		return
	}

	// Пересчитываем только для пользователей с балансом больше 0
	if user.Balance <= 0 {
		log.Printf("AUTO_BILLING: У пользователя %d баланс %.2f₽, пропускаем пересчет", telegramID, user.Balance)
		return
	}

	// Убираем проверку на реферальный бонус - ForceBalanceRecalculation должен создавать конфиги для всех пользователей с балансом

	// Вычисляем количество дней по балансу
	availableDays := int(user.Balance / float64(common.PRICE_PER_DAY))

	if availableDays <= 0 {
		log.Printf("AUTO_BILLING: У пользователя %d недостаточно средств для оплаты хотя бы одного дня (%.2f₽, нужно %d₽)",
			telegramID, user.Balance, common.PRICE_PER_DAY)
		return
	}

	now := time.Now()

	// Проверяем наличие активного конфига в любом из инбаундов
	hasAnyActiveConfig := user.HasActiveConfig || user.HasActiveSecondaryConfig

	// Если у пользователя нет активного конфига, создаем новый
	if !hasAnyActiveConfig {
		err := abs.createConfigFromBalance(user, availableDays)
		if err != nil {
			log.Printf("AUTO_BILLING: Ошибка создания конфига для пользователя %d: %v", user.TelegramID, err)
			return
		}
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
			err := abs.updateConfigExpiry(user, availableDays)
			if err != nil {
				log.Printf("AUTO_BILLING: Ошибка принудительного обновления конфига для пользователя %d: %v", user.TelegramID, err)
				return
			}
			log.Printf("AUTO_BILLING: Принудительно обновлен конфиг на %d дней для пользователя %d", availableDays, user.TelegramID)
		} else {
			log.Printf("AUTO_BILLING: Конфиг пользователя %d уже синхронизирован (до %s, доступно дней: %d)",
				user.TelegramID, currentExpiryTime.Format("2006-01-02 15:04"), availableDays)
		}
	}

	log.Printf("AUTO_BILLING: Пересчет дней для пользователя %d завершен", telegramID)
}

// createConfigFromBalance создает конфиг на основе баланса БЕЗ списания денег
func (abs *AutoBillingService) createConfigFromBalance(user *common.User, days int) error {
	log.Printf("AUTO_BILLING: ===== СОЗДАНИЕ КОНФИГА ИЗ БАЛАНСА БЕЗ СПИСАНИЯ =====")
	log.Printf("AUTO_BILLING: Пользователь: %d, Баланс: %.2f₽, Дни: %d", user.TelegramID, user.Balance, days)

	// Создаем конфиг через панель 3x-ui БЕЗ списания денег (как в пробном периоде)
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("AUTO_BILLING: Ошибка авторизации в панели для пользователя %d: %v", user.TelegramID, err)
		return fmt.Errorf("ошибка авторизации в панели: %v", err)
	}

	// Создаем конфиг БЕЗ списания денег (как в пробном периоде)
	err = common.AddTrialClient(sessionCookie, user, days)
	if err != nil {
		log.Printf("AUTO_BILLING: Ошибка создания конфига для пользователя %d: %v", user.TelegramID, err)
		return fmt.Errorf("ошибка создания конфига: %v", err)
	}

	// Создаем конфиг в дополнительном инбаунде, если он включен
	if common.SECONDARY_INBOUND_ENABLED {
		log.Printf("AUTO_BILLING: Создание конфига в дополнительном инбаунде для пользователя %d", user.TelegramID)
		err = common.AddSecondaryClient(sessionCookie, user, days)
		if err != nil {
			log.Printf("AUTO_BILLING: ⚠️ Ошибка создания конфига в дополнительном инбаунде для пользователя %d: %v", user.TelegramID, err)
			// Не прерываем выполнение, основной конфиг уже создан
		} else {
			log.Printf("AUTO_BILLING: ✅ Конфиг в дополнительном инбаунде создан для пользователя %d", user.TelegramID)
		}
	}

	// Обновляем данные пользователя в базе БЕЗ изменения баланса
	if err := common.UpdateUser(user); err != nil {
		log.Printf("AUTO_BILLING: Ошибка обновления пользователя: %v", err)
		return fmt.Errorf("ошибка обновления пользователя: %v", err)
	}

	configURL := fmt.Sprintf("%s%s", common.CONFIG_BASE_URL, user.SubID)
	log.Printf("AUTO_BILLING: ✅ Конфиг успешно создан для пользователя %d, URL: %s, баланс остался: %.2f₽",
		user.TelegramID, configURL, user.Balance)

	return nil
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

	// ===== УБРАНО: Принудительный сброс состояния в автосписании =====
	// ПРОБЛЕМА: ForceResetDepletedStatus вызывался при каждом обновлении конфига в автосписании,
	// что приводило к частому сбросу трафика клиентов (каждые 30 минут).
	//
	// РЕШЕНИЕ: Убираем принудительный сброс состояния из автосписания.
	// Синхронизация клиентов должна происходить только при периодическом сбросе трафика (раз в 7 дней).
	//
	// РЕЗУЛЬТАТ: Автосписание только обновляет время истечения, не сбрасывая трафик клиентов.
	log.Printf("AUTO_BILLING: Обновление конфига завершено для пользователя %d (без принудительного сброса состояния)", user.TelegramID)
	common.LogTraffic("AUTO_BILLING", "Конфиг обновлен для TelegramID=%d без сброса состояния", user.TelegramID)

	log.Printf("AUTO_BILLING: Конфигурация принудительно обновлена (время+синхронизация) для пользователя %d на %d дней (до %s) - FIX как в тарифах",
		user.TelegramID, days, time.UnixMilli(newExpiryTime).Format("2006-01-02 15:04"))

	return nil
}

// diagnoseAndFixSyncIssues диагностирует и исправляет рассинхронизацию между базой данных и панелью
func (abs *AutoBillingService) diagnoseAndFixSyncIssues() {
	// Проверяем, включена ли диагностика
	if !common.SYNC_DIAGNOSTIC_ENABLED {
		return
	}

	log.Printf("SYNC_DIAGNOSTIC: ===== НАЧАЛО ДИАГНОСТИКИ РАССИНХРОНИЗАЦИИ =====")

	// Получаем всех пользователей с активными конфигами
	users, err := common.GetAllUsers()
	if err != nil {
		log.Printf("SYNC_DIAGNOSTIC: ❌ Ошибка получения пользователей: %v", err)
		return
	}

	// Авторизуемся в панели
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("SYNC_DIAGNOSTIC: ❌ Ошибка авторизации в панели: %v", err)
		return
	}

	// Получаем inbound из панели
	targetInbound, err := common.GetInbound(sessionCookie)
	if err != nil {
		log.Printf("SYNC_DIAGNOSTIC: ❌ Ошибка получения inbound: %v", err)
		return
	}

	if targetInbound == nil {
		log.Printf("SYNC_DIAGNOSTIC: ❌ Inbound не найден")
		return
	}

	// Парсим settings
	var settings common.Settings
	if err := json.Unmarshal([]byte(targetInbound.Settings), &settings); err != nil {
		log.Printf("SYNC_DIAGNOSTIC: ❌ Ошибка парсинга settings: %v", err)
		return
	}

	log.Printf("SYNC_DIAGNOSTIC: Найдено %d клиентов в панели, %d пользователей в базе", len(settings.Clients), len(users))

	fixedCount := 0
	checkedCount := 0

	// Проверяем каждого пользователя с активным конфигом (в любом из инбаундов)
	for _, user := range users {
		hasAnyActiveConfig := user.HasActiveConfig || user.HasActiveSecondaryConfig
		if !hasAnyActiveConfig || user.Balance <= 0 {
			continue
		}

		checkedCount++
		log.Printf("SYNC_DIAGNOSTIC: Проверка пользователя %d (HasActiveConfig=%v, ClientID=%s, SubID=%s)",
			user.TelegramID, user.HasActiveConfig, user.ClientID, user.SubID)

		// Ищем клиента в панели
		foundInPanel := false
		for _, client := range settings.Clients {
			if client.Email == fmt.Sprintf("%d", user.TelegramID) ||
				(user.ClientID != "" && client.ID == user.ClientID) ||
				(user.SubID != "" && client.SubID == user.SubID) {
				foundInPanel = true
				log.Printf("SYNC_DIAGNOSTIC: ✅ Пользователь %d найден в панели: Email=%s, Enable=%v, ExpiryTime=%d",
					user.TelegramID, client.Email, client.Enable, client.ExpiryTime)
				break
			}
		}

		if !foundInPanel {
			log.Printf("SYNC_DIAGNOSTIC: ❌ Пользователь %d НЕ найден в панели, но HasActiveConfig=true в базе", user.TelegramID)
			log.Printf("SYNC_DIAGNOSTIC: 🔧 Исправление: сброс флага HasActiveConfig для пользователя %d", user.TelegramID)

			// Сбрасываем флаг в базе данных
			user.HasActiveConfig = false
			user.ClientID = ""
			user.SubID = ""
			user.Email = ""
			user.ExpiryTime = 0

			if err := common.UpdateUser(&user); err != nil {
				log.Printf("SYNC_DIAGNOSTIC: ❌ Ошибка обновления пользователя %d: %v", user.TelegramID, err)
			} else {
				log.Printf("SYNC_DIAGNOSTIC: ✅ Флаг HasActiveConfig сброшен для пользователя %d", user.TelegramID)
				fixedCount++
			}
		}
	}

	log.Printf("SYNC_DIAGNOSTIC: ===== ДИАГНОСТИКА ЗАВЕРШЕНА =====")
	log.Printf("SYNC_DIAGNOSTIC: Проверено пользователей: %d, исправлено: %d", checkedCount, fixedCount)
}
