# Настройка мультиподписок

## Обзор

Система мультиподписок позволяет пользователям создавать подписки, включающие несколько VPN серверов одновременно. При создании мультиподписки автоматически создаются клиенты в указанном инбаунде X-UI панели.

## Настройки в config.go

### Основные настройки мультиподписок

```go
// ========== МУЛЬТИПОДПИСКИ ==========
MULTI_SUBSCRIPTION_ENABLED = true                                    // Включены ли мультиподписки
MULTI_SUBSCRIPTION_MAX_SERVERS = 5                                   // Максимальное количество серверов
MULTI_SUBSCRIPTION_BASE_URL = "https://im.shadowfade.ru:8443/multi/" // Базовый URL для ссылок
MULTI_SUBSCRIPTION_CLEANUP_INTERVAL = 60                             // Интервал очистки состояний (мин)
```

### Настройки серверов мультиподписок

```go
// ========== НАСТРОЙКИ МУЛЬТИПОДПИСОК СЕРВЕРОВ ==========
MULTI_SERVER_INBOUND_ID = 19                                         // ID инбаунда для мультиподписок
MULTI_SERVER_AUTO_CREATE_CLIENTS = true                              // Автоматически создавать клиентов
MULTI_SERVER_CHECK_EXISTING = true                                   // Проверять существующих клиентов
MULTI_SERVER_DEFAULT_EXPIRY_DAYS = 30                                // Дней действия по умолчанию
```

## Настройка инбаунда

1. **Создайте инбаунд в X-UI панели** с ID 19 (или измените `MULTI_SERVER_INBOUND_ID`)
2. **Настройте протокол** - рекомендуется VLESS с XHTTP
3. **Убедитесь, что инбаунд активен**

## Добавление серверов

Серверы добавляются в таблицу `multi_servers`:

```sql
INSERT INTO multi_servers (id, name, country, country_code, flag, inbound_id, config_url, json_url, protocol, transport, enabled, priority) 
VALUES 
('germany', 'Германия', 'Germany', 'DE', '🇩🇪', 19, 'https://example.com/config', 'https://example.com/json', 'vless', 'xhttp', true, 100),
('finland', 'Финляндия', 'Finland', 'FI', '🇫🇮', 19, 'https://example.com/config', 'https://example.com/json', 'vless', 'xhttp', true, 90);
```

### Параметры сервера

- `id` - уникальный идентификатор сервера
- `name` - отображаемое название
- `country` - название страны
- `country_code` - код страны (ISO)
- `flag` - эмодзи флага
- `inbound_id` - ID инбаунда в X-UI панели
- `config_url` - URL для получения конфигурации
- `json_url` - URL для получения JSON конфигурации
- `protocol` - протокол (vless, vmess, etc.)
- `transport` - транспорт (xhttp, ws, etc.)
- `enabled` - включен ли сервер
- `priority` - приоритет отображения (больше = выше)

## Логика создания клиентов

### Автоматическое создание

При создании мультиподписки система:

1. **Проверяет настройку** `MULTI_SERVER_AUTO_CREATE_CLIENTS`
2. **Для каждого сервера** создает клиента в инбаунде
3. **Проверяет существующих клиентов** (если `MULTI_SERVER_CHECK_EXISTING = true`)
4. **Генерирует уникальный email** в формате: `{userID}_{serverID}_{subscriptionID}`

### Параметры клиента

```go
type Client struct {
    ID         string    // UUID клиента
    Flow       string    // Пустой для VLESS XHTTP
    Email      string    // Уникальный email
    TotalGB    int64     // 0 = безлимитный трафик
    ExpiryTime int64     // Время истечения (миллисекунды)
    Enable     bool      // Включен ли клиент
    TgID       int64     // ID пользователя Telegram
    SubID      string    // Уникальный SubID
    Reset      int64     // Сброс трафика
    Depleted   *bool     // Исчерпан ли трафик
    Exhausted  *bool     // Исчерпан ли лимит
    CreatedAt  int64     // Время создания
    UpdatedAt  int64     // Время обновления
}
```

## API Endpoints

### Получение мультиподписки

```
GET /api/multi-subscription?id={subscription_id}
```

Возвращает конфигурацию мультиподписки в формате, подходящем для импорта в VPN клиенты.

## Использование

1. **Пользователь выбирает** "Мультиподписка" в главном меню
2. **Выбирает серверы** из доступного списка
3. **Подтверждает выбор** и создает мультиподписку
4. **Получает ссылку** для автоимпорта
5. **Клиенты автоматически создаются** в X-UI панели

## Мониторинг

### Логи

Система ведет подробные логи:

```
CREATE_MULTI_SUBSCRIPTION_CLIENT: Создание клиента для мультиподписки {id}, сервер {server}, пользователь {user}
CREATE_MULTI_SUBSCRIPTION_CLIENT: Клиент с email {email} уже существует
CREATE_MULTI_SUBSCRIPTION_CLIENT: ✅ Клиент успешно создан в инбаунде {id}
```

### Очистка

Автоматическая очистка истекших состояний выбора серверов каждые 60 минут (настраивается в `MULTI_SUBSCRIPTION_CLEANUP_INTERVAL`).

## Безопасность

- **Уникальные email** для каждого клиента
- **Проверка существующих клиентов** перед созданием
- **Транзакционность** операций с базой данных
- **Валидация** входных данных

## Устранение неполадок

### Клиент не создается

1. Проверьте `MULTI_SERVER_AUTO_CREATE_CLIENTS = true`
2. Убедитесь, что инбаунд с ID `MULTI_SERVER_INBOUND_ID` существует
3. Проверьте логи на ошибки авторизации в панели

### Дублирование клиентов

1. Убедитесь, что `MULTI_SERVER_CHECK_EXISTING = true`
2. Проверьте логику генерации email

### Ошибки API

1. Проверьте настройки `MULTI_SUBSCRIPTION_BASE_URL`
2. Убедитесь, что серверы добавлены в `multi_servers`
3. Проверьте статус инбаундов