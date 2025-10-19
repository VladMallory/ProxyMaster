//go:build tools
// +build tools

package main

import (
	"log"

	"bot/common"
)

func main() {
	log.Println("🔄 Сброс флага пробного периода для пользователя 873925520")

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

	log.Printf("Текущий статус: HasUsedTrial = %v", user.HasUsedTrial)

	// Сбрасываем флаг пробного периода
	err = common.UpdateTrialFlag(873925520)
	if err != nil {
		log.Fatalf("Ошибка сброса флага пробного периода: %v", err)
	}

	log.Println("✅ Флаг пробного периода сброшен")
}
