#!/bin/bash

# IP Ban System - Скрипт запуска
# Автор: AI Assistant
# Описание: Удобный скрипт для запуска системы IP бана

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Функция для вывода сообщений
print_message() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

print_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING:${NC} $1"
}

print_info() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] INFO:${NC} $1"
}

# Функция показа справки
show_help() {
    echo "🎯 IP Ban System - Система управления конфигами"
    echo
    echo "Использование: $0 [команда] [опции]"
    echo
    echo "Команды:"
    echo "  start           Запустить сервис мониторинга"
    echo "  stats           Показать статистику"
    echo "  configs         Показать список конфигов"
    echo "  enable EMAIL    Включить конфиг по email"
    echo "  disable EMAIL   Отключить конфиг по email"
    echo "  test            Тестовый запуск (показать что будет сделано)"
    echo "  help            Показать эту справку"
    echo
    echo "Опции:"
    echo "  --max-ips N         Максимум IP на конфиг (по умолчанию: 2)"
    echo "  --check-interval N  Интервал проверки в минутах (по умолчанию: 5)"
    echo "  --grace-period N    Период ожидания в минутах (по умолчанию: 10)"
    echo "  --access-log PATH   Путь к access.log (по умолчанию: /usr/local/x-ui/access.log)"
    echo
    echo "Примеры:"
    echo "  $0 start"
    echo "  $0 stats"
    echo "  $0 enable 123456789"
    echo "  $0 start --max-ips 3 --check-interval 10"
}

# Функция проверки зависимостей
check_dependencies() {
    print_info "Проверка зависимостей..."
    
    if ! command -v go &> /dev/null; then
        print_error "Go не установлен. Установите Go и попробуйте снова."
        exit 1
    fi
    
    if [ ! -f "main.go" ]; then
        print_error "Файл main.go не найден. Запустите скрипт из папки ipBan."
        exit 1
    fi
    
    print_message "✅ Все зависимости найдены"
}

# Функция проверки файла access.log
check_access_log() {
    local log_path=${1:-"/usr/local/x-ui/access.log"}
    
    if [ ! -f "$log_path" ]; then
        print_warning "Файл access.log не найден: $log_path"
        print_info "Убедитесь, что путь правильный или используйте --access-log"
        return 1
    fi
    
    if [ ! -r "$log_path" ]; then
        print_error "Нет прав на чтение файла: $log_path"
        return 1
    fi
    
    print_message "✅ Файл access.log доступен: $log_path"
    return 0
}

# Функция запуска сервиса
start_service() {
    local max_ips=${1:-2}
    local check_interval=${2:-5}
    local grace_period=${3:-10}
    local access_log=${4:-"/usr/local/x-ui/access.log"}
    
    print_message "🚀 Запуск IP Ban сервиса..."
    print_info "Максимум IP на конфиг: $max_ips"
    print_info "Интервал проверки: ${check_interval}м"
    print_info "Период ожидания: ${grace_period}м"
    print_info "Файл лога: $access_log"
    
    if ! check_access_log "$access_log"; then
        print_error "Не удалось проверить файл access.log"
        exit 1
    fi
    
    go run . -max-ips "$max_ips" -check-interval "${check_interval}m" -grace-period "${grace_period}m" -access-log "$access_log"
}

# Функция показа статистики
show_stats() {
    local access_log=${1:-"/usr/local/x-ui/access.log"}
    local max_ips=${2:-2}
    
    print_message "📊 Анализ статистики..."
    
    if ! check_access_log "$access_log"; then
        print_error "Не удалось проверить файл access.log"
        exit 1
    fi
    
    go run . -stats -access-log "$access_log" -max-ips "$max_ips"
}

# Функция показа конфигов
show_configs() {
    print_message "📋 Получение списка конфигов..."
    go run . -list-configs
}

# Функция включения конфига
enable_config() {
    local email=$1
    
    if [ -z "$email" ]; then
        print_error "Не указан email для включения конфига"
        echo "Использование: $0 enable EMAIL"
        exit 1
    fi
    
    print_message "🔓 Включение конфига для email: $email"
    go run . -enable "$email"
}

# Функция отключения конфига
disable_config() {
    local email=$1
    
    if [ -z "$email" ]; then
        print_error "Не указан email для отключения конфига"
        echo "Использование: $0 disable EMAIL"
        exit 1
    fi
    
    print_message "🔒 Отключение конфига для email: $email"
    go run . -disable "$email"
}

# Функция тестового запуска
test_run() {
    local access_log=${1:-"/usr/local/x-ui/access.log"}
    local max_ips=${2:-2}
    
    print_message "🧪 Тестовый запуск (показывает что будет сделано)..."
    
    if ! check_access_log "$access_log"; then
        print_error "Не удалось проверить файл access.log"
        exit 1
    fi
    
    print_info "Анализ лога без изменений конфигов..."
    go run . -stats -access-log "$access_log" -max-ips "$max_ips"
}

# Основная логика
main() {
    local command=${1:-"help"}
    local max_ips=2
    local check_interval=5
    local grace_period=10
    local access_log="/usr/local/x-ui/access.log"
    
    # Парсинг аргументов
    shift 2>/dev/null || true
    while [[ $# -gt 0 ]]; do
        case $1 in
            --max-ips)
                max_ips="$2"
                shift 2
                ;;
            --check-interval)
                check_interval="$2"
                shift 2
                ;;
            --grace-period)
                grace_period="$2"
                shift 2
                ;;
            --access-log)
                access_log="$2"
                shift 2
                ;;
            *)
                if [ "$command" = "enable" ] || [ "$command" = "disable" ]; then
                    # Для команд enable/disable первый аргумент - это email
                    break
                fi
                shift
                ;;
        esac
    done
    
    # Проверка зависимостей
    check_dependencies
    
    # Выполнение команды
    case $command in
        start)
            start_service "$max_ips" "$check_interval" "$grace_period" "$access_log"
            ;;
        stats)
            show_stats "$access_log" "$max_ips"
            ;;
        configs)
            show_configs
            ;;
        enable)
            enable_config "$1"
            ;;
        disable)
            disable_config "$1"
            ;;
        test)
            test_run "$access_log" "$max_ips"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "Неизвестная команда: $command"
            echo
            show_help
            exit 1
            ;;
    esac
}

# Запуск основной функции
main "$@"
