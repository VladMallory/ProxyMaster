package main

import (
	"fmt"
	"log"
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
	log.Printf("Запуск IP Ban сервиса")
	log.Printf("Максимум IP на конфиг: %d", s.MaxIPs)
	log.Printf("Интервал проверки: %v", s.CheckInterval)
	log.Printf("Период ожидания: %v", s.GracePeriod)

	go s.monitorLoop()
	return nil
}

// Stop останавливает сервис мониторинга
func (s *IPBanService) Stop() {
	if !s.Running {
		return
	}

	log.Printf("Остановка IP Ban сервиса")
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
			log.Printf("IP Ban сервис остановлен")
			return
		}
	}
}

// performCheck выполняет проверку и управление конфигами
func (s *IPBanService) performCheck() {
	log.Printf("Выполнение проверки IP ban")

	// Анализируем лог файл
	stats, err := s.Analyzer.AnalyzeLog()
	if err != nil {
		log.Printf("Ошибка анализа лога: %v", err)
		return
	}

	if len(stats) == 0 {
		fmt.Println("📝 Нет данных для анализа")
		return
	}

	// Находим подозрительные конфиги (много IP)
	suspiciousEmails := s.Analyzer.GetSuspiciousEmails(s.MaxIPs)
	log.Printf("Найдено подозрительных конфигов: %d", len(suspiciousEmails))

	// Отключаем подозрительные конфиги
	for _, stats := range suspiciousEmails {
		s.handleSuspiciousConfig(stats)
	}

	// Находим нормальные конфиги (мало IP)
	normalEmails := s.Analyzer.GetNormalEmails(s.MaxIPs)
	log.Printf("Найдено нормальных конфигов: %d", len(normalEmails))

	// Включаем нормальные конфиги
	for _, stats := range normalEmails {
		s.handleNormalConfig(stats)
	}

	// Логируем общую статистику
	log.Printf("IP_BAN: Всего email: %d, Подозрительных: %d, Нормальных: %d", len(stats), len(suspiciousEmails), len(normalEmails))

	log.Printf("Проверка IP ban завершена")
}

// handleSuspiciousConfig обрабатывает подозрительный конфиг
func (s *IPBanService) handleSuspiciousConfig(stats *EmailIPStats) {
	log.Printf("Подозрительный конфиг: %s (IP адресов: %d)", stats.Email, stats.TotalIPs)

	// Собираем список IP адресов для логирования
	var ips []string
	for ip, activity := range stats.IPs {
		ips = append(ips, ip)
		log.Printf("IP %s: соединений %d, последний раз %s",
			ip, activity.Count, activity.LastSeen.Format("15:04:05"))
	}

	// Проверяем текущий статус конфига
	currentStatus, err := s.ConfigManager.GetConfigStatus(stats.Email)
	if err != nil {
		log.Printf("Ошибка получения статуса конфига %s: %v", stats.Email, err)
		return
	}

	// Если конфиг уже отключен, ничего не делаем
	if !currentStatus {
		log.Printf("Конфиг %s уже отключен", stats.Email)
		return
	}

	// Отключаем конфиг
	log.Printf("Отключение конфига %s...", stats.Email)
	if err := s.ConfigManager.DisableConfig(stats.Email); err != nil {
		log.Printf("Ошибка отключения конфига %s: %v", stats.Email, err)
	} else {
		log.Printf("IP_BAN: ВКЛЮЧЕН конфиг %s (IP адресов %d, IP: %v)", stats.Email, stats.TotalIPs, ips)
	}
}

// handleNormalConfig обрабатывает нормальный конфиг
func (s *IPBanService) handleNormalConfig(stats *EmailIPStats) {
	// Пропускаем конфиги с 0 IP (нет активности)
	if stats.TotalIPs == 0 {
		return
	}

	log.Printf("Нормальный конфиг: %s (IP адресов: %d)", stats.Email, stats.TotalIPs)

	// Собираем список IP адресов для логирования
	var ips []string
	for ip := range stats.IPs {
		ips = append(ips, ip)
	}

	// Проверяем текущий статус конфига
	currentStatus, err := s.ConfigManager.GetConfigStatus(stats.Email)
	if err != nil {
		log.Printf("Ошибка получения статуса конфига %s: %v", stats.Email, err)
		return
	}

	// Если конфиг уже включен, ничего не делаем
	if currentStatus {
		log.Printf("Конфиг %s уже включен", stats.Email)
		return
	}

	// Включаем конфиг
	log.Printf("Включение конфига %s...", stats.Email)
	if err := s.ConfigManager.EnableConfig(stats.Email); err != nil {
		log.Printf("Ошибка включения конфига %s: %v", stats.Email, err)
	} else {
		log.Printf("IP_BAN: ВКЛЮЧЕН конфиг %s (IP адресов %d, IP: %v)", stats.Email, stats.TotalIPs, ips)
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
