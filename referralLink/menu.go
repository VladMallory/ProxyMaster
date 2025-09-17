package referralLink

import (
	"fmt"
	"log"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ReferralMenu меню реферальной системы
type ReferralMenu struct {
	service *ReferralService
	bot     *tgbotapi.BotAPI
}

// NewReferralMenu создает новое меню реферальной системы
func NewReferralMenu(service *ReferralService, bot *tgbotapi.BotAPI) *ReferralMenu {
	return &ReferralMenu{
		service: service,
		bot:     bot,
	}
}

// SendReferralMenu отправляет главное меню реферальной системы
func (rm *ReferralMenu) SendReferralMenu(chatID int64, user *common.User) {
	log.Printf("REFERRAL_MENU: Отправка реферального меню для пользователя %d", user.TelegramID)

	// Получаем информацию о реферальной ссылке
	linkInfo, err := rm.service.GetReferralLinkInfo(user.TelegramID)
	if err != nil {
		log.Printf("REFERRAL_MENU: Ошибка получения реферальной ссылки: %v", err)
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка получения реферальной ссылки")
		rm.bot.Send(msg)
		return
	}

	// Получаем статистику
	stats, err := rm.service.GetReferralStats(user.TelegramID)
	if err != nil {
		log.Printf("REFERRAL_MENU: Ошибка получения статистики: %v", err)
		stats = &ReferralStats{}
	}

	// Формируем текст меню
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
			tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", "ref_refresh"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	if _, err := rm.bot.Send(msg); err != nil {
		log.Printf("REFERRAL_MENU: Ошибка отправки сообщения: %v", err)
	}
}

// EditReferralMenu редактирует реферальное меню
func (rm *ReferralMenu) EditReferralMenu(chatID int64, messageID int, user *common.User) {
	log.Printf("REFERRAL_MENU: Редактирование реферального меню для пользователя %d", user.TelegramID)

	// Получаем информацию о реферальной ссылке
	linkInfo, err := rm.service.GetReferralLinkInfo(user.TelegramID)
	if err != nil {
		log.Printf("REFERRAL_MENU: Ошибка получения реферальной ссылки: %v", err)
		return
	}

	// Получаем статистику
	stats, err := rm.service.GetReferralStats(user.TelegramID)
	if err != nil {
		log.Printf("REFERRAL_MENU: Ошибка получения статистики: %v", err)
		stats = &ReferralStats{}
	}

	// Формируем текст меню
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
			tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", "ref_refresh"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "main_menu"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ParseMode = "HTML"
	editMsg.ReplyMarkup = &keyboard

	if _, err := rm.bot.Send(editMsg); err != nil {
		log.Printf("REFERRAL_MENU: Ошибка редактирования сообщения: %v", err)
	}
}

// SendReferralStats отправляет статистику рефералов
func (rm *ReferralMenu) SendReferralStats(chatID int64, user *common.User) {
	stats, err := rm.service.GetReferralStats(user.TelegramID)
	if err != nil {
		log.Printf("REFERRAL_MENU: Ошибка получения статистики: %v", err)
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка получения статистики")
		rm.bot.Send(msg)
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

	rm.bot.Send(msg)
}

// SendReferralHistory отправляет историю реферальных бонусов
func (rm *ReferralMenu) SendReferralHistory(chatID int64, user *common.User) {
	bonuses, err := rm.service.GetReferralHistory(user.TelegramID, 10)
	if err != nil {
		log.Printf("REFERRAL_MENU: Ошибка получения истории: %v", err)
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка получения истории бонусов")
		rm.bot.Send(msg)
		return
	}

	text := "📋 <b>История реферальных бонусов</b>\n\n"

	if len(bonuses) == 0 {
		text += "📭 Пока нет бонусов\n"
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

	rm.bot.Send(msg)
}

// SendReferralShare отправляет информацию для поделиться ссылкой
func (rm *ReferralMenu) SendReferralShare(chatID int64, user *common.User) {
	linkInfo, err := rm.service.GetReferralLinkInfo(user.TelegramID)
	if err != nil {
		log.Printf("REFERRAL_MENU: Ошибка получения ссылки: %v", err)
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка получения реферальной ссылки")
		rm.bot.Send(msg)
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

	rm.bot.Send(msg)
}
