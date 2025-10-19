package payments

import (
	"fmt"
	"log"

	"bot/common"
	paymentCommon "bot/payments/common"
	"bot/payments/sitePayment"
	"bot/payments/telegramPayment"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// GlobalPaymentManager глобальный экземпляр менеджера платежей
var GlobalPaymentManager *PaymentManager

// PaymentManager расширенный менеджер платежей с дополнительной функциональностью
type PaymentManager struct {
	*paymentCommon.PaymentManager
	telegramProvider *telegramPayment.TelegramPaymentProvider
	yookassaProvider *sitePayment.YooKassaPaymentProvider
	onDemandService  *OnDemandPaymentService
}

// InitializePaymentManager инициализирует глобальный менеджер платежей
func InitializePaymentManager(bot *tgbotapi.BotAPI) error {
	log.Printf("PAYMENT_MANAGER: Инициализация менеджера платежей")

	// Создаем базовый менеджер
	baseManager := paymentCommon.NewPaymentManager()

	// Создаем расширенный менеджер
	manager := &PaymentManager{
		PaymentManager: baseManager,
	}

	// Инициализируем провайдеры
	if err := manager.initializeProviders(bot); err != nil {
		return fmt.Errorf("ошибка инициализации провайдеров: %v", err)
	}

	// Устанавливаем глобальный экземпляр
	GlobalPaymentManager = manager

	// Инициализируем сервис обработки платежей по требованию
	manager.onDemandService = NewOnDemandPaymentService(manager)

	log.Printf("PAYMENT_MANAGER: Менеджер платежей успешно инициализирован")
	return nil
}

// initializeProviders инициализирует всех провайдеров платежей
func (pm *PaymentManager) initializeProviders(bot *tgbotapi.BotAPI) error {
	// Инициализируем Telegram провайдер
	if common.TELEGRAM_PAYMENTS_ENABLED && common.YUKASSA_PROVIDER_TOKEN != "" {
		pm.telegramProvider = telegramPayment.NewTelegramPaymentProvider(bot)
		pm.RegisterProvider(pm.telegramProvider)
		log.Printf("PAYMENT_MANAGER: Telegram провайдер зарегистрирован (включен: %v)", pm.telegramProvider.IsEnabled())
	} else {
		log.Printf("PAYMENT_MANAGER: Telegram провайдер отключен (TELEGRAM_PAYMENTS_ENABLED=%v, TOKEN=%s)",
			common.TELEGRAM_PAYMENTS_ENABLED,
			func() string {
				if common.YUKASSA_PROVIDER_TOKEN != "" {
					return "установлен"
				}
				return "не установлен"
			}())
	}

	// Инициализируем ЮКасса API провайдер
	if common.YUKASSA_API_PAYMENTS_ENABLED && common.YUKASSA_SHOP_ID != "" && common.YUKASSA_SECRET_KEY != "" {
		pm.yookassaProvider = sitePayment.NewYooKassaPaymentProvider()
		pm.RegisterProvider(pm.yookassaProvider)
		log.Printf("PAYMENT_MANAGER: ЮКасса API провайдер зарегистрирован (включен: %v)", pm.yookassaProvider.IsEnabled())
	} else {
		log.Printf("PAYMENT_MANAGER: ЮКасса API провайдер отключен (YUKASSA_API_PAYMENTS_ENABLED=%v, SHOP_ID=%s, SECRET_KEY=%s)",
			common.YUKASSA_API_PAYMENTS_ENABLED,
			func() string {
				if common.YUKASSA_SHOP_ID != "" {
					return "установлен"
				}
				return "не установлен"
			}(),
			func() string {
				if common.YUKASSA_SECRET_KEY != "" {
					return "установлен"
				}
				return "не установлен"
			}())
	}

	// Проверяем, что хотя бы один провайдер доступен
	available := pm.GetAvailableProviders()
	if len(available) == 0 {
		return fmt.Errorf("ни один провайдер платежей не доступен")
	}

	log.Printf("PAYMENT_MANAGER: Доступные провайдеры: %v", available)
	return nil
}

// CreatePaymentWithPreferredMethod создает платеж используя предпочтительный метод
func (pm *PaymentManager) CreatePaymentWithPreferredMethod(userID int64, amount float64, description string) (*paymentCommon.PaymentInfo, paymentCommon.PaymentMethod, error) {
	method, err := pm.GetPreferredMethod()
	if err != nil {
		return nil, "", err
	}

	paymentInfo, err := pm.CreatePayment(method, userID, amount, description)
	return paymentInfo, method, err
}

// SendTelegramInvoice отправляет инвойс через Telegram (только для Telegram провайдера)
func (pm *PaymentManager) SendTelegramInvoice(chatID int64, paymentInfo *paymentCommon.PaymentInfo) error {
	if pm.telegramProvider == nil {
		return fmt.Errorf("Telegram провайдер не инициализирован")
	}

	if !pm.telegramProvider.IsEnabled() {
		return fmt.Errorf("Telegram провайдер отключен")
	}

	return pm.telegramProvider.SendInvoice(chatID, paymentInfo)
}

// ProcessTelegramSuccessfulPayment обрабатывает успешный платеж от Telegram
func (pm *PaymentManager) ProcessTelegramSuccessfulPayment(payment *tgbotapi.SuccessfulPayment, userID int64) (*paymentCommon.PaymentInfo, error) {
	if pm.telegramProvider == nil {
		return nil, fmt.Errorf("Telegram провайдер не инициализирован")
	}

	return pm.telegramProvider.ProcessSuccessfulPayment(payment, userID)
}

// SendTelegramPaymentConfirmation отправляет подтверждение платежа через Telegram
func (pm *PaymentManager) SendTelegramPaymentConfirmation(chatID int64, paymentInfo *paymentCommon.PaymentInfo, newBalance float64) error {
	if pm.telegramProvider == nil {
		return fmt.Errorf("Telegram провайдер не инициализирован")
	}

	return pm.telegramProvider.SendPaymentConfirmation(chatID, paymentInfo, newBalance)
}

// GetPaymentStatusText возвращает текстовое описание статуса платежа
func (pm *PaymentManager) GetPaymentStatusText(paymentInfo *paymentCommon.PaymentInfo) string {
	statusDescription := paymentCommon.GetPaymentStatusDescription(paymentInfo.Status)
	methodDescription := paymentCommon.GetMethodDescription(paymentInfo.Method)

	return fmt.Sprintf("Статус: %s\nМетод: %s\nСумма: %s",
		statusDescription, methodDescription, paymentCommon.FormatAmount(paymentInfo.Amount))
}

// IsAnyProviderEnabled проверяет, включен ли хотя бы один провайдер
func (pm *PaymentManager) IsAnyProviderEnabled() bool {
	return len(pm.GetAvailableProviders()) > 0
}

// GetProviderStatus возвращает статус всех провайдеров
func (pm *PaymentManager) GetProviderStatus() map[paymentCommon.PaymentMethod]bool {
	status := make(map[paymentCommon.PaymentMethod]bool)

	if pm.telegramProvider != nil {
		status[paymentCommon.PaymentMethodTelegram] = pm.telegramProvider.IsEnabled()
	} else {
		status[paymentCommon.PaymentMethodTelegram] = false
	}

	if pm.yookassaProvider != nil {
		status[paymentCommon.PaymentMethodAPI] = pm.yookassaProvider.IsEnabled()
	} else {
		status[paymentCommon.PaymentMethodAPI] = false
	}

	return status
}

// ProcessTopupRequest обрабатывает запрос на пополнение баланса
func (pm *PaymentManager) ProcessTopupRequest(userID int64, amount float64, chatID int64) error {
	log.Printf("PAYMENT_MANAGER: Обработка запроса пополнения для пользователя %d на сумму %.2f", userID, amount)

	// Получаем пользователя
	user, err := common.GetUserByTelegramID(userID)
	if err != nil {
		return fmt.Errorf("ошибка получения пользователя: %v", err)
	}

	description := fmt.Sprintf("Пополнение баланса на %.2f₽", amount)

	// Создаем платеж с предпочтительным методом
	paymentInfo, method, err := pm.CreatePaymentWithPreferredMethod(userID, amount, description)
	if err != nil {
		return fmt.Errorf("ошибка создания платежа: %v", err)
	}

	log.Printf("PAYMENT_MANAGER: Платеж создан (ID=%s, Method=%s)", paymentInfo.ID, method)

	// Обрабатываем в зависимости от метода
	switch method {
	case paymentCommon.PaymentMethodTelegram:
		// Отправляем инвойс через Telegram
		err = pm.SendTelegramInvoice(chatID, paymentInfo)
		if err != nil {
			return fmt.Errorf("ошибка отправки Telegram инвойса: %v", err)
		}
		log.Printf("PAYMENT_MANAGER: Telegram инвойс отправлен для платежа %s", paymentInfo.ID)

	case paymentCommon.PaymentMethodAPI:
		// Отправляем ссылку на оплату через ЮКасса API
		if paymentInfo.PaymentURL == "" {
			return fmt.Errorf("URL для оплаты не получен")
		}

		// Отправляем сообщение со ссылкой на оплату
		err = pm.sendYooKassaPaymentLink(chatID, paymentInfo, user)
		if err != nil {
			return fmt.Errorf("ошибка отправки ссылки на оплату: %v", err)
		}
		log.Printf("PAYMENT_MANAGER: Ссылка на оплату ЮКасса отправлена для платежа %s", paymentInfo.ID)

		// Запускаем мониторинг платежа
		if pm.onDemandService != nil {
			pm.onDemandService.StartPaymentMonitoring(paymentInfo.ID, paymentInfo.UserID, paymentInfo.Amount)
			log.Printf("PAYMENT_MANAGER: Запущен мониторинг платежа %s", paymentInfo.ID)
		}

	default:
		return fmt.Errorf("неподдерживаемый метод оплаты: %s", method)
	}

	return nil
}

// sendYooKassaPaymentLink отправляет ссылку на оплату через ЮКасса API
func (pm *PaymentManager) sendYooKassaPaymentLink(chatID int64, paymentInfo *paymentCommon.PaymentInfo, user *common.User) error {
	text := fmt.Sprintf("💳 <b>Пополнение баланса</b>\n\n"+
		"💰 Сумма: %s\n"+
		"🏦 Платежная система: %s\n"+
		"🆔 ID платежа: %s\n\n"+
		"Нажмите кнопку ниже для перехода к оплате:",
		paymentCommon.FormatAmount(paymentInfo.Amount),
		paymentCommon.GetMethodDescription(paymentInfo.Method),
		paymentInfo.ID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("💳 Оплатить", paymentInfo.PaymentURL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Проверить платеж", fmt.Sprintf("check_payment:%s", paymentInfo.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отменить", "main"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	_, err := common.GlobalBot.Send(msg)
	return err
}

// CheckPaymentStatus проверяет статус платежа
func (pm *PaymentManager) CheckPaymentStatus(method paymentCommon.PaymentMethod, paymentID string) (*paymentCommon.PaymentInfo, error) {
	log.Printf("PAYMENT_MANAGER: Проверка статуса платежа %s (метод: %s)", paymentID, method)

	paymentInfo, err := pm.GetPayment(method, paymentID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения информации о платеже: %v", err)
	}

	return paymentInfo, nil
}

// GetOnDemandService возвращает сервис обработки платежей по требованию
func (pm *PaymentManager) GetOnDemandService() *OnDemandPaymentService {
	return pm.onDemandService
}
