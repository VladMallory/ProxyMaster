#!/bin/bash

# Скрипт для быстрой настройки SSL сертификата
# Использование: ./setup_ssl.sh your-domain.com your-email@domain.com

set -e

DOMAIN=$1
EMAIL=$2

if [ -z "$DOMAIN" ] || [ -z "$EMAIL" ]; then
    echo "❌ Использование: $0 <domain> <email>"
    echo "   Пример: $0 im.акуавпн.рф admin@example.com"
    exit 1
fi

echo "🔧 Настройка SSL сертификата для домена: $DOMAIN"
echo "📧 Email: $EMAIL"
echo ""

# Проверяем права sudo
if [ "$EUID" -ne 0 ]; then
    echo "❌ Запустите скрипт с sudo: sudo $0 $DOMAIN $EMAIL"
    exit 1
fi

# Обновляем пакеты
echo "📦 Обновление пакетов..."
apt update

# Устанавливаем nginx, certbot и python3-idna для конвертации доменов
echo "🔧 Установка nginx, certbot и python3-idna..."
apt install -y nginx certbot python3-certbot-nginx python3-idna

# Останавливаем nginx если запущен
systemctl stop nginx 2>/dev/null || true

# Конвертируем домен в Punycode если нужно
echo "🔄 Проверка домена..."
PUNYCODE_DOMAIN=$(python3 -c "import idna; print(idna.encode('$DOMAIN').decode('ascii'))" 2>/dev/null || echo "$DOMAIN")

if [ "$PUNYCODE_DOMAIN" != "$DOMAIN" ]; then
    echo "📝 Кириллический домен конвертирован в Punycode: $PUNYCODE_DOMAIN"
    CERT_DOMAIN="$PUNYCODE_DOMAIN"
else
    echo "📝 Используется оригинальный домен: $DOMAIN"
    CERT_DOMAIN="$DOMAIN"
fi

# Создаем базовую конфигурацию nginx
echo "⚙️ Создание конфигурации nginx..."
if [ "$PUNYCODE_DOMAIN" != "$DOMAIN" ]; then
    # Для кириллических доменов поддерживаем оба варианта
    cat > /etc/nginx/sites-available/default << EOF
server {
    listen 80;
    server_name $CERT_DOMAIN $DOMAIN;
    
    location / {
        proxy_pass http://127.0.0.1:8081;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
else
    # Для обычных доменов используем только один
    cat > /etc/nginx/sites-available/default << EOF
server {
    listen 80;
    server_name $CERT_DOMAIN;
    
    location / {
        proxy_pass http://127.0.0.1:8081;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
fi

# Запускаем nginx
echo "🚀 Запуск nginx..."
systemctl start nginx
systemctl enable nginx

# Получаем SSL сертификат
echo "🔒 Получение SSL сертификата..."
certbot --nginx -d $CERT_DOMAIN --non-interactive --agree-tos --email $EMAIL

# Проверяем конфигурацию nginx
echo "✅ Проверка конфигурации nginx..."
nginx -t

# Обновляем конфигурацию nginx для поддержки кириллического домена
if [ "$PUNYCODE_DOMAIN" != "$DOMAIN" ]; then
    echo "🔧 Добавление поддержки кириллического домена в nginx..."
    # Обновляем конфигурацию nginx для поддержки обоих доменов
    sed -i "s/server_name $CERT_DOMAIN;/server_name $CERT_DOMAIN $DOMAIN;/g" /etc/nginx/sites-available/default
fi

# Проверяем, занят ли порт 443 (обычно xray)
echo "🔍 Проверка порта 443..."
if ss -tlnp | grep -q ":443 "; then
    echo "⚠️ Порт 443 занят, переключаемся на порт 8443..."
    sed -i 's/listen 443 ssl;/listen 8443 ssl;/g' /etc/nginx/sites-available/default
    SSL_PORT="8443"
else
    echo "✅ Порт 443 свободен"
    SSL_PORT="443"
fi

# Перезагружаем nginx
echo "🔄 Перезагрузка nginx..."
systemctl reload nginx

echo ""
echo "🎉 SSL сертификат успешно настроен!"
echo "🌐 Ваш сайт доступен по HTTPS: https://$DOMAIN:$SSL_PORT"
if [ "$PUNYCODE_DOMAIN" != "$DOMAIN" ]; then
    echo "🌐 Также доступен по Punycode: https://$PUNYCODE_DOMAIN:$SSL_PORT"
fi
echo ""
echo "📝 Не забудьте обновить REDIRECT_DOMAIN в common/config.go:"
echo "   REDIRECT_DOMAIN = \"$DOMAIN:$SSL_PORT\""
echo ""
echo "🔄 Автообновление сертификата настроено автоматически"
