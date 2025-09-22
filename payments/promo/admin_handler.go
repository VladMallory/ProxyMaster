package promo

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// AdminPromoHandler обработчик команд админа для промокодов
type AdminPromoHandler struct {
	service *PromoService
}

// NewAdminPromoHandler создает новый обработчик команд админа
func NewAdminPromoHandler() (*AdminPromoHandler, error) {
	service, err := NewPromoService()
	if err != nil {
		return nil, fmt.Errorf("ошибка создания сервиса промокодов: %v", err)
	}

	return &AdminPromoHandler{service: service}, nil
}

// HandlePromoSetCommand обрабатывает команду /promoset
func (h *AdminPromoHandler) HandlePromoSetCommand(chatID int64, userID int64) error {
	// Проверяем, что пользователь - админ
	if !IsAdmin(userID) {
		return h.sendMessage(chatID, "❌ У вас нет прав для выполнения этой команды.")
	}

	// Создаем клавиатуру с предопределенными суммами
	keyboard := h.createAmountKeyboard()

	text := "🎁 <b>Создание промокода</b>\n\n" +
		"На сколько денег сделать промокод?\n\n" +
		"Выберите сумму из предложенных вариантов:"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard

	_, err := common.GlobalBot.Send(msg)
	if err != nil {
		return fmt.Errorf("ошибка отправки сообщения: %v", err)
	}

	return nil
}

// HandlePromoSetCallback обрабатывает callback от кнопок выбора суммы
func (h *AdminPromoHandler) HandlePromoSetCallback(chatID int64, userID int64, callbackData string) error {
	// Проверяем, что пользователь - админ
	if !IsAdmin(userID) {
		return h.sendMessage(chatID, "❌ У вас нет прав для выполнения этой команды.")
	}

	// Извлекаем сумму из callback data
	parts := strings.Split(callbackData, ":")
	if len(parts) != 2 || parts[0] != "promo_amount" {
		return h.sendMessage(chatID, "❌ Неверный формат данных.")
	}

	amountStr := parts[1]
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return h.sendMessage(chatID, "❌ Неверная сумма.")
	}

	// Создаем промокод
	promo, err := h.service.CreatePromoCode(amount, userID)
	if err != nil {
		log.Printf("PROMO_ADMIN: Ошибка создания промокода: %v", err)
		return h.sendMessage(chatID, "❌ Ошибка создания промокода. Попробуйте позже.")
	}

	// Отправляем результат
	text := fmt.Sprintf("✅ <b>Промокод создан!</b>\n\n"+
		"🎁 <b>Код:</b> <code>%s</code>\n"+
		"💰 <b>Сумма:</b> %.2f₽\n"+
		"⏰ <b>Действует до:</b> %s\n"+
		"👥 <b>Максимум использований:</b> %d\n\n"+
		"Пользователь может активировать промокод командой:\n"+
		"<code>/promo %s</code>",
		promo.Code,
		promo.Amount,
		promo.ExpiresAt.Format("02.01.2006 15:04"),
		promo.MaxUses,
		promo.Code)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	// Добавляем кнопку для копирования промокода
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Копировать код", fmt.Sprintf("copy_promo:%s", promo.Code)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Статистика промокодов", "promo_stats"),
		),
	)
	msg.ReplyMarkup = keyboard

	_, err = common.GlobalBot.Send(msg)
	if err != nil {
		return fmt.Errorf("ошибка отправки результата: %v", err)
	}

	return nil
}

// HandlePromoStatsCallback обрабатывает запрос статистики промокодов
func (h *AdminPromoHandler) HandlePromoStatsCallback(chatID int64, userID int64) error {
	// Проверяем, что пользователь - админ
	if !IsAdmin(userID) {
		return h.sendMessage(chatID, "❌ У вас нет прав для выполнения этой команды.")
	}

	// Получаем статистику
	stats, err := h.service.GetPromoStats(userID)
	if err != nil {
		log.Printf("PROMO_ADMIN: Ошибка получения статистики: %v", err)
		return h.sendMessage(chatID, "❌ Ошибка получения статистики. Попробуйте позже.")
	}

	// Форматируем статистику
	text := fmt.Sprintf("📊 <b>Статистика промокодов</b>\n\n"+
		"🎁 <b>Всего создано:</b> %d\n"+
		"✅ <b>Использовано:</b> %d\n"+
		"💰 <b>Общая сумма выдана:</b> %.2f₽\n"+
		"📈 <b>Процент использования:</b> %.1f%%",
		stats["total_created"],
		stats["total_used"],
		stats["total_amount"],
		stats["usage_rate"])

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	// Добавляем кнопку для создания нового промокода
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎁 Создать новый промокод", "create_promo"),
		),
	)
	msg.ReplyMarkup = keyboard

	_, err = common.GlobalBot.Send(msg)
	if err != nil {
		return fmt.Errorf("ошибка отправки статистики: %v", err)
	}

	return nil
}

// HandleCopyPromoCallback обрабатывает копирование промокода
func (h *AdminPromoHandler) HandleCopyPromoCallback(chatID int64, userID int64, callbackData string) error {
	// Проверяем, что пользователь - админ
	if !IsAdmin(userID) {
		return h.sendMessage(chatID, "❌ У вас нет прав для выполнения этой команды.")
	}

	// Извлекаем промокод из callback data
	parts := strings.Split(callbackData, ":")
	if len(parts) != 2 || parts[0] != "copy_promo" {
		return h.sendMessage(chatID, "❌ Неверный формат данных.")
	}

	promoCode := parts[1]

	// Отправляем промокод отдельным сообщением для удобного копирования
	text := fmt.Sprintf("📋 <b>Промокод для копирования:</b>\n\n<code>%s</code>", promoCode)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	_, err := common.GlobalBot.Send(msg)
	if err != nil {
		return fmt.Errorf("ошибка отправки промокода: %v", err)
	}

	return nil
}

// createAmountKeyboard создает клавиатуру с предопределенными суммами
func (h *AdminPromoHandler) createAmountKeyboard() tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Создаем кнопки для каждой предопределенной суммы
	for _, amount := range PredefinedAmounts {
		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("💰 %.0f₽", amount),
			fmt.Sprintf("promo_amount:%.0f", amount),
		)
		rows = append(rows, []tgbotapi.InlineKeyboardButton{button})
	}

	// Добавляем кнопку для произвольной суммы
	customButton := tgbotapi.NewInlineKeyboardButtonData(
		"🔧 Произвольная сумма",
		"promo_custom",
	)
	rows = append(rows, []tgbotapi.InlineKeyboardButton{customButton})

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// HandleCustomAmountCallback обрабатывает ввод произвольной суммы
func (h *AdminPromoHandler) HandleCustomAmountCallback(chatID int64, userID int64) error {
	// Проверяем, что пользователь - админ
	if !IsAdmin(userID) {
		return h.sendMessage(chatID, "❌ У вас нет прав для выполнения этой команды.")
	}

	text := "🔧 <b>Произвольная сумма</b>\n\n" +
		"Введите сумму для промокода в рублях (например: 1500):"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	_, err := common.GlobalBot.Send(msg)
	if err != nil {
		return fmt.Errorf("ошибка отправки сообщения: %v", err)
	}

	// Здесь можно добавить логику для ожидания следующего сообщения от пользователя
	// и обработки произвольной суммы

	return nil
}

// sendMessage отправляет текстовое сообщение
func (h *AdminPromoHandler) sendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	_, err := common.GlobalBot.Send(msg)
	return err
}
