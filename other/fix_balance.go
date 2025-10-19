//go:build tools
// +build tools

package main

import (
	"log"

	"bot/common"
)

func main() {
	log.Println("💰 Исправление баланса пользователя 873925520")

	// Инициализируем базу данных
	if err := common.InitPostgreSQL(); err != nil {
		log.Fatalf("Ошибка инициализации базы данных: %v", err)
	}
	defer common.GetDB().Close()

	// Получаем пользователя
	user, err := common.GetUserByTelegramID(873925520)
	if err != nil {
		log.Fatalf("Ошибка получения пользователя: %v", err)
	}

	log.Printf("Текущий баланс: %.2f₽", user.Balance)

	// Устанавливаем правильный баланс (50₽ за пробный период)
	err = common.SetBalance(873925520, 50.0)
	if err != nil {
		log.Fatalf("Ошибка установки баланса: %v", err)
	}

	log.Println("✅ Баланс исправлен на 50₽")
}
