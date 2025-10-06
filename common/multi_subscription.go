package common

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// GetAvailableServers получает список доступных серверов для мультиподписок
func GetAvailableServers() ([]Server, error) {
	log.Printf("GET_AVAILABLE_SERVERS: Получение доступных серверов")

	query := `
		SELECT id, name, country, country_code, flag, inbound_id, 
		       config_url, json_url, protocol, transport, enabled, priority
		FROM multi_servers 
		WHERE enabled = true 
		ORDER BY priority DESC, name`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("GET_AVAILABLE_SERVERS: Ошибка выполнения запроса: %v", err)
		return nil, fmt.Errorf("ошибка получения серверов: %v", err)
	}
	defer rows.Close()

	var servers []Server
	for rows.Next() {
		var server Server
		err := rows.Scan(
			&server.ID, &server.Name, &server.Country, &server.CountryCode,
			&server.Flag, &server.InboundID, &server.ConfigURL, &server.JSONURL,
			&server.Protocol, &server.Transport, &server.Enabled, &server.Priority,
		)
		if err != nil {
			log.Printf("GET_AVAILABLE_SERVERS: Ошибка сканирования строки: %v", err)
			continue
		}
		servers = append(servers, server)
	}

	if err = rows.Err(); err != nil {
		log.Printf("GET_AVAILABLE_SERVERS: Ошибка итерации по строкам: %v", err)
		return nil, fmt.Errorf("ошибка обработки результатов: %v", err)
	}

	log.Printf("GET_AVAILABLE_SERVERS: Получено %d серверов", len(servers))
	return servers, nil
}

// GetServersByIDs получает серверы по их ID
func GetServersByIDs(serverIDs []string) ([]Server, error) {
	log.Printf("GET_SERVERS_BY_IDS: Получение серверов по ID: %v", serverIDs)

	if len(serverIDs) == 0 {
		return []Server{}, nil
	}

	// Создаем плейсхолдеры для IN запроса
	placeholders := make([]string, len(serverIDs))
	args := make([]interface{}, len(serverIDs))
	for i, id := range serverIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, name, country, country_code, flag, inbound_id, 
		       config_url, json_url, protocol, transport, enabled, priority
		FROM multi_servers 
		WHERE id IN (%s)
		ORDER BY priority DESC, name`,
		fmt.Sprintf("%s", placeholders[0]))

	// Заменяем первый плейсхолдер на все остальные
	for i := 1; i < len(placeholders); i++ {
		query = fmt.Sprintf("%s, %s", query, placeholders[i])
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("GET_SERVERS_BY_IDS: Ошибка выполнения запроса: %v", err)
		return nil, fmt.Errorf("ошибка получения серверов: %v", err)
	}
	defer rows.Close()

	var servers []Server
	for rows.Next() {
		var server Server
		err := rows.Scan(
			&server.ID, &server.Name, &server.Country, &server.CountryCode,
			&server.Flag, &server.InboundID, &server.ConfigURL, &server.JSONURL,
			&server.Protocol, &server.Transport, &server.Enabled, &server.Priority,
		)
		if err != nil {
			log.Printf("GET_SERVERS_BY_IDS: Ошибка сканирования строки: %v", err)
			continue
		}
		servers = append(servers, server)
	}

	if err = rows.Err(); err != nil {
		log.Printf("GET_SERVERS_BY_IDS: Ошибка итерации по строкам: %v", err)
		return nil, fmt.Errorf("ошибка обработки результатов: %v", err)
	}

	log.Printf("GET_SERVERS_BY_IDS: Получено %d серверов", len(servers))
	return servers, nil
}

// SaveServerSelectionState сохраняет состояние выбора серверов
func SaveServerSelectionState(state *ServerSelectionState) error {
	log.Printf("SAVE_SERVER_SELECTION_STATE: Сохранение состояния для пользователя %d", state.UserID)

	// Удаляем старое состояние пользователя
	_, err := db.Exec("DELETE FROM server_selection_states WHERE user_id = $1", state.UserID)
	if err != nil {
		log.Printf("SAVE_SERVER_SELECTION_STATE: Ошибка удаления старого состояния: %v", err)
	}

	// Сериализуем выбранные серверы в JSON
	selectedJSON, err := json.Marshal(state.Selected)
	if err != nil {
		log.Printf("SAVE_SERVER_SELECTION_STATE: Ошибка сериализации выбранных серверов: %v", err)
		return fmt.Errorf("ошибка сериализации данных: %v", err)
	}

	// Вставляем новое состояние
	query := `
		INSERT INTO server_selection_states (user_id, selected_servers, max_servers, step, expires_at)
		VALUES ($1, $2, $3, $4, NOW() + INTERVAL '1 hour')`

	_, err = db.Exec(query, state.UserID, string(selectedJSON), state.MaxServers, state.Step)
	if err != nil {
		log.Printf("SAVE_SERVER_SELECTION_STATE: Ошибка вставки состояния: %v", err)
		return fmt.Errorf("ошибка сохранения состояния: %v", err)
	}

	log.Printf("SAVE_SERVER_SELECTION_STATE: Состояние успешно сохранено")
	return nil
}

// GetServerSelectionState получает состояние выбора серверов
func GetServerSelectionState(userID int64) (*ServerSelectionState, error) {
	log.Printf("GET_SERVER_SELECTION_STATE: Получение состояния для пользователя %d", userID)

	query := `
		SELECT user_id, selected_servers, max_servers, step
		FROM server_selection_states 
		WHERE user_id = $1 AND expires_at > NOW()`

	var state ServerSelectionState
	var selectedJSON string

	err := db.QueryRow(query, userID).Scan(
		&state.UserID, &selectedJSON, &state.MaxServers, &state.Step,
	)
	if err == sql.ErrNoRows {
		log.Printf("GET_SERVER_SELECTION_STATE: Состояние не найдено или истекло")
		return nil, nil
	}
	if err != nil {
		log.Printf("GET_SERVER_SELECTION_STATE: Ошибка получения состояния: %v", err)
		return nil, fmt.Errorf("ошибка получения состояния: %v", err)
	}

	// Десериализуем выбранные серверы
	err = json.Unmarshal([]byte(selectedJSON), &state.Selected)
	if err != nil {
		log.Printf("GET_SERVER_SELECTION_STATE: Ошибка десериализации выбранных серверов: %v", err)
		return nil, fmt.Errorf("ошибка десериализации данных: %v", err)
	}

	log.Printf("GET_SERVER_SELECTION_STATE: Состояние получено, выбранных серверов: %d", len(state.Selected))
	return &state, nil
}

// CreateMultiSubscription создает мультиподписку
func CreateMultiSubscription(userID int64, serverIDs []string) (*MultiSubscription, error) {
	log.Printf("CREATE_MULTI_SUBSCRIPTION: Создание мультиподписки для пользователя %d, серверов: %d", userID, len(serverIDs))

	// Получаем информацию о серверах
	servers, err := GetServersByIDs(serverIDs)
	if err != nil {
		log.Printf("CREATE_MULTI_SUBSCRIPTION: Ошибка получения серверов: %v", err)
		return nil, fmt.Errorf("ошибка получения серверов: %v", err)
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("не найдено серверов для создания мультиподписки")
	}

	// Генерируем уникальный ID для мультиподписки
	subscriptionID := "multi_" + uuid.New().String()

	// Создаем URL мультиподписки
	subscriptionURL := MULTI_SUBSCRIPTION_BASE_URL + subscriptionID

	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		log.Printf("CREATE_MULTI_SUBSCRIPTION: Ошибка начала транзакции: %v", err)
		return nil, fmt.Errorf("ошибка начала транзакции: %v", err)
	}
	defer tx.Rollback()

	// Создаем мультиподписку
	query := `
		INSERT INTO multi_subscriptions (id, user_id, subscription_url, is_active, expiry_time)
		VALUES ($1, $2, $3, $4, $5)`

	expiryTime := time.Now().AddDate(0, 0, 30).UnixMilli() // 30 дней по умолчанию

	_, err = tx.Exec(query, subscriptionID, userID, subscriptionURL, true, expiryTime)
	if err != nil {
		log.Printf("CREATE_MULTI_SUBSCRIPTION: Ошибка создания мультиподписки: %v", err)
		return nil, fmt.Errorf("ошибка создания мультиподписки: %v", err)
	}

	// Связываем серверы с мультиподпиской
	for _, serverID := range serverIDs {
		linkQuery := `
			INSERT INTO multi_subscription_servers (subscription_id, server_id)
			VALUES ($1, $2)`

		_, err = tx.Exec(linkQuery, subscriptionID, serverID)
		if err != nil {
			log.Printf("CREATE_MULTI_SUBSCRIPTION: Ошибка связывания сервера %s: %v", serverID, err)
			return nil, fmt.Errorf("ошибка связывания серверов: %v", err)
		}
	}

	// Удаляем состояние выбора серверов
	_, err = tx.Exec("DELETE FROM server_selection_states WHERE user_id = $1", userID)
	if err != nil {
		log.Printf("CREATE_MULTI_SUBSCRIPTION: Ошибка удаления состояния выбора: %v", err)
		// Не возвращаем ошибку, так как мультиподписка уже создана
	}

	// Подтверждаем транзакцию
	err = tx.Commit()
	if err != nil {
		log.Printf("CREATE_MULTI_SUBSCRIPTION: Ошибка подтверждения транзакции: %v", err)
		return nil, fmt.Errorf("ошибка подтверждения транзакции: %v", err)
	}

	// Создаем объект мультиподписки
	subscription := &MultiSubscription{
		ID:              subscriptionID,
		UserID:          userID,
		Servers:         servers,
		SubscriptionURL: subscriptionURL,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		ExpiryTime:      expiryTime,
	}

	log.Printf("CREATE_MULTI_SUBSCRIPTION: Мультиподписка успешно создана: %s", subscriptionID)
	return subscription, nil
}

// GetUserMultiSubscription получает мультиподписку пользователя
func GetUserMultiSubscription(userID int64) (*MultiSubscription, error) {
	log.Printf("GET_USER_MULTI_SUBSCRIPTION: Получение мультиподписки для пользователя %d", userID)

	query := `
		SELECT ms.id, ms.subscription_url, ms.is_active, ms.created_at, ms.expiry_time,
		       COALESCE(
		           jsonb_agg(
		               jsonb_build_object(
		                   'id', s.id,
		                   'name', s.name,
		                   'country', s.country,
		                   'country_code', s.country_code,
		                   'flag', s.flag,
		                   'inbound_id', s.inbound_id,
		                   'config_url', s.config_url,
		                   'json_url', s.json_url,
		                   'protocol', s.protocol,
		                   'transport', s.transport,
		                   'enabled', s.enabled,
		                   'priority', s.priority
		               ) ORDER BY s.priority DESC, s.name
		           ) FILTER (WHERE s.id IS NOT NULL),
		           '[]'::jsonb
		       ) as servers
		FROM multi_subscriptions ms
		LEFT JOIN multi_subscription_servers mss ON ms.id = mss.subscription_id
		LEFT JOIN multi_servers s ON mss.server_id = s.id
		WHERE ms.user_id = $1
		GROUP BY ms.id, ms.subscription_url, ms.is_active, ms.created_at, ms.expiry_time`

	var subscription MultiSubscription
	var serversJSON string

	err := db.QueryRow(query, userID).Scan(
		&subscription.ID, &subscription.SubscriptionURL, &subscription.IsActive,
		&subscription.CreatedAt, &subscription.ExpiryTime, &serversJSON,
	)
	if err == sql.ErrNoRows {
		log.Printf("GET_USER_MULTI_SUBSCRIPTION: Мультиподписка не найдена")
		return nil, nil
	}
	if err != nil {
		log.Printf("GET_USER_MULTI_SUBSCRIPTION: Ошибка получения мультиподписки: %v", err)
		return nil, fmt.Errorf("ошибка получения мультиподписки: %v", err)
	}

	// Десериализуем серверы
	err = json.Unmarshal([]byte(serversJSON), &subscription.Servers)
	if err != nil {
		log.Printf("GET_USER_MULTI_SUBSCRIPTION: Ошибка десериализации серверов: %v", err)
		return nil, fmt.Errorf("ошибка десериализации серверов: %v", err)
	}

	subscription.UserID = userID
	subscription.UpdatedAt = time.Now()

	log.Printf("GET_USER_MULTI_SUBSCRIPTION: Мультиподписка получена: %s, серверов: %d", subscription.ID, len(subscription.Servers))
	return &subscription, nil
}

// CleanupExpiredServerSelectionStates очищает истекшие состояния выбора серверов
func CleanupExpiredServerSelectionStates() error {
	log.Printf("CLEANUP_EXPIRED_SERVER_SELECTION_STATES: Очистка истекших состояний выбора серверов")

	query := "DELETE FROM server_selection_states WHERE expires_at < NOW()"
	result, err := db.Exec(query)
	if err != nil {
		log.Printf("CLEANUP_EXPIRED_SERVER_SELECTION_STATES: Ошибка очистки: %v", err)
		return fmt.Errorf("ошибка очистки состояний: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("CLEANUP_EXPIRED_SERVER_SELECTION_STATES: Удалено %d истекших состояний", rowsAffected)
	return nil
}
