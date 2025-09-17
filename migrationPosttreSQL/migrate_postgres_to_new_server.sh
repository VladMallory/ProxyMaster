#!/bin/bash

# Скрипт для миграции PostgreSQL VPN бота на новый сервер
# Использование: ./migrate_postgres_to_new_server.sh NEW_SERVER_IP

set -e  # Остановка при ошибке

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Функция для вывода сообщений
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Проверка параметров
if [ $# -ne 1 ]; then
    error "Использование: $0 NEW_SERVER_IP"
    error "Пример: $0 192.168.1.100"
    exit 1
fi

NEW_SERVER_IP="$1"
BACKUP_DIR="./backups/migration_$(date +%Y%m%d_%H%M%S)"
CURRENT_DIR="/root/bot"

log "=== МИГРАЦИЯ PostgreSQL VPN БОТА НА НОВЫЙ СЕРВЕР ==="
log "Целевой сервер: $NEW_SERVER_IP"
log "Директория бэкапа: $BACKUP_DIR"

# Проверяем, что мы в правильной директории
if [ ! -f "main.go" ] || [ ! -f "common/postgres.go" ]; then
    error "Запустите скрипт из директории /root/bot"
    exit 1
fi

# Создаем директорию для бэкапа
mkdir -p "$BACKUP_DIR"

# Загружаем переменные окружения
if [ -f ".env.postgres" ]; then
    source .env.postgres
    log "Загружены переменные окружения из .env.postgres"
else
    error "Файл .env.postgres не найден!"
    exit 1
fi

log "Текущие настройки PostgreSQL:"
log "  Host: $PG_HOST"
log "  Port: $PG_PORT"
log "  User: $PG_USER"
log "  Database: $PG_DBNAME"

# Проверяем подключение к текущей базе данных
log "Проверка подключения к текущей базе данных..."
if ! PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -c "SELECT 1;" > /dev/null 2>&1; then
    error "Не удается подключиться к текущей базе данных PostgreSQL"
    exit 1
fi
success "Подключение к текущей базе данных успешно"

# Останавливаем бота
log "Остановка VPN бота..."
if pgrep -f "bot" > /dev/null; then
    pkill -f "bot" || true
    sleep 2
    log "Бот остановлен"
else
    warning "Бот не запущен"
fi

# Создаем полный бэкап базы данных
log "Создание полного бэкапа базы данных..."
BACKUP_FILE="$BACKUP_DIR/vpn_bot_full_backup.sql"

if PGPASSWORD="$PG_PASSWORD" pg_dump -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" --clean --create --if-exists > "$BACKUP_FILE"; then
    success "Бэкап создан: $BACKUP_FILE"
else
    error "Ошибка создания бэкапа"
    exit 1
fi

# Создаем бэкап только данных (без схемы)
log "Создание бэкапа данных..."
DATA_BACKUP_FILE="$BACKUP_DIR/vpn_bot_data_only.sql"

if PGPASSWORD="$PG_PASSWORD" pg_dump -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" --data-only --inserts > "$DATA_BACKUP_FILE"; then
    success "Бэкап данных создан: $DATA_BACKUP_FILE"
else
    error "Ошибка создания бэкапа данных"
    exit 1
fi

# Создаем бэкап схемы
log "Создание бэкапа схемы..."
SCHEMA_BACKUP_FILE="$BACKUP_DIR/vpn_bot_schema_only.sql"

if PGPASSWORD="$PG_PASSWORD" pg_dump -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" --schema-only > "$SCHEMA_BACKUP_FILE"; then
    success "Бэкап схемы создан: $SCHEMA_BACKUP_FILE"
else
    error "Ошибка создания бэкапа схемы"
    exit 1
fi

# Копируем конфигурационные файлы
log "Копирование конфигурационных файлов..."
cp .env.postgres "$BACKUP_DIR/"
cp postgres_schema.sql "$BACKUP_DIR/" 2>/dev/null || warning "postgres_schema.sql не найден"
cp setup_postgres.sh "$BACKUP_DIR/" 2>/dev/null || warning "setup_postgres.sh не найден"
cp -r common/ "$BACKUP_DIR/" 2>/dev/null || warning "Директория common/ не найдена"

# Создаем архив для передачи
log "Создание архива для передачи..."
ARCHIVE_FILE="vpn_bot_migration_$(date +%Y%m%d_%H%M%S).tar.gz"
cd "$BACKUP_DIR"
tar -czf "../$ARCHIVE_FILE" .
cd "$CURRENT_DIR"

success "Архив создан: $ARCHIVE_FILE"

# Копируем архив на новый сервер
log "Копирование архива на новый сервер..."
if scp "$ARCHIVE_FILE" "root@$NEW_SERVER_IP:/tmp/"; then
    success "Архив скопирован на новый сервер"
else
    error "Ошибка копирования архива на новый сервер"
    error "Убедитесь, что SSH ключи настроены и сервер доступен"
    exit 1
fi

# Создаем скрипт для восстановления на новом сервере
log "Создание скрипта восстановления для нового сервера..."
cat > "restore_on_new_server.sh" << EOF
#!/bin/bash
# Скрипт для восстановления PostgreSQL на новом сервере

set -e

echo "=== ВОССТАНОВЛЕНИЕ PostgreSQL VPN БОТА ==="

# Распаковываем архив
cd /tmp
ARCHIVE_FILE=\$(ls vpn_bot_migration_*.tar.gz | head -1)
if [ -z "\$ARCHIVE_FILE" ]; then
    echo "Ошибка: архив миграции не найден в /tmp/"
    exit 1
fi

echo "Распаковка архива: \$ARCHIVE_FILE"
tar -xzf "\$ARCHIVE_FILE"

# Устанавливаем PostgreSQL если не установлен
if ! command -v psql &> /dev/null; then
    echo "Установка PostgreSQL..."
    apt update
    apt install -y postgresql postgresql-contrib
    systemctl start postgresql
    systemctl enable postgresql
fi

# Загружаем переменные окружения
source .env.postgres

# Создаем пользователя и базу данных
echo "Создание пользователя и базы данных..."
sudo -u postgres psql -c "CREATE USER \$PG_USER WITH ENCRYPTED PASSWORD '\$PG_PASSWORD';" || true
sudo -u postgres psql -c "CREATE DATABASE \$PG_DBNAME OWNER \$PG_USER;" || true
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE \$PG_DBNAME TO \$PG_USER;"

# Восстанавливаем схему
echo "Восстановление схемы базы данных..."
if [ -f "postgres_schema.sql" ]; then
    sudo -u postgres psql -d "\$PG_DBNAME" -f postgres_schema.sql
else
    echo "Восстановление из бэкапа схемы..."
    PGPASSWORD="\$PG_PASSWORD" psql -h localhost -U "\$PG_USER" -d "\$PG_DBNAME" < vpn_bot_schema_only.sql
fi

# Восстанавливаем данные
echo "Восстановление данных..."
PGPASSWORD="\$PG_PASSWORD" psql -h localhost -U "\$PG_USER" -d "\$PG_DBNAME" < vpn_bot_data_only.sql

# Проверяем восстановление
echo "Проверка восстановления..."
USER_COUNT=\$(PGPASSWORD="\$PG_PASSWORD" psql -h localhost -U "\$PG_USER" -d "\$PG_DBNAME" -t -c "SELECT COUNT(*) FROM users;")
echo "Восстановлено пользователей: \$USER_COUNT"

# Копируем проект на новый сервер
echo "Копирование проекта..."
if [ -d "/root/bot" ]; then
    echo "Директория /root/bot уже существует, создаем бэкап..."
    mv /root/bot /root/bot.backup.\$(date +%Y%m%d_%H%M%S)
fi

# Здесь нужно скопировать весь проект на новый сервер
echo "ВНИМАНИЕ: Скопируйте весь проект /root/bot на новый сервер вручную"
echo "Или используйте rsync: rsync -avz /root/bot/ root@\$NEW_SERVER_IP:/root/bot/"

echo "=== ВОССТАНОВЛЕНИЕ ЗАВЕРШЕНО ==="
echo "Следующие шаги:"
echo "1. Скопируйте проект на новый сервер"
echo "2. Обновите .env.postgres с новыми настройками"
echo "3. Запустите бота: go run main.go"
EOF

chmod +x "restore_on_new_server.sh"

# Копируем скрипт восстановления на новый сервер
log "Копирование скрипта восстановления на новый сервер..."
scp "restore_on_new_server.sh" "root@$NEW_SERVER_IP:/tmp/"

# Создаем инструкции для завершения миграции
log "Создание инструкций для завершения миграции..."
cat > "MIGRATION_INSTRUCTIONS.md" << EOF
# Инструкции по завершению миграции PostgreSQL

## Что было сделано:
1. ✅ Создан полный бэкап базы данных
2. ✅ Создан бэкап только данных
3. ✅ Создан бэкап схемы
4. ✅ Скопированы конфигурационные файлы
5. ✅ Создан архив для передачи
6. ✅ Архив скопирован на новый сервер: $NEW_SERVER_IP
7. ✅ Скрипт восстановления скопирован на новый сервер

## Следующие шаги на НОВОМ СЕРВЕРЕ:

### 1. Подключитесь к новому серверу:
\`\`\`bash
ssh root@$NEW_SERVER_IP
\`\`\`

### 2. Выполните восстановление:
\`\`\`bash
cd /tmp
chmod +x restore_on_new_server.sh
./restore_on_new_server.sh
\`\`\`

### 3. Скопируйте проект на новый сервер:
\`\`\`bash
# С текущего сервера выполните:
rsync -avz --exclude='backups/' /root/bot/ root@$NEW_SERVER_IP:/root/bot/
\`\`\`

### 4. На новом сервере обновите настройки:
\`\`\`bash
cd /root/bot
# Обновите .env.postgres если нужно
nano .env.postgres

# Загрузите переменные окружения
source .env.postgres

# Обновите зависимости
go mod tidy
\`\`\`

### 5. Запустите бота:
\`\`\`bash
go run main.go
\`\`\`

### 6. Проверьте работу:
\`\`\`bash
# Проверьте подключение к базе данных
PGPASSWORD="\$PG_PASSWORD" psql -h localhost -U "\$PG_USER" -d "\$PG_DBNAME" -c "SELECT COUNT(*) FROM users;"

# Проверьте логи бота
tail -f /var/log/syslog | grep bot
\`\`\`

## Файлы бэкапа:
- Полный бэкап: $BACKUP_FILE
- Только данные: $DATA_BACKUP_FILE
- Только схема: $SCHEMA_BACKUP_FILE
- Архив: $ARCHIVE_FILE

## Откат (если что-то пошло не так):
\`\`\`bash
# Остановите бота на новом сервере
pkill -f bot

# Восстановите из полного бэкапа
PGPASSWORD="\$PG_PASSWORD" psql -h localhost -U "\$PG_USER" -d "\$PG_DBNAME" < $BACKUP_FILE

# Запустите бота
go run main.go
\`\`\`
EOF

success "=== МИГРАЦИЯ ПОДГОТОВЛЕНА ==="
log "Архив создан: $ARCHIVE_FILE"
log "Скрипт восстановления: restore_on_new_server.sh"
log "Инструкции: MIGRATION_INSTRUCTIONS.md"

echo ""
echo "Следующие шаги:"
echo "1. Подключитесь к новому серверу: ssh root@$NEW_SERVER_IP"
echo "2. Выполните: cd /tmp && ./restore_on_new_server.sh"
echo "3. Скопируйте проект: rsync -avz --exclude='backups/' /root/bot/ root@$NEW_SERVER_IP:/root/bot/"
echo "4. Запустите бота на новом сервере"

log "Миграция подготовлена успешно!"
