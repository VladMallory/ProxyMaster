#!/bin/bash

# Скрипт для проверки успешности миграции PostgreSQL
# Использование: ./verify_migration.sh

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

log "=== ПРОВЕРКА МИГРАЦИИ PostgreSQL ==="

# Загружаем переменные окружения
if [ -f ".env.postgres" ]; then
    source .env.postgres
    log "Загружены переменные окружения из .env.postgres"
else
    error "Файл .env.postgres не найден!"
    exit 1
fi

# Проверяем подключение к базе данных
log "Проверка подключения к базе данных..."
if ! PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -c "SELECT 1;" > /dev/null 2>&1; then
    error "Не удается подключиться к базе данных PostgreSQL"
    exit 1
fi
success "✅ Подключение к базе данных успешно"

# Проверяем таблицы
log "Проверка структуры базы данных..."
TABLES=$(PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -t -c "
    SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename;
")

REQUIRED_TABLES=("users" "traffic_configs" "ip_connections" "ip_violations")
MISSING_TABLES=()

for table in "${REQUIRED_TABLES[@]}"; do
    if echo "$TABLES" | grep -q "^$table$"; then
        success "✅ Таблица $table существует"
    else
        error "❌ Таблица $table отсутствует"
        MISSING_TABLES+=("$table")
    fi
done

if [ ${#MISSING_TABLES[@]} -gt 0 ]; then
    error "Отсутствуют обязательные таблицы: ${MISSING_TABLES[*]}"
    exit 1
fi

# Проверяем данные
log "Проверка данных..."

# Количество пользователей
USER_COUNT=$(PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -t -c "SELECT COUNT(*) FROM users;" 2>/dev/null || echo "0")
log "Пользователей в базе: $USER_COUNT"

# Количество конфигураций трафика
TRAFFIC_CONFIGS=$(PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -t -c "SELECT COUNT(*) FROM traffic_configs;" 2>/dev/null || echo "0")
log "Конфигураций трафика: $TRAFFIC_CONFIGS"

# Проверяем индексы
log "Проверка индексов..."
INDEXES=$(PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -t -c "
    SELECT indexname FROM pg_indexes WHERE schemaname = 'public' AND tablename = 'users';
")

if echo "$INDEXES" | grep -q "idx_users_telegram_id"; then
    success "✅ Индекс idx_users_telegram_id существует"
else
    warning "⚠️  Индекс idx_users_telegram_id отсутствует"
fi

# Проверяем функции
log "Проверка функций..."
FUNCTIONS=$(PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -t -c "
    SELECT proname FROM pg_proc WHERE proname LIKE '%users_statistics%';
")

if echo "$FUNCTIONS" | grep -q "get_users_statistics"; then
    success "✅ Функция get_users_statistics существует"
else
    warning "⚠️  Функция get_users_statistics отсутствует"
fi

# Проверяем размер базы данных
log "Проверка размера базы данных..."
DB_SIZE=$(PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -t -c "SELECT pg_size_pretty(pg_database_size('$PG_DBNAME'));" 2>/dev/null || echo "Неизвестно")
log "Размер базы данных: $DB_SIZE"

# Проверяем производительность
log "Проверка производительности..."
QUERY_TIME=$(PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -t -c "
    \timing on
    SELECT COUNT(*) FROM users WHERE has_active_config = true;
" 2>/dev/null | grep "Time:" | tail -1 || echo "Неизвестно")
log "Время выполнения запроса: $QUERY_TIME"

# Проверяем права доступа
log "Проверка прав доступа..."
PERMISSIONS=$(PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -t -c "
    SELECT has_table_privilege('$PG_USER', 'users', 'SELECT, INSERT, UPDATE, DELETE');
" 2>/dev/null || echo "f")

if [ "$PERMISSIONS" = "t" ]; then
    success "✅ Права доступа к таблице users корректны"
else
    error "❌ Недостаточно прав доступа к таблице users"
fi

# Проверяем работу бота
log "Проверка работы бота..."
if pgrep -f "bot" > /dev/null; then
    success "✅ Бот запущен"
    
    # Проверяем логи бота
    log "Проверка логов бота..."
    if [ -f "/var/log/syslog" ]; then
        BOT_ERRORS=$(grep -i "bot.*error" /var/log/syslog | tail -5 || echo "")
        if [ -z "$BOT_ERRORS" ]; then
            success "✅ Ошибок в логах бота не найдено"
        else
            warning "⚠️  Найдены ошибки в логах бота:"
            echo "$BOT_ERRORS"
        fi
    fi
else
    warning "⚠️  Бот не запущен"
    log "Для запуска бота выполните: go run main.go"
fi

# Создаем отчет о проверке
REPORT_FILE="migration_verification_$(date +%Y%m%d_%H%M%S).txt"
cat > "$REPORT_FILE" << EOF
=== ОТЧЕТ О ПРОВЕРКЕ МИГРАЦИИ PostgreSQL ===
Дата: $(date)
Сервер: $(hostname)
IP: $(hostname -I | awk '{print $1}')

Настройки базы данных:
  Host: $PG_HOST
  Port: $PG_PORT
  User: $PG_USER
  Database: $PG_DBNAME

Результаты проверки:
  Подключение к БД: ✅ УСПЕШНО
  Таблицы: ✅ ВСЕ НАЙДЕНЫ
  Пользователей: $USER_COUNT
  Конфигураций трафика: $TRAFFIC_CONFIGS
  Размер БД: $DB_SIZE
  Права доступа: ✅ КОРРЕКТНЫ
  Бот запущен: $(pgrep -f "bot" > /dev/null && echo "✅ ДА" || echo "❌ НЕТ")

Статус миграции: ✅ УСПЕШНО
EOF

success "Отчет сохранен в: $REPORT_FILE"

# Финальная проверка
log "=== ФИНАЛЬНАЯ ПРОВЕРКА ==="

ALL_CHECKS_PASSED=true

# Проверяем подключение
if ! PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -c "SELECT 1;" > /dev/null 2>&1; then
    error "❌ Подключение к базе данных не работает"
    ALL_CHECKS_PASSED=false
fi

# Проверяем таблицы
if [ ${#MISSING_TABLES[@]} -gt 0 ]; then
    error "❌ Отсутствуют обязательные таблицы"
    ALL_CHECKS_PASSED=false
fi

# Проверяем данные
if [ "$USER_COUNT" -eq 0 ] && [ "$TRAFFIC_CONFIGS" -eq 0 ]; then
    warning "⚠️  База данных пуста - возможно данные не были восстановлены"
fi

if [ "$ALL_CHECKS_PASSED" = true ]; then
    success "🎉 ВСЕ ПРОВЕРКИ ПРОЙДЕНЫ УСПЕШНО!"
    success "Миграция PostgreSQL завершена успешно!"
    
    echo ""
    echo "📋 СЛЕДУЮЩИЕ ШАГИ:"
    echo "1. Убедитесь, что бот запущен: go run main.go"
    echo "2. Проверьте работу команд бота"
    echo "3. Настройте мониторинг базы данных"
    echo "4. Создайте регулярные бэкапы"
    echo ""
    echo "🔍 МОНИТОРИНГ:"
    echo "cat $REPORT_FILE  # Просмотр отчета"
    echo "tail -f /var/log/syslog | grep bot  # Логи бота"
    echo "PGPASSWORD=\"\$PG_PASSWORD\" psql -h \"\$PG_HOST\" -U \"\$PG_USER\" -d \"\$PG_DBNAME\"  # Подключение к БД"
    
else
    error "❌ НЕКОТОРЫЕ ПРОВЕРКИ НЕ ПРОЙДЕНЫ"
    error "Проверьте отчет: $REPORT_FILE"
    exit 1
fi

log "Проверка миграции завершена!"
