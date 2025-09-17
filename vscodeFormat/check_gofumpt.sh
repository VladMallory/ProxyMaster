#!/bin/bash

# 🧪 Скрипт проверки работы gofumpt

echo "🔍 Проверка настройки gofumpt..."

# 1. Проверяем gofumpt
echo "1️⃣ Проверка gofumpt:"
if command -v gofumpt &> /dev/null; then
    echo "   ✅ gofumpt найден: $(gofumpt --version)"
else
    echo "   ❌ gofumpt не найден!"
    exit 1
fi

# 2. Проверяем настройки VS Code
echo ""
echo "2️⃣ Проверка настроек VS Code:"
VSCODE_SETTINGS="/root/.config/Code/User/settings.json"
if [ -f "$VSCODE_SETTINGS" ]; then
    echo "   ✅ Настройки VS Code найдены: $VSCODE_SETTINGS"
    if grep -q "go.formatTool.*gofumpt" "$VSCODE_SETTINGS"; then
        echo "   ✅ gofumpt настроен как форматтер"
    else
        echo "   ❌ gofumpt не настроен как форматтер"
    fi
    if grep -q "formatting.gofumpt.*true" "$VSCODE_SETTINGS"; then
        echo "   ✅ gopls настроен для gofumpt"
    else
        echo "   ❌ gopls не настроен для gofumpt"
    fi
else
    echo "   ❌ Настройки VS Code не найдены!"
fi

# 3. Создаем тестовый файл
echo ""
echo "3️⃣ Создание тестового файла:"
cat > test_auto_format.go << 'EOF'
package main

import "fmt"

func main() {
    //тест без пробела
    x := 42 //еще тест
    //и третий тест
    fmt.Println(x)
}
EOF
echo "   ✅ Тестовый файл создан: test_auto_format.go"

# 4. Тестируем gofumpt
echo ""
echo "4️⃣ Тест форматирования gofumpt:"
echo "   До форматирования:"
grep "//" test_auto_format.go

echo ""
echo "   Применяем gofumpt..."
gofumpt -w test_auto_format.go

echo "   После форматирования:"
grep "//" test_auto_format.go

# 5. Проверяем результат
echo ""
echo "5️⃣ Проверка результата:"
if grep -q "// тест" test_auto_format.go; then
    echo "   ✅ gofumpt работает правильно!"
    echo "   ✅ Комментарии отформатированы с пробелами"
else
    echo "   ❌ gofumpt не работает правильно"
fi

# 6. Инструкции для VS Code
echo ""
echo "6️⃣ Инструкции для VS Code:"
echo "   📋 Для проверки в VS Code:"
echo "   1. Перезапустите VS Code"
echo "   2. Ctrl+Shift+P → 'Go: Restart Language Server'"
echo "   3. Откройте test_auto_format.go"
echo "   4. Измените комментарии на //тест"
echo "   5. Сохраните файл (Ctrl+S)"
echo "   6. Должно стать // тест"

echo ""
echo "🎉 Проверка завершена!"
