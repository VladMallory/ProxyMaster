package common

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
)

// ReferralChecker структура для автоматической проверки реферальных кодов
type ReferralChecker struct {
	db                *sql.DB
	codeGenerator     ReferralCodeGeneratorInterface
	checkInterval     time.Duration
	isRunning         bool
	stopChan          chan bool
	lastCheckTime     time.Time
	totalUsers        int
	usersWithCodes    int
	usersWithoutCodes int
	createdCodes      int
}

// NewReferralChecker создает новый экземпляр проверяльщика реферальных кодов
func NewReferralChecker(db *sql.DB) *ReferralChecker {
	return &ReferralChecker{
		db:       db,
		stopChan: make(chan bool),
	}
}

// Start запускает автоматическую проверку реферальных кодов
func (rc *ReferralChecker) Start() error {
	if !REFERRAL_CHECK_ENABLED {
		log.Println("REFERRAL_CHECKER: Автоматическая проверка реферальных кодов отключена")
		return nil
	}

	if REFERRAL_CHECK_INTERVAL <= 0 {
		log.Println("REFERRAL_CHECKER: Интервал проверки не задан или равен 0")
		return nil
	}

	rc.checkInterval = time.Duration(REFERRAL_CHECK_INTERVAL) * time.Minute
	// Инициализируем генератор кодов
	rc.codeGenerator = &referralCodeGeneratorAdapter{db: rc.db}

	log.Printf("REFERRAL_CHECKER: Запуск автоматической проверки реферальных кодов каждые %d минут", REFERRAL_CHECK_INTERVAL)

	rc.isRunning = true
	go rc.run()

	return nil
}

// Stop останавливает автоматическую проверку
func (rc *ReferralChecker) Stop() {
	if rc.isRunning {
		log.Println("REFERRAL_CHECKER: Остановка автоматической проверки реферальных кодов")
		rc.isRunning = false
		rc.stopChan <- true
	}
}

// run основной цикл проверки
func (rc *ReferralChecker) run() {
	ticker := time.NewTicker(rc.checkInterval)
	defer ticker.Stop()

	// Выполняем первую проверку сразу при запуске
	rc.performCheck()

	for {
		select {
		case <-ticker.C:
			rc.performCheck()
		case <-rc.stopChan:
			log.Println("REFERRAL_CHECKER: Получен сигнал остановки")
			return
		}
	}
}

// performCheck выполняет проверку реферальных кодов
func (rc *ReferralChecker) performCheck() {
	log.Println("REFERRAL_CHECKER: ===== НАЧАЛО ПРОВЕРКИ РЕФЕРАЛЬНЫХ КОДОВ =====")
	startTime := time.Now()

	// Сбрасываем счетчики
	rc.totalUsers = 0
	rc.usersWithCodes = 0
	rc.usersWithoutCodes = 0
	rc.createdCodes = 0

	// Получаем всех пользователей
	query := `
		SELECT telegram_id, username, first_name, referral_code, balance, total_paid, created_at
		FROM users 
		ORDER BY telegram_id`

	rows, err := rc.db.Query(query)
	if err != nil {
		log.Printf("REFERRAL_CHECKER: ❌ Ошибка запроса пользователей: %v", err)
		return
	}
	defer rows.Close()

	// Список пользователей без кодов для создания кодов
	usersWithoutCodes := make([]UserInfo, 0)

	for rows.Next() {
		var telegramID int64
		var username, firstName string
		var referralCode sql.NullString
		var balance, totalPaid float64
		var createdAt string

		err := rows.Scan(&telegramID, &username, &firstName, &referralCode, &balance, &totalPaid, &createdAt)
		if err != nil {
			log.Printf("REFERRAL_CHECKER: ❌ Ошибка сканирования: %v", err)
			continue
		}

		rc.totalUsers++

		if !referralCode.Valid || referralCode.String == "" {
			rc.usersWithoutCodes++
			usersWithoutCodes = append(usersWithoutCodes, UserInfo{
				TelegramID: telegramID,
				Username:   username,
				FirstName:  firstName,
				Balance:    balance,
			})
		} else {
			rc.usersWithCodes++
		}
	}

	// Создаем реферальные коды для пользователей без кодов
	if len(usersWithoutCodes) > 0 {
		log.Printf("REFERRAL_CHECKER: 🔧 Создание реферальных кодов для %d пользователей без кодов...", len(usersWithoutCodes))

		for _, user := range usersWithoutCodes {
			// Создаем реферальный код используя генератор кодов
			referralCode, err := rc.codeGenerator.GenerateReferralCode(user.TelegramID)
			if err != nil {
				log.Printf("REFERRAL_CHECKER: ❌ Ошибка создания кода для %d (@%s): %v",
					user.TelegramID, user.Username, err)
			} else {
				log.Printf("REFERRAL_CHECKER: ✅ Создан код %s для @%s (%s)",
					referralCode, user.Username, user.FirstName)
				rc.createdCodes++
			}
		}
	}

	// Обновляем статистику
	rc.usersWithCodes = rc.totalUsers - rc.usersWithoutCodes + rc.createdCodes
	rc.usersWithoutCodes = rc.totalUsers - rc.usersWithCodes
	rc.lastCheckTime = time.Now()

	// Логируем результаты
	duration := time.Since(startTime)
	log.Printf("REFERRAL_CHECKER: ===== РЕЗУЛЬТАТЫ ПРОВЕРКИ =====")
	log.Printf("REFERRAL_CHECKER: 📊 Всего пользователей: %d", rc.totalUsers)
	log.Printf("REFERRAL_CHECKER: ✅ С реферальными кодами: %d", rc.usersWithCodes)
	log.Printf("REFERRAL_CHECKER: ❌ БЕЗ реферальных кодов: %d", rc.usersWithoutCodes)
	log.Printf("REFERRAL_CHECKER: 🔧 Создано новых кодов: %d", rc.createdCodes)
	log.Printf("REFERRAL_CHECKER: 📈 Процент покрытия: %.1f%%", float64(rc.usersWithCodes)/float64(rc.totalUsers)*100)
	log.Printf("REFERRAL_CHECKER: ⏱️ Время выполнения: %v", duration)
	log.Printf("REFERRAL_CHECKER: ===== ПРОВЕРКА ЗАВЕРШЕНА =====")

	// Отправляем уведомление администратору, если были созданы новые коды
	if rc.createdCodes > 0 && ADMIN_NOTIFICATIONS_ENABLED {
		rc.sendAdminNotification()
	}
}

// sendAdminNotification отправляет уведомление администратору
func (rc *ReferralChecker) sendAdminNotification() {
	if GlobalBot == nil || ADMIN_ID == 0 {
		return
	}

	message := fmt.Sprintf(
		"🔧 <b>Автоматическая проверка реферальных кодов</b>\n\n"+
			"📊 <b>Результаты:</b>\n"+
			"• Всего пользователей: %d\n"+
			"• С реферальными кодами: %d\n"+
			"• БЕЗ реферальных кодов: %d\n"+
			"• Создано новых кодов: %d\n"+
			"• Процент покрытия: %.1f%%\n\n"+
			"⏰ Время проверки: %s",
		rc.totalUsers,
		rc.usersWithCodes,
		rc.usersWithoutCodes,
		rc.createdCodes,
		float64(rc.usersWithCodes)/float64(rc.totalUsers)*100,
		rc.lastCheckTime.Format("2006-01-02 15:04:05"),
	)

	msg := tgbotapi.NewMessage(ADMIN_ID, message)
	msg.ParseMode = "HTML"

	_, err := GlobalBot.Send(msg)
	if err != nil {
		log.Printf("REFERRAL_CHECKER: ❌ Ошибка отправки уведомления админу: %v", err)
	}
}

// GetStats возвращает статистику последней проверки
func (rc *ReferralChecker) GetStats() ReferralCheckStats {
	return ReferralCheckStats{
		TotalUsers:        rc.totalUsers,
		UsersWithCodes:    rc.usersWithCodes,
		UsersWithoutCodes: rc.usersWithoutCodes,
		CreatedCodes:      rc.createdCodes,
		LastCheckTime:     rc.lastCheckTime,
		IsRunning:         rc.isRunning,
	}
}

// UserInfo информация о пользователе для создания реферального кода
type UserInfo struct {
	TelegramID int64
	Username   string
	FirstName  string
	Balance    float64
}

// ReferralCheckStats статистика проверки реферальных кодов
type ReferralCheckStats struct {
	TotalUsers        int       `json:"total_users"`
	UsersWithCodes    int       `json:"users_with_codes"`
	UsersWithoutCodes int       `json:"users_without_codes"`
	CreatedCodes      int       `json:"created_codes"`
	LastCheckTime     time.Time `json:"last_check_time"`
	IsRunning         bool      `json:"is_running"`
}

// Глобальный экземпляр проверяльщика реферальных кодов
var GlobalReferralChecker *ReferralChecker

// InitReferralChecker инициализирует глобальный проверяльщик реферальных кодов
func InitReferralChecker(db *sql.DB) error {
	GlobalReferralChecker = NewReferralChecker(db)
	return GlobalReferralChecker.Start()
}

// StopReferralChecker останавливает глобальный проверяльщик реферальных кодов
func StopReferralChecker() {
	if GlobalReferralChecker != nil {
		GlobalReferralChecker.Stop()
	}
}

// referralCodeGeneratorAdapter адаптер для генерации реферальных кодов
type referralCodeGeneratorAdapter struct {
	db *sql.DB
}

// GenerateReferralCode генерирует реферальный код (упрощенная версия без циклических импортов)
func (r *referralCodeGeneratorAdapter) GenerateReferralCode(telegramID int64) (string, error) {
	// Проверяем, есть ли уже код у пользователя
	var existingCode sql.NullString
	query := "SELECT referral_code FROM users WHERE telegram_id = $1"
	err := r.db.QueryRow(query, telegramID).Scan(&existingCode)

	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("ошибка проверки существующего кода: %v", err)
	}

	// Если код уже есть, возвращаем его
	if existingCode.Valid && existingCode.String != "" {
		return existingCode.String, nil
	}

	// Генерируем новый код
	code := fmt.Sprintf("%d%03d", telegramID, int(telegramID%1000))

	// Проверяем уникальность кода
	query = "SELECT EXISTS(SELECT 1 FROM users WHERE referral_code = $1)"
	var exists bool
	err = r.db.QueryRow(query, code).Scan(&exists)
	if err != nil {
		log.Printf("REFERRAL_CHECKER: Ошибка проверки уникальности кода: %v", err)
		return code, nil
	}

	// Если код уже существует, добавляем случайное число
	if exists {
		code = fmt.Sprintf("%d%03d%d", telegramID, int(telegramID%1000), int(telegramID%100))
	}

	// Сохраняем код в БД
	updateQuery := "UPDATE users SET referral_code = $1 WHERE telegram_id = $2"
	_, err = r.db.Exec(updateQuery, code, telegramID)
	if err != nil {
		return "", fmt.Errorf("ошибка сохранения реферального кода: %v", err)
	}

	log.Printf("REFERRAL_CHECKER: Сгенерирован реферальный код %s для пользователя %d", code, telegramID)
	return code, nil
}
