package common

import (
	"errors"
)

// PaymentStatus представляет статус платежа
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"   // Ожидает оплаты
	PaymentStatusSucceeded PaymentStatus = "succeeded" // Оплачен
	PaymentStatusCanceled  PaymentStatus = "canceled"  // Отменен
	PaymentStatusFailed    PaymentStatus = "failed"    // Ошибка оплаты
)

// PaymentMethod представляет метод оплаты
type PaymentMethod string

const (
	PaymentMethodTelegram PaymentMethod = "telegram" // Через Telegram Bot API
	PaymentMethodAPI      PaymentMethod = "api"      // Через прямое API ЮКассы
)

// PaymentInfo содержит информацию о платеже
type PaymentInfo struct {
	ID          string                 `json:"id"`          // Уникальный ID платежа
	UserID      int64                  `json:"user_id"`     // ID пользователя Telegram
	Amount      float64                `json:"amount"`      // Сумма платежа в рублях
	Currency    string                 `json:"currency"`    // Валюта (RUB)
	Status      PaymentStatus          `json:"status"`      // Статус платежа
	Method      PaymentMethod          `json:"method"`      // Метод оплаты
	Description string                 `json:"description"` // Описание платежа
	CreatedAt   string                 `json:"created_at"`  // Время создания
	UpdatedAt   string                 `json:"updated_at"`  // Время обновления
	PaymentURL  string                 `json:"payment_url"` // URL для оплаты (только для API метода)
	Metadata    map[string]interface{} `json:"metadata"`    // Дополнительные данные
}

// PaymentProvider интерфейс для провайдеров платежей
type PaymentProvider interface {
	// Создать платеж
	CreatePayment(userID int64, amount float64, description string) (*PaymentInfo, error)

	// Получить информацию о платеже
	GetPayment(paymentID string) (*PaymentInfo, error)

	// Обработать уведомление о платеже
	ProcessWebhook(data []byte) (*PaymentInfo, error)

	// Проверить, поддерживается ли данный метод
	IsEnabled() bool

	// Получить название метода
	GetMethod() PaymentMethod
}

// PaymentManager управляет различными провайдерами платежей
type PaymentManager struct {
	providers map[PaymentMethod]PaymentProvider
}

// NewPaymentManager создает новый менеджер платежей
func NewPaymentManager() *PaymentManager {
	return &PaymentManager{
		providers: make(map[PaymentMethod]PaymentProvider),
	}
}

// RegisterProvider регистрирует провайдера платежей
func (pm *PaymentManager) RegisterProvider(provider PaymentProvider) {
	pm.providers[provider.GetMethod()] = provider
}

// GetAvailableProviders возвращает список доступных провайдеров
func (pm *PaymentManager) GetAvailableProviders() []PaymentMethod {
	var methods []PaymentMethod
	for method, provider := range pm.providers {
		if provider.IsEnabled() {
			methods = append(methods, method)
		}
	}
	return methods
}

// CreatePayment создает платеж используя предпочтительный метод
func (pm *PaymentManager) CreatePayment(method PaymentMethod, userID int64, amount float64, description string) (*PaymentInfo, error) {
	provider, exists := pm.providers[method]
	if !exists {
		return nil, errors.New("платежный провайдер не найден")
	}

	if !provider.IsEnabled() {
		return nil, errors.New("платежный провайдер отключен")
	}

	return provider.CreatePayment(userID, amount, description)
}

// GetPayment получает информацию о платеже
func (pm *PaymentManager) GetPayment(method PaymentMethod, paymentID string) (*PaymentInfo, error) {
	provider, exists := pm.providers[method]
	if !exists {
		return nil, errors.New("платежный провайдер не найден")
	}

	return provider.GetPayment(paymentID)
}

// ProcessWebhook обрабатывает уведомление от платежного провайдера
func (pm *PaymentManager) ProcessWebhook(method PaymentMethod, data []byte) (*PaymentInfo, error) {
	provider, exists := pm.providers[method]
	if !exists {
		return nil, errors.New("платежный провайдер не найден")
	}

	return provider.ProcessWebhook(data)
}

// GetPreferredMethod возвращает предпочтительный метод оплаты
func (pm *PaymentManager) GetPreferredMethod() (PaymentMethod, error) {
	available := pm.GetAvailableProviders()
	if len(available) == 0 {
		return "", errors.New("нет доступных методов оплаты")
	}

	// Приоритет: Telegram Bot API > Прямое API
	for _, method := range []PaymentMethod{PaymentMethodTelegram, PaymentMethodAPI} {
		for _, available := range available {
			if method == available {
				return method, nil
			}
		}
	}

	return available[0], nil
}

// Errors
var (
	ErrPaymentNotFound    = errors.New("платеж не найден")
	ErrProviderNotEnabled = errors.New("провайдер платежей отключен")
	ErrInvalidAmount      = errors.New("неверная сумма платежа")
	ErrInvalidWebhookData = errors.New("неверные данные webhook")
)
