# ⚡ Быстрая миграция PostgreSQL на новый сервер

## 🚀 За 3 команды

### На текущем сервере:
```bash
cd /root/bot
./migrate_postgres_to_new_server.sh NEW_SERVER_IP
```

### На новом сервере:
```bash
ssh root@NEW_SERVER_IP
cd /tmp && tar -xzf vpn_bot_migration_*.tar.gz && ./restore_on_new_server.sh
```

### Копирование проекта:
```bash
rsync -avz --exclude='backups/' /root/bot/ root@NEW_SERVER_IP:/root/bot/
```

## 📋 Что делают скрипты

| Скрипт | Описание |
|--------|----------|
| `migrate_postgres_to_new_server.sh` | Создает бэкап и копирует на новый сервер |
| `setup_postgres_new_server.sh` | Настраивает PostgreSQL на новом сервере |
| `restore_postgres_data.sh` | Восстанавливает данные из бэкапа |
| `verify_migration.sh` | Проверяет успешность миграции |

## ✅ Проверка

```bash
# Проверьте миграцию
./verify_migration.sh

# Запустите бота
source .env.postgres && go run main.go
```

## 🔧 Если что-то пошло не так

```bash
# Проверьте логи
tail -f /var/log/syslog | grep bot

# Проверьте базу данных
PGPASSWORD="$PG_PASSWORD" psql -h localhost -U vpn_bot_user -d vpn_bot -c "SELECT COUNT(*) FROM users;"
```

**Готово!** 🎉
