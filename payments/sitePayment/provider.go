package sitePayment

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"bot/common"
	paymentCommon "bot/payments/common"
)

const (
	YooKassaAPIURL = "https://api.yookassa.ru/v3"
)

// YooKassaPaymentProvider реализует платежи через прямое API ЮКассы
type YooKassaPaymentProvider struct {
	shopID    string
	secretKey string
	client    *http.Client
}

// YooKassaPaymentRequest структура запроса создания платежа
type YooKassaPaymentRequest struct {
	Amount       Amount                 `json:"amount"`
	Currency     string                 `json:"currency"`
	Description  string                 `json:"description"`
	Confirmation Confirmation           `json:"confirmation"`
	Capture      bool                   `json:"capture"`
	Receipt      *Receipt               `json:"receipt,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Receipt структура чека для 54-ФЗ
type Receipt struct {
	Customer Customer `json:"customer"`
	Items    []Item   `json:"items"`
}

// Customer структура покупателя
type Customer struct {
	Email string `json:"email"`
	Phone string `json:"phone,omitempty"`
}

// Item структура товара/услуги в чеке
type Item struct {
	Description    string        `json:"description"`
	Amount         Amount        `json:"amount"`
	VATCode        int           `json:"vat_code"`
	PaymentSubject string        `json:"payment_subject"`
	PaymentMode    string        `json:"payment_mode"`
	Quantity       string        `json:"quantity"`
	MarkCodeInfo   *MarkCodeInfo `json:"mark_code_info,omitempty"`
}

// MarkCodeInfo структура маркировочного кода
type MarkCodeInfo struct {
	MarkCodeRaw string `json:"mark_code_raw,omitempty"`
}

// Amount структура суммы платежа
type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

// Confirmation структура подтверждения платежа
type Confirmation struct {
	Type      string `json:"type"`
	ReturnURL string `json:"return_url,omitempty"`
}

// YooKassaPaymentResponse структура ответа от ЮКассы
type YooKassaPaymentResponse struct {
	ID           string                 `json:"id"`
	Status       string                 `json:"status"`
	Paid         bool                   `json:"paid"`
	Amount       Amount                 `json:"amount"`
	Description  string                 `json:"description"`
	Confirmation ConfirmationResponse   `json:"confirmation"`
	CreatedAt    string                 `json:"created_at"`
	Metadata     map[string]interface{} `json:"metadata"`
	Test         bool                   `json:"test"`
}

// ConfirmationResponse структура подтверждения в ответе
type ConfirmationResponse struct {
	Type            string `json:"type"`
	ConfirmationURL string `json:"confirmation_url,omitempty"`
}

// WebhookNotification структура уведомления от ЮКассы
type WebhookNotification struct {
	Type   string                  `json:"type"`
	Event  string                  `json:"event"`
	Object YooKassaPaymentResponse `json:"object"`
}

// NewYooKassaPaymentProvider создает новый провайдер ЮКассы API
func NewYooKassaPaymentProvider() *YooKassaPaymentProvider {
	return &YooKassaPaymentProvider{
		shopID:    common.YUKASSA_SHOP_ID,
		secretKey: common.YUKASSA_SECRET_KEY,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsEnabled проверяет, включены ли платежи через ЮКассы API
func (y *YooKassaPaymentProvider) IsEnabled() bool {
	return common.YUKASSA_API_PAYMENTS_ENABLED &&
		y.shopID != "" &&
		y.secretKey != ""
}

// GetMethod возвращает метод оплаты
func (y *YooKassaPaymentProvider) GetMethod() paymentCommon.PaymentMethod {
	return paymentCommon.PaymentMethodAPI
}

// CreatePayment создает платеж через API ЮКассы
func (y *YooKassaPaymentProvider) CreatePayment(userID int64, amount float64, description string) (*paymentCommon.PaymentInfo, error) {
	paymentCommon.LogPaymentEvent("INFO", paymentCommon.PaymentMethodAPI,
		"Создание платежа для пользователя %d на сумму %.2f", userID, amount)

	// Валидация суммы
	if err := paymentCommon.ValidateAmount(amount); err != nil {
		return nil, err
	}

	// Генерируем идемпотентный ключ
	idempotencyKey := y.generateIdempotencyKey(userID, amount)

	// Подготавливаем запрос
	request := YooKassaPaymentRequest{
		Amount: Amount{
			Value:    fmt.Sprintf("%.2f", amount),
			Currency: "RUB",
		},
		Currency:    "RUB",
		Description: paymentCommon.SanitizeDescription(description),
		Confirmation: Confirmation{
			Type:      "redirect",
			ReturnURL: fmt.Sprintf("https://t.me/%s", strings.TrimPrefix(common.BOT_TOKEN, "")), // Возврат в бота
		},
		Capture: true,
		Receipt: y.createReceipt(userID, amount, description),
		Metadata: paymentCommon.CreatePaymentMetadata(userID, map[string]interface{}{
			"idempotency_key": idempotencyKey,
		}),
	}

	// Отправляем запрос
	response, err := y.sendAPIRequest("POST", "/payments", request, idempotencyKey)
	if err != nil {
		paymentCommon.LogPaymentEvent("ERROR", paymentCommon.PaymentMethodAPI,
			"Ошибка создания платежа: %v", err)
		return nil, err
	}

	var paymentResponse YooKassaPaymentResponse
	if err := json.Unmarshal(response, &paymentResponse); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа ЮКассы: %v", err)
	}

	// Конвертируем статус ЮКассы в наш формат
	status := y.convertYooKassaStatus(paymentResponse.Status)

	// Создаем информацию о платеже
	paymentInfo := &paymentCommon.PaymentInfo{
		ID:          paymentResponse.ID,
		UserID:      userID,
		Amount:      amount,
		Currency:    paymentResponse.Amount.Currency,
		Status:      status,
		Method:      paymentCommon.PaymentMethodAPI,
		Description: paymentResponse.Description,
		CreatedAt:   paymentResponse.CreatedAt,
		UpdatedAt:   paymentCommon.GetCurrentTimestamp(),
		PaymentURL:  paymentResponse.Confirmation.ConfirmationURL,
		Metadata:    paymentResponse.Metadata,
	}

	paymentCommon.LogPaymentEvent("INFO", paymentCommon.PaymentMethodAPI,
		"Платеж создан: ID=%s, UserID=%d, Amount=%.2f, URL=%s",
		paymentResponse.ID, userID, amount, paymentResponse.Confirmation.ConfirmationURL)

	return paymentInfo, nil
}

// GetPayment получает информацию о платеже
func (y *YooKassaPaymentProvider) GetPayment(paymentID string) (*paymentCommon.PaymentInfo, error) {
	paymentCommon.LogPaymentEvent("INFO", paymentCommon.PaymentMethodAPI,
		"Получение информации о платеже %s", paymentID)

	response, err := y.sendAPIRequest("GET", "/payments/"+paymentID, nil, "")
	if err != nil {
		return nil, err
	}

	var paymentResponse YooKassaPaymentResponse
	if err := json.Unmarshal(response, &paymentResponse); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа ЮКассы: %v", err)
	}

	// Извлекаем userID из метаданных
	var userID int64
	if userIDValue, exists := paymentResponse.Metadata["user_id"]; exists {
		if userIDFloat, ok := userIDValue.(float64); ok {
			userID = int64(userIDFloat)
		}
	}

	amount := 0.0
	if amountValue := paymentResponse.Amount.Value; amountValue != "" {
		fmt.Sscanf(amountValue, "%f", &amount)
	}

	status := y.convertYooKassaStatus(paymentResponse.Status)

	paymentInfo := &paymentCommon.PaymentInfo{
		ID:          paymentResponse.ID,
		UserID:      userID,
		Amount:      amount,
		Currency:    paymentResponse.Amount.Currency,
		Status:      status,
		Method:      paymentCommon.PaymentMethodAPI,
		Description: paymentResponse.Description,
		CreatedAt:   paymentResponse.CreatedAt,
		UpdatedAt:   paymentCommon.GetCurrentTimestamp(),
		PaymentURL:  paymentResponse.Confirmation.ConfirmationURL,
		Metadata:    paymentResponse.Metadata,
	}

	return paymentInfo, nil
}

// ProcessWebhook обрабатывает уведомления от ЮКассы
func (y *YooKassaPaymentProvider) ProcessWebhook(data []byte) (*paymentCommon.PaymentInfo, error) {
	paymentCommon.LogPaymentEvent("INFO", paymentCommon.PaymentMethodAPI,
		"Обработка webhook уведомления от ЮКассы")

	var notification WebhookNotification
	if err := json.Unmarshal(data, &notification); err != nil {
		return nil, fmt.Errorf("ошибка парсинга webhook данных: %v", err)
	}

	paymentCommon.LogPaymentEvent("DEBUG", paymentCommon.PaymentMethodAPI,
		"Webhook: Type=%s, Event=%s, PaymentID=%s, Status=%s",
		notification.Type, notification.Event, notification.Object.ID, notification.Object.Status)

	// Обрабатываем только уведомления о платежах
	if notification.Type != "notification" {
		return nil, fmt.Errorf("неподдерживаемый тип уведомления: %s", notification.Type)
	}

	payment := notification.Object

	// Извлекаем userID из метаданных
	var userID int64
	if userIDValue, exists := payment.Metadata["user_id"]; exists {
		if userIDFloat, ok := userIDValue.(float64); ok {
			userID = int64(userIDFloat)
		}
	}

	amount := 0.0
	if amountValue := payment.Amount.Value; amountValue != "" {
		fmt.Sscanf(amountValue, "%f", &amount)
	}

	status := y.convertYooKassaStatus(payment.Status)

	paymentInfo := &paymentCommon.PaymentInfo{
		ID:          payment.ID,
		UserID:      userID,
		Amount:      amount,
		Currency:    payment.Amount.Currency,
		Status:      status,
		Method:      paymentCommon.PaymentMethodAPI,
		Description: payment.Description,
		CreatedAt:   payment.CreatedAt,
		UpdatedAt:   paymentCommon.GetCurrentTimestamp(),
		Metadata:    payment.Metadata,
	}

	// Если платеж успешен, пополняем баланс
	if status == paymentCommon.PaymentStatusSucceeded && userID > 0 {
		err := common.AddBalance(userID, amount)
		if err != nil {
			paymentCommon.LogPaymentEvent("ERROR", paymentCommon.PaymentMethodAPI,
				"Ошибка пополнения баланса для пользователя %d: %v", userID, err)
			return paymentInfo, fmt.Errorf("ошибка пополнения баланса: %v", err)
		}

		paymentCommon.LogPaymentEvent("INFO", paymentCommon.PaymentMethodAPI,
			"Баланс пользователя %d пополнен на %.2f₽", userID, amount)
	}

	return paymentInfo, nil
}

// sendAPIRequest отправляет запрос к API ЮКассы
func (y *YooKassaPaymentProvider) sendAPIRequest(method, endpoint string, body interface{}, idempotencyKey string) ([]byte, error) {
	url := YooKassaAPIURL + endpoint

	var requestBody []byte
	var err error

	if body != nil {
		requestBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("ошибка кодирования JSON: %v", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(y.shopID, y.secretKey)

	if idempotencyKey != "" {
		req.Header.Set("Idempotence-Key", idempotencyKey)
	}

	// Отправляем запрос
	resp, err := y.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %v", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ошибка API ЮКассы (код %d): %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// generateIdempotencyKey генерирует идемпотентный ключ
func (y *YooKassaPaymentProvider) generateIdempotencyKey(userID int64, amount float64) string {
	timestamp := time.Now().Unix()
	source := fmt.Sprintf("%d_%d_%.2f_%d", userID, timestamp, amount, userID)

	hash := sha256.Sum256([]byte(source))
	return hex.EncodeToString(hash[:16])
}

// createReceipt создает чек для платежа
func (y *YooKassaPaymentProvider) createReceipt(userID int64, amount float64, description string) *Receipt {
	// Если отправка чеков отключена, возвращаем nil
	if !common.YUKASSA_RECEIPT_ENABLED {
		return nil
	}

	return &Receipt{
		Customer: Customer{
			Email: fmt.Sprintf("user_%d@vpnbot.local", userID), // Временный email для чека
		},
		Items: []Item{
			{
				Description: paymentCommon.SanitizeDescription(description),
				Amount: Amount{
					Value:    fmt.Sprintf("%.2f", amount),
					Currency: "RUB",
				},
				VATCode:        common.YUKASSA_VAT_CODE,        // НДС из конфигурации
				PaymentSubject: common.YUKASSA_PAYMENT_SUBJECT, // Предмет расчета из конфигурации
				PaymentMode:    common.YUKASSA_PAYMENT_MODE,    // Способ расчета из конфигурации
				Quantity:       "1",
			},
		},
	}
}

// convertYooKassaStatus конвертирует статус ЮКассы в наш формат
func (y *YooKassaPaymentProvider) convertYooKassaStatus(status string) paymentCommon.PaymentStatus {
	switch status {
	case "pending":
		return paymentCommon.PaymentStatusPending
	case "waiting_for_capture":
		return paymentCommon.PaymentStatusPending
	case "succeeded":
		return paymentCommon.PaymentStatusSucceeded
	case "canceled":
		return paymentCommon.PaymentStatusCanceled
	default:
		return paymentCommon.PaymentStatusFailed
	}
}
