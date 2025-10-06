# Система мультиподписок

## Описание

Система мультиподписок позволяет пользователям выбирать несколько серверов из разных стран для создания единой подписки. Это обеспечивает лучшую производительность, стабильность и возможность обхода блокировок.

## Основные возможности

- 🌍 **Выбор нескольких серверов** - пользователь может выбрать до 5 серверов из разных стран
- ⚡ **Автоматическое переключение** - приложение автоматически выбирает лучший сервер
- 🔄 **Автоматическое обновление** - конфигурации обновляются автоматически
- 📱 **Простой импорт** - поддержка Happ и V2rayTun приложений
- 🎯 **Умный выбор** - серверы сортируются по приоритету и производительности

## Архитектура

### Структуры данных

#### Server
```go
type Server struct {
    ID          string `json:"id" bson:"id"`
    Name        string `json:"name" bson:"name"`
    Country     string `json:"country" bson:"country"`
    CountryCode string `json:"country_code" bson:"country_code"`
    Flag        string `json:"flag" bson:"flag"`
    InboundID   int    `json:"inbound_id" bson:"inbound_id"`
    ConfigURL   string `json:"config_url" bson:"config_url"`
    JSONURL     string `json:"json_url" bson:"json_url"`
    Protocol    string `json:"protocol" bson:"protocol"`
    Transport   string `json:"transport" bson:"transport"`
    Enabled     bool   `json:"enabled" bson:"enabled"`
    Priority    int    `json:"priority" bson:"priority"`
}
```

#### MultiSubscription
```go
type MultiSubscription struct {
    ID              string    `json:"id" bson:"id"`
    UserID          int64     `json:"user_id" bson:"user_id"`
    Servers         []Server  `json:"servers" bson:"servers"`
    SubscriptionURL string    `json:"subscription_url" bson:"subscription_url"`
    IsActive        bool      `json:"is_active" bson:"is_active"`
    CreatedAt       time.Time `json:"created_at" bson:"created_at"`
    UpdatedAt       time.Time `json:"updated_at" bson:"updated_at"`
    ExpiryTime      int64     `json:"expiry_time" bson:"expiry_time"`
}
```

### База данных

#### Таблицы
- `multi_servers` - серверы для мультиподписок
- `multi_subscriptions` - мультиподписки пользователей
- `multi_subscription_servers` - связь мультиподписок с серверами (many-to-many)
- `server_selection_states` - временные состояния выбора серверов

#### Индексы
- `idx_multi_servers_enabled` - для быстрого поиска активных серверов
- `idx_multi_servers_priority` - для сортировки по приоритету
- `idx_multi_subscriptions_user` - для поиска мультиподписок пользователя
- `idx_server_selection_states_user` - для поиска состояний выбора

## Настройки

### Конфигурация в `common/config.go`

```go
// ========== МУЛЬТИПОДПИСКИ ==========
MULTI_SUBSCRIPTION_ENABLED = true        // Мультиподписки включены
MULTI_SUBSCRIPTION_MAX_SERVERS = 5       // Максимальное количество серверов
MULTI_SUBSCRIPTION_BASE_URL = "https://im.shadowfade.ru:8443/multi/" // Базовый URL
MULTI_SUBSCRIPTION_CLEANUP_INTERVAL = 60 // Интервал очистки состояний (мин)
```

## API Endpoints

### GET /api/multi-subscription?id={subscription_id}

Возвращает конфигурации для мультиподписки.

**Параметры:**
- `id` - ID мультиподписки

**Ответ:**
- `200 OK` - конфигурации в формате base64
- `400 Bad Request` - не указан ID
- `404 Not Found` - мультиподписка не найдена или неактивна
- `500 Internal Server Error` - ошибка генерации конфигураций

## Пользовательский интерфейс

### Меню выбора серверов

1. **Список серверов** - отображаются все доступные серверы с флагами стран
2. **Выбор серверов** - пользователь может выбрать до 5 серверов
3. **Подтверждение** - показывается информация о выбранных серверах
4. **Создание** - создается мультиподписка и списываются средства

### HTML страница импорта

- **Автоматическое определение приложения** - Happ или V2rayTun
- **Красивый интерфейс** - современный дизайн с анимациями
- **Информация о серверах** - список включенных серверов
- **Обработка ошибок** - понятные сообщения об ошибках

## Процесс создания мультиподписки

1. **Инициация** - пользователь нажимает "🌍 Мультиподписка"
2. **Выбор серверов** - отображается меню с доступными серверами
3. **Подтверждение** - показывается информация о выбранных серверах
4. **Создание** - создается мультиподписка в базе данных
5. **Списание средств** - списывается стоимость подписки
6. **Уведомление** - пользователь получает ссылку на мультиподписку

## Файлы системы

### Основные файлы
- `common/types.go` - структуры данных
- `common/multi_subscription.go` - функции работы с мультиподписками
- `common/config.go` - настройки мультиподписок

### Меню и обработчики
- `menus/multi_subscription_menu.go` - меню выбора серверов
- `handlers/multi_subscription_handler.go` - обработчики callback'ов
- `menus/main_menu_updated.go` - обновленное главное меню

### API и веб-интерфейс
- `app/init.go` - API endpoints для мультиподписок
- `importRedirect/redirect_multi_subscription.html` - HTML страница импорта

### База данных
- `postgres_schema.sql` - схема базы данных с таблицами мультиподписок

## Использование

### Для пользователей

1. Откройте бота и нажмите "🌍 Мультиподписка"
2. Выберите серверы из разных стран (до 5 штук)
3. Подтвердите выбор
4. Нажмите "Подключить мультиподписку" для импорта

### Для администраторов

1. Добавьте серверы в таблицу `multi_servers`
2. Настройте параметры в `common/config.go`
3. Перезапустите бота

## Мониторинг и обслуживание

### Очистка состояний
Система автоматически очищает истекшие состояния выбора серверов каждые 60 минут.

### Логирование
Все операции логируются с префиксами:
- `MULTI_SUBSCRIPTION_*` - основные операции
- `GET_AVAILABLE_SERVERS` - получение серверов
- `CREATE_MULTI_SUBSCRIPTION` - создание мультиподписок
- `GENERATE_MULTI_SUBSCRIPTION_CONFIGS` - генерация конфигураций

## Безопасность

- Проверка прав доступа к мультиподпискам
- Валидация выбранных серверов
- Защита от SQL-инъекций через параметризованные запросы
- Ограничение времени жизни состояний выбора

## Производительность

- Индексы для быстрого поиска серверов и мультиподписок
- Кэширование состояний выбора
- Асинхронная генерация конфигураций
- Оптимизированные SQL-запросы

## Расширение

### Добавление новых серверов
```sql
INSERT INTO multi_servers (id, name, country, country_code, flag, inbound_id, config_url, json_url, protocol, transport, enabled, priority) 
VALUES ('server_new_1', 'Новый сервер', 'Страна', 'XX', '🏳️', 6, 'https://example.com/config', 'https://example.com/json', 'vless', 'websocket', true, 50);
```

### Настройка приоритетов
Серверы сортируются по убыванию приоритета. Чем выше значение `priority`, тем выше сервер в списке.

### Добавление новых протоколов
Обновите структуру `Server` и добавьте поддержку в функции генерации конфигураций.
