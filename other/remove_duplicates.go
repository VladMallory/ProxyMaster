//go:build tools
// +build tools

package main

import (
	"log"

	"bot/common"
)

func main() {
	log.Println("🧹 Запуск очистки дубликатов в панели 3x-ui")

	// Инициализируем базу данных
	if err := common.InitPostgreSQL(); err != nil {
		log.Fatalf("Ошибка инициализации базы данных: %v", err)
	}
	defer common.GetDB().Close()

	// Удаляем дубликаты
	if err := common.RemoveDuplicateClients(); err != nil {
		log.Fatalf("Ошибка удаления дубликатов: %v", err)
	}

	log.Println("✅ Дубликаты успешно удалены!")
}
