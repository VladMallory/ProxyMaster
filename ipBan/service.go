package main

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// IPBanService основной сервис для мониторинга и управления IP банами
type IPBanService struct {
	Analyzer      *LogAnalyzer
	ConfigManager *ConfigManager
	MaxIPs        int
	CheckInterval time.Duration
	GracePeriod   time.Duration
	Running       bool
	StopChan      chan bool
}

// NewIPBanService создает новый сервис IP бана
func NewIPBanService(analyzer *LogAnalyzer, configManager *ConfigManager, maxIPs int, checkInterval, gracePeriod time.Duration) *IPBanService {
	return &IPBanService{
		Analyzer:      analyzer,
		ConfigManager: configManager,
		MaxIPs:        maxIPs,
		CheckInterval: checkInterval,
		GracePeriod:   gracePeriod,
		Running:       false,
		StopChan:      make(chan bool, 1),
	}
}

// Start запускает сервис мониторинга
func (s *IPBanService) Start() error {
	if s.Running {
		return fmt.Errorf("сервис уже запущен")
	}

	s.Running = true
	fmt.Printf("🚀 Запуск IP Ban сервиса...\n")
	fmt.Printf("📊 Максимум IP на конфиг: %d\n", s.MaxIPs)
	fmt.Printf("⏰ Интервал проверки: %v\n", s.CheckInterval)
	fmt.Printf("⏳ Период ожидания: %v\n", s.GracePeriod)
	fmt.Println(strings.Repeat("=", 50))

	go s.monitorLoop()
	return nil
}

// Stop останавливает сервис мониторинга
func (s *IPBanService) Stop() {
	if !s.Running {
		return
	}

	fmt.Println("🛑 Остановка IP Ban сервиса...")
	s.Running = false
	s.StopChan <- true
}

// monitorLoop основной цикл мониторинга
func (s *IPBanService) monitorLoop() {
	ticker := time.NewTicker(s.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performCheck()
		case <-s.StopChan:
			fmt.Println("✅ IP Ban сервис остановлен")
			return
		}
	}
}

// performCheck выполняет проверку и управление конфигами
func (s *IPBanService) performCheck() {
	fmt.Printf("\n🔍 [%s] Выполнение проверки...\n", time.Now().Format("2006-01-02 15:04:05"))

	// Анализируем лог файл
	stats, err := s.Analyzer.AnalyzeLog()
	if err != nil {
		log.Printf("❌ Ошибка анализа лога: %v", err)
		return
	}

	if len(stats) == 0 {
		fmt.Println("📝 Нет данных для анализа")
		return
	}

	// Находим подозрительные конфиги (много IP)
	suspiciousEmails := s.Analyzer.GetSuspiciousEmails(s.MaxIPs)
	fmt.Printf("🚨 Найдено подозрительных конфигов: %d\n", len(suspiciousEmails))

	// Отключаем подозрительные конфиги
	for _, stats := range suspiciousEmails {
		s.handleSuspiciousConfig(stats)
	}

	// Находим нормальные конфиги (мало IP)
	normalEmails := s.Analyzer.GetNormalEmails(s.MaxIPs)
	fmt.Printf("✅ Найдено нормальных конфигов: %d\n", len(normalEmails))

	// Включаем нормальные конфиги
	for _, stats := range normalEmails {
		s.handleNormalConfig(stats)
	}

	fmt.Println("✅ Проверка завершена")
}

// handleSuspiciousConfig обрабатывает подозрительный конфиг
func (s *IPBanService) handleSuspiciousConfig(stats *EmailIPStats) {
	fmt.Printf("🚨 Подозрительный конфиг: %s (IP адресов: %d)\n", stats.Email, stats.TotalIPs)

	// Выводим список IP адресов
	for ip, activity := range stats.IPs {
		fmt.Printf("   📍 %s (соединений: %d, последний раз: %s)\n",
			ip,
			activity.Count,
			activity.LastSeen.Format("15:04:05"))
	}

	// Проверяем текущий статус конфига
	currentStatus, err := s.ConfigManager.GetConfigStatus(stats.Email)
	if err != nil {
		log.Printf("❌ Ошибка получения статуса конфига %s: %v", stats.Email, err)
		return
	}

	// Если конфиг уже отключен, ничего не делаем
	if !currentStatus {
		fmt.Printf("   ℹ️  Конфиг %s уже отключен\n", stats.Email)
		return
	}

	// Отключаем конфиг
	fmt.Printf("   🔒 Отключение конфига %s...\n", stats.Email)
	if err := s.ConfigManager.DisableConfig(stats.Email); err != nil {
		log.Printf("❌ Ошибка отключения конфига %s: %v", stats.Email, err)
	} else {
		fmt.Printf("   ✅ Конфиг %s успешно отключен\n", stats.Email)
	}
}

// handleNormalConfig обрабатывает нормальный конфиг
func (s *IPBanService) handleNormalConfig(stats *EmailIPStats) {
	// Пропускаем конфиги с 0 IP (нет активности)
	if stats.TotalIPs == 0 {
		return
	}

	fmt.Printf("✅ Нормальный конфиг: %s (IP адресов: %d)\n", stats.Email, stats.TotalIPs)

	// Проверяем текущий статус конфига
	currentStatus, err := s.ConfigManager.GetConfigStatus(stats.Email)
	if err != nil {
		log.Printf("❌ Ошибка получения статуса конфига %s: %v", stats.Email, err)
		return
	}

	// Если конфиг уже включен, ничего не делаем
	if currentStatus {
		fmt.Printf("   ℹ️  Конфиг %s уже включен\n", stats.Email)
		return
	}

	// Включаем конфиг
	fmt.Printf("   🔓 Включение конфига %s...\n", stats.Email)
	if err := s.ConfigManager.EnableConfig(stats.Email); err != nil {
		log.Printf("❌ Ошибка включения конфига %s: %v", stats.Email, err)
	} else {
		fmt.Printf("   ✅ Конфиг %s успешно включен\n", stats.Email)
	}
}

// GetStatus возвращает текущий статус сервиса
func (s *IPBanService) GetStatus() map[string]interface{} {
	stats, err := s.Analyzer.AnalyzeLog()
	if err != nil {
		return map[string]interface{}{
			"running": s.Running,
			"error":   err.Error(),
		}
	}

	suspiciousCount := len(s.Analyzer.GetSuspiciousEmails(s.MaxIPs))
	normalCount := len(s.Analyzer.GetNormalEmails(s.MaxIPs))

	return map[string]interface{}{
		"running":            s.Running,
		"total_emails":       len(stats),
		"suspicious_count":   suspiciousCount,
		"normal_count":       normalCount,
		"max_ips_per_config": s.MaxIPs,
		"check_interval":     s.CheckInterval.String(),
		"grace_period":       s.GracePeriod.String(),
	}
}

// PrintCurrentStats выводит текущую статистику
func (s *IPBanService) PrintCurrentStats() {
	fmt.Println("\n📊 Текущая статистика:")

	stats, err := s.Analyzer.AnalyzeLog()
	if err != nil {
		fmt.Printf("❌ Ошибка получения статистики: %v\n", err)
		return
	}

	if len(stats) == 0 {
		fmt.Println("📝 Нет данных для отображения")
		return
	}

	suspiciousEmails := s.Analyzer.GetSuspiciousEmails(s.MaxIPs)
	normalEmails := s.Analyzer.GetNormalEmails(s.MaxIPs)

	fmt.Printf("📈 Всего email: %d\n", len(stats))
	fmt.Printf("🚨 Подозрительных: %d\n", len(suspiciousEmails))
	fmt.Printf("✅ Нормальных: %d\n", len(normalEmails))

	fmt.Println("\n📋 Детальная статистика:")
	for email, emailStats := range stats {
		status := "✅ Нормальный"
		if emailStats.TotalIPs > s.MaxIPs {
			status = "🚨 Подозрительный"
		}

		fmt.Printf("  %s %s: %d IP\n", status, email, emailStats.TotalIPs)
	}
}
