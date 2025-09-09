#!/bin/bash

echo "🚀 Запуск нагрузочного тестирования базы данных VPN бота"
echo "========================================================"

# Проверяем, что мы в правильной директории
if [ ! -f "main.go" ]; then
    echo "❌ Ошибка: Запустите скрипт из директории loadTest"
    exit 1
fi

# Проверяем подключение к базе данных
echo "🔍 Проверка подключения к базе данных..."
psql -h localhost -U your_db_user -d your_database_name -c "SELECT 1;" > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "❌ Ошибка: Не удается подключиться к базе данных PostgreSQL"
    echo "Убедитесь, что:"
    echo "  - PostgreSQL запущен"
    echo "  - База данных 'your_database_name' существует"
    echo "  - Пользователь 'your_db_user' создан и имеет права доступа"
    echo "  - Пароль правильный"
    exit 1
fi

echo "✅ Подключение к базе данных успешно"

# Создаем тестовые данные если их нет
echo "📊 Проверка тестовых данных..."
USER_COUNT=$(psql -h localhost -U your_db_user -d your_database_name -t -c "SELECT COUNT(*) FROM users;" 2>/dev/null | tr -d ' ')
if [ "$USER_COUNT" -lt 10 ]; then
    echo "📝 Создание тестовых данных..."
    psql -h localhost -U your_db_user -d your_database_name -c "
        INSERT INTO users (telegram_id, username, first_name, last_name, balance, total_paid, 
                          configs_count, has_active_config, has_used_trial, created_at, updated_at)
        SELECT 
            generate_series(1000001, 1000100) as telegram_id,
            'testuser_' || generate_series(1000001, 1000100) as username,
            'Test' || generate_series(1000001, 1000100) as first_name,
            'User' || generate_series(1000001, 1000100) as last_name,
            random() * 1000 as balance,
            random() * 500 as total_paid,
            floor(random() * 5) as configs_count,
            random() < 0.3 as has_active_config,
            random() < 0.5 as has_used_trial,
            NOW() as created_at,
            NOW() as updated_at
        ON CONFLICT (telegram_id) DO NOTHING;
    " > /dev/null 2>&1
    echo "✅ Тестовые данные созданы"
else
    echo "✅ Тестовые данные уже существуют ($USER_COUNT пользователей)"
fi

# Компилируем и запускаем нагрузочное тестирование
echo "🔨 Компиляция программы нагрузочного тестирования..."
go build -o loadtest main.go
if [ $? -ne 0 ]; then
    echo "❌ Ошибка компиляции"
    exit 1
fi

echo "✅ Компиляция успешна"

# Запускаем мониторинг в фоне
echo "🔍 Запуск мониторинга базы данных в фоне..."
go run monitor.go &
MONITOR_PID=$!

# Небольшая пауза для запуска мониторинга
sleep 2

echo "🔥 Запуск нагрузочного тестирования..."
echo "   - Продолжительность: 5 минут"
echo "   - Одновременных пользователей: 200"
echo "   - Операции: чтение, запись, обновление"
echo ""

# Запускаем нагрузочное тестирование
./loadtest

# Останавливаем мониторинг
echo "🛑 Остановка мониторинга..."
kill $MONITOR_PID 2>/dev/null

echo ""
echo "✅ Нагрузочное тестирование завершено!"
echo "📊 Результаты сохранены в логах выше"
