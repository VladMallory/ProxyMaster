# Архитектура системы мультиподписок

## Общая схема

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Telegram Bot  │    │   HTTP Server    │    │   PostgreSQL    │
│                 │    │                  │    │                 │
│ ┌─────────────┐ │    │ ┌──────────────┐ │    │ ┌─────────────┐ │
│ │ Main Menu   │ │    │ │ API Endpoint │ │    │ │ multi_servers│ │
│ │ + Multi Btn │ │    │ │ /api/multi-  │ │    │ │             │ │
│ └─────────────┘ │    │ │ subscription │ │    │ └─────────────┘ │
│                 │    │ └──────────────┘ │    │                 │
│ ┌─────────────┐ │    │                  │    │ ┌─────────────┐ │
│ │ Multi Menu  │ │    │ ┌──────────────┐ │    │ │multi_subscr.│ │
│ │ Server Sel. │ │    │ │ HTML Page    │ │    │ │             │ │
│ └─────────────┘ │    │ │ /multi/      │ │    │ └─────────────┘ │
│                 │    │ └──────────────┘ │    │                 │
│ ┌─────────────┐ │    │                  │    │ ┌─────────────┐ │
│ │ Callback    │ │    │                  │    │ │multi_subscr.│ │
│ │ Handlers    │ │    │                  │    │ │_servers     │ │
│ └─────────────┘ │    │                  │    │ └─────────────┘ │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Поток данных

### 1. Инициация мультиподписки
```
User → Bot → Main Menu → Multi Button → Multi Menu
```

### 2. Выбор серверов
```
User → Server Selection → Callback Handler → State Update → Menu Update
```

### 3. Создание мультиподписки
```
User → Confirm → Create Handler → Database → Payment → Success
```

### 4. Импорт конфигурации
```
User → Import Button → HTML Page → API Call → Config Generation → App Import
```

## Компоненты системы

### Frontend (Telegram Bot)
- **Main Menu** - главное меню с кнопкой мультиподписки
- **Multi Menu** - меню выбора серверов
- **Callback Handlers** - обработчики нажатий кнопок
- **State Management** - управление состоянием выбора

### Backend (HTTP Server)
- **API Endpoint** - `/api/multi-subscription?id={id}`
- **Config Generator** - генерация конфигураций для серверов
- **HTML Page** - страница импорта мультиподписки

### Database (PostgreSQL)
- **multi_servers** - справочник серверов
- **multi_subscriptions** - мультиподписки пользователей
- **multi_subscription_servers** - связь подписок с серверами
- **server_selection_states** - временные состояния выбора

## API Endpoints

### GET /api/multi-subscription
```
Request:  GET /api/multi-subscription?id=multi_12345
Response: 200 OK
          Content-Type: text/plain
          [Base64 encoded configs for all servers]
```

### HTML Page
```
URL: https://domain.com/multi/?id=multi_12345&app=happ
Features:
- Auto-detect app type (Happ/V2rayTun)
- Server information display
- One-click import
- Error handling
```

## База данных

### Таблица multi_servers
```sql
CREATE TABLE multi_servers (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    country VARCHAR(100) NOT NULL,
    country_code VARCHAR(3) NOT NULL,
    flag VARCHAR(10) NOT NULL,
    inbound_id INTEGER NOT NULL,
    config_url VARCHAR(500) NOT NULL,
    json_url VARCHAR(500) NOT NULL,
    protocol VARCHAR(20) NOT NULL DEFAULT 'vless',
    transport VARCHAR(20) NOT NULL DEFAULT 'websocket',
    enabled BOOLEAN DEFAULT TRUE,
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Таблица multi_subscriptions
```sql
CREATE TABLE multi_subscriptions (
    id VARCHAR(50) PRIMARY KEY,
    user_id BIGINT NOT NULL,
    subscription_url VARCHAR(500) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    expiry_time BIGINT,
    FOREIGN KEY (user_id) REFERENCES users(telegram_id) ON DELETE CASCADE
);
```

### Таблица multi_subscription_servers
```sql
CREATE TABLE multi_subscription_servers (
    id SERIAL PRIMARY KEY,
    subscription_id VARCHAR(50) NOT NULL,
    server_id VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (subscription_id) REFERENCES multi_subscriptions(id) ON DELETE CASCADE,
    FOREIGN KEY (server_id) REFERENCES multi_servers(id) ON DELETE CASCADE,
    UNIQUE(subscription_id, server_id)
);
```

## Конфигурация

### Настройки в common/config.go
```go
// ========== МУЛЬТИПОДПИСКИ ==========
MULTI_SUBSCRIPTION_ENABLED = true        // Включены ли мультиподписки
MULTI_SUBSCRIPTION_MAX_SERVERS = 5       // Максимальное количество серверов
MULTI_SUBSCRIPTION_BASE_URL = "https://im.shadowfade.ru:8443/multi/" // Базовый URL
MULTI_SUBSCRIPTION_CLEANUP_INTERVAL = 60 // Интервал очистки состояний (мин)
```

## Безопасность

### Проверки
- Валидация выбранных серверов
- Проверка лимитов (максимум серверов)
- Проверка баланса пользователя
- Валидация ID мультиподписки

### Защита
- SQL-инъекции: параметризованные запросы
- XSS: экранирование HTML
- CSRF: проверка источников запросов
- Rate limiting: ограничение частоты запросов

## Производительность

### Оптимизации
- Индексы для быстрого поиска
- Кэширование состояний выбора
- Асинхронная генерация конфигураций
- Батчинг операций с базой данных

### Мониторинг
- Логирование всех операций
- Метрики производительности
- Алерты при ошибках
- Статистика использования

## Масштабирование

### Горизонтальное
- Несколько экземпляров бота
- Балансировщик нагрузки
- Репликация базы данных

### Вертикальное
- Увеличение ресурсов сервера
- Оптимизация запросов к БД
- Кэширование конфигураций

## Мониторинг и логирование

### Логи
- `MULTI_SUBSCRIPTION_*` - основные операции
- `GET_AVAILABLE_SERVERS` - получение серверов
- `CREATE_MULTI_SUBSCRIPTION` - создание подписок
- `GENERATE_MULTI_SUBSCRIPTION_CONFIGS` - генерация конфигов

### Метрики
- Количество созданных мультиподписок
- Популярность серверов
- Время генерации конфигураций
- Ошибки API

## Развертывание

### Требования
- Go 1.19+
- PostgreSQL 12+
- Telegram Bot API
- HTTP сервер (встроенный)

### Шаги
1. Обновить схему базы данных
2. Настроить конфигурацию
3. Добавить серверы
4. Перезапустить бота
5. Протестировать функциональность
