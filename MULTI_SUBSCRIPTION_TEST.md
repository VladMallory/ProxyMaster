# Тестирование мультиподписок

## ✅ Что было сделано

1. **Заменили кнопку "Подключить"** на "🌍 Мультиподписка" в главном меню
2. **Добавили обработку callback** `multi_subscription` в `callback_handler.go`
3. **Создали таблицы** в базе данных:
   - `multi_servers` - список доступных серверов
   - `multi_subscriptions` - мультиподписки пользователей
   - `multi_subscription_servers` - связь серверов с подписками
   - `server_selection_states` - состояния выбора серверов
4. **Добавили тестовые серверы**:
   - 🇩🇪 Основной сервер (inbound_id: 19)
   - 🇩🇪 Сервер #19 (inbound_id: 1)

## 🧪 Как протестировать

1. **Запустите бота** (если еще не запущен)
2. **Отправьте команду** `/start` или `/menu`
3. **Нажмите кнопку** "🌍 Мультиподписка"
4. **Выберите серверы** из списка (можно выбрать несколько)
5. **Подтвердите выбор** и создайте мультиподписку
6. **Получите ссылку** для автоимпорта

## 🔧 Настройки

Все настройки находятся в `common/config.go`:

```go
// Основные настройки
MULTI_SUBSCRIPTION_ENABLED = true        // Включены ли мультиподписки
MULTI_SUBSCRIPTION_MAX_SERVERS = 2       // Максимум серверов в подписке
MULTI_SUBSCRIPTION_BASE_URL = "https://im.shadowfade.ru:8443/multi/"

// Настройки серверов
MULTI_SERVER_INBOUND_ID = 19             // ID инбаунда для создания клиентов
MULTI_SERVER_AUTO_CREATE_CLIENTS = true  // Автоматически создавать клиентов
MULTI_SERVER_CHECK_EXISTING = true       // Проверять существующих клиентов
MULTI_SERVER_DEFAULT_EXPIRY_DAYS = 30    // Дней действия по умолчанию
```

## 📊 Логи

При тестировании следите за логами:

```
HANDLE_MULTI_SUBSCRIPTION_CALLBACK: Обработка callback: multi_subscription
SEND_MULTI_SUBSCRIPTION_MENU: Отправка меню мультиподписки
GET_AVAILABLE_SERVERS: Получение доступных серверов
CREATE_MULTI_SUBSCRIPTION: Создание мультиподписки
CREATE_MULTI_SUBSCRIPTION_CLIENT: Создание клиента для мультиподписки
```

## 🐛 Устранение неполадок

### Ошибка "relation does not exist"
- Убедитесь, что таблицы созданы: `\dt multi_*` в psql
- Если нет, выполните: `psql -f postgres_schema.sql`

### Ошибка "Ошибка загрузки серверов"
- Проверьте, что в таблице `multi_servers` есть записи
- Убедитесь, что `enabled = true` для серверов

### Клиенты не создаются
- Проверьте `MULTI_SERVER_AUTO_CREATE_CLIENTS = true`
- Убедитесь, что инбаунд с ID 19 существует в X-UI панели
- Проверьте логи авторизации в панели

## 📝 Добавление новых серверов

```sql
INSERT INTO multi_servers (id, name, country, country_code, flag, inbound_id, config_url, json_url, protocol, transport, enabled, priority) 
VALUES ('new_server', 'Новый сервер', 'Country', 'CC', '🏳️', 19, 'https://example.com/config', 'https://example.com/json', 'vless', 'xhttp', true, 50);
```

## 🎯 Результат

После успешного тестирования пользователи смогут:
1. Выбирать несколько серверов в одной подписке
2. Автоматически получать клиентов в X-UI панели
3. Импортировать конфигурацию в VPN приложения
4. Управлять мультиподписками через бота
