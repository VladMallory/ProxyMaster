# 🚀 Руководство по миграции PostgreSQL на новый сервер

## 📋 Обзор

Это руководство поможет вам безопасно мигрировать PostgreSQL базу данных VPN бота на новый сервер.

## 🛠️ Подготовленные скрипты

1. **`migrate_postgres_to_new_server.sh`** - Основной скрипт миграции
2. **`setup_postgres_new_server.sh`** - Настройка PostgreSQL на новом сервере
3. **`restore_postgres_data.sh`** - Восстановление данных на новом сервере
4. **`verify_migration.sh`** - Проверка успешности миграции

## 🚀 Быстрая миграция

### Шаг 1: Подготовка на текущем сервере

```bash
cd /root/bot

# Запустите основной скрипт миграции
./migrate_postgres_to_new_server.sh NEW_SERVER_IP

# Пример:
./migrate_postgres_to_new_server.sh 192.168.1.100
```

Скрипт автоматически:
- ✅ Создаст полный бэкап базы данных
- ✅ Создаст бэкап только данных
- ✅ Создаст бэкап схемы
- ✅ Скопирует конфигурационные файлы
- ✅ Создаст архив для передачи
- ✅ Скопирует все на новый сервер

### Шаг 2: Настройка на новом сервере

```bash
# Подключитесь к новому серверу
ssh root@NEW_SERVER_IP

# Перейдите в директорию с архивом
cd /tmp

# Распакуйте архив
tar -xzf vpn_bot_migration_*.tar.gz

# Настройте PostgreSQL
./setup_postgres_new_server.sh

# Восстановите данные
./restore_postgres_data.sh vpn_bot_data_only.sql
```

### Шаг 3: Копирование проекта

```bash
# На текущем сервере скопируйте проект
rsync -avz --exclude='backups/' /root/bot/ root@NEW_SERVER_IP:/root/bot/

# На новом сервере
cd /root/bot
source .env.postgres
go mod tidy
go run main.go
```

### Шаг 4: Проверка миграции

```bash
# Проверьте успешность миграции
./verify_migration.sh
```

## 📊 Детальная миграция

### Вариант 1: Полная миграция (рекомендуется)

```bash
# 1. На текущем сервере
cd /root/bot
./migrate_postgres_to_new_server.sh NEW_SERVER_IP

# 2. На новом сервере
ssh root@NEW_SERVER_IP
cd /tmp
tar -xzf vpn_bot_migration_*.tar.gz
./restore_on_new_server.sh

# 3. Копирование проекта
rsync -avz --exclude='backups/' /root/bot/ root@NEW_SERVER_IP:/root/bot/

# 4. Запуск на новом сервере
cd /root/bot
source .env.postgres
go mod tidy
go run main.go
```

### Вариант 2: Ручная миграция

```bash
# 1. Создание бэкапа
PGPASSWORD="$PG_PASSWORD" pg_dump -h localhost -U vpn_bot_user vpn_bot > backup.sql

# 2. Копирование на новый сервер
scp backup.sql root@NEW_SERVER_IP:/tmp/

# 3. Настройка PostgreSQL на новом сервере
./setup_postgres_new_server.sh

# 4. Восстановление данных
./restore_postgres_data.sh /tmp/backup.sql
```

## 🔍 Проверка и мониторинг

### Проверка базы данных

```bash
# Подключение к базе данных
source .env.postgres
PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -U "$PG_USER" -d "$PG_DBNAME"

# Проверка пользователей
SELECT COUNT(*) FROM users;

# Проверка конфигураций
SELECT * FROM traffic_configs;

# Статистика
SELECT * FROM get_users_statistics();
```

### Мониторинг производительности

```bash
# Размер базы данных
PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -U "$PG_USER" -d "$PG_DBNAME" -c "SELECT pg_size_pretty(pg_database_size('$PG_DBNAME'));"

# Активные подключения
PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -U "$PG_USER" -d "$PG_DBNAME" -c "SELECT * FROM pg_stat_activity WHERE datname='$PG_DBNAME';"

# Статистика таблиц
PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -U "$PG_USER" -d "$PG_DBNAME" -c "SELECT schemaname, tablename, n_tup_ins, n_tup_upd, n_tup_del FROM pg_stat_user_tables;"
```

## 🛠️ Устранение проблем

### Проблема: Ошибка подключения

```bash
# Проверьте статус PostgreSQL
systemctl status postgresql

# Перезапустите PostgreSQL
systemctl restart postgresql

# Проверьте настройки подключения
sudo -u postgres psql -c "SELECT current_setting('listen_addresses');"
```

### Проблема: Недостаточно прав

```bash
# Выдайте права пользователю
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE vpn_bot TO vpn_bot_user;"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO vpn_bot_user;"
```

### Проблема: Бот не запускается

```bash
# Проверьте переменные окружения
source .env.postgres
echo $PG_USER $PG_PASSWORD $PG_DBNAME

# Проверьте подключение к базе
PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -U "$PG_USER" -d "$PG_DBNAME" -c "SELECT 1;"

# Запустите с логированием
go run main.go 2>&1 | tee bot.log
```

## 📈 Оптимизация PostgreSQL

### Настройка postgresql.conf

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

### Перезапуск PostgreSQL

```bash
sudo systemctl restart postgresql
```

## 🔄 Автоматические бэкапы

### Настройка cron для бэкапов

```bash
# Добавьте в crontab
crontab -e

# Бэкап каждый час
0 * * * * /root/bot/backup_postgres.sh

# Бэкап каждый день в 2:00
0 2 * * * /root/bot/backup_postgres_daily.sh
```

### Скрипт для автоматического бэкапа

```bash
#!/bin/bash
# backup_postgres.sh

source /root/bot/.env.postgres
BACKUP_DIR="/root/bot/backups/auto"
mkdir -p "$BACKUP_DIR"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/vpn_bot_auto_$TIMESTAMP.sql"

PGPASSWORD="$PG_PASSWORD" pg_dump -h "$PG_HOST" -U "$PG_USER" -d "$PG_DBNAME" > "$BACKUP_FILE"

# Удаляем старые бэкапы (старше 7 дней)
find "$BACKUP_DIR" -name "vpn_bot_auto_*.sql" -mtime +7 -delete

echo "Бэкап создан: $BACKUP_FILE"
```

## ✅ Чек-лист после миграции

- [ ] PostgreSQL работает корректно
- [ ] Все данные мигрированы (проверить количество пользователей)
- [ ] Бот запускается без ошибок
- [ ] Все команды бота работают
- [ ] Backup создается автоматически
- [ ] Мониторинг настроен
- [ ] Старые данные на исходном сервере можно удалить

## 🎯 Заключение

После успешной миграции:

1. **MongoDB можно отключить** (если использовалась):
   ```bash
   sudo systemctl stop mongod
   sudo systemctl disable mongod
   ```

2. **Настройте автозапуск PostgreSQL**:
   ```bash
   sudo systemctl enable postgresql
   ```

3. **Обновите скрипты мониторинга** для работы с PostgreSQL

4. **Создайте регулярные бэкапы**

## 📞 Поддержка

Если возникли проблемы:
1. Проверьте раздел "Устранение проблем" выше
2. Изучите логи: `tail -f /var/log/postgresql/postgresql-*-main.log`
3. Проверьте подключение: `psql -h localhost -U vpn_bot_user -d vpn_bot`

🎉 **Поздравляем! Миграция завершена!**

Ваш VPN бот теперь работает на новом сервере с PostgreSQL.
