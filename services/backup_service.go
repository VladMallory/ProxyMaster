package services

import (
	"log"
	"time"

	"bot/common"
)

// StartPeriodicBackup запускает периодический бэкап
func StartPeriodicBackup() {
	log.Printf("START_PERIODIC_BACKUP: Запуск периодического бэкапа")
	ticker := time.NewTicker(1 * time.Hour) // Бэкап каждый час
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			log.Printf("START_PERIODIC_BACKUP: Выполнение периодического бэкапа")
			if err := common.BackupMongoDB(); err != nil {
				log.Printf("START_PERIODIC_BACKUP: Ошибка периодического бэкапа: %v", err)
			} else {
				log.Printf("START_PERIODIC_BACKUP: Периодический бэкап успешно создан")
			}
		}
	}()
	log.Println("START_PERIODIC_BACKUP: Запущен сервис периодического бэкапа (каждый час)")
}
