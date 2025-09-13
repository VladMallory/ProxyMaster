#!/bin/bash

# 🔄 Скрипт для перезапуска языкового сервера Go в VS Code

echo "🔄 Перезапуск языкового сервера Go в VS Code..."

# 1. Проверяем, что VS Code запущен
if ! pgrep -f "code" > /dev/null; then
    echo "❌ VS Code не запущен. Запустите VS Code сначала."
    exit 1
fi

echo "✅ VS Code запущен"

# 2. Отправляем команду перезапуска языкового сервера
echo "🔄 Отправляем команду перезапуска языкового сервера..."

# Попробуем через code CLI
code --command "go.restartLanguageServer" 2>/dev/null || echo "⚠️ Не удалось отправить команду через CLI"

# 3. Альтернативный способ - через gopls
echo "🔄 Перезапуск gopls..."
pkill -f gopls 2>/dev/null || echo "gopls не запущен"

# 4. Проверяем инструменты Go
echo "🔍 Проверка инструментов Go..."
code --command "go.locateTools" 2>/dev/null || echo "⚠️ Не удалось проверить инструменты через CLI"

echo ""
echo "✅ Перезапуск завершен!"
echo ""
echo "📋 Что дальше:"
echo "1. В VS Code нажмите Ctrl+Shift+P"
echo "2. Введите 'Go: Restart Language Server'"
echo "3. Выберите команду"
echo "4. Попробуйте сохранить .go файл (Ctrl+S)"
echo ""
echo "🧪 Тест:"
echo "1. Откройте test_format.go"
echo "2. Измените комментарий на //тест"
echo "3. Сохраните файл (Ctrl+S)"
echo "4. Должно стать // тест"
