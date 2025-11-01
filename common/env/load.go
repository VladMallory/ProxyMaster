// Пакет env: загрузка конфигурации из файла .env и применение к глобальным настройкам.
// Ответственность файла: парсинг .env и предоставление удобного интерфейса.
//
// Как использовать (в main.go):
//   cfg := env.MustLoad()        // читаем .env
//   cfg.ApplyToCommon()          // применяем к глобальным переменным пакета common
//   common.InitGlobals()         // инициализации, которые зависят от конфигурации
//
// Формат .env:
//   - строки вида KEY=VALUE, допускаются пробелы вокруг '=' и кавычки вокруг значений
//   - комментарии начинаются с '#'
//
// Поведение:
//   - поля чисел (например, ADMIN_ID, INBOUND_ID) парсятся из строк
//   - ApplyToCommon НЕ трогает значения, если из .env пришли пустые строки
//   - при ошибке чтения MustLoad аварийно завершает приложение с понятным сообщением
//
// Важно: утилиты из каталога tools/ тоже должны вызывать MustLoad().ApplyToCommon(),
// иначе они будут использовать значения по умолчанию из common/config.go.
package env

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"

	"bot/common"
)

// Config представляет набор настроек, читаемых из .env
type Config struct {
	// Бот
	BotToken    string
	AdminID     int64
	SupportLink string

	// Панель X-UI
	PanelURL  string
	PanelUser string
	PanelPass string
	InboundID int

	// Ссылки на конфиги и редиректы
	ConfigBaseURL  string
	ConfigJSONURL  string
	RedirectDomain string
	RedirectImport string

	// Реферальная система
	ReferralLinkBaseURL string

	// ЮКасса
	YukassaShopID     string
	YukassaSecretKey  string
	YukassaWebhookURL string
}

// Load загружает конфигурацию из указанного .env файла
func Load(path string) (Config, error) {
    f, err := os.Open(path)
    if err != nil {
        return Config{}, err
    }
    defer f.Close()

    // kv — все пары ключ-значение, считанные из .env
    kv := map[string]string{}
    s := bufio.NewScanner(f)
    for s.Scan() {
        line := strings.TrimSpace(s.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        // Допустим формат с пробелами: KEY = value
        parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            continue
        }
        key := strings.TrimSpace(parts[0])
        val := strings.TrimSpace(parts[1])
        // Убираем кавычки, если есть
        val = strings.Trim(val, "\"'")
        kv[key] = val
    }
    if err := s.Err(); err != nil {
        return Config{}, err
    }

	cfg := Config{
		BotToken:            kv["BOT_TOKEN"],
		SupportLink:         kv["SUPPORT_LINK"],
		PanelURL:            kv["PANEL_URL"],
		PanelUser:           kv["PANEL_USER"],
		PanelPass:           kv["PANEL_PASS"],
		ConfigBaseURL:       kv["CONFIG_BASE_URL"],
		ConfigJSONURL:       kv["CONFIG_JSON_URL"],
		RedirectDomain:      kv["REDIRECT_DOMAIN"],
		RedirectImport:      kv["REDIRECT_IMPORT"],
		ReferralLinkBaseURL: kv["REFERRAL_LINK_BASE_URL"],
		YukassaShopID:       kv["YUKASSA_SHOP_ID"],
		YukassaSecretKey:    kv["YUKASSA_SECRET_KEY"],
		YukassaWebhookURL:   kv["YUKASSA_WEBHOOK_URL"],
	}

    // Безопасно парсим целочисленные поля
    if v := strings.TrimSpace(kv["INBOUND_ID"]); v != "" {
        if n, err := strconv.Atoi(v); err == nil {
            cfg.InboundID = n
        } else {
            return cfg, err
        }
    }
    if v := strings.TrimSpace(kv["ADMIN_ID"]); v != "" {
        if n, err := strconv.ParseInt(v, 10, 64); err == nil {
            cfg.AdminID = n
        } else {
            return cfg, err
        }
    }

    return cfg, nil
}

// MustLoad загружает конфиг из стандартного пути и падает при ошибке.
// Используйте в начале исполнения программы, чтобы гарантировать наличие конфигурации.
func MustLoad() Config {
    const defaultPath = "/root/bot/.env"
    cfg, err := Load(defaultPath)
    if err != nil {
        log.Fatalf("ENV: не удалось загрузить %s: %v", defaultPath, err)
    }
    return cfg
}

// ApplyToCommon применяет загруженные значения к глобальным переменным пакета common.
// Пустые значения НЕ перезаписывают текущие, чтобы сохранять разумные дефолты.
func (c Config) ApplyToCommon() {
    // Бот
    if c.BotToken != "" {
        common.BOT_TOKEN = c.BotToken
    }
	if c.AdminID != 0 {
		common.ADMIN_ID = c.AdminID
	}
	if c.SupportLink != "" {
		common.SUPPORT_LINK = c.SupportLink
	}

	// Панель
	if c.PanelURL != "" {
		common.PANEL_URL = c.PanelURL
	}
	if c.PanelUser != "" {
		common.PANEL_USER = c.PanelUser
	}
	if c.PanelPass != "" {
		common.PANEL_PASS = c.PanelPass
	}
	if c.InboundID != 0 {
		common.INBOUND_ID = c.InboundID
	}

	// Ссылки
	if c.ConfigBaseURL != "" {
		common.CONFIG_BASE_URL = c.ConfigBaseURL
	}
	if c.ConfigJSONURL != "" {
		common.CONFIG_JSON_URL = c.ConfigJSONURL
	}
	if c.RedirectDomain != "" {
		common.REDIRECT_DOMAIN = c.RedirectDomain
	}
	if c.RedirectImport != "" {
		common.REDIRECT_IMPORT = c.RedirectImport
	}

	// Рефералы
	if c.ReferralLinkBaseURL != "" {
		common.REFERRAL_LINK_BASE_URL = c.ReferralLinkBaseURL
	}

	// ЮКасса
	if c.YukassaShopID != "" {
		common.YUKASSA_SHOP_ID = c.YukassaShopID
	}
	if c.YukassaSecretKey != "" {
		common.YUKASSA_SECRET_KEY = c.YukassaSecretKey
	}
	if c.YukassaWebhookURL != "" {
		common.YUKASSA_WEBHOOK_URL = c.YukassaWebhookURL
	}
}
