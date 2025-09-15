#!/bin/bash

echo "🔍 ПРОВЕРКА СИСТЕМЫ БЭКАПОВ"
echo "=========================="

# Проверяем структуру папок
echo "📁 Структура папок:"
ls -la backups/ 2>/dev/null || echo "❌ Папка backups не найдена"

echo ""
echo "📊 Бэкапы в backupdb:"
ls -la backups/backupdb/ 2>/dev/null || echo "❌ Папка backupdb не найдена"

echo ""
echo "⚡ Последний бэкап (latest):"
ls -la backups/latest/ 2>/dev/null || echo "❌ Папка latest не найдена"

echo ""
echo "📈 Статистика бэкапов:"
backup_count=$(ls -1 backups/backupdb/ 2>/dev/null | wc -l)
echo "   Количество бэкапов: $backup_count (хранятся бессрочно)"

if [ -d "backups/latest/vpn_bot" ]; then
    echo "   ✅ Latest бэкап существует"
    latest_size=$(du -sh backups/latest/ 2>/dev/null | cut -f1)
    echo "   📦 Размер latest бэкапа: $latest_size"
else
    echo "   ❌ Latest бэкап отсутствует"
fi

echo ""
echo "🕐 Последние изменения:"
if [ -d "backups/backupdb" ]; then
    latest_backup=$(ls -1t backups/backupdb/ | head -1)
    if [ ! -z "$latest_backup" ]; then
        echo "   Последний бэкап: $latest_backup"
        echo "   Время создания: $(stat -c %y backups/backupdb/$latest_backup 2>/dev/null)"
    fi
fi

echo ""
echo "🧪 Способы тестирования:"
echo "   1. Ручной бэкап: go run -c 'package main; import \"bot/common\"; func main() { common.BackupMongoDB() }'"
echo "   2. Команда в Telegram: /backup (только для админа)"
echo "   3. Автоматический бэкап: каждый час (хранятся бессрочно)"
echo "   4. Восстановление: при запуске бота автоматически"

echo ""
echo "📋 Логи бэкапов:"
echo "   grep 'BACKUP_MONGODB\\|RESTORE_MONGODB' /var/log/syslog 2>/dev/null || echo 'Логи не найдены'"
