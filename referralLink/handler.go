package referralLink

import (
	"fmt"
	"log"
	"strings"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ReferralHandler обработчик команд и callback'ов реферальной системы
type ReferralHandler struct {
	service *ReferralService
	bot     *tgbotapi.BotAPI
}

// NewReferralHandler создает новый обработчик реферальной системы
func NewReferralHandler(service *ReferralService, bot *tgbotapi.BotAPI) *ReferralHandler {
	return &ReferralHandler{
		service: service,
		bot:     bot,
	}
}

// HandleRefCommand обрабатывает команду /ref
func (rh *ReferralHandler) HandleRefCommand(chatID int64, user *common.User) {
	log.Printf("REFERRAL_HANDLER: Обработка команды /ref для пользователя %d", user.TelegramID)

	// Проверяем, включена ли реферальная система
	if !common.REFERRAL_SYSTEM_ENABLED {
		msg := tgbotapi.NewMessage(chatID, "❌ Реферальная система временно отключена")
		rh.bot.Send(msg)
		return
	}

	// Получаем информацию о реферальной ссылке
	linkInfo, err := rh.service.GetReferralLinkInfo(user.TelegramID)
	if err != nil {
		log.Printf("REFERRAL_HANDLER: Ошибка получения реферальной ссылки для %d: %v", user.TelegramID, err)
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка получения реферальной ссылки")
		rh.bot.Send(msg)
		return
	}

	// Получаем статистику рефералов
	stats, err := rh.service.GetReferralStats(user.TelegramID)
	if err != nil {
		log.Printf("REFERRAL_HANDLER: Ошибка получения статистики для %d: %v", user.TelegramID, err)
		stats = &ReferralStats{} // Используем пустую статистику
	}

	// Формируем сообщение
	text := fmt.Sprintf("🎯 <b>Реферальная система</b>\n\n")
	text += "💰 <b>Ваш бонус за приглашение:</b> " + fmt.Sprintf("%.0f", common.REFERRAL_BONUS_AMOUNT) + "₽\n"
	text += "🎁 <b>Бонус для друга:</b> " + fmt.Sprintf("%.0f", common.REFERRAL_WELCOME_BONUS) + "₽\n\n"

	text += "📊 <b>Ваша статистика:</b>\n"
	text += "👥 Приглашено друзей: " + fmt.Sprintf("%d", stats.TotalReferrals) + "\n"

	text += "🔗 <b>Ваша реферальная ссылка:</b>\n"
	text += "<code>" + linkInfo.ReferralLink + "</code>\n\n"

	text += "📱 <b>Как пригласить друга:</b>\n"
	text += "1️⃣ Отправьте ссылку другу\n"
	text += "2️⃣ Друг переходит по ссылке и регистрируется\n"
	text += "3️⃣ Вы оба получаете бонусы!\n\n"

	// Создаем клавиатуру
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", "ref_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 История бонусов", "ref_history"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔗 Поделиться ссылкой", "ref_share"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	if _, err := rh.bot.Send(msg); err != nil {
		log.Printf("REFERRAL_HANDLER: Ошибка отправки сообщения: %v", err)
	}
}

// HandleRefCallback обрабатывает callback'и реферальной системы
func (rh *ReferralHandler) HandleRefCallback(chatID int64, userID int64, data string) {
	log.Printf("REFERRAL_HANDLER: Обработка callback %s для пользователя %d", data, userID)

	user, err := common.GetUserByTelegramID(userID)
	if err != nil {
		log.Printf("REFERRAL_HANDLER: Ошибка получения пользователя %d: %v", userID, err)
		return
	}

	switch data {
	case "ref_stats":
		rh.handleStatsCallback(chatID, user)
	case "ref_history":
		rh.handleHistoryCallback(chatID, user)
	case "ref_share":
		rh.handleShareCallback(chatID, user)
	default:
		log.Printf("REFERRAL_HANDLER: Неизвестный callback: %s", data)
	}
}

// handleStatsCallback обрабатывает callback статистики
func (rh *ReferralHandler) handleStatsCallback(chatID int64, user *common.User) {
	stats, err := rh.service.GetReferralStats(user.TelegramID)
	if err != nil {
		log.Printf("REFERRAL_HANDLER: Ошибка получения статистики: %v", err)
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка получения статистики")
		rh.bot.Send(msg)
		return
	}

	text := "📊 <b>Статистика рефералов</b>\n\n"
	text += "👥 <b>Всего приглашено:</b> " + fmt.Sprintf("%d", stats.TotalReferrals) + "\n"
	text += "✅ <b>Успешных приглашений:</b> " + fmt.Sprintf("%d", stats.SuccessfulReferrals) + "\n"
	text += "⏳ <b>Ожидающих:</b> " + fmt.Sprintf("%d", stats.PendingReferrals) + "\n"

	text += "💰 <b>Бонусы:</b>\n"
	text += "• За приглашение: " + fmt.Sprintf("%.0f", common.REFERRAL_BONUS_AMOUNT) + "₽\n"
	text += "• Другу за регистрацию: " + fmt.Sprintf("%.0f", common.REFERRAL_WELCOME_BONUS) + "₽\n"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "ref_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	rh.bot.Send(msg)
}

// handleHistoryCallback обрабатывает callback истории бонусов
func (rh *ReferralHandler) handleHistoryCallback(chatID int64, user *common.User) {
	bonuses, err := rh.service.GetReferralHistory(user.TelegramID, 10)
	if err != nil {
		log.Printf("REFERRAL_HANDLER: Ошибка получения истории: %v", err)
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка получения истории бонусов")
		rh.bot.Send(msg)
		return
	}

	text := "📋 <b>История реферальных бонусов</b>\n\n"

	if len(bonuses) == 0 {
		text += "📭 Пока нет бонусов\n"
		text += "Пригласите друзей, чтобы начать зарабатывать!"
	} else {
		for i, bonus := range bonuses {
			text += fmt.Sprintf("%d. %s: <b>+%.2f₽</b>\n", i+1, bonus.Description, bonus.Amount)
			text += "   📅 " + bonus.CreatedAt.Format("02.01.2006 15:04") + "\n\n"
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "ref_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	rh.bot.Send(msg)
}

// handleShareCallback обрабатывает callback поделиться ссылкой
func (rh *ReferralHandler) handleShareCallback(chatID int64, user *common.User) {
	linkInfo, err := rh.service.GetReferralLinkInfo(user.TelegramID)
	if err != nil {
		log.Printf("REFERRAL_HANDLER: Ошибка получения ссылки: %v", err)
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка получения реферальной ссылки")
		rh.bot.Send(msg)
		return
	}

	text := "🔗 <b>Поделиться реферальной ссылкой</b>\n\n"
	text += "Скопируйте ссылку ниже и отправьте другу:\n\n"
	text += "<code>" + linkInfo.ReferralLink + "</code>\n\n"
	text += "💡 <i>При регистрации по этой ссылке вы оба получите бонусы!</i>"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "ref_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	rh.bot.Send(msg)
}

// ProcessReferralStart обрабатывает команду /start с реферальным кодом
func (rh *ReferralHandler) ProcessReferralStart(chatID int64, user *common.User, referralCode string) {
	log.Printf("REFERRAL_HANDLER: Обработка реферального перехода для пользователя %d, код: %s", user.TelegramID, referralCode)

	// Получаем информацию о пригласившем
	referrer, err := rh.service.GetReferrerByCode(referralCode)
	if err != nil {
		log.Printf("REFERRAL_HANDLER: Ошибка получения пригласившего по коду %s: %v", referralCode, err)
		return
	}

	// Проверяем, что пользователь не приглашает сам себя
	if referrer.TelegramID == user.TelegramID {
		log.Printf("REFERRAL_HANDLER: Пользователь %d пытается пригласить сам себя", user.TelegramID)
		return
	}

	// Обрабатываем реферальный переход
	err = rh.service.ProcessReferralTransition(referrer.TelegramID, user.TelegramID, referralCode)
	if err != nil {
		log.Printf("REFERRAL_HANDLER: Ошибка обработки реферального перехода: %v", err)
		return
	}

	// Начисляем бонусы
	err = rh.service.AwardReferralBonuses(referrer.TelegramID, user.TelegramID, referralCode)
	if err != nil {
		log.Printf("REFERRAL_HANDLER: Ошибка начисления бонусов: %v", err)
		return
	}

	// Отправляем уведомление приглашенному
	text := fmt.Sprintf("🎉 <b>Добро пожаловать!</b>\n\n")
	text += fmt.Sprintf("Вы зарегистрировались по реферальной ссылке от %s!\n", referrer.FirstName)
	text += fmt.Sprintf("🎁 На ваш баланс начислен приветственный бонус: <b>%.0f₽</b>\n\n", common.REFERRAL_WELCOME_BONUS)
	text += "Спасибо, что присоединились к нашему сервису!"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	rh.bot.Send(msg)

	// Отправляем уведомление пригласившему
	referrerText := fmt.Sprintf("🎉 <b>Новый реферал!</b>\n\n")
	referrerText += fmt.Sprintf("Пользователь %s зарегистрировался по вашей ссылке!\n", user.FirstName)
	referrerText += fmt.Sprintf("💰 Вам начислен бонус: <b>%.0f₽</b>\n\n", common.REFERRAL_BONUS_AMOUNT)
	referrerText += "Продолжайте приглашать друзей и зарабатывайте больше!"

	referrerMsg := tgbotapi.NewMessage(referrer.TelegramID, referrerText)
	referrerMsg.ParseMode = "HTML"
	rh.bot.Send(referrerMsg)

	log.Printf("REFERRAL_HANDLER: Успешно обработан реферальный переход %d -> %d", referrer.TelegramID, user.TelegramID)
}

// IsReferralCallback проверяет, является ли callback реферальным
func (rh *ReferralHandler) IsReferralCallback(data string) bool {
	referralCallbacks := []string{
		"ref_stats", "ref_history", "ref_share", "ref_menu",
	}

	for _, callback := range referralCallbacks {
		if data == callback {
			return true
		}
	}

	return false
}

// IsReferralCommand проверяет, является ли команда реферальной
func (rh *ReferralHandler) IsReferralCommand(command string) bool {
	return command == "ref"
}

// IsReferralStart проверяет, является ли команда /start с реферальным кодом
func (rh *ReferralHandler) IsReferralStart(text string) bool {
	return strings.HasPrefix(text, "/start ref_")
}

// ExtractReferralCode извлекает реферальный код из команды /start
func (rh *ReferralHandler) ExtractReferralCode(text string) string {
	parts := strings.Fields(text)
	if len(parts) >= 2 && parts[0] == "/start" && strings.HasPrefix(parts[1], "ref_") {
		return parts[1]
	}
	return ""
}
