# Быстрая настройка мультиподписок

## 1. Обновление базы данных

Выполните SQL-скрипт для создания таблиц мультиподписок:

```bash
psql -U vpn_bot_user -d vpn_bot -f postgres_schema.sql
```

## 2. Настройка конфигурации

В файле `common/config.go` убедитесь, что настройки мультиподписок включены:

```go
// ========== МУЛЬТИПОДПИСКИ ==========
MULTI_SUBSCRIPTION_ENABLED = true        // Мультиподписки включены
MULTI_SUBSCRIPTION_MAX_SERVERS = 5       // Максимальное количество серверов
MULTI_SUBSCRIPTION_BASE_URL = "https://im.shadowfade.ru:8443/multi/" // Базовый URL
MULTI_SUBSCRIPTION_CLEANUP_INTERVAL = 60 // Интервал очистки состояний (мин)
```

## 3. Добавление серверов

Добавьте серверы в таблицу `multi_servers`:

```sql
INSERT INTO multi_servers (id, name, country, country_code, flag, inbound_id, config_url, json_url, protocol, transport, enabled, priority) VALUES
('server_de_1', 'Германия #1', 'Германия', 'DE', '🇩🇪', 1, 'https://your-server.com/config/de1', 'https://your-server.com/json/de1', 'vless', 'websocket', true, 100),
('server_fi_1', 'Финляндия #1', 'Финляндия', 'FI', '🇫🇮', 2, 'https://your-server.com/config/fi1', 'https://your-server.com/json/fi1', 'vless', 'websocket', true, 90),
('server_nl_1', 'Нидерланды #1', 'Нидерланды', 'NL', '🇳🇱', 3, 'https://your-server.com/config/nl1', 'https://your-server.com/json/nl1', 'vless', 'websocket', true, 80);
```

## 4. Обновление главного меню

Замените вызовы `SendMainMenu` на `SendMainMenuWithMultiSubscription` в обработчиках:

```go
// Вместо
menus.SendMainMenu(bot, chatID, user)

// Используйте
menus.SendMainMenuWithMultiSubscription(bot, chatID, user)
```

## 5. Перезапуск бота

```bash
# Остановите бота
pkill -f "go run main.go"

# Запустите заново
go run main.go
```

## 6. Тестирование

1. Откройте бота в Telegram
2. Нажмите "🌍 Мультиподписка"
3. Выберите несколько серверов
4. Подтвердите выбор
5. Проверьте создание мультиподписки

## 7. Мониторинг

Проверьте логи на наличие ошибок:

```bash
tail -f /root/bot/logs/console.log | grep MULTI_SUBSCRIPTION
```

## Возможные проблемы

### Ошибка "Мультиподписки временно недоступны"
- Проверьте, что `MULTI_SUBSCRIPTION_ENABLED = true`
- Перезапустите бота

### Ошибка "Не найдено серверов"
- Проверьте таблицу `multi_servers`
- Убедитесь, что есть серверы с `enabled = true`

### Ошибка API "/api/multi-subscription"
- Проверьте, что HTTP сервер запущен на порту 8081
- Проверьте логи на наличие ошибок

### Ошибка импорта в приложение
- Проверьте, что URL серверов доступны
- Убедитесь, что приложение поддерживает импорт подписок

## Дополнительные настройки

### Изменение максимального количества серверов
```go
MULTI_SUBSCRIPTION_MAX_SERVERS = 10  // До 10 серверов
```

### Изменение базового URL
```go
MULTI_SUBSCRIPTION_BASE_URL = "https://your-domain.com/multi/"
```

### Добавление новых стран
```sql
INSERT INTO multi_servers (id, name, country, country_code, flag, inbound_id, config_url, json_url, protocol, transport, enabled, priority) 
VALUES ('server_uk_1', 'Великобритания #1', 'Великобритания', 'GB', '🇬🇧', 6, 'https://your-server.com/config/uk1', 'https://your-server.com/json/uk1', 'vless', 'websocket', true, 70);
```

## Поддержка

При возникновении проблем:
1. Проверьте логи бота
2. Убедитесь, что все настройки корректны
3. Проверьте доступность серверов
4. Обратитесь к документации `MULTI_SUBSCRIPTION_README.md`
