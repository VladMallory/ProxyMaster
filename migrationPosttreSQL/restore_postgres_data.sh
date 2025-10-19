#!/bin/bash

# Скрипт для восстановления данных PostgreSQL на новом сервере
# Использование: ./restore_postgres_data.sh BACKUP_FILE

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
    error "Использование: $0 BACKUP_FILE"
    error "Пример: $0 vpn_bot_full_backup.sql"
    exit 1
fi

BACKUP_FILE="$1"

log "=== ВОССТАНОВЛЕНИЕ ДАННЫХ PostgreSQL ==="
log "Файл бэкапа: $BACKUP_FILE"

# Проверяем существование файла бэкапа
if [ ! -f "$BACKUP_FILE" ]; then
    error "Файл бэкапа не найден: $BACKUP_FILE"
    exit 1
fi

# Загружаем переменные окружения
if [ -f ".env.postgres" ]; then
    source .env.postgres
    log "Загружены переменные окружения из .env.postgres"
else
    error "Файл .env.postgres не найден!"
    exit 1
fi

log "Настройки PostgreSQL:"
log "  Host: $PG_HOST"
log "  Port: $PG_PORT"
log "  User: $PG_USER"
log "  Database: $PG_DBNAME"

# Проверяем подключение к базе данных
log "Проверка подключения к базе данных..."
if ! PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -c "SELECT 1;" > /dev/null 2>&1; then
    error "Не удается подключиться к базе данных PostgreSQL"
    error "Убедитесь, что PostgreSQL запущен и настройки корректны"
    exit 1
fi
success "Подключение к базе данных успешно"

# Создаем бэкап текущих данных (на всякий случай)
log "Создание бэкапа текущих данных..."
CURRENT_BACKUP="backup_before_restore_$(date +%Y%m%d_%H%M%S).sql"
if PGPASSWORD="$PG_PASSWORD" pg_dump -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" > "$CURRENT_BACKUP"; then
    success "Текущие данные сохранены в: $CURRENT_BACKUP"
else
    warning "Не удалось создать бэкап текущих данных"
fi

# Определяем тип бэкапа
if grep -q "CREATE DATABASE" "$BACKUP_FILE"; then
    BACKUP_TYPE="full"
    log "Обнаружен полный бэкап (с созданием базы данных)"
elif grep -q "INSERT INTO" "$BACKUP_FILE"; then
    BACKUP_TYPE="data"
    log "Обнаружен бэкап данных (только INSERT запросы)"
else
    BACKUP_TYPE="schema"
    log "Обнаружен бэкап схемы"
fi

# Останавливаем бота если запущен
log "Проверка запущенных процессов бота..."
if pgrep -f "bot" > /dev/null; then
    warning "Обнаружен запущенный бот, останавливаем..."
    pkill -f "bot" || true
    sleep 2
    log "Бот остановлен"
fi

# Восстанавливаем данные в зависимости от типа бэкапа
case $BACKUP_TYPE in
    "full")
        log "Восстановление полного бэкапа..."
        log "ВНИМАНИЕ: Это пересоздаст базу данных!"
        
        # Создаем временную базу данных
        TEMP_DB="vpn_bot_temp_$(date +%Y%m%d_%H%M%S)"
        log "Создание временной базы данных: $TEMP_DB"
        
        sudo -u postgres psql -c "CREATE DATABASE $TEMP_DB OWNER $PG_USER;" || true
        
        # Восстанавливаем в временную базу
        log "Восстановление в временную базу данных..."
        if PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$TEMP_DB" < "$BACKUP_FILE"; then
            success "Данные восстановлены в временную базу"
            
            # Переименовываем базы данных
            log "Переключение на восстановленную базу данных..."
            sudo -u postgres psql -c "ALTER DATABASE $PG_DBNAME RENAME TO ${PG_DBNAME}_old;"
            sudo -u postgres psql -c "ALTER DATABASE $TEMP_DB RENAME TO $PG_DBNAME;"
            
            success "База данных успешно переключена"
        else
            error "Ошибка восстановления полного бэкапа"
            # Восстанавливаем старую базу
            sudo -u postgres psql -c "DROP DATABASE IF EXISTS $TEMP_DB;"
            exit 1
        fi
        ;;
        
    "data")
        log "Восстановление данных..."
        
        # Очищаем существующие данные
        log "Очистка существующих данных..."
        PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -c "
            TRUNCATE TABLE ip_violations CASCADE;
            TRUNCATE TABLE ip_connections CASCADE;
            TRUNCATE TABLE users CASCADE;
            TRUNCATE TABLE traffic_configs CASCADE;
        " || warning "Не удалось очистить некоторые таблицы"
        
        # Восстанавливаем данные
        if PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" < "$BACKUP_FILE"; then
            success "Данные восстановлены успешно"
        else
            error "Ошибка восстановления данных"
            exit 1
        fi
        ;;
        
    "schema")
        log "Восстановление схемы..."
        if PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" < "$BACKUP_FILE"; then
            success "Схема восстановлена успешно"
        else
            error "Ошибка восстановления схемы"
            exit 1
        fi
        ;;
esac

# Проверяем восстановление
log "Проверка восстановления..."

# Проверяем подключение
if ! PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -c "SELECT 1;" > /dev/null 2>&1; then
    error "Ошибка подключения после восстановления"
    exit 1
fi

# Проверяем таблицы
log "Проверка таблиц..."
TABLES=$(PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -t -c "
    SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename;
")

if [ -z "$TABLES" ]; then
    error "Таблицы не найдены после восстановления"
    exit 1
fi

log "Найденные таблицы:"
echo "$TABLES" | while read table; do
    if [ ! -z "$table" ]; then
        log "  - $table"
    fi
done

# Проверяем количество пользователей
USER_COUNT=$(PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -t -c "SELECT COUNT(*) FROM users;" 2>/dev/null || echo "0")
log "Восстановлено пользователей: $USER_COUNT"

# Проверяем конфигурации трафика
TRAFFIC_CONFIGS=$(PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -t -c "SELECT COUNT(*) FROM traffic_configs;" 2>/dev/null || echo "0")
log "Конфигураций трафика: $TRAFFIC_CONFIGS"

# Создаем отчет о восстановлении
REPORT_FILE="restore_report_$(date +%Y%m%d_%H%M%S).txt"
cat > "$REPORT_FILE" << EOF
=== ОТЧЕТ О ВОССТАНОВЛЕНИИ PostgreSQL ===
Дата: $(date)
Файл бэкапа: $BACKUP_FILE
Тип бэкапа: $BACKUP_TYPE

Настройки базы данных:
  Host: $PG_HOST
  Port: $PG_PORT
  User: $PG_USER
  Database: $PG_DBNAME

Результаты восстановления:
  Пользователей: $USER_COUNT
  Конфигураций трафика: $TRAFFIC_CONFIGS
  Таблиц: $(echo "$TABLES" | wc -l)

Файлы:
  Текущий бэкап: $CURRENT_BACKUP
  Отчет: $REPORT_FILE

Статус: УСПЕШНО
EOF

success "Отчет сохранен в: $REPORT_FILE"

# Проверяем целостность данных
log "Проверка целостности данных..."
if PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DBNAME" -c "VACUUM ANALYZE;" > /dev/null 2>&1; then
    success "Проверка целостности прошла успешно"
else
    warning "Предупреждения при проверке целостности"
fi

success "=== ВОССТАНОВЛЕНИЕ ЗАВЕРШЕНО ==="

echo ""
echo "📋 СЛЕДУЮЩИЕ ШАГИ:"
echo "1. Проверьте данные: PGPASSWORD=\"\$PG_PASSWORD\" psql -h \"\$PG_HOST\" -U \"\$PG_USER\" -d \"\$PG_DBNAME\" -c \"SELECT COUNT(*) FROM users;\""
echo "2. Запустите бота: go run main.go"
echo "3. Проверьте работу бота"
echo ""
echo "🔍 ПРОВЕРКА:"
echo "cat $REPORT_FILE  # Просмотр отчета"
echo "tail -f /var/log/syslog | grep bot  # Логи бота"
echo ""

log "Восстановление данных завершено успешно!"
