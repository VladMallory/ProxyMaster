package services

import (
	"log"
	"time"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// DuplicateCleanupService сервис для автоматической очистки дубликатов в панели 3x-ui
type DuplicateCleanupService struct {
	bot     *tgbotapi.BotAPI
	ticker  *time.Ticker
	stopCh  chan bool
	running bool
}

// NewDuplicateCleanupService создает новый сервис очистки дубликатов
func NewDuplicateCleanupService(bot *tgbotapi.BotAPI) *DuplicateCleanupService {
	return &DuplicateCleanupService{
		bot:     bot,
		stopCh:  make(chan bool),
		running: false,
	}
}

// Start запускает сервис очистки дубликатов
func (dcs *DuplicateCleanupService) Start() {
	if dcs.running {
		log.Printf("DUPLICATE_CLEANUP: Сервис уже запущен")
		return
	}

	dcs.running = true
	dcs.ticker = time.NewTicker(time.Duration(common.DUPLICATE_CLEANUP_INTERVAL) * time.Minute)

	log.Printf("DUPLICATE_CLEANUP: Запуск сервиса очистки дубликатов (интервал: %d минут)", common.DUPLICATE_CLEANUP_INTERVAL)

	// Запускаем первую очистку сразу
	go dcs.runCleanup()

	// Запускаем периодическую очистку
	go func() {
		for {
			select {
			case <-dcs.ticker.C:
				dcs.runCleanup()
			case <-dcs.stopCh:
				log.Printf("DUPLICATE_CLEANUP: Получен сигнал остановки")
				return
			}
		}
	}()
}

// Stop останавливает сервис очистки дубликатов
func (dcs *DuplicateCleanupService) Stop() {
	if !dcs.running {
		return
	}

	log.Printf("DUPLICATE_CLEANUP: Остановка сервиса очистки дубликатов")
	dcs.running = false

	if dcs.ticker != nil {
		dcs.ticker.Stop()
	}

	select {
	case dcs.stopCh <- true:
	default:
	}
}

// runCleanup выполняет очистку дубликатов
func (dcs *DuplicateCleanupService) runCleanup() {
	log.Printf("DUPLICATE_CLEANUP: Начало очистки дубликатов")

	// Выполняем очистку дубликатов
	if err := common.RemoveDuplicateClients(); err != nil {
		log.Printf("DUPLICATE_CLEANUP: Ошибка очистки дубликатов: %v", err)

		// Отправляем уведомление администратору об ошибке
		if dcs.bot != nil && common.ADMIN_ID != 0 {
			msg := tgbotapi.NewMessage(common.ADMIN_ID,
				"❌ <b>Ошибка очистки дубликатов</b>\n\n"+
					"Произошла ошибка при автоматической очистке дубликатов в панели 3x-ui.\n\n"+
					"<code>"+err.Error()+"</code>")
			msg.ParseMode = "HTML"
			dcs.bot.Send(msg)
		}
	} else {
		log.Printf("DUPLICATE_CLEANUP: Очистка дубликатов завершена успешно")
	}
}
