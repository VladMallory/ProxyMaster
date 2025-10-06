#!/usr/bin/env python3
import sqlite3
import json
import sys

DB_PATH = "/etc/x-ui/x-ui.db"

def main():
    print("=== УДАЛЕНИЕ ДУБЛИКАТА 873925520_server2 ===\n")
    
    # Подключаемся к базе
    conn = sqlite3.connect(DB_PATH, timeout=30)
    cursor = conn.cursor()
    
    try:
        # Получаем inbound 2
        cursor.execute("SELECT id, settings FROM inbounds WHERE id = 2")
        row = cursor.fetchone()
        
        if not row:
            print("❌ Inbound 2 не найден")
            return
        
        inbound_id, settings_str = row
        settings = json.loads(settings_str)
        
        print(f"Inbound {inbound_id}: всего клиентов = {len(settings['clients'])}")
        
        # Ищем дубликаты 873925520_server2
        duplicates = [c for c in settings['clients'] if c['email'] == '873925520_server2']
        print(f"Найдено записей с email '873925520_server2': {len(duplicates)}\n")
        
        if len(duplicates) <= 1:
            print("✓ Дубликатов нет, всё в порядке")
            return
        
        # Показываем дубликаты
        for i, dup in enumerate(duplicates):
            print(f"Дубликат {i+1}:")
            print(f"  ID: {dup['id']}")
            print(f"  SubID: {dup['subId']}")
            print(f"  ExpiryTime: {dup['expiryTime']}")
            print(f"  Enable: {dup['enable']}")
            print()
        
        # Оставляем только самый новый (с большим expiryTime)
        duplicates.sort(key=lambda x: x['expiryTime'], reverse=True)
        keep = duplicates[0]
        remove = duplicates[1:]
        
        print(f"✓ Оставляем запись с expiryTime={keep['expiryTime']}, SubID={keep['subId']}")
        for r in remove:
            print(f"✗ Удаляем запись с expiryTime={r['expiryTime']}, SubID={r['subId']}")
        
        # Удаляем дубликаты из списка
        new_clients = [c for c in settings['clients'] if not (c['email'] == '873925520_server2' and c != keep)]
        
        print(f"\nБыло клиентов: {len(settings['clients'])}")
        print(f"Стало клиентов: {len(new_clients)}")
        
        # Обновляем settings
        settings['clients'] = new_clients
        new_settings_str = json.dumps(settings)
        
        # Сохраняем в базу
        cursor.execute("UPDATE inbounds SET settings = ? WHERE id = ?", (new_settings_str, inbound_id))
        conn.commit()
        
        print("\n✅ Дубликат успешно удалён!")
        print("⚠  Перезапустите X-UI: systemctl restart x-ui")
        
    except Exception as e:
        print(f"❌ Ошибка: {e}")
        conn.rollback()
        sys.exit(1)
    finally:
        conn.close()

if __name__ == "__main__":
    main()
