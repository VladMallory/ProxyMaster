#!/bin/bash

# Скрипт для настройки PostgreSQL для VPN бота
# Выполните этот скрипт для подготовки базы данных

echo "=== НАСТРОЙКА PostgreSQL ДЛЯ VPN БОТА ==="

# Проверяем, установлен ли PostgreSQL
if ! command -v psql &> /dev/null; then
    echo "❌ PostgreSQL не установлен. Пожалуйста, установите PostgreSQL:"
    echo "   Ubuntu/Debian: sudo apt update && sudo apt install postgresql postgresql-contrib"
    echo "   CentOS/RHEL: sudo yum install postgresql-server postgresql-contrib"
    echo "   или следуйте инструкциям на https://www.postgresql.org/download/"
    exit 1
fi

echo "✅ PostgreSQL найден"

# Переменные
DB_NAME="vpn_bot"
DB_USER="vpn_bot_user"
DB_PASSWORD="your_secure_password"

echo ""
echo "📋 Настройки базы данных:"
echo "   База данных: $DB_NAME"
echo "   Пользователь: $DB_USER"
echo "   Пароль: $DB_PASSWORD"
echo ""

# Проверяем переменные окружения
if [ ! -z "$PG_PASSWORD" ]; then
    DB_PASSWORD="$PG_PASSWORD"
    echo "🔧 Используется пароль из переменной окружения PG_PASSWORD"
fi

if [ ! -z "$PG_USER" ]; then
    DB_USER="$PG_USER"
    echo "🔧 Используется пользователь из переменной окружения PG_USER"
fi

if [ ! -z "$PG_DBNAME" ]; then
    DB_NAME="$PG_DBNAME"
    echo "🔧 Используется база данных из переменной окружения PG_DBNAME"
fi

echo ""
read -p "Продолжить настройку? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Настройка отменена"
    exit 1
fi

echo ""
echo "🚀 Начинаем настройку..."

# Функция для выполнения SQL команд как postgres пользователь
run_sql() {
    echo "   Выполняется: $1"
    sudo -u postgres psql -c "$1"
}

# Функция для выполнения SQL из файла
run_sql_file() {
    echo "   Выполняется SQL файл: $1"
    sudo -u postgres psql -d "$DB_NAME" -f "$1"
}

echo ""
echo "1️⃣ Создание пользователя базы данных..."
run_sql "CREATE USER $DB_USER WITH ENCRYPTED PASSWORD '$DB_PASSWORD';" || true

echo ""
echo "2️⃣ Создание базы данных..."
run_sql "CREATE DATABASE $DB_NAME OWNER $DB_USER;" || true

echo ""
echo "3️⃣ Выдача прав пользователю..."
run_sql "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;"
run_sql "ALTER USER $DB_USER CREATEDB;"

echo ""
echo "4️⃣ Создание схемы базы данных..."
if [ -f "postgres_schema.sql" ]; then
    run_sql_file "postgres_schema.sql"
    echo "✅ Схема создана из postgres_schema.sql"
else
    echo "❌ Файл postgres_schema.sql не найден!"
    echo "   Пожалуйста, выполните postgres_schema.sql вручную"
fi

echo ""
echo "5️⃣ Выдача дополнительных прав..."
sudo -u postgres psql -d "$DB_NAME" -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $DB_USER;"
sudo -u postgres psql -d "$DB_NAME" -c "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $DB_USER;"
sudo -u postgres psql -d "$DB_NAME" -c "GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO $DB_USER;"

echo ""
echo "6️⃣ Настройка соединений..."

# Обновляем pg_hba.conf для локальных соединений
PG_VERSION=$(sudo -u postgres psql -t -c "SELECT version();" | grep -oE '[0-9]+\.[0-9]+' | head -1)
PG_HBA_PATH="/etc/postgresql/$PG_VERSION/main/pg_hba.conf"

if [ -f "$PG_HBA_PATH" ]; then
    echo "   Обновляем $PG_HBA_PATH..."
    
    # Проверяем, есть ли уже настройка для нашего пользователя
    if ! grep -q "$DB_USER" "$PG_HBA_PATH"; then
        echo "local   $DB_NAME      $DB_USER                                md5" | sudo tee -a "$PG_HBA_PATH"
        echo "host    $DB_NAME      $DB_USER        127.0.0.1/32            md5" | sudo tee -a "$PG_HBA_PATH"
        echo "host    $DB_NAME      $DB_USER        ::1/128                 md5" | sudo tee -a "$PG_HBA_PATH"
    fi
    
    echo "   Перезапускаем PostgreSQL..."
    sudo systemctl reload postgresql
else
    echo "⚠️  Не удалось найти pg_hba.conf, возможно потребуется настройка соединений вручную"
fi

echo ""
echo "7️⃣ Тестирование соединения..."
export PGPASSWORD="$DB_PASSWORD"
if psql -h localhost -U "$DB_USER" -d "$DB_NAME" -c "SELECT 'Подключение успешно!' as status;"; then
    echo "✅ Тестовое подключение успешно!"
else
    echo "❌ Ошибка подключения. Проверьте настройки."
fi

echo ""
echo "📝 Создание переменных окружения..."
cat > .env.postgres << EOF
# PostgreSQL настройки для VPN бота
export PG_HOST=localhost
export PG_PORT=5432
export PG_USER=$DB_USER
export PG_PASSWORD=$DB_PASSWORD
export PG_DBNAME=$DB_NAME
EOF

echo "✅ Переменные окружения сохранены в .env.postgres"
echo "   Для использования выполните: source .env.postgres"

echo ""
echo "🎯 ДОПОЛНИТЕЛЬНЫЕ РЕКОМЕНДАЦИИ:"
echo ""
echo "1. Для безопасности измените пароль:"
echo "   sudo -u postgres psql -c \"ALTER USER $DB_USER WITH PASSWORD 'ваш_новый_пароль';\""
echo ""
echo "2. Для использования переменных окружения:"
echo "   source .env.postgres"
echo ""
echo "3. Для миграции данных из MongoDB:"
echo "   go run migrate_to_postgres.go"
echo ""
echo "4. Для тестирования нового бота:"
echo "   go run main.go"
echo ""

if [ -f "migrate_to_postgres.go" ]; then
    echo "🔄 МИГРАЦИЯ ДАННЫХ:"
    echo ""
    read -p "Выполнить миграцию данных из MongoDB сейчас? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "🚀 Запускаем миграцию..."
        echo "   Примечание: убедитесь что MongoDB запущена и доступна"
        echo ""
        
        if [ -f "migration_go.mod" ]; then
            mv migration_go.mod go.mod.migration
            echo "go mod tidy -modfile=go.mod.migration"
            go mod tidy -modfile=go.mod.migration
            echo "go run -modfile=go.mod.migration migrate_to_postgres.go"
            go run -modfile=go.mod.migration migrate_to_postgres.go
            mv go.mod.migration migration_go.mod
        else
            echo "❌ Файл migration_go.mod не найден!"
        fi
    fi
fi

echo ""
echo "🎉 НАСТРОЙКА PostgreSQL ЗАВЕРШЕНА!"
echo ""
echo "📋 СЛЕДУЮЩИЕ ШАГИ:"
echo "1. source .env.postgres              # Загрузить переменные окружения"
echo "2. go mod tidy                      # Обновить зависимости Go"
echo "3. go run main.go                   # Запустить бота"
echo ""
echo "🔍 ПРОВЕРКА СТАТУСА:"
echo "systemctl status postgresql         # Статус PostgreSQL"
echo "psql -h localhost -U $DB_USER -d $DB_NAME  # Подключение к БД"
echo ""

exit 0
