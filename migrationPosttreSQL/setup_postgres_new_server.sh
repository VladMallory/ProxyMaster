#!/bin/bash

# Скрипт для настройки PostgreSQL на новом сервере для VPN бота
# Использование: ./setup_postgres_new_server.sh [DB_PASSWORD]

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

log "=== НАСТРОЙКА PostgreSQL НА НОВОМ СЕРВЕРЕ ==="

# Параметры базы данных
DB_NAME="vpn_bot"
DB_USER="vpn_bot_user"
DB_PASSWORD="${1:-$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)}"

log "Настройки базы данных:"
log "  База данных: $DB_NAME"
log "  Пользователь: $DB_USER"
log "  Пароль: $DB_PASSWORD"

# Проверяем, установлен ли PostgreSQL
if ! command -v psql &> /dev/null; then
    log "Установка PostgreSQL..."
    
    # Обновляем пакеты
    apt update
    
    # Устанавливаем PostgreSQL
    apt install -y postgresql postgresql-contrib
    
    # Запускаем и включаем автозапуск
    systemctl start postgresql
    systemctl enable postgresql
    
    success "PostgreSQL установлен и запущен"
else
    log "PostgreSQL уже установлен"
fi

# Проверяем статус PostgreSQL
if ! systemctl is-active --quiet postgresql; then
    log "Запуск PostgreSQL..."
    systemctl start postgresql
fi

success "PostgreSQL запущен"

# Функция для выполнения SQL команд как postgres пользователь
run_sql() {
    log "Выполняется: $1"
    sudo -u postgres psql -c "$1"
}

# Функция для выполнения SQL из файла
run_sql_file() {
    log "Выполняется SQL файл: $1"
    sudo -u postgres psql -d "$DB_NAME" -f "$1"
}

log "Создание пользователя базы данных..."
run_sql "CREATE USER $DB_USER WITH ENCRYPTED PASSWORD '$DB_PASSWORD';" || warning "Пользователь уже существует"

log "Создание базы данных..."
run_sql "CREATE DATABASE $DB_NAME OWNER $DB_USER;" || warning "База данных уже существует"

log "Выдача прав пользователю..."
run_sql "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;"
run_sql "ALTER USER $DB_USER CREATEDB;"

log "Создание схемы базы данных..."
if [ -f "postgres_schema.sql" ]; then
    run_sql_file "postgres_schema.sql"
    success "Схема создана из postgres_schema.sql"
else
    warning "Файл postgres_schema.sql не найден, создаем базовую схему..."
    
    # Создаем базовую схему
    sudo -u postgres psql -d "$DB_NAME" << EOF
-- Основная таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    balance DECIMAL(10,2) DEFAULT 0.00,
    total_paid DECIMAL(10,2) DEFAULT 0.00,
    configs_count INTEGER DEFAULT 0,
    has_active_config BOOLEAN DEFAULT FALSE,
    client_id VARCHAR(255),
    sub_id VARCHAR(255),
    email VARCHAR(255),
    config_created_at TIMESTAMP,
    expiry_time BIGINT,
    has_used_trial BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Настройки трафика
CREATE TABLE IF NOT EXISTS traffic_configs (
    id VARCHAR(50) PRIMARY KEY DEFAULT 'default',
    enabled BOOLEAN DEFAULT TRUE,
    daily_limit_gb INTEGER,
    weekly_limit_gb INTEGER,
    monthly_limit_gb INTEGER,
    limit_gb INTEGER,
    reset_days INTEGER,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- IP подключения
CREATE TABLE IF NOT EXISTS ip_connections (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT,
    ip_address INET,
    connection_data JSONB,
    timestamp TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (telegram_id) REFERENCES users(telegram_id) ON DELETE CASCADE
);

-- IP нарушения
CREATE TABLE IF NOT EXISTS ip_violations (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT,
    ip_address INET,
    is_blocked BOOLEAN DEFAULT FALSE,
    violation_count INTEGER DEFAULT 1,
    violation_type VARCHAR(100),
    violation_data JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (telegram_id) REFERENCES users(telegram_id) ON DELETE CASCADE
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);
CREATE INDEX IF NOT EXISTS idx_users_has_active_config ON users(has_active_config);

-- Вставка конфигурации трафика по умолчанию
INSERT INTO traffic_configs (id, enabled, daily_limit_gb, weekly_limit_gb, monthly_limit_gb, limit_gb, reset_days)
VALUES ('default', true, 0, 0, 0, 0, 30)
ON CONFLICT (id) DO NOTHING;
EOF
    success "Базовая схема создана"
fi

log "Выдача дополнительных прав..."
sudo -u postgres psql -d "$DB_NAME" -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $DB_USER;"
sudo -u postgres psql -d "$DB_NAME" -c "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $DB_USER;"
sudo -u postgres psql -d "$DB_NAME" -c "GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO $DB_USER;"

log "Настройка соединений..."

# Обновляем pg_hba.conf для локальных соединений
PG_VERSION=$(sudo -u postgres psql -t -c "SELECT version();" | grep -oE '[0-9]+\.[0-9]+' | head -1)
PG_HBA_PATH="/etc/postgresql/$PG_VERSION/main/pg_hba.conf"

if [ -f "$PG_HBA_PATH" ]; then
    log "Обновляем $PG_HBA_PATH..."
    
    # Проверяем, есть ли уже настройка для нашего пользователя
    if ! grep -q "$DB_USER" "$PG_HBA_PATH"; then
        echo "local   $DB_NAME      $DB_USER                                md5" | sudo tee -a "$PG_HBA_PATH"
        echo "host    $DB_NAME      $DB_USER        127.0.0.1/32            md5" | sudo tee -a "$PG_HBA_PATH"
        echo "host    $DB_NAME      $DB_USER        ::1/128                 md5" | sudo tee -a "$PG_HBA_PATH"
    fi
    
    log "Перезапускаем PostgreSQL..."
    systemctl reload postgresql
else
    warning "Не удалось найти pg_hba.conf, возможно потребуется настройка соединений вручную"
fi

log "Тестирование соединения..."
export PGPASSWORD="$DB_PASSWORD"
if psql -h localhost -U "$DB_USER" -d "$DB_NAME" -c "SELECT 'Подключение успешно!' as status;" > /dev/null 2>&1; then
    success "Тестовое подключение успешно!"
else
    error "Ошибка подключения. Проверьте настройки."
    exit 1
fi

log "Создание переменных окружения..."
cat > .env.postgres << EOF
# PostgreSQL настройки для VPN бота
export PG_HOST=localhost
export PG_PORT=5432
export PG_USER=$DB_USER
export PG_PASSWORD=$DB_PASSWORD
export PG_DBNAME=$DB_NAME
EOF

success "Переменные окружения сохранены в .env.postgres"

log "Создание скрипта для загрузки переменных окружения..."
cat > load_env.sh << EOF
#!/bin/bash
# Скрипт для загрузки переменных окружения PostgreSQL
source .env.postgres
echo "Переменные окружения PostgreSQL загружены"
echo "PG_HOST=\$PG_HOST"
echo "PG_PORT=\$PG_PORT"
echo "PG_USER=\$PG_USER"
echo "PG_DBNAME=\$PG_DBNAME"
EOF

chmod +x load_env.sh

log "Создание скрипта для проверки базы данных..."
cat > check_db.sh << EOF
#!/bin/bash
# Скрипт для проверки состояния базы данных

source .env.postgres

echo "=== ПРОВЕРКА БАЗЫ ДАННЫХ ==="
echo "Host: \$PG_HOST"
echo "Port: \$PG_PORT"
echo "User: \$PG_USER"
echo "Database: \$PG_DBNAME"
echo ""

echo "Проверка подключения..."
if PGPASSWORD="\$PG_PASSWORD" psql -h "\$PG_HOST" -U "\$PG_USER" -d "\$PG_DBNAME" -c "SELECT 1;" > /dev/null 2>&1; then
    echo "✅ Подключение успешно"
else
    echo "❌ Ошибка подключения"
    exit 1
fi

echo ""
echo "Статистика таблиц:"
PGPASSWORD="\$PG_PASSWORD" psql -h "\$PG_HOST" -U "\$PG_USER" -d "\$PG_DBNAME" -c "
SELECT 
    schemaname,
    tablename,
    n_tup_ins as inserts,
    n_tup_upd as updates,
    n_tup_del as deletes
FROM pg_stat_user_tables
ORDER BY tablename;
"

echo ""
echo "Количество пользователей:"
USER_COUNT=\$(PGPASSWORD="\$PG_PASSWORD" psql -h "\$PG_HOST" -U "\$PG_USER" -d "\$PG_DBNAME" -t -c "SELECT COUNT(*) FROM users;")
echo "Пользователей в базе: \$USER_COUNT"

echo ""
echo "Размер базы данных:"
PGPASSWORD="\$PG_PASSWORD" psql -h "\$PG_HOST" -U "\$PG_USER" -d "\$PG_DBNAME" -c "SELECT pg_size_pretty(pg_database_size('$DB_NAME'));"
EOF

chmod +x check_db.sh

success "=== НАСТРОЙКА PostgreSQL ЗАВЕРШЕНА ==="

echo ""
echo "📋 СЛЕДУЮЩИЕ ШАГИ:"
echo "1. source .env.postgres              # Загрузить переменные окружения"
echo "2. ./check_db.sh                     # Проверить состояние базы данных"
echo "3. go mod tidy                       # Обновить зависимости Go"
echo "4. go run main.go                    # Запустить бота"
echo ""
echo "🔍 ПРОВЕРКА СТАТУСА:"
echo "systemctl status postgresql          # Статус PostgreSQL"
echo "psql -h localhost -U $DB_USER -d $DB_NAME  # Подключение к БД"
echo ""

log "Настройка PostgreSQL завершена успешно!"
