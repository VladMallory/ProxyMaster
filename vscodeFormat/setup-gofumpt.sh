#!/bin/bash

# 🚀 Универсальный скрипт настройки gofumpt для любого проекта Go
# Работает как локально, так и через Remote SSH

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Функция для цветного вывода
print_status() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_header() {
    echo -e "${BLUE}"
    echo "🚀 Настройка gofumpt для проекта Go"
    echo "=================================="
    echo -e "${NC}"
}

# Показываем справку
show_help() {
    echo "Использование: $0 [опции] [путь_к_проекту]"
    echo ""
    echo "Опции:"
    echo "  -h, --help     Показать эту справку"
    echo "  -f, --force    Перезаписать существующие настройки"
    echo "  -g, --global   Настроить глобально для VS Code"
    echo "  -c, --check    Проверить текущую настройку"
    echo ""
    echo "Примеры:"
    echo "  $0                    # Настроить текущую папку"
    echo "  $0 /path/to/project   # Настроить указанную папку"
    echo "  $0 --global           # Настроить глобально"
    echo "  $0 --check            # Проверить настройки"
    echo ""
}

# Проверяем аргументы
PROJECT_PATH="."
FORCE=false
GLOBAL=false
CHECK=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -f|--force)
            FORCE=true
            shift
            ;;
        -g|--global)
            GLOBAL=true
            shift
            ;;
        -c|--check)
            CHECK=true
            shift
            ;;
        -*)
            print_error "Неизвестная опция: $1"
            show_help
            exit 1
            ;;
        *)
            PROJECT_PATH="$1"
            shift
            ;;
    esac
done

# Функция проверки настройки
check_setup() {
    print_header
    print_status "Проверка текущей настройки gofumpt..."
    
    # Проверяем gofumpt
    if command -v gofumpt &> /dev/null; then
        print_success "gofumpt установлен: $(gofumpt --version)"
    else
        print_error "gofumpt не найден!"
        echo "Установите его командой: go install mvdan.cc/gofumpt@latest"
        exit 1
    fi
    
    # Проверяем VS Code
    if command -v code &> /dev/null; then
        print_success "VS Code найден"
    else
        print_warning "VS Code не найден в PATH"
    fi
    
    # Проверяем локальные настройки
    if [ -f "$PROJECT_PATH/.vscode/settings.json" ]; then
        print_success "Локальные настройки VS Code найдены"
        if grep -q "go.formatTool.*gofumpt" "$PROJECT_PATH/.vscode/settings.json"; then
            print_success "gofumpt настроен как форматтер"
        else
            print_warning "gofumpt не настроен как форматтер"
        fi
    else
        print_warning "Локальные настройки VS Code не найдены"
    fi
    
    # Проверяем глобальные настройки
    GLOBAL_SETTINGS="$HOME/.config/Code/User/settings.json"
    if [ -f "$GLOBAL_SETTINGS" ]; then
        print_success "Глобальные настройки VS Code найдены"
        if grep -q "go.formatTool.*gofumpt" "$GLOBAL_SETTINGS"; then
            print_success "gofumpt настроен глобально"
        else
            print_warning "gofumpt не настроен глобально"
        fi
    else
        print_warning "Глобальные настройки VS Code не найдены"
    fi
    
    echo ""
    print_status "Проверка завершена!"
}

# Функция настройки глобально
setup_global() {
    print_header
    print_status "Настройка gofumpt глобально для VS Code..."
    
    GLOBAL_SETTINGS="$HOME/.config/Code/User/settings.json"
    GLOBAL_DIR="$(dirname "$GLOBAL_SETTINGS")"
    
    # Создаем директорию если не существует
    mkdir -p "$GLOBAL_DIR"
    
    # Создаем или обновляем настройки
    if [ -f "$GLOBAL_SETTINGS" ] && [ "$FORCE" = false ]; then
        if grep -q "go.formatTool" "$GLOBAL_SETTINGS"; then
            print_warning "Глобальные настройки уже существуют"
            print_status "Используйте --force для перезаписи"
            return 0
        fi
    fi
    
    # Создаем бэкап если файл существует
    if [ -f "$GLOBAL_SETTINGS" ]; then
        BACKUP_NAME="settings.json.backup.$(date +%Y%m%d_%H%M%S)"
        cp "$GLOBAL_SETTINGS" "$GLOBAL_DIR/$BACKUP_NAME"
        print_success "Бэкап создан: $BACKUP_NAME"
    fi
    
    # Создаем настройки
    cat > "$GLOBAL_SETTINGS" << 'EOF'
{
  "go.formatTool": "gofumpt",
  "[go]": {
    "editor.formatOnSave": true,
    "editor.defaultFormatter": "golang.go",
    "editor.codeActionsOnSave": {
      "source.organizeImports": "explicit"
    },
    "editor.insertSpaces": false,
    "editor.tabSize": 4
  },
  "gopls": {
    "formatting.gofumpt": true
  },
  "go.useCodeSnippetsOnFunctionSuggest": false,
  "editor.formatOnSave": true,
  "go.alternateTools": {
    "gofumpt": "gofumpt"
  }
}
EOF
    
    print_success "Глобальные настройки созданы: $GLOBAL_SETTINGS"
}

# Функция настройки локально
setup_local() {
    print_header
    print_status "Настройка gofumpt для проекта: $PROJECT_PATH"
    
    # Проверяем, что это Go проект
    if [ ! -f "$PROJECT_PATH/go.mod" ] && [ ! -f "$PROJECT_PATH/*.go" ]; then
        print_warning "Не похоже на Go проект (нет go.mod или .go файлов)"
        print_status "Продолжаем настройку..."
    fi
    
    # Создаем директорию .vscode
    VSCODE_DIR="$PROJECT_PATH/.vscode"
    mkdir -p "$VSCODE_DIR"
    
    # Проверяем существующие настройки
    if [ -f "$VSCODE_DIR/settings.json" ] && [ "$FORCE" = false ]; then
        if grep -q "go.formatTool" "$VSCODE_DIR/settings.json"; then
            print_warning "Локальные настройки уже существуют"
            print_status "Используйте --force для перезаписи"
            return 0
        fi
    fi
    
    # Создаем бэкап если файл существует
    if [ -f "$VSCODE_DIR/settings.json" ]; then
        BACKUP_NAME="settings.json.backup.$(date +%Y%m%d_%H%M%S)"
        cp "$VSCODE_DIR/settings.json" "$VSCODE_DIR/$BACKUP_NAME"
        print_success "Бэкап создан: $BACKUP_NAME"
    fi
    
    # Определяем путь к gofumpt
    GOFUMPT_PATH="gofumpt"
    if command -v gofumpt &> /dev/null; then
        GOFUMPT_PATH="$(which gofumpt)"
    fi
    
    # Создаем настройки
    cat > "$VSCODE_DIR/settings.json" << EOF
{
  "go.formatTool": "gofumpt",
  "[go]": {
    "editor.formatOnSave": true,
    "editor.defaultFormatter": "golang.go",
    "editor.codeActionsOnSave": {
      "source.organizeImports": "explicit"
    },
    "editor.insertSpaces": false,
    "editor.tabSize": 4
  },
  "gopls": {
    "formatting.gofumpt": true
  },
  "go.useCodeSnippetsOnFunctionSuggest": false,
  "editor.formatOnSave": true,
  "go.alternateTools": {
    "gofumpt": "$GOFUMPT_PATH"
  },
  "go.toolsEnvVars": {
    "PATH": "\$GOPATH/bin:/usr/local/go/bin:\${env:PATH}"
  }
}
EOF
    
    print_success "Локальные настройки созданы: $VSCODE_DIR/settings.json"
    
    # Создаем файл рекомендуемых расширений
    cat > "$VSCODE_DIR/extensions.json" << 'EOF'
{
  "recommendations": [
    "golang.go"
  ]
}
EOF
    
    print_success "Файл рекомендуемых расширений создан"
}

# Функция создания тестового файла
create_test_file() {
    TEST_FILE="$PROJECT_PATH/test_gofumpt.go"
    cat > "$TEST_FILE" << 'EOF'
package main

import "fmt"

func main() {
	//тест без пробела
	x := 42 //еще тест
	//и третий тест
	fmt.Println(x)
}
EOF
    print_success "Тестовый файл создан: test_gofumpt.go"
}

# Основная логика
if [ "$CHECK" = true ]; then
    check_setup
    exit 0
fi

if [ "$GLOBAL" = true ]; then
    setup_global
    print_status "Перезапустите VS Code для применения глобальных настроек"
else
    setup_local
    create_test_file
    
    echo ""
    print_status "Настройка завершена!"
    echo ""
    print_status "Что дальше:"
    echo "1. Перезапустите VS Code или языковой сервер Go"
    echo "2. Откройте test_gofumpt.go"
    echo "3. Измените комментарии на //тест"
    echo "4. Сохраните файл (Ctrl+S)"
    echo "5. Должно стать // тест"
    echo ""
    print_status "Для перезапуска языкового сервера:"
    echo "Ctrl+Shift+P → 'Go: Restart Language Server'"
fi
