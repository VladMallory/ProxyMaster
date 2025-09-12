package common

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// YukassaAPI структура для работы с API ЮКассы
type YukassaAPI struct {
	shopID  string
	baseURL string
}

// NewYukassaAPI создает новый экземпляр API ЮКассы
func NewYukassaAPI() *YukassaAPI {
	baseURL := "https://api.yookassa.ru/v3"
	if YUKASSA_TEST_MODE {
		// В тестовом режиме используем тот же URL, но с тестовыми данными
		log.Printf("YUKASSA: Используется тестовый режим")
	}

	return &YukassaAPI{
		shopID:  YUKASSA_SHOP_ID,
		baseURL: baseURL,
	}
}

// PaymentRequest структура запроса создания платежа
type PaymentRequest struct {
	Amount       Amount       `json:"amount"`
	Currency     string       `json:"currency"`
	Description  string       `json:"description"`
	Receipt      *Receipt     `json:"receipt,omitempty"`
	Confirmation Confirmation `json:"confirmation"`
	Capture      bool         `json:"capture"`
	Metadata     Metadata     `json:"metadata"`
}

// Amount структура суммы платежа
type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

// Receipt структура чека для ФЗ-54
type Receipt struct {
	Customer Customer `json:"customer"`
	Items    []Item   `json:"items"`
}

// Customer структура покупателя
type Customer struct {
	Email string `json:"email,omitempty"`
}

// Item структура товара в чеке
type Item struct {
	Description string `json:"description"`
	Quantity    string `json:"quantity"`
	Amount      Amount `json:"amount"`
	VatCode     int    `json:"vat_code"`
}

// Confirmation структура подтверждения платежа
type Confirmation struct {
	Type            string `json:"type"`
	ReturnURL       string `json:"return_url,omitempty"`
	ConfirmationURL string `json:"confirmation_url,omitempty"`
}

// Metadata дополнительная информация о платеже
type Metadata struct {
	TelegramID string `json:"telegram_id"`
	UserID     string `json:"user_id"`
	OrderType  string `json:"order_type"`
}

// PaymentResponse структура ответа создания платежа
type PaymentResponse struct {
	ID           string       `json:"id"`
	Status       string       `json:"status"`
	Amount       Amount       `json:"amount"`
	Description  string       `json:"description"`
	Confirmation Confirmation `json:"confirmation"`
	CreatedAt    string       `json:"created_at"`
	Metadata     Metadata     `json:"metadata"`
	Paid         bool         `json:"paid"`
}

// PaymentStatus структура статуса платежа
type PaymentStatus struct {
	ID       string   `json:"id"`
	Status   string   `json:"status"`
	Amount   Amount   `json:"amount"`
	Metadata Metadata `json:"metadata"`
	Paid     bool     `json:"paid"`
}

// CreatePayment создает новый платеж в ЮКассе
func (y *YukassaAPI) CreatePayment(userID int64, amount float64, description string) (*PaymentResponse, error) {
	log.Printf("YUKASSA: Создание платежа для пользователя %d на сумму %.2f", userID, amount)

	// Генерируем уникальный ключ идемпотентности
	idempotencyKey := uuid.New().String()

	// Формируем запрос
	amountStr := fmt.Sprintf("%.2f", amount)
	request := PaymentRequest{
		Amount: Amount{
			Value:    amountStr,
			Currency: "RUB",
		},
		Currency:    "RUB",
		Description: description,
		Receipt: &Receipt{
			Customer: Customer{
				Email: fmt.Sprintf("%d@telegram.user", userID), // Используем Telegram ID как email
			},
			Items: []Item{
				{
					Description: description,
					Quantity:    "1",
					Amount: Amount{
						Value:    amountStr,
						Currency: "RUB",
					},
					VatCode: 1, // НДС не облагается
				},
			},
		},
		Confirmation: Confirmation{
			Type:      "redirect",
			ReturnURL: fmt.Sprintf("https://t.me/%s", GlobalBot.Self.UserName), // Возврат в бота
		},
		Capture: true,
		Metadata: Metadata{
			TelegramID: fmt.Sprintf("%d", userID),
			UserID:     fmt.Sprintf("%d", userID),
			OrderType:  "balance_topup",
		},
	}

	// Преобразуем в JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		log.Printf("YUKASSA: Ошибка сериализации запроса: %v", err)
		return nil, fmt.Errorf("ошибка сериализации запроса: %v", err)
	}

	log.Printf("YUKASSA: Отправляем запрос: %s", string(jsonData))

	// Создаем HTTP запрос
	req, err := http.NewRequest("POST", y.baseURL+"/payments", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("YUKASSA: Ошибка создания HTTP запроса: %v", err)
		return nil, fmt.Errorf("ошибка создания HTTP запроса: %v", err)
	}

	// Устанавливаем заголовки (ЮКасса: shopId как логин, секретный ключ как пароль)
	auth := base64.StdEncoding.EncodeToString([]byte(y.shopID + ":" + YUKASSA_SECRET))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotence-Key", idempotencyKey)

	// Отправляем запрос
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("YUKASSA: Ошибка отправки HTTP запроса: %v", err)
		return nil, fmt.Errorf("ошибка отправки HTTP запроса: %v", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("YUKASSA: Ошибка чтения ответа: %v", err)
		return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	log.Printf("YUKASSA: Ответ от API (код %d): %s", resp.StatusCode, string(body))

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка API ЮКассы (код %d): %s", resp.StatusCode, string(body))
	}

	// Парсим ответ
	var payment PaymentResponse
	if err := json.Unmarshal(body, &payment); err != nil {
		log.Printf("YUKASSA: Ошибка парсинга ответа: %v", err)
		return nil, fmt.Errorf("ошибка парсинга ответа: %v", err)
	}

	log.Printf("YUKASSA: Платеж успешно создан, ID: %s", payment.ID)
	return &payment, nil
}

// GetPaymentStatus получает статус платежа по ID
func (y *YukassaAPI) GetPaymentStatus(paymentID string) (*PaymentStatus, error) {
	log.Printf("YUKASSA: Получение статуса платежа %s", paymentID)

	// Создаем HTTP запрос
	req, err := http.NewRequest("GET", y.baseURL+"/payments/"+paymentID, nil)
	if err != nil {
		log.Printf("YUKASSA: Ошибка создания HTTP запроса: %v", err)
		return nil, fmt.Errorf("ошибка создания HTTP запроса: %v", err)
	}

	// Устанавливаем заголовки (ЮКасса: shopId как логин, секретный ключ как пароль)
	auth := base64.StdEncoding.EncodeToString([]byte(y.shopID + ":" + YUKASSA_SECRET))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	// Отправляем запрос
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("YUKASSA: Ошибка отправки HTTP запроса: %v", err)
		return nil, fmt.Errorf("ошибка отправки HTTP запроса: %v", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("YUKASSA: Ошибка чтения ответа: %v", err)
		return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	log.Printf("YUKASSA: Ответ статуса (код %d): %s", resp.StatusCode, string(body))

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка API ЮКассы (код %d): %s", resp.StatusCode, string(body))
	}

	// Парсим ответ
	var status PaymentStatus
	if err := json.Unmarshal(body, &status); err != nil {
		log.Printf("YUKASSA: Ошибка парсинга ответа: %v", err)
		return nil, fmt.Errorf("ошибка парсинга ответа: %v", err)
	}

	log.Printf("YUKASSA: Статус платежа %s: %s", paymentID, status.Status)
	return &status, nil
}

// VerifyCallback проверяет подпись callback-а от ЮКассы
func (y *YukassaAPI) VerifyCallback(body []byte, signature string) bool {
	if YUKASSA_SECRET == "" {
		log.Printf("YUKASSA: Секретный ключ не настроен, пропускаем проверку подписи")
		return true // Если секрет не настроен, считаем callback валидным
	}

	// Формируем строку для проверки
	data := string(body) + YUKASSA_SECRET

	// Вычисляем SHA-256 хеш
	hash := sha256.Sum256([]byte(data))
	expectedSignature := fmt.Sprintf("%x", hash)

	// Сравниваем подписи
	return strings.EqualFold(signature, expectedSignature)
}

// ProcessCallback обрабатывает callback от ЮКассы
func (y *YukassaAPI) ProcessCallback(body []byte) error {
	log.Printf("YUKASSA: Обработка callback: %s", string(body))

	var payment PaymentResponse
	if err := json.Unmarshal(body, &payment); err != nil {
		log.Printf("YUKASSA: Ошибка парсинга callback: %v", err)
		return fmt.Errorf("ошибка парсинга callback: %v", err)
	}

	// Проверяем статус платежа
	if payment.Status == "succeeded" && payment.Paid {
		log.Printf("YUKASSA: Платеж %s успешно оплачен", payment.ID)

		// Получаем ID пользователя из метаданных
		telegramIDStr := payment.Metadata.TelegramID
		if telegramIDStr == "" {
			log.Printf("YUKASSA: Telegram ID не найден в метаданных платежа %s", payment.ID)
			return fmt.Errorf("Telegram ID не найден в метаданных")
		}

		// Конвертируем строку в int64
		var telegramID int64
		if _, err := fmt.Sscanf(telegramIDStr, "%d", &telegramID); err != nil {
			log.Printf("YUKASSA: Ошибка конвертации Telegram ID %s: %v", telegramIDStr, err)
			return fmt.Errorf("ошибка конвертации Telegram ID: %v", err)
		}

		// Получаем сумму платежа
		var amount float64
		if _, err := fmt.Sscanf(payment.Amount.Value, "%f", &amount); err != nil {
			log.Printf("YUKASSA: Ошибка конвертации суммы %s: %v", payment.Amount.Value, err)
			return fmt.Errorf("ошибка конвертации суммы: %v", err)
		}

		log.Printf("YUKASSA: Пополняем баланс пользователя %d на сумму %.2f", telegramID, amount)

		// Пополняем баланс пользователя
		err := AddBalance(telegramID, amount)
		if err != nil {
			log.Printf("YUKASSA: Ошибка пополнения баланса для пользователя %d: %v", telegramID, err)
			return fmt.Errorf("ошибка пополнения баланса: %v", err)
		}

		// Получаем пользователя для уведомлений
		user, err := GetUserByTelegramID(telegramID)
		if err != nil {
			log.Printf("YUKASSA: Ошибка получения пользователя %d: %v", telegramID, err)
		} else {
			// Отправляем уведомление администратору
			SendBalanceTopupNotificationToAdmin(user, amount)

			// Принудительно запускаем пересчет баланса
			ForceBalanceRecalculation(telegramID)
		}

		log.Printf("YUKASSA: Callback успешно обработан для платежа %s", payment.ID)
	} else {
		log.Printf("YUKASSA: Платеж %s не оплачен, статус: %s, paid: %v", payment.ID, payment.Status, payment.Paid)
	}

	return nil
}

// GetPaymentURL возвращает URL для оплаты из ответа создания платежа
func GetPaymentURL(payment *PaymentResponse) string {
	if payment.Confirmation.Type == "redirect" {
		return payment.Confirmation.ConfirmationURL
	}
	return ""
}
