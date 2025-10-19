package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"bot/common"
	"bot/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Использование:")
		fmt.Println("  go run main.go stats                    - показать статистику напоминаний")
		fmt.Println("  go run main.go clean                    - очистить старые записи лога")
		fmt.Println("  go run main.go test <user_id>           - отправить тестовое напоминание")
		fmt.Println("  go run main.go check <user_id>          - проверить статус пользователя")
		fmt.Println("  go run main.go force <user_id>          - принудительно отправить напоминание")
		return
	}

	command := os.Args[1]

	// Инициализируем глобальные переменные
	common.InitGlobals()

	// Инициализируем базу данных
	err := common.InitPostgreSQL()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer common.DisconnectPostgreSQL()

	// Инициализируем бота для тестирования
	bot, err := tgbotapi.NewBotAPI(common.BOT_TOKEN)
	if err != nil {
		log.Fatalf("Ошибка инициализации бота: %v", err)
	}

	// Устанавливаем глобального бота для уведомлений администратору
	common.GlobalBot = bot

	// Создаем сервис напоминаний
	reminderService := services.NewUniversalReminderService(bot)

	switch command {
	case "stats":
		showStats(reminderService)
	case "clean":
		cleanOldLogs(reminderService)
	case "test":
		if len(os.Args) < 3 {
			fmt.Println("Ошибка: укажите user_id для тестового напоминания")
			return
		}
		userID, err := strconv.ParseInt(os.Args[2], 10, 64)
		if err != nil {
			fmt.Printf("Ошибка: неверный user_id: %v\n", err)
			return
		}
		sendTestReminder(reminderService, userID)
	case "check":
		if len(os.Args) < 3 {
			fmt.Println("Ошибка: укажите user_id для проверки")
			return
		}
		userID, err := strconv.ParseInt(os.Args[2], 10, 64)
		if err != nil {
			fmt.Printf("Ошибка: неверный user_id: %v\n", err)
			return
		}
		checkUserStatus(userID)
	case "force":
		if len(os.Args) < 3 {
			fmt.Println("Ошибка: укажите user_id для принудительной отправки")
			return
		}
		userID, err := strconv.ParseInt(os.Args[2], 10, 64)
		if err != nil {
			fmt.Printf("Ошибка: неверный user_id: %v\n", err)
			return
		}
		forceSendReminder(reminderService, userID)
	default:
		fmt.Printf("Неизвестная команда: %s\n", command)
	}
}

func showStats(reminderService *services.UniversalReminderService) {
	fmt.Println("=== Статистика напоминаний ===")

	count, err := reminderService.GetReminderStats()
	if err != nil {
		fmt.Printf("Ошибка получения статистики: %v\n", err)
		return
	}

	fmt.Printf("Всего отправлено напоминаний: %d\n", count)

	// Показываем последние 10 записей
	file, err := os.Open(common.UNIVERSAL_REMINDER_LOG_PATH)
	if err != nil {
		fmt.Printf("Лог файл не найден: %v\n", err)
		return
	}
	defer file.Close()

	var entries []services.ReminderLogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var entry services.ReminderLogEntry
		if err := json.Unmarshal([]byte(scanner.Text()), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	fmt.Printf("\nПоследние %d напоминаний:\n", min(10, len(entries)))
	for i := len(entries) - min(10, len(entries)); i < len(entries); i++ {
		entry := entries[i]
		fmt.Printf("  %s - Пользователь %d: %d дней %d часов (до %s)\n",
			entry.SentAt.Format("2006-01-02 15:04:05"),
			entry.UserID,
			entry.DaysLeft,
			entry.HoursLeft,
			time.UnixMilli(entry.ExpiryTime).Format("2006-01-02 15:04:05"))
	}
}

func cleanOldLogs(reminderService *services.UniversalReminderService) {
	fmt.Println("Очистка старых записей лога...")

	err := reminderService.CleanOldLogs()
	if err != nil {
		fmt.Printf("Ошибка очистки лога: %v\n", err)
		return
	}

	fmt.Println("Очистка завершена")
}

func sendTestReminder(reminderService *services.UniversalReminderService, userID int64) {
	fmt.Printf("Отправка тестового напоминания пользователю %d...\n", userID)

	// Создаем тестовое сообщение
	message := "🧪 <b>Тестовое напоминание</b>\n\n" +
		"До окончания вашей подписки осталось: <b>2 дней 5 часов</b>\n\n" +
		"Это тестовое сообщение для проверки системы напоминаний.\n\n" +
		"Нажмите /balance для просмотра баланса и продления."

	err := reminderService.SendReminder(userID, message)
	if err != nil {
		fmt.Printf("Ошибка отправки тестового напоминания: %v\n", err)
		return
	}

	fmt.Println("Тестовое напоминание отправлено успешно")
}

func checkUserStatus(userID int64) {
	fmt.Printf("Проверка статуса пользователя %d...\n", userID)

	user, err := common.GetUserByTelegramID(userID)
	if err != nil {
		fmt.Printf("Ошибка получения пользователя: %v\n", err)
		return
	}

	fmt.Printf("Пользователь: %s (ID: %d)\n", user.FirstName, user.TelegramID)
	fmt.Printf("Активный конфиг: %v\n", user.HasActiveConfig)
	fmt.Printf("Баланс: %.2f₽\n", user.Balance)

	if user.ExpiryTime > 0 {
		expiry := time.UnixMilli(user.ExpiryTime)
		now := time.Now()
		diff := expiry.Sub(now)

		days := int(diff.Hours() / 24)
		hours := int(diff.Hours()) % 24

		fmt.Printf("Время истечения: %s\n", expiry.Format("2006-01-02 15:04:05"))
		fmt.Printf("Осталось: %d дней %d часов\n", days, hours)

		// Проверяем, нужно ли отправить напоминание
		if days <= common.UNIVERSAL_REMINDER_DAYS_BEFORE && diff > 0 {
			fmt.Printf("✅ Пользователю нужно отправить напоминание\n")
		} else {
			fmt.Printf("❌ Пользователю не нужно напоминание\n")
		}
	} else {
		fmt.Printf("Время истечения не установлено\n")
	}
}

func forceSendReminder(reminderService *services.UniversalReminderService, userID int64) {
	fmt.Printf("Принудительная отправка напоминания пользователю %d...\n", userID)

	// Получаем информацию о пользователе
	user, err := common.GetUserByTelegramID(userID)
	if err != nil {
		fmt.Printf("Ошибка получения пользователя: %v\n", err)
		return
	}

	if !user.HasActiveConfig || user.ExpiryTime <= 0 {
		fmt.Printf("❌ У пользователя нет активной подписки или время истечения не установлено\n")
		return
	}

	// Вычисляем оставшееся время
	now := time.Now()
	expiry := time.UnixMilli(user.ExpiryTime)
	diff := expiry.Sub(now)

	if diff <= 0 {
		fmt.Printf("❌ Подписка уже истекла\n")
		return
	}

	daysLeft := int(diff.Hours() / 24)
	hoursLeft := int(diff.Hours()) % 24

	// Создаем сообщение напоминания с подстановкой времени
	message := common.UNIVERSAL_REMINDER_MESSAGE
	message = strings.ReplaceAll(message, "{DAYS}", strconv.Itoa(daysLeft))
	message = strings.ReplaceAll(message, "{HOURS}", strconv.Itoa(hoursLeft))

	// Отправляем напоминание
	err = reminderService.SendReminder(userID, message)
	if err != nil {
		fmt.Printf("❌ Ошибка отправки напоминания: %v\n", err)
		return
	}

	// Отправляем уведомление администратору
	common.SendReminderNotificationToAdmin(user, daysLeft, hoursLeft)

	// Записываем в лог (создаем временную запись для лога)
	reminderService.LogReminderSent(user, daysLeft, hoursLeft)

	fmt.Printf("✅ Напоминание успешно отправлено пользователю %d (осталось %d дней %d часов)\n",
		userID, daysLeft, hoursLeft)
	fmt.Printf("✅ Администратор получил уведомление о отправке\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
