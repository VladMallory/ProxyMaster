# 🔄 Настройка системы редиректа с нуля

Этот документ описывает полную настройку системы редиректа для Telegram бота VPN с нуля на новом сервере.

## 📋 Обзор системы

Система редиректа состоит из:
- **Telegram бот** - генерирует ссылки на конфигурации
- **HTTP сервер** (Go, порт 8081) - обслуживает HTML файлы редиректа
- **Nginx** (порт 8443) - проксирует HTTPS трафик
- **HTML файлы** - обрабатывают импорт в приложения (Happ, v2raytun)

## 🚀 Пошаговая настройка

### Шаг 1: Подготовка сервера

```bash
# Обновление системы
sudo apt update && sudo apt upgrade -y

# Установка необходимых пакетов
sudo apt install -y nginx certbot python3-certbot-nginx curl wget git
```

### Шаг 2: Настройка домена

1. **Настройте DNS запись** для вашего домена:
   ```
   A    redirect.yourdomain.com    -> IP_ВАШЕГО_СЕРВЕРА
   ```

2. **Проверьте доступность домена**:
   ```bash
   ping redirect.yourdomain.com
   ```

### Шаг 3: Установка SSL сертификата

```bash
# Получение SSL сертификата через Let's Encrypt
sudo certbot --nginx -d redirect.yourdomain.com --non-interactive --agree-tos --email your-email@domain.com

# Проверка статуса сертификата
sudo certbot certificates
```

### Шаг 4: Настройка Nginx

1. **Создайте конфигурацию для редиректа**:
   ```bash
   sudo nano /etc/nginx/sites-available/redirect
   ```

2. **Вставьте следующую конфигурацию**:
   ```nginx
   server {
       listen 80;
       server_name redirect.yourdomain.com;
       return 301 https://$server_name$request_uri;
   }

   server {
       listen 8443 ssl http2;
       server_name redirect.yourdomain.com;

       # SSL конфигурация (управляется Certbot)
       ssl_certificate /etc/letsencrypt/live/redirect.yourdomain.com/fullchain.pem;
       ssl_certificate_key /etc/letsencrypt/live/redirect.yourdomain.com/privkey.pem;
       include /etc/letsencrypt/options-ssl-nginx.conf;
       ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;

       # Основная конфигурация
       location / {
           # Проксируем все запросы на локальный HTTP сервер
           proxy_pass http://127.0.0.1:8081;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
           
           # Таймауты
           proxy_connect_timeout 30s;
           proxy_send_timeout 30s;
           proxy_read_timeout 30s;
       }

       # Логирование
       access_log /var/log/nginx/redirect_access.log;
       error_log /var/log/nginx/redirect_error.log;
   }
   ```

3. **Активируйте конфигурацию**:
   ```bash
   # Создайте символическую ссылку
   sudo ln -s /etc/nginx/sites-available/redirect /etc/nginx/sites-enabled/
   
   # Удалите дефолтную конфигурацию (если нужно)
   sudo rm /etc/nginx/sites-enabled/default
   
   # Проверьте конфигурацию
   sudo nginx -t
   
   # Перезапустите nginx
   sudo systemctl reload nginx
   ```

### Шаг 5: Настройка брандмауэра

```bash
# Разрешите необходимые порты
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp   # HTTP (для редиректа на HTTPS)
sudo ufw allow 443/tcp  # HTTPS (если используете стандартный порт)
sudo ufw allow 8443/tcp # HTTPS для редиректа
sudo ufw allow 8081/tcp # HTTP сервер бота (только локально)

# Включите брандмауэр
sudo ufw enable
```

### Шаг 6: Настройка конфигурации бота

1. **Отредактируйте `common/config.go`**:
   ```go
   // ========== НАСТРОЙКИ ПАНЕЛИ УПРАВЛЕНИЯ ==========
   REDIRECT_DOMAIN = "redirect.yourdomain.com:8443"
   REDIRECT_IMPORT = "happ"  // или "v2raytun"
   ```

2. **Проверьте другие настройки**:
   ```go
   CONFIG_BASE_URL = "https://yourdomain.com:2096/sub/"
   CONFIG_JSON_URL = "https://yourdomain.com:2096/json/"
   ```

### Шаг 7: Создание HTML файлов редиректа

Убедитесь, что в проекте есть папка `importRedirect/` с файлами:

- `redirect_happ.html` - для Happ (iOS)
- `redirect_happ_test.html` - для Happ (Android, улучшенная версия)
- `redirect_v2raytun.html` - для v2raytun

### Шаг 8: Запуск бота

```bash
# Перейдите в папку проекта
cd /path/to/your/bot

# Запустите бота
go run main.go
```

### Шаг 9: Проверка работы

1. **Проверьте HTTP сервер**:
   ```bash
   curl http://127.0.0.1:8081/redirect_happ_test.html
   ```

2. **Проверьте HTTPS редирект**:
   ```bash
   curl https://redirect.yourdomain.com:8443/redirect_happ_test.html
   ```

3. **Проверьте в браузере**:
   ```
   https://redirect.yourdomain.com:8443/redirect_happ_test.html?url=test
   ```

## 🔧 Дополнительные настройки

### Автоматическое обновление SSL сертификата

```bash
# Добавьте в crontab
sudo crontab -e

# Добавьте строку:
0 12 * * * /usr/bin/certbot renew --quiet && /usr/bin/systemctl reload nginx
```

### Мониторинг логов

```bash
# Просмотр логов nginx
sudo tail -f /var/log/nginx/redirect_access.log
sudo tail -f /var/log/nginx/redirect_error.log

# Просмотр логов бота
tail -f /root/bot/logs/console.log
```

### Настройка для разных портов

Если порт 443 занят (например, xray):

1. **Измените порт в nginx**:
   ```nginx
   listen 8443 ssl http2;  # вместо 443
   ```

2. **Обновите конфигурацию бота**:
   ```go
   REDIRECT_DOMAIN = "redirect.yourdomain.com:8443"
   ```

3. **Откройте порт в брандмауэре**:
   ```bash
   sudo ufw allow 8443/tcp
   ```

## 🐛 Решение проблем

### Проблема: SSL сертификат не работает

```bash
# Проверьте статус сертификата
sudo certbot certificates

# Обновите сертификат вручную
sudo certbot renew --force-renewal

# Проверьте конфигурацию nginx
sudo nginx -t
```

### Проблема: Nginx не проксирует на бота

```bash
# Проверьте, что бот слушает на порту 8081
sudo netstat -tlnp | grep 8081

# Проверьте логи nginx
sudo tail -f /var/log/nginx/redirect_error.log
```

### Проблема: Приложения не открываются

1. **Проверьте URL в браузере**:
   ```
   https://redirect.yourdomain.com:8443/redirect_happ_test.html?url=test
   ```

2. **Проверьте JavaScript консоль** в браузере на наличие ошибок

3. **Убедитесь, что приложение установлено** на устройстве

## 📱 Поддерживаемые приложения

### Happ
- **iOS**: Автоматический импорт через `happ://add/URL`
- **Android**: Ручной импорт через кнопку (решает проблемы с base64)

### v2raytun
- **Все платформы**: Импорт через `v2raytun://import/URL`

## 🔒 Безопасность

1. **Используйте только HTTPS** - приложения не работают с HTTP
2. **Регулярно обновляйте SSL сертификаты**
3. **Мониторьте логи** на предмет подозрительной активности
4. **Ограничьте доступ** к порту 8081 только локально

## 📊 Мониторинг

### Проверка статуса сервисов

```bash
# Статус nginx
sudo systemctl status nginx

# Статус бота (если используете systemd)
sudo systemctl status your-bot-service

# Проверка портов
sudo netstat -tlnp | grep -E "(8081|8443)"
```

### Метрики производительности

```bash
# Нагрузка на nginx
sudo nginx -T | grep worker_processes

# Использование памяти
free -h

# Использование диска
df -h
```

## 🎯 Итоговая конфигурация

После настройки у вас будет:

- ✅ **HTTPS редирект** на порту 8443
- ✅ **Автоматическое обновление SSL**
- ✅ **Проксирование через Nginx**
- ✅ **Поддержка Happ и v2raytun**
- ✅ **Логирование и мониторинг**

URL для редиректа будет выглядеть так:
```
https://redirect.yourdomain.com:8443/redirect_happ_test.html?url=CONFIG_URL
```

## 📞 Поддержка

При возникновении проблем:

1. Проверьте логи nginx и бота
2. Убедитесь, что все порты открыты
3. Проверьте SSL сертификат
4. Убедитесь, что бот запущен и слушает порт 8081

---

**Примечание**: Этот документ описывает настройку системы редиректа с нуля. Убедитесь, что у вас уже настроены основные компоненты бота (база данных, панель управления, платежи).
