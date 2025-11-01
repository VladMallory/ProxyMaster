package main

import (
	"bot/common"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	log.Println("--- ЗАПУСК ИНСТРУМЕНТА СБРОСА ЛОЖНЫХ СТАТУСОВ ---")

	// Инициализация
	common.InitMongoDB()
	defer common.DisconnectMongoDB()
	common.InitGlobals()

	// Получаем всех пользователей с активной подпиской из БД
	log.Println("1. Получение списка активных пользователей из базы данных...")
	users, err := common.GetUsersWithActiveConfigs()
	if err != nil {
		log.Fatalf("Ошибка получения пользователей: %v", err)
	}

	var usersToFix []common.User
	now := time.Now()
	for _, user := range users {
		if user.ExpiryTime > now.UnixMilli() {
			usersToFix = append(usersToFix, user)
		}
	}

	if len(usersToFix) == 0 {
		log.Println("Активных пользователей для сброса не найдено.")
		return
	}

	log.Printf("2. Найдено %d активных пользователей для принудительного сброса.", len(usersToFix))

	// Проверяем, передан ли аргумент --force для пропуска ожидания
	force := false
	if len(os.Args) > 1 && os.Args[1] == "--force" {
		force = true
	}

	if !force {
		log.Println("ВНИМАНИЕ! Этот инструмент вызовет кратковременное прерывание соединения у ВСЕХ активных пользователей.")
		log.Println("Для отмены закройте программу (Ctrl+C) в течение 10 секунд.")

		for i := 10; i > 0; i-- {
			fmt.Printf("\rПродолжение через %d секунд...", i)
			time.Sleep(1 * time.Second)
		}
		fmt.Println("\rПродолжение через 0 секунд...")
	} else {
		log.Println("Аргумент --force указан, ожидание пропущено.")
	}

	// Авторизуемся в панели один раз
	log.Println("3. Авторизация в панели управления...")
	sessionCookie, err := common.Login()
	if err != nil {
		log.Fatalf("Ошибка авторизации в панели: %v", err)
	}
	log.Println("Авторизация прошла успешно.")

	// Запускаем сброс
	log.Println("4. Начало процесса принудительного сброса...")
	fixedCount := 0
	for i, user := range usersToFix {
		log.Printf(" -> [%d/%d] Сброс для пользователя ID: %d...", i+1, len(usersToFix), user.TelegramID)
		if err := common.ForceResetDepletedStatus(sessionCookie, user.TelegramID); err != nil {
			log.Printf("  [!] ОШИБКА при сбросе для пользователя %d: %v", user.TelegramID, err)
		} else {
			log.Printf("  [✓] Успешно сброшено для пользователя %d.", user.TelegramID)
			fixedCount++
		}
		// Небольшая пауза между запросами, чтобы не перегружать API панели
		time.Sleep(500 * time.Millisecond)
	}

	log.Println("--- ЗАВЕРШЕНИЕ РАБОТЫ ---")
	log.Printf("Всего было обработано: %d", len(usersToFix))
	log.Printf("Успешно сброшено: %d", fixedCount)
	log.Printf("С ошибками: %d", len(usersToFix)-fixedCount)
	log.Println("---------------------------")
}
