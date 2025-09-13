package payments

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"bot/common"
	paymentCommon "bot/payments/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// WebhookHandlers содержит обработчики веб-хуков для разных платежных систем
type WebhookHandlers struct {
	paymentManager *PaymentManager
}

// NewWebhookHandlers создает новый экземпляр обработчиков веб-хуков
func NewWebhookHandlers(paymentManager *PaymentManager) *WebhookHandlers {
	return &WebhookHandlers{
		paymentManager: paymentManager,
	}
}

// HandleYooKassaWebhook обрабатывает веб-хуки от ЮКассы
func (wh *WebhookHandlers) HandleYooKassaWebhook(w http.ResponseWriter, r *http.Request) {
	log.Printf("WEBHOOK_YOOKASSA: Получен webhook от ЮКассы")
	log.Printf("WEBHOOK_YOOKASSA: Method: %s, URL: %s, Headers: %+v", r.Method, r.URL.String(), r.Header)

	// Проверяем метод запроса
	if r.Method != http.MethodPost {
		log.Printf("WEBHOOK_YOOKASSA: Неверный метод запроса: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("WEBHOOK_YOOKASSA: Ошибка чтения тела запроса: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("WEBHOOK_YOOKASSA: Получено тело запроса (длина: %d байт)", len(body))
	log.Printf("WEBHOOK_YOOKASSA: Тело запроса: %s", string(body))

	// Проверяем, что платежи через API включены
	if !common.YUKASSA_API_PAYMENTS_ENABLED {
		log.Printf("WEBHOOK_YOOKASSA: Платежи через API ЮКассы отключены")
		wh.sendSuccessResponse(w)
		return
	}

	// Обрабатываем webhook
	paymentInfo, err := wh.paymentManager.ProcessWebhook(paymentCommon.PaymentMethodAPI, body)
	if err != nil {
		log.Printf("WEBHOOK_YOOKASSA: Ошибка обработки webhook: %v", err)
		http.Error(w, "Webhook processing error", http.StatusBadRequest)
		return
	}

	if paymentInfo == nil {
		log.Printf("WEBHOOK_YOOKASSA: Webhook обработан, но информация о платеже не получена")
		wh.sendSuccessResponse(w)
		return
	}

	log.Printf("WEBHOOK_YOOKASSA: Обработан платеж ID=%s, UserID=%d, Status=%s, Amount=%.2f",
		paymentInfo.ID, paymentInfo.UserID, paymentInfo.Status, paymentInfo.Amount)

	// Отправляем уведомление пользователю, если платеж успешен
	if paymentInfo.Status == paymentCommon.PaymentStatusSucceeded && paymentInfo.UserID > 0 {
		err = wh.sendPaymentNotificationToUser(paymentInfo)
		if err != nil {
			log.Printf("WEBHOOK_YOOKASSA: Ошибка отправки уведомления пользователю: %v", err)
			// Не возвращаем ошибку, так как основная обработка прошла успешно
		}

		// Отправляем уведомление администратору
		err = wh.sendPaymentNotificationToAdmin(paymentInfo)
		if err != nil {
			log.Printf("WEBHOOK_YOOKASSA: Ошибка отправки уведомления администратору: %v", err)
		}
	}

	// Возвращаем успешный ответ
	wh.sendSuccessResponse(w)
	log.Printf("WEBHOOK_YOOKASSA: Webhook успешно обработан")
}

// sendPaymentNotificationToUser отправляет уведомление о платеже пользователю
func (wh *WebhookHandlers) sendPaymentNotificationToUser(paymentInfo *paymentCommon.PaymentInfo) error {
	user, err := common.GetUserByTelegramID(paymentInfo.UserID)
	if err != nil {
		return fmt.Errorf("ошибка получения пользователя: %v", err)
	}

	// Формируем текст уведомления в зависимости от статуса
	var text string
	var keyboard *tgbotapi.InlineKeyboardMarkup

	switch paymentInfo.Status {
	case paymentCommon.PaymentStatusSucceeded:
		text = fmt.Sprintf("✅ <b>Платеж успешно выполнен!</b>\n\n"+
			"💰 Пополнено: %s\n"+
			"💳 Новый баланс: %.2f₽\n"+
			"🏦 Платежная система: %s\n"+
			"🆔 ID платежа: %s\n\n"+
			"Спасибо за пополнение! Теперь вы можете пользоваться нашими услугами.",
			paymentCommon.FormatAmount(paymentInfo.Amount), user.Balance,
			paymentCommon.GetMethodDescription(paymentInfo.Method), paymentInfo.ID)

		keyboardButtons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
			),
		)
		keyboard = &keyboardButtons

	case paymentCommon.PaymentStatusCanceled:
		text = fmt.Sprintf("❌ <b>Платеж отменен</b>\n\n"+
			"💰 Сумма: %s\n"+
			"🆔 ID платежа: %s\n\n"+
			"Если у вас есть вопросы, обратитесь в поддержку.",
			paymentCommon.FormatAmount(paymentInfo.Amount), paymentInfo.ID)

		keyboardButtons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💳 Попробовать снова", "topup"),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
			),
		)
		keyboard = &keyboardButtons

	case paymentCommon.PaymentStatusFailed:
		text = fmt.Sprintf("❌ <b>Ошибка платежа</b>\n\n"+
			"💰 Сумма: %s\n"+
			"🆔 ID платежа: %s\n\n"+
			"Произошла ошибка при обработке платежа. Попробуйте еще раз или обратитесь в поддержку.",
			paymentCommon.FormatAmount(paymentInfo.Amount), paymentInfo.ID)

		keyboardButtons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💳 Попробовать снова", "topup"),
				tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
			),
		)
		keyboard = &keyboardButtons

	default:
		// Для других статусов не отправляем уведомления
		return nil
	}

	msg := tgbotapi.NewMessage(paymentInfo.UserID, text)
	msg.ParseMode = "HTML"
	if keyboard != nil {
		msg.ReplyMarkup = keyboard
	}

	_, err = common.GlobalBot.Send(msg)
	if err != nil {
		return fmt.Errorf("ошибка отправки сообщения пользователю: %v", err)
	}

	return nil
}

// sendPaymentNotificationToAdmin отправляет уведомление о платеже администратору
func (wh *WebhookHandlers) sendPaymentNotificationToAdmin(paymentInfo *paymentCommon.PaymentInfo) error {
	// Проверяем, включены ли уведомления администратора
	if !common.ADMIN_NOTIFICATIONS_ENABLED || !common.ADMIN_BALANCE_TOPUP_ENABLED {
		return nil
	}

	// Отправляем уведомление только для успешных платежей
	if paymentInfo.Status != paymentCommon.PaymentStatusSucceeded {
		return nil
	}

	user, err := common.GetUserByTelegramID(paymentInfo.UserID)
	if err != nil {
		return fmt.Errorf("ошибка получения данных пользователя для уведомления: %v", err)
	}

	notificationText := fmt.Sprintf(
		"💰 <b>Пополнение баланса</b>\n\n"+
			"👤 Пользователь: %s %s\n"+
			"🆔 Telegram ID: %d\n"+
			"💵 Сумма: %s\n"+
			"💳 Новый баланс: %.2f₽\n"+
			"🏦 Платежная система: %s\n"+
			"📅 ID платежа: %s",
		user.FirstName, user.LastName, paymentInfo.UserID,
		paymentCommon.FormatAmount(paymentInfo.Amount), user.Balance,
		paymentCommon.GetMethodDescription(paymentInfo.Method), paymentInfo.ID)

	msg := tgbotapi.NewMessage(common.ADMIN_ID, notificationText)
	msg.ParseMode = "HTML"

	_, err = common.GlobalBot.Send(msg)
	if err != nil {
		return fmt.Errorf("ошибка отправки уведомления администратору: %v", err)
	}

	return nil
}

// sendSuccessResponse отправляет успешный ответ на webhook
func (wh *WebhookHandlers) sendSuccessResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]string{"status": "ok"}
	json.NewEncoder(w).Encode(response)
}

// RegisterWebhookRoutes регистрирует маршруты для веб-хуков
func RegisterWebhookRoutes(mux *http.ServeMux, paymentManager *PaymentManager) {
	handlers := NewWebhookHandlers(paymentManager)

	// Регистрируем обработчик для ЮКассы
	mux.HandleFunc("/yukassa/webhook", handlers.HandleYooKassaWebhook)

	log.Printf("WEBHOOK_ROUTES: Зарегистрированы маршруты веб-хуков:")
	log.Printf("WEBHOOK_ROUTES: - POST /yukassa/webhook - обработка уведомлений от ЮКассы")
}

// HandleCheckPayment обрабатывает проверку статуса платежа
func (wh *WebhookHandlers) HandleCheckPayment(paymentID string, chatID int64, messageID int) error {
	log.Printf("WEBHOOK_CHECK: Проверка статуса платежа %s", paymentID)

	// Пробуем получить платеж через API ЮКассы
	paymentInfo, err := wh.paymentManager.CheckPaymentStatus(paymentCommon.PaymentMethodAPI, paymentID)
	if err != nil {
		log.Printf("WEBHOOK_CHECK: Ошибка проверки платежа %s: %v", paymentID, err)

		// Отправляем сообщение об ошибке
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID,
			fmt.Sprintf("❌ Ошибка проверки платежа\n\n🆔 ID: %s\n\nПопробуйте позже или обратитесь в поддержку.", paymentID))

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Проверить снова", fmt.Sprintf("check_payment:%s", paymentID)),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
			),
		)
		editMsg.ReplyMarkup = &keyboard

		if _, err := common.GlobalBot.Send(editMsg); err != nil {
			log.Printf("WEBHOOK_CHECK: Ошибка отправки сообщения об ошибке: %v", err)
		}
		return err
	}

	// Формируем сообщение в зависимости от статуса
	var text string
	var keyboard *tgbotapi.InlineKeyboardMarkup

	switch paymentInfo.Status {
	case paymentCommon.PaymentStatusSucceeded:
		// Извлекаем userID из метаданных если он не установлен
		userID := paymentInfo.UserID
		if userID == 0 {
			if userIDValue, exists := paymentInfo.Metadata["user_id"]; exists {
				switch v := userIDValue.(type) {
				case float64:
					userID = int64(v)
				case int64:
					userID = v
				case int:
					userID = int64(v)
				case string:
					fmt.Sscanf(v, "%d", &userID)
				}
			}
		}

		if userID == 0 {
			log.Printf("WEBHOOK_CHECK: Не удалось определить UserID для платежа %s", paymentID)
			text = fmt.Sprintf("❌ <b>Ошибка обработки платежа</b>\n\n🆔 ID: %s\n\nНе удалось определить пользователя. Обратитесь в поддержку.", paymentID)
		} else {
			// Зачисляем средства если они еще не зачислены
			user, err := common.GetUserByTelegramID(userID)
			if err != nil {
				log.Printf("WEBHOOK_CHECK: Ошибка получения пользователя %d: %v", userID, err)
				text = fmt.Sprintf("❌ <b>Ошибка обработки платежа</b>\n\n🆔 ID: %s\n\nОшибка получения данных пользователя.", paymentID)
			} else {
				// Зачисляем средства
				err = common.AddBalance(userID, paymentInfo.Amount)
				if err != nil {
					log.Printf("WEBHOOK_CHECK: Ошибка зачисления средств для платежа %s: %v", paymentID, err)
					text = fmt.Sprintf("❌ <b>Ошибка зачисления средств</b>\n\n🆔 ID: %s\n\nОшибка зачисления на баланс.", paymentID)
				} else {
					// Получаем обновленные данные пользователя
					user, err = common.GetUserByTelegramID(userID)
					if err != nil {
						log.Printf("WEBHOOK_CHECK: Ошибка получения обновленных данных пользователя %d: %v", userID, err)
						text = fmt.Sprintf("✅ <b>Платеж обработан!</b>\n\n💰 Пополнено: %s\n🆔 ID: %s", paymentCommon.FormatAmount(paymentInfo.Amount), paymentID)
					} else {
						text = fmt.Sprintf("✅ <b>Платеж выполнен успешно!</b>\n\n"+
							"💰 Пополнено: %s\n"+
							"💳 Новый баланс: %.2f₽\n"+
							"🆔 ID платежа: %s\n\n"+
							"Спасибо за пополнение!",
							paymentCommon.FormatAmount(paymentInfo.Amount), user.Balance, paymentID)

						log.Printf("WEBHOOK_CHECK: Средства зачислены для платежа %s, пользователь %d, баланс: %.2f₽", paymentID, userID, user.Balance)
					}
				}
			}
		}

		keyboardButtons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
			),
		)
		keyboard = &keyboardButtons

	case paymentCommon.PaymentStatusPending:
		text = fmt.Sprintf("⏳ <b>Платеж обрабатывается</b>\n\n"+
			"💰 Сумма: %s\n"+
			"🆔 ID платежа: %s\n\n"+
			"Пожалуйста, подождите. Проверьте статус через несколько минут.",
			paymentCommon.FormatAmount(paymentInfo.Amount), paymentInfo.ID)

		keyboardButtons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Проверить снова", fmt.Sprintf("check_payment:%s", paymentID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
			),
		)
		keyboard = &keyboardButtons

	case paymentCommon.PaymentStatusCanceled:
		text = fmt.Sprintf("❌ <b>Платеж отменен</b>\n\n"+
			"💰 Сумма: %s\n"+
			"🆔 ID платежа: %s",
			paymentCommon.FormatAmount(paymentInfo.Amount), paymentInfo.ID)

		keyboardButtons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💳 Новый платеж", "topup"),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
			),
		)
		keyboard = &keyboardButtons

	case paymentCommon.PaymentStatusFailed:
		text = fmt.Sprintf("❌ <b>Ошибка платежа</b>\n\n"+
			"💰 Сумма: %s\n"+
			"🆔 ID платежа: %s\n\n"+
			"Произошла ошибка. Попробуйте создать новый платеж.",
			paymentCommon.FormatAmount(paymentInfo.Amount), paymentInfo.ID)

		keyboardButtons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💳 Новый платеж", "topup"),
				tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
			),
		)
		keyboard = &keyboardButtons

	default:
		text = fmt.Sprintf("❓ <b>Неизвестный статус платежа</b>\n\n"+
			"🆔 ID платежа: %s\n\n"+
			"Обратитесь в поддержку для уточнения.",
			paymentInfo.ID)

		keyboardButtons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", common.SUPPORT_LINK),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main"),
			),
		)
		keyboard = &keyboardButtons
	}

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ParseMode = "HTML"
	if keyboard != nil {
		editMsg.ReplyMarkup = keyboard
	}

	_, err = common.GlobalBot.Send(editMsg)
	if err != nil {
		log.Printf("WEBHOOK_CHECK: Ошибка отправки обновленного сообщения: %v", err)
		return err
	}

	log.Printf("WEBHOOK_CHECK: Статус платежа %s успешно проверен: %s", paymentID, paymentInfo.Status)
	return nil
}
