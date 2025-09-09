package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Конфигурация (в реальном проекте эти значения должны загружаться из config.go)
const (
	DefaultPanelURL      = "https://domen:123/path/"
	DefaultPanelUser     = "C7QKEuj"
	DefaultPanelPass     = "cXFEwoD"
	DefaultInboundID     = 1
	DefaultAccessLog     = "/usr/local/x-ui/access.log"
	DefaultMaxIPs        = 2
	DefaultCheckInterval = 5 * time.Minute
	DefaultGracePeriod   = 10 * time.Minute
)

func main() {
	// Парсинг аргументов командной строки
	var (
		panelURL      = flag.String("panel-url", DefaultPanelURL, "URL панели x-ui")
		panelUser     = flag.String("panel-user", DefaultPanelUser, "Пользователь панели")
		panelPass     = flag.String("panel-pass", DefaultPanelPass, "Пароль панели")
		inboundID     = flag.Int("inbound-id", DefaultInboundID, "ID inbound")
		accessLog     = flag.String("access-log", DefaultAccessLog, "Путь к файлу access.log")
		maxIPs        = flag.Int("max-ips", DefaultMaxIPs, "Максимальное количество IP на конфиг")
		checkInterval = flag.Duration("check-interval", DefaultCheckInterval, "Интервал проверки")
		gracePeriod   = flag.Duration("grace-period", DefaultGracePeriod, "Период ожидания перед отключением")
		showStats     = flag.Bool("stats", false, "Показать статистику и выйти")
		showConfigs   = flag.Bool("list-configs", false, "Показать список конфигов и выйти")
		enableEmail   = flag.String("enable", "", "Включить конфиг по email")
		disableEmail  = flag.String("disable", "", "Отключить конфиг по email")
	)
	flag.Parse()

	// Создаем анализатор логов
	analyzer := NewLogAnalyzer(*accessLog)

	// Создаем менеджер конфигураций
	configManager := NewConfigManager(*panelURL, *panelUser, *panelPass, *inboundID)

	// Обработка специальных команд
	if *showStats {
		handleShowStats(analyzer, *maxIPs)
		return
	}

	if *showConfigs {
		handleShowConfigs(configManager)
		return
	}

	if *enableEmail != "" {
		handleEnableConfig(configManager, *enableEmail)
		return
	}

	if *disableEmail != "" {
		handleDisableConfig(configManager, *disableEmail)
		return
	}

	// Создаем и запускаем сервис
	service := NewIPBanService(analyzer, configManager, *maxIPs, *checkInterval, *gracePeriod)

	// Обработка сигналов для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем сервис
	if err := service.Start(); err != nil {
		log.Fatalf("❌ Ошибка запуска сервиса: %v", err)
	}

	fmt.Println("🎯 IP Ban сервис запущен. Нажмите Ctrl+C для остановки.")

	// Ожидаем сигнал остановки
	<-sigChan
	fmt.Println("\n🛑 Получен сигнал остановки...")
	service.Stop()
}

// handleShowStats показывает статистику
func handleShowStats(analyzer *LogAnalyzer, maxIPs int) {
	fmt.Println("📊 Анализ статистики IP адресов")
	fmt.Println(strings.Repeat("=", 50))

	stats, err := analyzer.AnalyzeLog()
	if err != nil {
		log.Fatalf("❌ Ошибка анализа лога: %v", err)
	}

	if len(stats) == 0 {
		fmt.Println("📝 Нет данных для анализа")
		return
	}

	// Показываем общую статистику
	suspiciousEmails := analyzer.GetSuspiciousEmails(maxIPs)
	normalEmails := analyzer.GetNormalEmails(maxIPs)

	fmt.Printf("📈 Всего email: %d\n", len(stats))
	fmt.Printf("🚨 Подозрительных (IP > %d): %d\n", maxIPs, len(suspiciousEmails))
	fmt.Printf("✅ Нормальных (IP ≤ %d): %d\n", maxIPs, len(normalEmails))
	fmt.Println()

	// Показываем детальную статистику
	analyzer.PrintStats()
}

// handleShowConfigs показывает список конфигураций
func handleShowConfigs(configManager *ConfigManager) {
	fmt.Println("📋 Список конфигураций")
	fmt.Println(strings.Repeat("=", 50))

	if err := configManager.ListAllConfigs(); err != nil {
		log.Fatalf("❌ Ошибка получения списка конфигураций: %v", err)
	}
}

// handleEnableConfig включает конфиг
func handleEnableConfig(configManager *ConfigManager, email string) {
	fmt.Printf("🔓 Включение конфига для email: %s\n", email)

	if err := configManager.EnableConfig(email); err != nil {
		log.Fatalf("❌ Ошибка включения конфига: %v", err)
	}

	fmt.Println("✅ Конфиг успешно включен")
}

// handleDisableConfig отключает конфиг
func handleDisableConfig(configManager *ConfigManager, email string) {
	fmt.Printf("🔒 Отключение конфига для email: %s\n", email)

	if err := configManager.DisableConfig(email); err != nil {
		log.Fatalf("❌ Ошибка отключения конфига: %v", err)
	}

	fmt.Println("✅ Конфиг успешно отключен")
}

// printUsage выводит справку по использованию
func printUsage() {
	fmt.Println("🎯 IP Ban System - Система автоматического управления конфигами")
	fmt.Println()
	fmt.Println("Использование:")
	fmt.Println("  go run . [опции]")
	fmt.Println()
	fmt.Println("Опции:")
	fmt.Println("  -panel-url string")
	fmt.Println("        URL панели x-ui (по умолчанию: " + DefaultPanelURL + ")")
	fmt.Println("  -panel-user string")
	fmt.Println("        Пользователь панели (по умолчанию: " + DefaultPanelUser + ")")
	fmt.Println("  -panel-pass string")
	fmt.Println("        Пароль панели")
	fmt.Println("  -inbound-id int")
	fmt.Println("        ID inbound (по умолчанию: 3)")
	fmt.Println("  -access-log string")
	fmt.Println("        Путь к файлу access.log (по умолчанию: " + DefaultAccessLog + ")")
	fmt.Println("  -max-ips int")
	fmt.Println("        Максимальное количество IP на конфиг (по умолчанию: 2)")
	fmt.Println("  -check-interval duration")
	fmt.Println("        Интервал проверки (по умолчанию: 5m)")
	fmt.Println("  -grace-period duration")
	fmt.Println("        Период ожидания перед отключением (по умолчанию: 10m)")
	fmt.Println("  -stats")
	fmt.Println("        Показать статистику и выйти")
	fmt.Println("  -list-configs")
	fmt.Println("        Показать список конфигов и выйти")
	fmt.Println("  -enable string")
	fmt.Println("        Включить конфиг по email")
	fmt.Println("  -disable string")
	fmt.Println("        Отключить конфиг по email")
	fmt.Println()
	fmt.Println("Примеры:")
	fmt.Println("  go run .                                    # Запуск сервиса")
	fmt.Println("  go run . -stats                             # Показать статистику")
	fmt.Println("  go run . -list-configs                      # Показать конфиги")
	fmt.Println("  go run . -enable 123456789                  # Включить конфиг")
	fmt.Println("  go run . -disable 123456789                 # Отключить конфиг")
	fmt.Println("  go run . -max-ips 3 -check-interval 10m     # Настройка параметров")
}
