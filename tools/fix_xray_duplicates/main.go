package main

import (
	"database/sql"
	"encoding/json"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

const (
	DB_PATH = "/etc/x-ui/x-ui.db"
)

type Client struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	SubID      string `json:"subId"`
	Enable     bool   `json:"enable"`
	ExpiryTime int64  `json:"expiryTime"`
	Flow       string `json:"flow"`
	LimitIP    int    `json:"limitIp"`
	TotalGB    int    `json:"totalGB"`
	Reset      int    `json:"reset"`
	TgID       int    `json:"tgId"`
	CreatedAt  int64  `json:"created_at,omitempty"`
	UpdatedAt  int64  `json:"updated_at,omitempty"`
}

type Settings struct {
	Clients    []Client `json:"clients"`
	Decryption string   `json:"decryption"`
}

type Inbound struct {
	ID       int    `json:"id"`
	Settings string `json:"settings"`
}

func main() {
	log.Println("=== ПОИСК И УДАЛЕНИЕ ДУБЛИКАТОВ В X-UI ===")

	// Подключаемся к базе данных X-UI
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		log.Fatalf("Ошибка открытия БД: %v", err)
	}
	defer db.Close()

	// Получаем все inbound
	rows, err := db.Query("SELECT id, settings FROM inbounds WHERE id IN (2, 5)")
	if err != nil {
		log.Fatalf("Ошибка запроса inbound: %v", err)
	}
	defer rows.Close()

	totalDuplicates := 0
	totalFixed := 0

	for rows.Next() {
		var inbound Inbound
		if err := rows.Scan(&inbound.ID, &inbound.Settings); err != nil {
			log.Printf("Ошибка сканирования: %v", err)
			continue
		}

		log.Printf("\n========== INBOUND ID: %d ==========", inbound.ID)

		// Парсим settings
		var settings Settings
		if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
			log.Printf("Ошибка парсинга settings: %v", err)
			continue
		}

		log.Printf("Всего клиентов: %d", len(settings.Clients))

		// Ищем дубликаты
		emailMap := make(map[string][]int) // email -> индексы
		for i, client := range settings.Clients {
			baseEmail := client.Email
			emailMap[baseEmail] = append(emailMap[baseEmail], i)
		}

		// Находим дубликаты
		var duplicateFound bool
		newClients := []Client{}

		for email, indices := range emailMap {
			if len(indices) > 1 {
				duplicateFound = true
				totalDuplicates += len(indices) - 1
				log.Printf("🔴 Найден дубликат: %s (записей: %d)", email, len(indices))

				// Оставляем только первую запись
				for i, idx := range indices {
					client := settings.Clients[idx]
					if i == 0 {
						log.Printf("  ✓ Оставляем: %s (enable=%v, expiryTime=%d)", client.Email, client.Enable, client.ExpiryTime)
						newClients = append(newClients, client)
					} else {
						log.Printf("  ✗ Удаляем: %s (enable=%v, expiryTime=%d)", client.Email, client.Enable, client.ExpiryTime)
					}
				}
			} else {
				// Нет дубликатов - оставляем как есть
				newClients = append(newClients, settings.Clients[indices[0]])
			}
		}

		if duplicateFound {
			log.Printf("Обновление inbound %d: было %d клиентов, стало %d",
				inbound.ID, len(settings.Clients), len(newClients))

			// Обновляем settings
			settings.Clients = newClients
			newSettingsJSON, err := json.Marshal(settings)
			if err != nil {
				log.Printf("Ошибка сериализации settings: %v", err)
				continue
			}

			// Обновляем в базе данных
			_, err = db.Exec("UPDATE inbounds SET settings = ? WHERE id = ?", string(newSettingsJSON), inbound.ID)
			if err != nil {
				log.Printf("Ошибка обновления inbound: %v", err)
			} else {
				log.Printf("✓ Inbound %d обновлен", inbound.ID)
				totalFixed++
			}
		} else {
			log.Printf("✓ Дубликатов не найдено в inbound %d", inbound.ID)
		}
	}

	log.Printf("\n========== ИТОГИ ==========")
	log.Printf("Найдено дубликатов: %d", totalDuplicates)
	log.Printf("Исправлено inbound: %d", totalFixed)

	if totalFixed > 0 {
		log.Println("\n⚠ ВАЖНО: Перезапустите X-UI для применения изменений:")
		log.Println("   systemctl restart x-ui")
	} else {
		log.Println("\n✓ Дубликатов не обнаружено")
	}
}
