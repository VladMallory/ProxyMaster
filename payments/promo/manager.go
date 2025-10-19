package promo

import (
	"fmt"
	"log"
	"strings"
	"time"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// PromoManager главный менеджер промокодов
type PromoManager struct {
	adminHandler *AdminPromoHandler
	userHandler  *UserPromoHandler
	service      *PromoService
}

// GlobalPromoManager глобальный экземпляр менеджера промокодов
var GlobalPromoManager *PromoManager

// InitializePromoManager инициализирует глобальный менеджер промокодов
func InitializePromoManager() error {
	log.Printf("PROMO_MANAGER: Инициализация менеджера промокодов")

	// Создаем сервис промокодов
	service, err := NewPromoService()
	if err != nil {
		return fmt.Errorf("ошибка создания сервиса промокодов: %v", err)
	}

	// Создаем обработчики
	adminHandler, err := NewAdminPromoHandler()
	if err != nil {
		return fmt.Errorf("ошибка создания обработчика админа: %v", err)
	}

	userHandler, err := NewUserPromoHandler()
	if err != nil {
		return fmt.Errorf("ошибка создания обработчика пользователей: %v", err)
	}

	// Создаем менеджер
	manager := &PromoManager{
		adminHandler: adminHandler,
		userHandler:  userHandler,
		service:      service,
	}

	// Устанавливаем глобальный экземпляр
	GlobalPromoManager = manager

	// Запускаем очистку истекших промокодов
	go manager.startCleanupRoutine()

	log.Printf("PROMO_MANAGER: Менеджер промокодов успешно инициализирован")
	return nil
}

// HandleCommand обрабатывает команды промокодов
func (pm *PromoManager) HandleCommand(chatID int64, userID int64, command string, args []string) error {
	switch command {
	case "promoset":
		return pm.adminHandler.HandlePromoSetCommand(chatID, userID)
	case "promo":
		return pm.userHandler.HandlePromoCommand(chatID, userID, args)
	case "promohistory":
		return pm.userHandler.HandlePromoHistoryCommand(chatID, userID)
	default:
		return fmt.Errorf("неизвестная команда промокодов: %s", command)
	}
}

// HandleCallback обрабатывает callback'и от inline клавиатур
func (pm *PromoManager) HandleCallback(chatID int64, userID int64, callbackData string) error {
	if strings.HasPrefix(callbackData, "promo_amount:") {
		return pm.adminHandler.HandlePromoSetCallback(chatID, userID, callbackData)
	}

	switch callbackData {
	case "promo_stats":
		return pm.adminHandler.HandlePromoStatsCallback(chatID, userID)
	case "create_promo":
		return pm.adminHandler.HandlePromoSetCommand(chatID, userID)
	case "promo_custom":
		return pm.adminHandler.HandleCustomAmountCallback(chatID, userID)
	}

	if strings.HasPrefix(callbackData, "copy_promo:") {
		return pm.adminHandler.HandleCopyPromoCallback(chatID, userID, callbackData)
	}

	return fmt.Errorf("неизвестный callback промокодов: %s", callbackData)
}

// IsPromoCommand проверяет, является ли команда командой промокодов
func (pm *PromoManager) IsPromoCommand(command string) bool {
	promoCommands := []string{"promoset", "promo", "promohistory"}

	for _, promoCmd := range promoCommands {
		if command == promoCmd {
			return true
		}
	}

	return false
}

// IsPromoCallback проверяет, является ли callback связанным с промокодами
func (pm *PromoManager) IsPromoCallback(callbackData string) bool {
	promoCallbacks := []string{
		"promo_amount:",
		"promo_stats",
		"create_promo",
		"promo_custom",
		"copy_promo:",
	}

	for _, promoCallback := range promoCallbacks {
		if strings.HasPrefix(callbackData, promoCallback) {
			return true
		}
	}

	return false
}

// GetPromoStats возвращает статистику промокодов для админа
func (pm *PromoManager) GetPromoStats(adminID int64) (map[string]interface{}, error) {
	return pm.service.GetPromoStats(adminID)
}

// CleanupExpiredPromos очищает истекшие промокоды
func (pm *PromoManager) CleanupExpiredPromos() error {
	return pm.service.CleanupExpiredPromos()
}

// startCleanupRoutine запускает фоновую задачу очистки истекших промокодов
func (pm *PromoManager) startCleanupRoutine() {
	log.Printf("PROMO_MANAGER: Запуск фоновой задачи очистки истекших промокодов")

	// Очищаем истекшие промокоды каждые 24 часа
	ticker := time.NewTicker(24 * time.Hour)

	for {
		select {
		case <-ticker.C:
			if err := pm.CleanupExpiredPromos(); err != nil {
				log.Printf("PROMO_MANAGER: Ошибка очистки истекших промокодов: %v", err)
			}
		}
	}
}

// SendPromoNotification отправляет уведомление о создании промокода (опционально)
func (pm *PromoManager) SendPromoNotification(chatID int64, promo *PromoCode) error {
	text := fmt.Sprintf("🎁 <b>Новый промокод!</b>\n\n"+
		"💰 <b>Сумма:</b> %.2f₽\n"+
		"⏰ <b>Действует до:</b> %s\n\n"+
		"Для активации используйте команду:\n"+
		"<code>/promo %s</code>",
		promo.Amount,
		promo.ExpiresAt.Format("02.01.2006 15:04"),
		promo.Code)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	_, err := common.GlobalBot.Send(msg)
	return err
}

// ValidatePromoCode проверяет валидность промокода без его использования
func (pm *PromoManager) ValidatePromoCode(code string, userID int64) (*PromoCode, PromoCodeStatus, error) {
	return pm.service.ValidatePromoCode(code, userID)
}

// GetService возвращает сервис промокодов (для внутреннего использования)
func (pm *PromoManager) GetService() *PromoService {
	return pm.service
}
