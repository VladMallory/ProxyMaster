package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"bot/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func main() {
	log.Println("=== ТЕСТОВЫЙ СКРИПТ /start С СУФФИКСОМ _server2 ===")

	// Инициализируем базу данных
	if err := common.InitPostgreSQL(); err != nil {
		log.Fatalf("Ошибка инициализации PostgreSQL: %v", err)
	}
	defer func() {
		// Закрываем соединение с базой данных
		log.Println("Закрываем соединение с PostgreSQL")
	}()

	// Создаем бота
	bot, err := tgbotapi.NewBotAPI(common.BOT_TOKEN)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	bot.Debug = true
	log.Printf("Авторизован как %s", bot.Self.UserName)

	// Настраиваем обновления
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	log.Println("Ожидаем команду /start...")

	// Обрабатываем обновления
	for update := range updates {
		if update.Message != nil {
			// Обрабатываем команду /start
			if update.Message.IsCommand() && update.Message.Command() == "start" {
				telegramID := update.Message.From.ID
				username := update.Message.From.UserName
				if username == "" {
					username = update.Message.From.FirstName
				}

				log.Printf("Получена команда /start от пользователя %s (ID: %d)", username, telegramID)

				// Создаем или получаем пользователя
				user, err := common.GetOrCreateUser(telegramID, username, update.Message.From.FirstName, update.Message.From.LastName)
				if err != nil {
					log.Printf("Ошибка получения/создания пользователя: %v", err)
					sendMessage(bot, update.Message.Chat.ID, "❌ Ошибка обработки запроса. Попробуйте позже.")
					continue
				}

				log.Printf("Пользователь: %s (ID: %d), Баланс: %.2f₽", user.Username, user.TelegramID, user.Balance)

				// Создаем подписку с суффиксом _server2 и timestamp для уникальности
				email := fmt.Sprintf("%d_server2_%d", telegramID, time.Now().Unix())
				subID, err := createSubscriptionWithServer2Suffix(telegramID, email)
				if err != nil {
					log.Printf("Ошибка создания подписки: %v", err)
					sendMessage(bot, update.Message.Chat.ID, "❌ Ошибка создания подписки. Попробуйте позже.")
					continue
				}

				// Обновляем данные пользователя
				user.HasActiveConfig = true
				user.ClientID = subID
				user.Email = email
				user.SubID = subID
				user.ConfigCreatedAt = time.Now()
				user.ExpiryTime = time.Now().AddDate(0, 0, 30).UnixMilli() // 30 дней

				if err := common.UpdateUser(user); err != nil {
					log.Printf("Ошибка обновления пользователя: %v", err)
				}

				// Отправляем сообщение об успехе
				message := fmt.Sprintf("✅ Подписка создана успешно!\n\n📧 Email: %s\n🔑 SubID: %s\n⏰ Срок действия: 30 дней", email, subID)
				sendMessage(bot, update.Message.Chat.ID, message)

				log.Printf("✅ Подписка создана: Email=%s, SubID=%s", email, subID)
			}
		}
	}
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

func createSubscriptionWithServer2Suffix(telegramID int64, email string) (string, error) {
	log.Printf("🔧 Создание подписки с суффиксом _server2 для пользователя %d", telegramID)

	// Авторизуемся в панели
	sessionCookie, err := common.Login()
	if err != nil {
		return "", fmt.Errorf("ошибка авторизации: %v", err)
	}

	// Получаем inbound (используем ID=19)
	inbound, err := getInboundByID(sessionCookie, 19)
	if err != nil {
		return "", fmt.Errorf("ошибка получения inbound: %v", err)
	}

	// Парсим settings
	var settings common.Settings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return "", fmt.Errorf("ошибка парсинга settings: %v", err)
	}

	log.Printf("📋 Анализ текущего inbound:")
	log.Printf("   🆔 Inbound ID: %d", inbound.ID)
	log.Printf("   📡 Порт: %d", inbound.Port)
	log.Printf("   🔧 Протокол: %s", inbound.Protocol)
	log.Printf("   🏷️ Tag: %s", inbound.Tag)
	log.Printf("   🌐 StreamSettings: %s", inbound.StreamSettings)
	log.Printf("   📊 Количество клиентов: %d", len(settings.Clients))

	// Email теперь уникальный с timestamp, дубликатов быть не должно
	log.Printf("📧 Используем уникальный email: %s", email)

	// Создаем нового клиента
	clientUUID := uuid.New().String()
	subID := generateSubID()
	expiryTime := time.Now().AddDate(0, 0, 30).UnixMilli() // 30 дней
	falseValue := false

	newClient := common.Client{
		ID:         clientUUID,
		Flow:       "", // Пустой flow для VLESS XHTTP
		Email:      email,
		TotalGB:    0, // Безлимитный трафик
		ExpiryTime: expiryTime,
		Enable:     true,
		TgID:       0,
		SubID:      subID,
		Reset:      0,
		Depleted:   &falseValue,
		Exhausted:  &falseValue,
		CreatedAt:  time.Now().UnixMilli(),
		UpdatedAt:  time.Now().UnixMilli(),
	}

	// Добавляем клиента в список
	settings.Clients = append(settings.Clients, newClient)

	log.Printf("🔧 Создаем клиента с настройками:")
	log.Printf("   📧 Email: %s", email)
	log.Printf("   🔑 SubID: %s", subID)
	log.Printf("   🆔 ClientID: %s", clientUUID)
	log.Printf("   🌊 Flow: '%s' (пустой как указано)", newClient.Flow)
	log.Printf("   ⏰ ExpiryTime: %d (%s)", expiryTime, time.UnixMilli(expiryTime).Format("2006-01-02 15:04:05"))
	log.Printf("   ✅ Enable: %v", newClient.Enable)
	log.Printf("   📊 TotalGB: %d (0 = безлимит)", newClient.TotalGB)

	// Сериализуем обновленные settings
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации settings: %v", err)
	}
	inbound.Settings = string(settingsJSON)

	log.Printf("📤 Отправляем обновленные настройки в панель:")
	log.Printf("   📋 Количество клиентов: %d", len(settings.Clients))
	log.Printf("   🔧 JSON размер: %d байт", len(settingsJSON))
	log.Printf("   📡 Inbound ID: %d", inbound.ID)

	// Обновляем inbound
	if err := common.UpdateInbound(sessionCookie, *inbound); err != nil {
		return "", fmt.Errorf("ошибка обновления inbound: %v", err)
	}

	log.Printf("✅ Inbound успешно обновлен в панели")

	// Проверяем, что клиент создался
	updatedInbound, err := getInboundByID(sessionCookie, 19)
	if err != nil {
		log.Printf("⚠️ Не удалось получить обновленный inbound для проверки: %v", err)
	} else {
		var updatedSettings common.Settings
		if err := json.Unmarshal([]byte(updatedInbound.Settings), &updatedSettings); err != nil {
			log.Printf("⚠️ Ошибка парсинга обновленных settings: %v", err)
		} else {
			// Ищем нашего клиента
			found := false
			for _, client := range updatedSettings.Clients {
				if client.Email == email {
					log.Printf("✅ НАШ КЛИЕНТ найден в панели:")
					log.Printf("   📧 Email: %s", client.Email)
					log.Printf("   🔑 SubID: %s", client.SubID)
					log.Printf("   🌊 Flow: '%s'", client.Flow)
					log.Printf("   ✅ Enable: %v", client.Enable)
					log.Printf("   ⏰ ExpiryTime: %d (%s)", client.ExpiryTime, time.UnixMilli(client.ExpiryTime).Format("2006-01-02 15:04:05"))
					found = true
					break
				}
			}
			if !found {
				log.Printf("❌ Клиент %s не найден в панели после создания", email)
			}
		}
	}

	return subID, nil
}

func generateSubID() string {
	return uuid.New().String()
}

func getInboundByID(sessionCookie string, inboundID int) (*common.Inbound, error) {
	log.Printf("📡 Получение inbound ID=%d", inboundID)
	log.Printf("   🔗 URL: %s", common.PANEL_URL+"panel/api/inbounds/get/"+fmt.Sprintf("%d", inboundID))
	log.Printf("   🍪 Cookie: %s...", sessionCookie[:50])

	req, err := http.NewRequest("GET", common.PANEL_URL+"panel/api/inbounds/get/"+fmt.Sprintf("%d", inboundID), nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Cookie", sessionCookie)

	log.Printf("📡 Выполняем запрос получения inbound...")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	log.Printf("📥 Получен ответ от панели:")
	log.Printf("   📊 Статус код: %d", resp.StatusCode)
	log.Printf("   📄 Размер ответа: %d байт", len(body))
	log.Printf("   📝 Тело ответа: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("некорректный статус ответа: %d, body=%s", resp.StatusCode, string(body))
	}

	var inboundInfo common.InboundInfo
	if err := json.Unmarshal(body, &inboundInfo); err != nil {
		return nil, fmt.Errorf("ошибка десериализации ответа: %v, body=%s", err, string(body))
	}

	log.Printf("📋 Результат получения inbound:")
	log.Printf("   ✅ Success: %v", inboundInfo.Success)
	log.Printf("   💬 Message: %s", inboundInfo.Msg)

	if !inboundInfo.Success {
		return nil, fmt.Errorf("получение inbound не удалось: %s", inboundInfo.Msg)
	}

	log.Printf("✅ Успешно получен inbound: ID=%d", inboundInfo.Obj.ID)
	return &inboundInfo.Obj, nil
}
