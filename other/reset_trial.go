package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"bot/common"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Использование: go run reset_trial.go <telegram_id>")
		os.Exit(1)
	}

	telegramID, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		log.Fatalf("Ошибка парсинга Telegram ID: %v", err)
	}

	// Инициализируем базу данных
	if err := common.InitPostgreSQL(); err != nil {
		log.Fatalf("Ошибка инициализации базы данных: %v", err)
	}
	defer common.GetDB().Close()

	// Сбрасываем флаг пробного периода
	if err := common.ResetTrialFlag(telegramID); err != nil {
		log.Fatalf("Ошибка сброса флага пробного периода: %v", err)
	}

	fmt.Printf("Флаг пробного периода сброшен для пользователя %d\n", telegramID)
}
