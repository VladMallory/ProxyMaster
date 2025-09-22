package telegram_bot

import (
	"fmt"
	"log"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SupportConfig конфигурация поддержки
type SupportConfig struct {
	SupportText   string
	SupportURL    string
	SupportButton string
}

// DefaultSupportConfig возвращает конфигурацию поддержки по умолчанию
func DefaultSupportConfig() SupportConfig {
	return SupportConfig{
		SupportText:   "Чтобы обратиться в поддержку, нажмите на ->",
		SupportURL:    common.SUPPORT_LINK,
		SupportButton: "🆘 Поддержка",
	}
}

// AddSupportToMessage добавляет кнопку поддержки к сообщению
func AddSupportToMessage(msg tgbotapi.MessageConfig, config SupportConfig) tgbotapi.MessageConfig {
	if msg.ReplyMarkup == nil {
		// Если нет клавиатуры, создаем новую
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL(config.SupportButton, config.SupportURL),
			),
		)
		msg.ReplyMarkup = &keyboard
	} else {
		// Если есть клавиатура, добавляем кнопку поддержки
		if msg.ReplyMarkup != nil {
			// Добавляем новую строку с кнопкой поддержки
			newRow := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL(config.SupportButton, config.SupportURL),
			)
			if keyboard, ok := msg.ReplyMarkup.(*tgbotapi.InlineKeyboardMarkup); ok {
				keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, newRow)
			}
		}
	}
	return msg
}

// AddSupportToEditMessage добавляет кнопку поддержки к редактируемому сообщению
func AddSupportToEditMessage(msg tgbotapi.EditMessageTextConfig, config SupportConfig) tgbotapi.EditMessageTextConfig {
	if msg.ReplyMarkup == nil {
		// Если нет клавиатуры, создаем новую
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL(config.SupportButton, config.SupportURL),
			),
		)
		msg.ReplyMarkup = &keyboard
	} else {
		// Если есть клавиатура, добавляем кнопку поддержки
		if msg.ReplyMarkup != nil {
			// Добавляем новую строку с кнопкой поддержки
			newRow := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL(config.SupportButton, config.SupportURL),
			)
			msg.ReplyMarkup.InlineKeyboard = append(msg.ReplyMarkup.InlineKeyboard, newRow)
		}
	}
	return msg
}

// SendSupportMessage отправляет сообщение с поддержкой
func SendSupportMessage(bot *tgbotapi.BotAPI, chatID int64, text string, config SupportConfig) error {
	log.Printf("SEND_SUPPORT_MESSAGE: Отправка сообщения с поддержкой для ChatID=%d", chatID)

	// Добавляем текст поддержки к основному тексту
	fullText := fmt.Sprintf("%s\n\n%s", text, config.SupportText)

	msg := tgbotapi.NewMessage(chatID, fullText)
	msg = AddSupportToMessage(msg, config)

	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("SEND_SUPPORT_MESSAGE: Ошибка отправки сообщения: %v", err)
		return err
	}

	log.Printf("SEND_SUPPORT_MESSAGE: Сообщение с поддержкой успешно отправлено")
	return nil
}

// EditSupportMessage редактирует сообщение с поддержкой
func EditSupportMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int, text string, config SupportConfig) error {
	log.Printf("EDIT_SUPPORT_MESSAGE: Редактирование сообщения с поддержкой для ChatID=%d, MessageID=%d", chatID, messageID)

	// Добавляем текст поддержки к основному тексту
	fullText := fmt.Sprintf("%s\n\n%s", text, config.SupportText)

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, fullText)
	editMsg = AddSupportToEditMessage(editMsg, config)

	_, err := bot.Send(editMsg)
	if err != nil {
		log.Printf("EDIT_SUPPORT_MESSAGE: Ошибка редактирования сообщения: %v", err)
		return err
	}

	log.Printf("EDIT_SUPPORT_MESSAGE: Сообщение с поддержкой успешно отредактировано")
	return nil
}

// CreateSupportKeyboard создает клавиатуру только с кнопкой поддержки
func CreateSupportKeyboard(config SupportConfig) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(config.SupportButton, config.SupportURL),
		),
	)
}

// AddSupportToExistingKeyboard добавляет кнопку поддержки к существующей клавиатуре
func AddSupportToExistingKeyboard(keyboard *tgbotapi.InlineKeyboardMarkup, config SupportConfig) *tgbotapi.InlineKeyboardMarkup {
	if keyboard == nil {
		newKeyboard := CreateSupportKeyboard(config)
		return &newKeyboard
	}

	// Добавляем новую строку с кнопкой поддержки
	newRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonURL(config.SupportButton, config.SupportURL),
	)
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, newRow)
	return keyboard
}
