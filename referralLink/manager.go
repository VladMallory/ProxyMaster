package referralLink

import (
	"database/sql"
	"fmt"
	"log"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ReferralManager глобальный менеджер реферальной системы
type ReferralManager struct {
	service *ReferralService
	handler *ReferralHandler
	menu    *ReferralMenu
	bot     *tgbotapi.BotAPI
}

// GlobalReferralManager глобальный экземпляр менеджера рефералов
var GlobalReferralManager *ReferralManager

// InitReferralSystem инициализирует реферальную систему
func InitReferralSystem(db *sql.DB, bot *tgbotapi.BotAPI) error {
	log.Printf("REFERRAL_MANAGER: Инициализация реферальной системы")

	// Проверяем, включена ли реферальная система
	if !common.REFERRAL_SYSTEM_ENABLED {
		log.Printf("REFERRAL_MANAGER: Реферальная система отключена в конфигурации")
		return nil
	}

	// Создаем сервис
	service := NewReferralService(db)

	// Создаем обработчик
	handler := NewReferralHandler(service, bot)

	// Создаем меню
	menu := NewReferralMenu(service, bot)

	// Создаем менеджер
	GlobalReferralManager = &ReferralManager{
		service: service,
		handler: handler,
		menu:    menu,
		bot:     bot,
	}

	// Сохраняем ссылку в глобальной переменной common
	common.GlobalReferralManager = GlobalReferralManager

	log.Printf("REFERRAL_MANAGER: Реферальная система успешно инициализирована")
	return nil
}

// HandleCommand обрабатывает команды реферальной системы
func (rm *ReferralManager) HandleCommand(chatID int64, user *common.User, command string) {
	if !common.REFERRAL_SYSTEM_ENABLED {
		return
	}

	if rm.handler.IsReferralCommand(command) {
		rm.handler.HandleRefCommand(chatID, user)
	}
}

// HandleCallback обрабатывает callback'и реферальной системы
func (rm *ReferralManager) HandleCallback(chatID int64, userID int64, data string) {
	log.Printf("REFERRAL_MANAGER: ===== ОБРАБОТКА CALLBACK =====")
	log.Printf("REFERRAL_MANAGER: ChatID=%d, UserID=%d, Data='%s'", chatID, userID, data)

	if !common.REFERRAL_SYSTEM_ENABLED {
		log.Printf("REFERRAL_MANAGER: ❌ Реферальная система отключена в конфигурации")
		return
	}
	log.Printf("REFERRAL_MANAGER: ✅ Реферальная система включена")

	if rm.handler.IsReferralCallback(data) {
		log.Printf("REFERRAL_MANAGER: ✅ Callback '%s' является реферальным, передаем обработчику", data)
		rm.handler.HandleRefCallback(chatID, userID, data)
	} else {
		log.Printf("REFERRAL_MANAGER: ❌ Callback '%s' не является реферальным", data)
	}
}

// HandleStartCommand обрабатывает команду /start с возможным реферальным кодом
func (rm *ReferralManager) HandleStartCommand(chatID int64, user *common.User, text string) {
	if !common.REFERRAL_SYSTEM_ENABLED {
		return
	}

	if rm.handler.IsReferralStart(text) {
		referralCode := rm.handler.ExtractReferralCode(text)
		if referralCode != "" {
			rm.handler.ProcessReferralStart(chatID, user, referralCode)
		}
	}
}

// IsReferralCommand проверяет, является ли команда реферальной
func (rm *ReferralManager) IsReferralCommand(command string) bool {
	if !common.REFERRAL_SYSTEM_ENABLED {
		return false
	}
	return rm.handler.IsReferralCommand(command)
}

// IsReferralCallback проверяет, является ли callback реферальным
func (rm *ReferralManager) IsReferralCallback(data string) bool {
	if !common.REFERRAL_SYSTEM_ENABLED {
		return false
	}
	return rm.handler.IsReferralCallback(data)
}

// IsReferralStart проверяет, является ли команда /start с реферальным кодом
func (rm *ReferralManager) IsReferralStart(text string) bool {
	if !common.REFERRAL_SYSTEM_ENABLED {
		return false
	}
	return rm.handler.IsReferralStart(text)
}

// ExtractReferralCode извлекает реферальный код из команды /start
func (rm *ReferralManager) ExtractReferralCode(text string) string {
	log.Printf("REFERRAL_MANAGER: Извлечение реферального кода из текста: '%s'", text)
	code := rm.handler.ExtractReferralCode(text)
	log.Printf("REFERRAL_MANAGER: Извлеченный код: '%s'", code)
	return code
}

// SendReferralMenu отправляет реферальное меню
func (rm *ReferralManager) SendReferralMenu(chatID int64, user *common.User) {
	log.Printf("REFERRAL_MANAGER: ===== ОТПРАВКА РЕФЕРАЛЬНОГО МЕНЮ =====")
	log.Printf("REFERRAL_MANAGER: ChatID=%d, UserID=%d", chatID, user.TelegramID)

	if !common.REFERRAL_SYSTEM_ENABLED {
		log.Printf("REFERRAL_MANAGER: ❌ Реферальная система отключена")
		return
	}

	if rm.menu != nil {
		log.Printf("REFERRAL_MANAGER: ✅ Отправка меню через ReferralMenu")
		rm.menu.SendReferralMenu(chatID, user)
	} else {
		log.Printf("REFERRAL_MANAGER: ❌ ReferralMenu не инициализирован")
	}
}

// ProcessReferralTransition обрабатывает реферальный переход
func (rm *ReferralManager) ProcessReferralTransition(referrerID, referredID int64, referralCode string) error {
	log.Printf("REFERRAL_MANAGER: ===== ОБРАБОТКА РЕФЕРАЛЬНОГО ПЕРЕХОДА =====")
	log.Printf("REFERRAL_MANAGER: ReferrerID=%d, ReferredID=%d, Code='%s'", referrerID, referredID, referralCode)

	if !common.REFERRAL_SYSTEM_ENABLED {
		log.Printf("REFERRAL_MANAGER: ❌ Реферальная система отключена")
		return fmt.Errorf("реферальная система отключена")
	}

	if rm.service != nil {
		log.Printf("REFERRAL_MANAGER: ✅ Обработка перехода через ReferralService")
		err := rm.service.ProcessReferralTransition(referrerID, referredID, referralCode)
		if err != nil {
			log.Printf("REFERRAL_MANAGER: ❌ Ошибка обработки перехода: %v", err)
		} else {
			log.Printf("REFERRAL_MANAGER: ✅ Переход успешно обработан")
		}
		return err
	} else {
		log.Printf("REFERRAL_MANAGER: ❌ ReferralService не инициализирован")
		return fmt.Errorf("сервис рефералов не инициализирован")
	}
}

// AwardReferralBonuses начисляет реферальные бонусы
func (rm *ReferralManager) AwardReferralBonuses(referrerID, referredID int64, referralCode string) error {
	log.Printf("REFERRAL_MANAGER: ===== НАЧИСЛЕНИЕ РЕФЕРАЛЬНЫХ БОНУСОВ =====")
	log.Printf("REFERRAL_MANAGER: ReferrerID=%d, ReferredID=%d, Code='%s'", referrerID, referredID, referralCode)

	if !common.REFERRAL_SYSTEM_ENABLED {
		log.Printf("REFERRAL_MANAGER: ❌ Реферальная система отключена")
		return fmt.Errorf("реферальная система отключена")
	}

	if rm.service != nil {
		log.Printf("REFERRAL_MANAGER: ✅ Начисление бонусов через ReferralService")
		err := rm.service.AwardReferralBonuses(referrerID, referredID, referralCode)
		if err != nil {
			log.Printf("REFERRAL_MANAGER: ❌ Ошибка начисления бонусов: %v", err)
		} else {
			log.Printf("REFERRAL_MANAGER: ✅ Бонусы успешно начислены")
		}
		return err
	} else {
		log.Printf("REFERRAL_MANAGER: ❌ ReferralService не инициализирован")
		return fmt.Errorf("сервис рефералов не инициализирован")
	}
}

// GetReferralLinkInfo получает информацию о реферальной ссылке пользователя
func (rm *ReferralManager) GetReferralLinkInfo(telegramID int64) (*ReferralLinkInfo, error) {
	if !common.REFERRAL_SYSTEM_ENABLED {
		return nil, nil
	}
	return rm.service.GetReferralLinkInfo(telegramID)
}

// GetReferralStats получает статистику рефералов пользователя
func (rm *ReferralManager) GetReferralStats(telegramID int64) (*ReferralStats, error) {
	if !common.REFERRAL_SYSTEM_ENABLED {
		return &ReferralStats{}, nil
	}
	return rm.service.GetReferralStats(telegramID)
}

// EditReferralMenu редактирует реферальное меню
func (rm *ReferralManager) EditReferralMenu(chatID int64, messageID int, user *common.User) {
	if !common.REFERRAL_SYSTEM_ENABLED {
		return
	}
	rm.menu.EditReferralMenu(chatID, messageID, user)
}
