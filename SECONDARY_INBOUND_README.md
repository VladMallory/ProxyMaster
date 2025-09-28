# Дополнительный инбаунд - Документация

## Обзор

Добавлена возможность синхронизации с дополнительным инбаундом (номер 3, VLESS + WebSocket) в дополнение к основному инбаунду (номер 2). Это позволяет клиентам получать конфиги для двух различных инбаундов одновременно.

## Настройки в config.go

### Новые переменные конфигурации:

```go
// ========== НАСТРОЙКИ ДОПОЛНИТЕЛЬНОГО ИНБАУНДА ==========
SECONDARY_INBOUND_ENABLED    bool   // Включена ли синхронизация с дополнительным инбаундом
SECONDARY_INBOUND_ID         int    // ID дополнительного инбаунда
SECONDARY_CONFIG_BASE_URL    string // Базовый URL для конфигураций дополнительного инбаунда
SECONDARY_CONFIG_JSON_URL    string // URL для JSON конфигураций дополнительного инбаунда
SECONDARY_REDIRECT_DOMAIN    string // Домен для редиректа импорта дополнительного инбаунда
SECONDARY_REDIRECT_IMPORT    string // Тип импорта для дополнительного инбаунда
SECONDARY_INBOUND_PROTOCOL   string // Протокол дополнительного инбаунда (vless, vmess, etc.)
SECONDARY_INBOUND_TRANSPORT  string // Транспорт дополнительного инбаунда (websocket, tcp, etc.)
```

### Текущие настройки:

```go
SECONDARY_INBOUND_ENABLED = true
SECONDARY_INBOUND_ID = 3
SECONDARY_CONFIG_BASE_URL = "https://shadowfade.ru:6187/E4x7DnWKY8QnRdDoc3/"
SECONDARY_CONFIG_JSON_URL = "https://shadowfade.ru:6187/htJyYoEQLqf3pfwGgs/"
SECONDARY_REDIRECT_DOMAIN = "im.shadowfade.ru:8443"
SECONDARY_REDIRECT_IMPORT = "happ"
SECONDARY_INBOUND_PROTOCOL = "vless"
SECONDARY_INBOUND_TRANSPORT = "websocket"
```

## Расширение модели пользователя

В структуру `User` добавлены новые поля:

```go
// Дополнительный инбаунд
SecondaryClientID        string    `bson:"secondary_client_id" json:"secondary_client_id"`
SecondarySubID           string    `bson:"secondary_sub_id" json:"secondary_sub_id"`
SecondaryEmail           string    `bson:"secondary_email" json:"secondary_email"`
SecondaryConfigCreatedAt time.Time `bson:"secondary_config_created_at" json:"secondary_config_created_at"`
SecondaryExpiryTime      int64     `bson:"secondary_expiry_time" json:"secondary_expiry_time"`
HasActiveSecondaryConfig bool      `bson:"has_active_secondary_config" json:"has_active_secondary_config"`
```

## Обновление базы данных

Выполните миграцию для добавления новых полей:

```bash
psql -d vpn_bot -f migrations/add_secondary_inbound_fields.sql
```

## Новые функции

### Основные функции для работы с дополнительным инбаундом:

1. **GetSecondaryInbound(sessionCookie)** - получение настроек дополнительного инбаунда
2. **AddSecondaryClient(sessionCookie, user, days)** - создание клиента в дополнительном инбаунде
3. **SyncUserWithSecondaryPanel(user)** - синхронизация пользователя с дополнительным инбаундом
4. **IsSecondaryConfigActive(user)** - проверка активности конфига в дополнительном инбаунде
5. **GetSecondaryConfigURL(user)** - получение URL конфигурации дополнительного инбаунда

### Универсальные функции:

1. **HasAnyActiveConfig(user)** - проверка наличия активного конфига в любом из инбаундов
2. **GetActiveConfigInfo(user)** - получение полной информации об активных конфигах

## Логика работы

### Создание конфигов

При создании конфига (пробный период, пополнение баланса, автосписание):

1. Создается конфиг в основном инбаунде (ID: 2)
2. Если `SECONDARY_INBOUND_ENABLED = true`, создается конфиг в дополнительном инбаунде (ID: 3)
3. Обновляются соответствующие поля в базе данных

### Синхронизация

При каждом `/start` и проверке конфигов:

1. Синхронизируется основной инбаунд
2. Если дополнительный инбаунд включен, синхронизируется дополнительный инбаунд
3. Обновляются флаги `HasActiveConfig` и `HasActiveSecondaryConfig`

### Проверка активности

Система теперь учитывает активность конфигов в обоих инбаундах:

- `HasActiveConfig` - активность основного конфига
- `HasActiveSecondaryConfig` - активность дополнительного конфига
- `HasAnyActiveConfig()` - наличие активного конфига в любом из инбаундов

## Управление

### Включение/отключение

Для отключения дополнительного инбаунда установите:

```go
SECONDARY_INBOUND_ENABLED = false
```

### Изменение ID инбаунда

Для использования другого инбаунда измените:

```go
SECONDARY_INBOUND_ID = 4  // Новый ID инбаунда
```

## Логирование

Все операции с дополнительным инбаундом логируются с префиксами:

- `GET_SECONDARY_INBOUND:` - получение настроек инбаунда
- `ADD_SECONDARY_CLIENT:` - создание клиента
- `SYNC_SECONDARY_PANEL:` - синхронизация
- `SECONDARY_INBOUND:` - общие операции

## Мониторинг

Для мониторинга состояния дополнительного инбаунда используйте:

```go
configInfo := GetActiveConfigInfo(user)
fmt.Printf("Primary active: %v\n", configInfo["primary_active"])
fmt.Printf("Secondary active: %v\n", configInfo["secondary_active"])
fmt.Printf("Any active: %v\n", configInfo["any_active"])
```

## Обработка ошибок

Система спроектирована так, чтобы ошибки в дополнительном инбаунде не влияли на основной:

- Если основной конфиг создан успешно, но дополнительный не удался - основной остается активным
- Логируется предупреждение, но выполнение продолжается
- Пользователь получает доступ к основному конфигу

## Совместимость

- Полная обратная совместимость с существующими пользователями
- Новые поля имеют значения по умолчанию
- При отключении дополнительного инбаунда система работает как раньше
