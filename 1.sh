#!/bin/bash

# 🌍 Скрипт для глобальной установки gofumpt в VS Code
# Настройки будут применяться ко всем проектам Go

set -e

echo "🚀 Установка gofumpt глобально для VS Code..."

# 1. Устанавливаем gofumpt
echo "📦 Установка gofumpt..."
go install mvdan.cc/gofumpt@latest

# 2. Проверяем установку
if ! command -v gofumpt &> /dev/null; then
    echo "❌ Ошибка: gofumpt не установился корректно"
    exit 1
fi

echo "✅ gofumpt установлен: $(gofumpt --version)"

# 3. Определяем путь к настройкам VS Code
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    VSCODE_SETTINGS="$HOME/.config/Code/User/settings.json"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    VSCODE_SETTINGS="$HOME/Library/Application Support/Code/User/settings.json"
elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
    VSCODE_SETTINGS="$APPDATA/Code/User/settings.json"
else
    echo "❌ Неподдерживаемая OS: $OSTYPE"
    exit 1
fi

echo "📍 Путь к настройкам VS Code: $VSCODE_SETTINGS"

# 4. Создаем директорию если не существует
mkdir -p "$(dirname "$VSCODE_SETTINGS")"

# 5. Создаем или обновляем настройки
if [ -f "$VSCODE_SETTINGS" ]; then
    echo "⚙️  Обновление существующих настроек..."
    # Создаем бэкап
    cp "$VSCODE_SETTINGS" "$VSCODE_SETTINGS.backup.$(date +%Y%m%d_%H%M%S)"
    echo "📁 Бэкап создан: $VSCODE_SETTINGS.backup.$(date +%Y%m%d_%H%M%S)"
    
    # Проверяем, есть ли уже настройки gofumpt
    if grep -q "go.formatTool" "$VSCODE_SETTINGS"; then
        echo "⚠️  Настройки Go уже существуют. Проверьте их вручную."
        echo "   Файл настроек: $VSCODE_SETTINGS"
        exit 0
    fi
else
    echo "📝 Создание новых настроек..."
fi

# 6. Добавляем настройки gofumpt
cat >> "$VSCODE_SETTINGS" << 'EOF'
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
  "editor.formatOnSave": true
}
EOF

echo "✅ Настройки VS Code обновлены!"

# 7. Проверяем расширение Go
echo ""
echo "🔍 Проверка расширения Go в VS Code..."
if command -v code &> /dev/null; then
    code --install-extension golang.go
    echo "✅ Расширение Go установлено/обновлено"
else
    echo "⚠️  VS Code не найден в PATH. Установите расширение Go вручную:"
    echo "   Ctrl+Shift+X → поиск 'Go' → установить от Google"
fi

echo ""
echo "🎉 Глобальная настройка завершена!"
echo ""
echo "📋 Что дальше:"
echo "1. Перезапустите VS Code"
echo "2. Откройте любой .go файл"
echo "3. Проверьте автоформатирование при Ctrl+S"
echo ""
echo "🧪 Тест:"
echo "   Создайте комментарий: //тест"
echo "   Сохраните файл (Ctrl+S)"
echo "   Должно стать: // тест"

