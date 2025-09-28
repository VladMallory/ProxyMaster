package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"bot/common"
)

func main() {
	log.Printf("=== ТЕСТ СИНХРОНИЗАЦИИ ПОЛЬЗОВАТЕЛЯ 873925520 ===")

	// Создаем тестового пользователя с активным основным конфигом
	testUser := &common.User{
		TelegramID:               873925520, // Реальный ID пользователя
		Username:                 "test_user_873925520",
		FirstName:                "Test User 873925520",
		Balance:                  100.0,
		HasActiveConfig:          true,
		ClientID:                 "test-main-client-id-873925520",
		Email:                    "873925520",
		SubID:                    "test-main-sub-id-873925520",
		ExpiryTime:               1761686400000, // Через 30 дней
		HasActiveSecondaryConfig: false,         // Пока нет дополнительного конфига
	}

	log.Printf("Тестовый пользователь:")
	log.Printf("  TelegramID: %d", testUser.TelegramID)
	log.Printf("  HasActiveConfig: %v", testUser.HasActiveConfig)
	log.Printf("  Email: %s", testUser.Email)
	log.Printf("  HasActiveSecondaryConfig: %v", testUser.HasActiveSecondaryConfig)
	log.Printf("  SecondaryEmail: %s", testUser.SecondaryEmail)

	// Проверяем HasAnyActiveConfig до синхронизации
	hasAnyConfigBefore := common.HasAnyActiveConfig(testUser)
	log.Printf("\n=== ДО СИНХРОНИЗАЦИИ ===")
	log.Printf("HasAnyActiveConfig: %v", hasAnyConfigBefore)

	// Выполняем синхронизацию с дополнительным инбаундом
	log.Printf("\n=== СИНХРОНИЗАЦИЯ С ДОПОЛНИТЕЛЬНЫМ ИНБАУНДОМ ===")

	// Используем версию без сохранения в БД для тестирования
	err := syncUserWithSecondaryPanelNoDB(testUser)
	if err != nil {
		log.Printf("❌ Ошибка синхронизации: %v", err)
		return
	}

	log.Printf("✅ Синхронизация завершена")

	// Проверяем результат
	log.Printf("\n=== ПОСЛЕ СИНХРОНИЗАЦИИ ===")
	log.Printf("HasActiveSecondaryConfig: %v", testUser.HasActiveSecondaryConfig)
	log.Printf("SecondaryEmail: %s", testUser.SecondaryEmail)
	log.Printf("SecondarySubID: %s", testUser.SecondarySubID)
	log.Printf("SecondaryClientID: %s", testUser.SecondaryClientID)

	// Проверяем HasAnyActiveConfig после синхронизации
	hasAnyConfigAfter := common.HasAnyActiveConfig(testUser)
	log.Printf("HasAnyActiveConfig: %v", hasAnyConfigAfter)

	// Проверяем URL конфигурации
	secondaryConfigURL := common.GetSecondaryConfigURL(testUser)
	log.Printf("SecondaryConfigURL: %s", secondaryConfigURL)

	// Проверяем, что дополнительный конфиг активен
	isSecondaryActive := common.IsSecondaryConfigActive(testUser)
	log.Printf("IsSecondaryConfigActive: %v", isSecondaryActive)

	log.Printf("\n=== РЕЗУЛЬТАТ ===")
	if testUser.HasActiveSecondaryConfig && testUser.SecondaryEmail == "873925520_1" {
		log.Printf("✅ ИНТЕГРАЦИЯ РАБОТАЕТ КОРРЕКТНО!")
		log.Printf("   - Основной конфиг: %s", testUser.Email)
		log.Printf("   - Дополнительный конфиг: %s", testUser.SecondaryEmail)
		log.Printf("   - HasAnyActiveConfig: %v", hasAnyConfigAfter)
	} else {
		log.Printf("❌ ИНТЕГРАЦИЯ НЕ РАБОТАЕТ!")
		log.Printf("   - HasActiveSecondaryConfig: %v", testUser.HasActiveSecondaryConfig)
		log.Printf("   - SecondaryEmail: %s", testUser.SecondaryEmail)
	}
}

// syncUserWithSecondaryPanelNoDB - версия SyncUserWithSecondaryPanel без сохранения в БД
func syncUserWithSecondaryPanelNoDB(user *common.User) error {
	if !common.SECONDARY_INBOUND_ENABLED {
		return fmt.Errorf("дополнительный инбаунд отключен")
	}

	log.Printf("SYNC_SECONDARY_PANEL_NO_DB: ===== НАЧАЛО СИНХРОНИЗАЦИИ С ДОПОЛНИТЕЛЬНЫМ ИНБАУНДОМ =====")
	log.Printf("SYNC_SECONDARY_PANEL_NO_DB: Пользователь: %d, HasActiveSecondaryConfig=%v, SecondaryClientID=%s, SecondarySubID=%s",
		user.TelegramID, user.HasActiveSecondaryConfig, user.SecondaryClientID, user.SecondarySubID)

	// Авторизуемся в панели
	sessionCookie, err := common.Login()
	if err != nil {
		log.Printf("SYNC_SECONDARY_PANEL_NO_DB: Ошибка авторизации для пользователя %d: %v", user.TelegramID, err)
		return fmt.Errorf("ошибка авторизации: %v", err)
	}

	// Получаем дополнительный inbound
	targetInbound, err := common.GetSecondaryInbound(sessionCookie)
	if err != nil {
		log.Printf("SYNC_SECONDARY_PANEL_NO_DB: Ошибка получения дополнительного inbound для пользователя %d: %v", user.TelegramID, err)
		return fmt.Errorf("ошибка получения дополнительного inbound: %v", err)
	}

	if targetInbound == nil {
		log.Printf("SYNC_SECONDARY_PANEL_NO_DB: Дополнительный inbound с ID %d не найден для пользователя %d", common.SECONDARY_INBOUND_ID, user.TelegramID)
		return fmt.Errorf("дополнительный inbound с ID %d не найден", common.SECONDARY_INBOUND_ID)
	}

	// Парсим settings
	var settings common.Settings
	if err := json.Unmarshal([]byte(targetInbound.Settings), &settings); err != nil {
		log.Printf("SYNC_SECONDARY_PANEL_NO_DB: Ошибка парсинга settings для пользователя %d: %v", user.TelegramID, err)
		return fmt.Errorf("ошибка парсинга settings: %v", err)
	}

	// Ищем клиента пользователя в дополнительном инбаунде
	expectedEmail := fmt.Sprintf("%d_1", user.TelegramID)
	existingClient := common.FindClientByEmail(settings.Clients, expectedEmail)

	// Если не найден клиент с суффиксом _1, ищем клиента без суффикса
	if existingClient == nil {
		legacyEmail := fmt.Sprintf("%d", user.TelegramID)
		existingClient = common.FindClientByEmail(settings.Clients, legacyEmail)
		if existingClient != nil {
			log.Printf("SYNC_SECONDARY_PANEL_NO_DB: Найден клиент без суффикса _1 для пользователя %d: Email=%s, переименовываем в %s",
				user.TelegramID, existingClient.Email, expectedEmail)
			// Переименовываем клиента, добавляя суффикс _1
			existingClient.Email = expectedEmail

			// Обновляем inbound с переименованным клиентом
			log.Printf("SYNC_SECONDARY_PANEL_NO_DB: Обновление inbound с переименованным клиентом для пользователя %d", user.TelegramID)

			// Обновляем settings в targetInbound
			settingsBytes, err := json.Marshal(settings)
			if err != nil {
				log.Printf("SYNC_SECONDARY_PANEL_NO_DB: ❌ Ошибка маршалинга settings для пользователя %d: %v", user.TelegramID, err)
				return fmt.Errorf("ошибка маршалинга settings: %v", err)
			}
			targetInbound.Settings = string(settingsBytes)

			if err := common.UpdateInbound(sessionCookie, *targetInbound); err != nil {
				log.Printf("SYNC_SECONDARY_PANEL_NO_DB: ❌ Ошибка обновления inbound с переименованным клиентом для пользователя %d: %v", user.TelegramID, err)
				return fmt.Errorf("ошибка обновления inbound с переименованным клиентом: %v", err)
			}
			log.Printf("SYNC_SECONDARY_PANEL_NO_DB: ✅ Inbound обновлен с переименованным клиентом для пользователя %d", user.TelegramID)
		}
	}

	if existingClient != nil {
		log.Printf("SYNC_SECONDARY_PANEL_NO_DB: ✅ Найден клиент в дополнительном инбаунде для пользователя %d: Email=%s, SubID=%s, Enable=%v, ExpiryTime=%d",
			user.TelegramID, existingClient.Email, existingClient.SubID, existingClient.Enable, existingClient.ExpiryTime)

		// Обновляем данные пользователя из дополнительного инбаунда
		user.SecondaryClientID = existingClient.ID
		user.SecondarySubID = existingClient.SubID
		user.SecondaryEmail = existingClient.Email
		user.SecondaryExpiryTime = existingClient.ExpiryTime
		user.HasActiveSecondaryConfig = existingClient.Enable && time.Now().UnixMilli() < existingClient.ExpiryTime

		log.Printf("SYNC_SECONDARY_PANEL_NO_DB: ✅ Пользователь %d успешно синхронизирован с дополнительным инбаундом, HasActiveSecondaryConfig=%v",
			user.TelegramID, user.HasActiveSecondaryConfig)
	} else {
		log.Printf("SYNC_SECONDARY_PANEL_NO_DB: ❌ Конфиг %s в дополнительном инбаунде для пользователя %d не найден", expectedEmail, user.TelegramID)
		log.Printf("SYNC_SECONDARY_PANEL_NO_DB: Доступные клиенты в дополнительном инбаунде:")
		for i, client := range settings.Clients {
			log.Printf("SYNC_SECONDARY_PANEL_NO_DB:   [%d] Email=%s, SubID=%s, Enable=%v", i, client.Email, client.SubID, client.Enable)
		}

		// Если у пользователя есть активный основной конфиг, создаем клиента в дополнительном инбаунде
		if user.HasActiveConfig {
			log.Printf("SYNC_SECONDARY_PANEL_NO_DB: У пользователя %d есть активный основной конфиг, создаем клиента в дополнительном инбаунде", user.TelegramID)

			// Определяем количество дней на основе основного конфига
			days := 30 // По умолчанию
			if user.ExpiryTime > 0 {
				// Если есть основной конфиг, используем его оставшееся время
				remainingDays := int((user.ExpiryTime - time.Now().UnixMilli()) / (24 * 60 * 60 * 1000))
				if remainingDays > 0 {
					days = remainingDays
				}
			}

			log.Printf("SYNC_SECONDARY_PANEL_NO_DB: Создание клиента в дополнительном инбаунде для пользователя %d, дней: %d", user.TelegramID, days)

			err = common.AddSecondaryClient(sessionCookie, user, days)
			if err != nil {
				log.Printf("SYNC_SECONDARY_PANEL_NO_DB: ❌ Ошибка создания клиента в дополнительном инбаунде для пользователя %d: %v", user.TelegramID, err)
				return err
			} else {
				log.Printf("SYNC_SECONDARY_PANEL_NO_DB: ✅ Клиент создан в дополнительном инбаунде для пользователя %d: Email=%s, SubID=%s",
					user.TelegramID, user.SecondaryEmail, user.SecondarySubID)
			}
		} else {
			// Сбрасываем данные дополнительного инбаунда, если нет основного конфига
			user.HasActiveSecondaryConfig = false
			user.SecondaryClientID = ""
			user.SecondarySubID = ""
			user.SecondaryEmail = ""
			user.SecondaryExpiryTime = 0
		}
	}

	log.Printf("SYNC_SECONDARY_PANEL_NO_DB: ===== КОНЕЦ СИНХРОНИЗАЦИИ С ДОПОЛНИТЕЛЬНЫМ ИНБАУНДОМ =====")
	return nil
}
