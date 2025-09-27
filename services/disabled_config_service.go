package services

import (
	"fmt"
	"log"
	"time"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// DisabledConfigService управляет автоматическим включением отключенных конфигов
type DisabledConfigService struct {
	bot           *tgbotapi.BotAPI
	configManager *common.ConfigManager
	checkTicker   *time.Ticker
}

// NewDisabledConfigService создает новый сервис проверки отключенных конфигов
func NewDisabledConfigService(bot *tgbotapi.BotAPI, configManager *common.ConfigManager) *DisabledConfigService {
	return &DisabledConfigService{
		bot:           bot,
		configManager: configManager,
	}
}

// Start запускает сервис проверки отключенных конфигов
func (dcs *DisabledConfigService) Start() {
	if !common.DISABLED_CONFIG_CHECK_ENABLED {
		log.Printf("DISABLED_CONFIG: Проверка отключенных конфигов отключена в конфигурации")
		return
	}

	log.Printf("DISABLED_CONFIG: Запуск сервиса проверки отключенных конфигов")
	log.Printf("DISABLED_CONFIG: Интервал проверки: %d минут", common.DISABLED_CONFIG_CHECK_INTERVAL)

	// Создаем тикер для периодической проверки
	dcs.checkTicker = time.NewTicker(time.Duration(common.DISABLED_CONFIG_CHECK_INTERVAL) * time.Minute)

	// Запускаем горутину для периодических проверок
	go func() {
		// Первая проверка сразу при запуске
		dcs.checkDisabledConfigs()

		// Затем проверяем по расписанию
		for range dcs.checkTicker.C {
			dcs.checkDisabledConfigs()
		}
	}()

	log.Printf("DISABLED_CONFIG: Сервис проверки отключенных конфигов успешно запущен")
}

// Stop останавливает сервис
func (dcs *DisabledConfigService) Stop() {
	if dcs.checkTicker != nil {
		dcs.checkTicker.Stop()
		log.Printf("DISABLED_CONFIG: Сервис проверки отключенных конфигов остановлен")
	}
}

// checkDisabledConfigs проверяет всех пользователей на отключенные конфиги с достаточным балансом
func (dcs *DisabledConfigService) checkDisabledConfigs() {
	log.Printf("DISABLED_CONFIG: Начинаем проверку отключенных конфигов...")

	// Получаем ВСЕХ пользователей (не только с активными конфигами)
	users, err := common.GetAllUsers()
	if err != nil {
		log.Printf("DISABLED_CONFIG: Ошибка получения пользователей: %v", err)
		return
	}

	enabledCount := 0
	checkedCount := 0

	for _, user := range users {
		checkedCount++

		// Проверяем условия для включения конфига
		if shouldEnable, reason := dcs.shouldEnableConfig(&user); shouldEnable {
			log.Printf("DISABLED_CONFIG: Найден отключенный конфиг с достаточным балансом для пользователя %d (email: %s, баланс: %.2f₽, причина: %s)",
				user.TelegramID, user.Email, user.Balance, reason)

			// Включаем конфиг пользователя
			if err := dcs.enableUserConfig(&user); err != nil {
				log.Printf("DISABLED_CONFIG: Ошибка включения конфига для пользователя %d: %v",
					user.TelegramID, err)
			} else {
				enabledCount++
				log.Printf("DISABLED_CONFIG: Конфиг успешно включен для пользователя %d", user.TelegramID)
			}
		}
	}

	if enabledCount > 0 {
		log.Printf("DISABLED_CONFIG: Проверка завершена. Проверено пользователей: %d, включено конфигов: %d",
			checkedCount, enabledCount)
	} else {
		log.Printf("DISABLED_CONFIG: Проверка завершена. Проверено пользователей: %d, отключенных конфигов с достаточным балансом не найдено",
			checkedCount)
	}
}

// shouldEnableConfig проверяет, нужно ли включить конфиг пользователя
func (dcs *DisabledConfigService) shouldEnableConfig(user *common.User) (bool, string) {
	// Проверяем, что у пользователя есть конфиг (ClientID и SubID)
	if user.ClientID == "" || user.SubID == "" {
		return false, "нет ClientID или SubID"
	}

	// Проверяем, что у пользователя есть email
	if user.Email == "" {
		return false, "нет email"
	}

	// Проверяем, что баланс больше PRICE_PER_DAY
	if user.Balance < float64(common.PRICE_PER_DAY) {
		return false, fmt.Sprintf("недостаточный баланс %.2f₽ < %d₽", user.Balance, common.PRICE_PER_DAY)
	}

	// Проверяем статус конфига в панели управления
	configStatus, err := dcs.configManager.GetConfigStatus(user.Email)
	if err != nil {
		log.Printf("DISABLED_CONFIG: Ошибка получения статуса конфига для пользователя %d: %v",
			user.TelegramID, err)
		return false, fmt.Sprintf("ошибка получения статуса: %v", err)
	}

	// Если конфиг отключен в панели, но у пользователя есть достаточный баланс - включаем
	if !configStatus {
		return true, fmt.Sprintf("конфиг отключен в панели, баланс %.2f₽ достаточен", user.Balance)
	}

	return false, "конфиг уже включен в панели"
}

// enableUserConfig включает конфиг пользователя
func (dcs *DisabledConfigService) enableUserConfig(user *common.User) error {
	log.Printf("DISABLED_CONFIG: Включение конфига для пользователя %d", user.TelegramID)

	// Включаем конфиг в панели управления
	err := dcs.configManager.EnableConfig(user.Email)
	if err != nil {
		return fmt.Errorf("ошибка включения конфига в панели: %v", err)
	}

	// Сбрасываем статус "исчерпано" в панели управления
	log.Printf("DISABLED_CONFIG: Сброс статуса 'исчерпано' для пользователя %d", user.TelegramID)
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("DISABLED_CONFIG: Ошибка авторизации для сброса статуса 'исчерпано' для пользователя %d: %v",
			user.TelegramID, err)
		// Не возвращаем ошибку, так как конфиг уже включен
	} else {
		if err := common.ForceResetDepletedStatus(sessionCookie, user.TelegramID); err != nil {
			log.Printf("DISABLED_CONFIG: Ошибка сброса статуса 'исчерпано' для пользователя %d: %v",
				user.TelegramID, err)
			// Не возвращаем ошибку, так как конфиг уже включен
		} else {
			log.Printf("DISABLED_CONFIG: Статус 'исчерпано' успешно сброшен для пользователя %d", user.TelegramID)
		}
	}

	// Обновляем статус пользователя в базе данных
	user.HasActiveConfig = true

	// Если время истечения не установлено или истекло, устанавливаем новое
	now := time.Now()
	if user.ExpiryTime <= now.UnixMilli() {
		// Вычисляем количество дней по балансу
		availableDays := int(user.Balance / float64(common.PRICE_PER_DAY))
		if availableDays > 0 {
			// Устанавливаем время истечения на доступное количество дней
			user.ExpiryTime = now.Add(time.Duration(availableDays) * 24 * time.Hour).UnixMilli()
			log.Printf("DISABLED_CONFIG: Установлено время истечения для пользователя %d: %d дней (до %s)",
				user.TelegramID, availableDays,
				time.UnixMilli(user.ExpiryTime).Format("2006-01-02 15:04:05"))
		}
	}

	err = common.UpdateUser(user)
	if err != nil {
		return fmt.Errorf("ошибка обновления пользователя в базе данных: %v", err)
	}

	// Отправляем уведомление пользователю
	if dcs.bot != nil {
		dcs.sendConfigEnabledNotification(user)
	}

	// Отправляем уведомление администратору
	if common.ADMIN_NOTIFICATIONS_ENABLED && common.ADMIN_CONFIG_BLOCKING_ENABLED {
		dcs.sendAdminNotification(user)
	}

	return nil
}

// sendConfigEnabledNotification отправляет уведомление пользователю о включении конфига
func (dcs *DisabledConfigService) sendConfigEnabledNotification(user *common.User) {
	availableDays := int(user.Balance / float64(common.PRICE_PER_DAY))
	expiryTime := time.UnixMilli(user.ExpiryTime).Format("2006-01-02 15:04:05")

	message := "✅ <b>Ваш VPN снова активен!</b>\n\n" +
		"Ваш конфиг был автоматически включен после пополнения баланса.\n" +
		"Статус \"Исчерпано\" сброшен, трафик восстановлен.\n\n" +
		fmt.Sprintf("💰 Баланс: %.2f₽\n", user.Balance) +
		fmt.Sprintf("📅 Доступно дней: %d\n", availableDays) +
		fmt.Sprintf("⏰ Истекает: %s\n\n", expiryTime) +
		"Наслаждайтесь стабильным соединением!"

	msg := tgbotapi.NewMessage(user.TelegramID, message)
	msg.ParseMode = tgbotapi.ModeHTML

	_, err := dcs.bot.Send(msg)
	if err != nil {
		log.Printf("DISABLED_CONFIG: Ошибка отправки уведомления пользователю %d: %v",
			user.TelegramID, err)
	} else {
		log.Printf("DISABLED_CONFIG: Уведомление о включении конфига отправлено пользователю %d",
			user.TelegramID)
	}
}

// sendAdminNotification отправляет уведомление администратору о включении конфига
func (dcs *DisabledConfigService) sendAdminNotification(user *common.User) {
	availableDays := int(user.Balance / float64(common.PRICE_PER_DAY))
	expiryTime := time.UnixMilli(user.ExpiryTime).Format("2006-01-02 15:04:05")

	message := fmt.Sprintf("🔔 <b>Конфиг автоматически включен</b>\n\n"+
		"📧 Email: %s\n"+
		"👤 Пользователь: %s (ID: %d)\n"+
		"💰 Баланс: %.2f₽\n"+
		"📅 Доступно дней: %d\n"+
		"⏰ Истекает: %s\n\n"+
		"Конфиг был автоматически включен из-за достаточного баланса.\n"+
		"Статус \"Исчерпано\" сброшен, трафик восстановлен.",
		user.Email, user.FirstName, user.TelegramID, user.Balance, availableDays, expiryTime)

	msg := tgbotapi.NewMessage(common.ADMIN_ID, message)
	msg.ParseMode = tgbotapi.ModeHTML

	_, err := dcs.bot.Send(msg)
	if err != nil {
		log.Printf("DISABLED_CONFIG: Ошибка отправки уведомления администратору: %v", err)
	} else {
		log.Printf("DISABLED_CONFIG: Уведомление о включении конфига отправлено администратору")
	}
}

// ForceCheckDisabledConfigs экспортированный метод для принудительной проверки отключенных конфигов
func (dcs *DisabledConfigService) ForceCheckDisabledConfigs() {
	log.Printf("DISABLED_CONFIG: Принудительная проверка отключенных конфигов")
	dcs.checkDisabledConfigs()
}
