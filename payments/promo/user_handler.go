package promo

import (
	"fmt"
	"log"
	"strings"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// UserPromoHandler обработчик команд пользователей для промокодов
type UserPromoHandler struct {
	service *PromoService
}

// NewUserPromoHandler создает новый обработчик команд пользователей
func NewUserPromoHandler() (*UserPromoHandler, error) {
	service, err := NewPromoService()
	if err != nil {
		return nil, fmt.Errorf("ошибка создания сервиса промокодов: %v", err)
	}

	return &UserPromoHandler{service: service}, nil
}

// HandlePromoCommand обрабатывает команду /promo
func (h *UserPromoHandler) HandlePromoCommand(chatID int64, userID int64, args []string) error {
	// Проверяем, что пользователь существует
	_, err := common.GetUserByTelegramID(userID)
	if err != nil {
		return h.sendMessage(chatID, "❌ Ошибка получения данных пользователя.")
	}

	// Проверяем наличие промокода в аргументах
	if len(args) == 0 {
		return h.sendPromoUsageHelp(chatID)
	}

	promoCode := strings.TrimSpace(args[0])
	if promoCode == "" {
		return h.sendPromoUsageHelp(chatID)
	}

	// Проверяем формат промокода (должен быть 8 символов)
	if len(promoCode) != 8 {
		return h.sendMessage(chatID, "❌ Неверный формат промокода. Промокод должен содержать 8 символов.")
	}

	// Используем промокод
	promo, err := h.service.UsePromoCode(promoCode, userID)
	if err != nil {
		log.Printf("PROMO_USER: Ошибка использования промокода %s пользователем %d: %v",
			promoCode, userID, err)

		// Определяем тип ошибки и отправляем соответствующее сообщение
		errorMsg := h.getErrorMessage(err)
		return h.sendMessage(chatID, errorMsg)
	}

	// Получаем обновленный баланс пользователя
	updatedUser, err := common.GetUserByTelegramID(userID)
	if err != nil {
		log.Printf("PROMO_USER: Ошибка получения обновленного баланса пользователя %d: %v", userID, err)
		// Не возвращаем ошибку, так как промокод уже использован
	}

	// Отправляем сообщение об успешном использовании
	text := fmt.Sprintf("✅ <b>Промокод успешно активирован!</b>\n\n"+
		"🎁 <b>Код:</b> <code>%s</code>\n"+
		"💰 <b>Получено:</b> %.2f₽\n",
		promo.Code, promo.Amount)

	if updatedUser != nil {
		text += fmt.Sprintf("💳 <b>Текущий баланс:</b> %.2f₽", updatedUser.Balance)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	_, err = common.GlobalBot.Send(msg)
	if err != nil {
		return fmt.Errorf("ошибка отправки сообщения об успехе: %v", err)
	}

	// Логируем успешное использование
	log.Printf("PROMO_USER: Промокод %s успешно использован пользователем %d (%.2f₽)",
		promo.Code, userID, promo.Amount)

	return nil
}

// HandlePromoHistoryCommand обрабатывает команду для просмотра истории промокодов
func (h *UserPromoHandler) HandlePromoHistoryCommand(chatID int64, userID int64) error {
	// Получаем историю использования промокодов
	history, err := h.service.GetUserPromoHistory(userID, 10) // Последние 10 использований
	if err != nil {
		log.Printf("PROMO_USER: Ошибка получения истории промокодов для пользователя %d: %v", userID, err)
		return h.sendMessage(chatID, "❌ Ошибка получения истории промокодов.")
	}

	if len(history) == 0 {
		return h.sendMessage(chatID, "📝 <b>История промокодов</b>\n\n"+
			"У вас пока нет использованных промокодов.")
	}

	// Формируем текст с историей
	text := "📝 <b>История промокодов</b>\n\n"

	var totalAmount float64
	for i, usage := range history {
		text += fmt.Sprintf("%d. %.2f₽ - %s\n",
			i+1,
			usage.Amount,
			usage.UsedAt.Format("02.01.2006 15:04"))
		totalAmount += usage.Amount
	}

	text += fmt.Sprintf("\n💰 <b>Всего получено:</b> %.2f₽", totalAmount)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	_, err = common.GlobalBot.Send(msg)
	if err != nil {
		return fmt.Errorf("ошибка отправки истории: %v", err)
	}

	return nil
}

// sendPromoUsageHelp отправляет справку по использованию промокодов
func (h *UserPromoHandler) sendPromoUsageHelp(chatID int64) error {
	text := "🎁 <b>Использование промокодов</b>\n\n" +
		"Для активации промокода используйте команду:\n" +
		"<code>/promo КОД_ПРОМОКОДА</code>\n\n" +
		"<b>Пример:</b>\n" +
		"<code>/promo 245nmao1</code>\n\n" +
		"📋 <b>Правила использования:</b>\n" +
		"• Один промокод можно использовать только один раз\n" +
		"• Один пользователь может активировать промокод раз в 24 часа\n" +
		"• Промокод действует 14 дней с момента создания\n" +
		"• Деньги зачисляются на ваш баланс мгновенно\n\n" +
		"Для просмотра истории использования:\n" +
		"<code>/promohistory</code>"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	_, err := common.GlobalBot.Send(msg)
	return err
}

// getErrorMessage возвращает понятное сообщение об ошибке
func (h *UserPromoHandler) getErrorMessage(err error) string {
	errorStr := err.Error()

	switch {
	case strings.Contains(errorStr, "не может быть использован"):
		if strings.Contains(errorStr, "Истек") {
			return "❌ Промокод истек. Срок действия промокода составляет 14 дней."
		}
		if strings.Contains(errorStr, "Уже использован") {
			return "❌ Вы уже использовали этот промокод или активировали другой промокод в последние 24 часа."
		}
		if strings.Contains(errorStr, "Не найден") {
			return "❌ Промокод не найден или неактивен. Проверьте правильность ввода."
		}
		if strings.Contains(errorStr, "Достигнут лимит") {
			return "❌ Промокод исчерпал лимит использований."
		}
		return "❌ Промокод не может быть использован."

	case strings.Contains(errorStr, "ошибка валидации"):
		return "❌ Ошибка проверки промокода. Попробуйте позже."

	case strings.Contains(errorStr, "ошибка пополнения баланса"):
		return "❌ Ошибка пополнения баланса. Обратитесь в поддержку."

	case strings.Contains(errorStr, "ошибка транзакции"):
		return "❌ Ошибка обработки промокода. Попробуйте позже."

	default:
		return "❌ Произошла ошибка при активации промокода. Попробуйте позже."
	}
}

// sendMessage отправляет текстовое сообщение
func (h *UserPromoHandler) sendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	_, err := common.GlobalBot.Send(msg)
	return err
}

// ValidatePromoCodeFormat проверяет формат промокода
func (h *UserPromoHandler) ValidatePromoCodeFormat(code string) error {
	// Промокод должен содержать ровно 8 символов
	if len(code) != 8 {
		return fmt.Errorf("промокод должен содержать 8 символов")
	}

	// Проверяем, что все символы являются буквами и цифрами
	for _, char := range code {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
			return fmt.Errorf("промокод может содержать только строчные буквы и цифры")
		}
	}

	return nil
}
