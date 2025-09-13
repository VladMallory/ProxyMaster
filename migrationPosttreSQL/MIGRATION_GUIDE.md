# 🚀 Руководство по миграции с MongoDB на PostgreSQL

## 📋 Обзор

Этот документ содержит пошаговые инструкции для полной миграции VPN бота с MongoDB на PostgreSQL. Все ваши данные будут сохранены, а функциональность останется точно такой же.

## ✅ Преимущества миграции

- **🚀 Лучшая производительность** - PostgreSQL быстрее для операций поиска и сортировки
- **🔒 ACID транзакции** - гарантия целостности данных
- **📊 Более строгая типизация** данных
- **🛠️ Лучшие инструменты мониторинга**
- **💰 Меньше потребление ресурсов**

## 📋 Требования

### Системные требования
- PostgreSQL 12 или выше
- Go 1.19 или выше  
- Доступ к существующей MongoDB (для миграции данных)

### Установка PostgreSQL

#### Ubuntu/Debian:
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
```

#### CentOS/RHEL:
```bash
sudo yum install postgresql-server postgresql-contrib
sudo postgresql-setup initdb
sudo systemctl enable postgresql
sudo systemctl start postgresql
```

## 🔧 Пошаговая миграция

### Шаг 1: Подготовка

1. **Создайте бэкап MongoDB** (на всякий случай):
```bash
mongodump --uri mongodb://localhost:27017 --db vpn_bot --out ./backup_before_migration
```

2. **Остановите VPN бота** (если он запущен):
```bash
# Остановите процесс бота
killall bot
# или используйте systemctl если бот настроен как сервис
sudo systemctl stop vpn-bot
```

### Шаг 2: Настройка PostgreSQL

Выполните автоматическую настройку:

```bash
cd /root/bot
./setup_postgres.sh
```

Скрипт автоматически:
- Создаст пользователя базы данных
- Создаст базу данных
- Применит SQL схему
- Настроит права доступа
- Создаст переменные окружения

### Шаг 3: Загрузка переменных окружения

```bash
source .env.postgres
```

Или создайте свои переменные:
```bash
export PG_HOST=localhost
export PG_PORT=5432
export PG_USER=vpn_bot_user
export PG_PASSWORD=your_secure_password
export PG_DBNAME=vpn_bot
```

### Шаг 4: Миграция данных

Выполните миграцию данных из MongoDB:

```bash
# Убедитесь, что MongoDB запущена
sudo systemctl start mongod

# Запустите миграцию
go mod tidy -modfile=migration_go.mod
go run -modfile=migration_go.mod migrate_to_postgres.go
```

Миграция перенесет:
- ✅ Всех пользователей
- ✅ Настройки трафика
- ✅ Конфигурации

### Шаг 5: Обновление зависимостей

```bash
go mod tidy
```

### Шаг 6: Тестирование

Запустите бота для проверки:

```bash
go run main.go
```

Проверьте:
- ✅ Бот запускается без ошибок
- ✅ База данных подключается
- ✅ Все пользователи доступны
- ✅ Команды работают корректно

## 🔍 Проверка миграции

### Проверка данных в PostgreSQL

```bash
# Подключитесь к базе данных
psql -h localhost -U vpn_bot_user -d vpn_bot

# Проверьте таблицы
\dt

# Проверьте количество пользователей
SELECT COUNT(*) FROM users;

# Проверьте несколько пользователей
SELECT telegram_id, username, first_name, balance, has_active_config 
FROM users ORDER BY created_at DESC LIMIT 5;

# Проверьте конфигурации трафика
SELECT * FROM traffic_configs;

# Выйдите из psql
\q
```

### Сравнение с MongoDB

```bash
# Проверьте количество пользователей в MongoDB
mongo vpn_bot --eval "db.users.count()"

# Сравните с PostgreSQL
psql -h localhost -U vpn_bot_user -d vpn_bot -c "SELECT COUNT(*) FROM users;"
```

## 🛠️ Использование новых инструментов

### PostgreSQL cleanup tool

```bash
cd tools
go run cleanup_tool_postgres.go
```

Функции:
- Показать всех пользователей
- Сбросить флаги пробных периодов
- Удалить пользователей
- Очистить базу данных

### Мониторинг PostgreSQL

```bash
# Статус PostgreSQL
sudo systemctl status postgresql

# Подключения к базе данных
sudo -u postgres psql -c "SELECT * FROM pg_stat_activity WHERE datname='vpn_bot';"

# Размер базы данных
sudo -u postgres psql -c "SELECT pg_size_pretty(pg_database_size('vpn_bot'));"
```

## 📊 Настройка производительности PostgreSQL

### Оптимизация postgresql.conf

```bash
sudo nano /etc/postgresql/*/main/postgresql.conf
```

Рекомендуемые настройки:
```
# Память
shared_buffers = 256MB
effective_cache_size = 1GB
work_mem = 4MB

# Соединения
max_connections = 100

# Логирование
log_statement = 'mod'
log_duration = on
log_min_duration_statement = 1000
```

Перезапустите PostgreSQL:
```bash
sudo systemctl restart postgresql
```

## 🔄 Backup и восстановление

### Автоматический backup

Бот теперь автоматически создает backup PostgreSQL каждый час.

### Ручной backup

```bash
# Создание backup
PGPASSWORD=$PG_PASSWORD pg_dump -h localhost -U vpn_bot_user vpn_bot > backup_$(date +%Y%m%d_%H%M%S).sql

# Восстановление из backup
PGPASSWORD=$PG_PASSWORD psql -h localhost -U vpn_bot_user -d vpn_bot < backup_file.sql
```

## 🚨 Устранение проблем

### Проблема: Ошибка подключения к PostgreSQL

**Решение:**
```bash
# Проверьте статус PostgreSQL
sudo systemctl status postgresql

# Перезапустите если нужно
sudo systemctl restart postgresql

# Проверьте настройки подключения
sudo -u postgres psql -c "SELECT current_setting('listen_addresses');"
```

### Проблема: Пользователь не может подключиться

**Решение:**
```bash
# Проверьте пользователя
sudo -u postgres psql -c "SELECT usename FROM pg_user WHERE usename='vpn_bot_user';"

# Проверьте права
sudo -u postgres psql -c "SELECT datname FROM pg_database WHERE datname='vpn_bot';"

# Проверьте pg_hba.conf
sudo nano /etc/postgresql/*/main/pg_hba.conf
```

### Проблема: Миграция не завершилась

**Решение:**
```bash
# Проверьте логи миграции
# Убедитесь что MongoDB запущена
sudo systemctl start mongod

# Проверьте доступность данных MongoDB
mongo vpn_bot --eval "db.users.findOne()"

# Повторите миграцию
go run migrate_to_postgres.go
```

### Проблема: Бот не запускается

**Решение:**
```bash
# Проверьте переменные окружения
echo $PG_USER $PG_PASSWORD $PG_DBNAME

# Проверьте зависимости
go mod tidy

# Проверьте подключение к базе
psql -h localhost -U $PG_USER -d $PG_DBNAME -c "SELECT 1;"

# Запустите с логированием
go run main.go 2>&1 | tee bot.log
```

## 📈 Мониторинг после миграции

### Логи PostgreSQL

```bash
# Просмотр логов PostgreSQL
sudo tail -f /var/log/postgresql/postgresql-*-main.log
```

### Производительность

```bash
# Статистика по таблицам
sudo -u postgres psql vpn_bot -c "SELECT schemaname, tablename, n_tup_ins, n_tup_upd, n_tup_del FROM pg_stat_user_tables;"

# Индексы
sudo -u postgres psql vpn_bot -c "SELECT schemaname, tablename, indexname, idx_scan FROM pg_stat_user_indexes;"
```

### Размер базы данных

```bash
# Размер таблиц
sudo -u postgres psql vpn_bot -c "SELECT tablename, pg_size_pretty(pg_total_relation_size(tablename::regclass)) FROM pg_tables WHERE schemaname='public';"
```

## ✅ Проверочный список после миграции

- [ ] PostgreSQL работает корректно
- [ ] Все данные мигрированы (проверить количество пользователей)
- [ ] Бот запускается без ошибок
- [ ] Все команды бота работают
- [ ] Backup создается автоматически
- [ ] Cleanup tools работают
- [ ] Мониторинг настроен

## 🎯 Заключение

После успешной миграции:

1. **MongoDB можно отключить** (но сначала убедитесь что всё работает):
```bash
sudo systemctl stop mongod
sudo systemctl disable mongod
```

2. **Настройте автозапуск PostgreSQL**:
```bash
sudo systemctl enable postgresql
```

3. **Обновите скрипты мониторинга** для работы с PostgreSQL

4. **Обновите документацию** для новых участников команды

## 📞 Поддержка

Если возникли проблемы:
1. Проверьте раздел "Устранение проблем" выше
2. Изучите логи: `tail -f /var/log/postgresql/postgresql-*-main.log`
3. Проверьте подключение: `psql -h localhost -U vpn_bot_user -d vpn_bot`

🎉 **Поздравляем! Миграция завершена!** 

Ваш VPN бот теперь работает на более быстрой и надежной PostgreSQL базе данных.
