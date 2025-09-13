package telegram_bot

import (
	"log"

	"bot/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot представляет экземпляр Telegram бота
type Bot struct {
	API     *tgbotapi.BotAPI
	Updates tgbotapi.UpdatesChannel
}

// NewBot создает новый экземпляр бота
func NewBot(token string) (*Bot, error) {
	log.Printf("TELEGRAM_BOT: Инициализация Telegram бота с токеном %s", token)

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	bot.Debug = false
	log.Printf("TELEGRAM_BOT: Авторизован как @%s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	log.Printf("TELEGRAM_BOT: Запущен канал обновлений Telegram")

	return &Bot{
		API:     bot,
		Updates: updates,
	}, nil
}

// Start запускает основной цикл обработки обновлений
func (b *Bot) Start() {
	log.Printf("TELEGRAM_BOT: Запуск основного цикла обработки обновлений")

	for update := range b.Updates {
		if update.Message != nil {
			// Проверяем, есть ли успешный платеж
			if update.Message.SuccessfulPayment != nil {
				log.Printf("TELEGRAM_BOT: Получен успешный платеж от пользователя TelegramID=%d", update.Message.From.ID)
				handlers.HandleSuccessfulPayment(b.API, update.Message)
			} else {
				log.Printf("TELEGRAM_BOT: Получено сообщение от пользователя TelegramID=%d, текст='%s'", update.Message.From.ID, update.Message.Text)
				handlers.HandleMessage(b.API, update.Message)
			}
		}

		if update.CallbackQuery != nil {
			log.Printf("TELEGRAM_BOT: Получен callback от пользователя TelegramID=%d, данные='%s'", update.CallbackQuery.From.ID, update.CallbackQuery.Data)
			handlers.HandleCallback(b.API, update.CallbackQuery)
		}
	}
}

// SetBotCommands устанавливает команды бота в боковом меню
func SetBotCommands(bot *tgbotapi.BotAPI) error {
	log.Printf("TELEGRAM_BOT: Настройка команд бота")

	commands := []tgbotapi.BotCommand{
		{
			Command:     "start",
			Description: "🚀 Запустить бота и открыть главное меню",
		},
		{
			Command:     "balance",
			Description: "💰 Показать баланс и информацию о подписке",
		},
	}

	config := tgbotapi.NewSetMyCommands(commands...)
	_, err := bot.Request(config)
	if err != nil {
		log.Printf("TELEGRAM_BOT: Ошибка настройки команд: %v", err)
		return err
	}

	log.Printf("TELEGRAM_BOT: Команды бота успешно настроены")
	return nil
}
