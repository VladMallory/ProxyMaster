# Запуск проекта
## env
Создать env в дериктории проекта 
```bash
vim .env
```

*Обязательно указать все данные чтобы бот работал корректно*
```env
BOT_TOKEN=123

ADMIN_ID=123
SUPPORT_LINK=https://t.me/

PANEL_URL=https://domen.ru:123/asd/
PANEL_USER=login
PANEL_PASS=password
INBOUND_ID=1
CONFIG_BASE_URL=https://domen.ru:123/asd/


REDIRECT_DOMAIN=im.domen.ru:8443
REFERRAL_LINK_BASE_URL=https://t.me/bot?start=ref_

YUKASSA_SHOP_ID=123
YUKASSA_SECRET_KEY=123123

# Логи
# Включение/выключение и путь для exhausted-лога
EXHAUSTED_LOG_ENABLED=true
EXHAUSTED_LOG_PATH=/root/bot/logs/exhausted.log
```

```bash
go run main.go
```
