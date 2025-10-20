package common

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// IPBanService основной сервис для мониторинга и управления IP банами
type IPBanService struct {
	Analyzer      *LogAnalyzer
	ConfigManager *ConfigManager
	BanManager    *BanManager
	IPTables      *IPTablesManager // Менеджер для работы с iptables
	MaxIPs        int
	CheckInterval time.Duration
	GracePeriod   time.Duration
	Running       bool
	StopChan      chan bool
}

// NewIPBanService создает новый сервис IP бана
func NewIPBanService(analyzer *LogAnalyzer, configManager *ConfigManager, banManager *BanManager, iptables *IPTablesManager, maxIPs int, checkInterval, gracePeriod time.Duration) *IPBanService {
	return &IPBanService{
		Analyzer:      analyzer,
		ConfigManager: configManager,
		BanManager:    banManager,
		IPTables:      iptables,
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

	// Получаем все конфиги из панели
	allConfigs, err := s.ConfigManager.GetConfigs()
	if err != nil {
		log.Printf("❌ Ошибка получения конфигов из панели: %v", err)
		return
	}

	if len(allConfigs) == 0 {
		fmt.Println("📝 Нет конфигов для анализа")
		return
	}

	// Анализируем лог файл для получения статистики IP
	logStats, err := s.Analyzer.AnalyzeLog()
	if err != nil {
		log.Printf("❌ Ошибка анализа лога: %v", err)
		return
	}

	// Создаем карту статистики IP по email
	ipStatsMap := make(map[string]*EmailIPStats)
	for _, stats := range logStats {
		ipStatsMap[stats.Email] = stats
	}

	// Очищаем истекшие баны
	s.BanManager.CleanupExpiredBans()

	// Очищаем старые баны (которые истекли дольше IP_COUNTER_RETENTION назад)
	s.BanManager.CleanupOldBans(IP_COUNTER_RETENTION)

	// Обрабатываем каждый конфиг из панели
	suspiciousCount := 0
	normalCount := 0
	enabledCount := 0
	bannedCount := 0

	for _, config := range allConfigs {
		// Проверяем, не забанен ли пользователь
		if s.BanManager.IsBanned(config.Email) {
			banInfo := s.BanManager.GetBanInfo(config.Email)
			fmt.Printf("🚫 Забаненный конфиг: %s (бан до: %s)\n", config.Email, banInfo.ExpiresAt.Format("15:04:05 02.01.2006"))
			bannedCount++

			// ВАЖНО: Проверяем, включен ли забаненный конфиг в панели, и отключаем его
			if config.Enable {
				fmt.Printf("   🔒 Забаненный конфиг %s включен в панели - отключаем!\n", config.Email)
				// Логируем в bot.log: начало отключения забаненного конфига
				LogIPBanInfo("Отключение забаненного конфига %s (превышение лимита IP)", config.Email)
				if err := s.ConfigManager.DisableConfig(config.Email); err != nil {
					fmt.Printf("❌ Ошибка отключения забаненного конфига %s: %v\n", config.Email, err)
					// Логируем в bot.log: ошибка отключения конфига
					LogIPBanError("Ошибка отключения забаненного конфига %s: %v", config.Email, err)
				} else {
					fmt.Printf("   ✅ Забаненный конфиг %s успешно отключен в панели\n", config.Email)
					// Логируем в bot.log: успешное отключение конфига (IP адреса недоступны для забаненных конфигов)
					LogIPBanAction("ОТКЛЮЧЕН", config.Email, 0, []string{})
				}
			} else {
				fmt.Printf("   ℹ️  Забаненный конфиг %s уже отключен в панели\n", config.Email)
			}
			continue
		}

		// Получаем статистику IP для этого конфига
		ipStats, hasActivity := ipStatsMap[config.Email]

		if hasActivity {
			// Конфиг имеет активность в логах
			if ipStats.TotalIPs > s.MaxIPs {
				// Подозрительный конфиг - баним
				suspiciousCount++
				s.handleSuspiciousConfig(ipStats)
			} else {
				// Нормальный конфиг - включаем
				normalCount++
				s.handleNormalConfig(ipStats)
			}
		} else {
			// Конфиг не имеет активности в логах
			if !config.Enable {
				// Отключенный конфиг без активности - включаем
				fmt.Printf("✅ Конфиг без активности: %s (отключен, включаем)\n", config.Email)
				// Логируем в bot.log: начало включения конфига без активности
				LogIPBanInfo("Включение конфига без активности %s", config.Email)
				if err := s.ConfigManager.EnableConfig(config.Email); err != nil {
					log.Printf("❌ Ошибка включения конфига %s: %v", config.Email, err)
					// Логируем в bot.log: ошибка включения конфига
					LogIPBanError("Ошибка включения конфига без активности %s: %v", config.Email, err)
				} else {
					fmt.Printf("   ✅ Конфиг %s успешно включен\n", config.Email)
					// Логируем в bot.log: успешное включение конфига без активности
					LogIPBanAction("ВКЛЮЧЕН", config.Email, 0, []string{})
					enabledCount++
				}
			} else {
				// Включенный конфиг без активности - оставляем как есть
				fmt.Printf("ℹ️  Конфиг без активности: %s (включен, оставляем)\n", config.Email)
			}
		}
	}

	fmt.Printf("🚨 Подозрительных конфигов: %d\n", suspiciousCount)
	fmt.Printf("✅ Нормальных конфигов: %d\n", normalCount)
	fmt.Printf("🔓 Включено отключенных: %d\n", enabledCount)
	fmt.Printf("🚫 Забаненных конфигов: %d\n", bannedCount)
	fmt.Println("✅ Проверка завершена")
}

// handleSuspiciousConfig обрабатывает подозрительный конфиг
func (s *IPBanService) handleSuspiciousConfig(stats *EmailIPStats) {
	fmt.Printf("🚨 Подозрительный конфиг: %s (IP адресов: %d, максимум: %d)\n",
		stats.Email, stats.TotalIPs, s.MaxIPs)

	// Собираем список IP адресов для уведомления
	var ipAddresses []string
	for ip, activity := range stats.IPs {
		fmt.Printf("   📍 %s (соединений: %d, последний раз: %s)\n",
			ip,
			activity.Count,
			activity.LastSeen.Format("15:04:05"))
		ipAddresses = append(ipAddresses, ip)
	}

	// Проверяем, не забанен ли уже пользователь
	if s.BanManager.IsBanned(stats.Email) {
		banInfo := s.BanManager.GetBanInfo(stats.Email)
		fmt.Printf("   ℹ️  Пользователь %s уже забанен до %s, пропускаем повторный бан\n",
			stats.Email, banInfo.ExpiresAt.Format("15:04:05 02.01.2006"))
		return
	}

	// Баним пользователя
	reason := fmt.Sprintf("Превышение лимита IP адресов: %d (максимум: %d)", stats.TotalIPs, s.MaxIPs)
	// Логируем в bot.log: начало банирования пользователя
	LogIPBanInfo("Начало банирования пользователя %s (IP адресов: %d, лимит: %d)", stats.Email, stats.TotalIPs, s.MaxIPs)

	if err := s.BanManager.BanUser(stats.Email, reason, ipAddresses); err != nil {
		log.Printf("❌ Ошибка бана пользователя %s: %v", stats.Email, err)
		// Логируем в bot.log: ошибка банирования
		LogIPBanError("Ошибка банирования пользователя %s: %v", stats.Email, err)
		return
	}

	fmt.Printf("   🚫 Пользователь %s забанен на %d минут\n", stats.Email, IP_BAN_DURATION)
	// Логируем в bot.log: успешное банирование
	LogIPBanInfo("Пользователь %s успешно забанен на %d минут", stats.Email, IP_BAN_DURATION)

	// Мгновенно отключаем конфиг и ротируем UUID, чтобы обрубить активные сессии без рестарта Xray
	fmt.Printf("   🔒 Отключение и ротация UUID для %s...\n", stats.Email)
	if _, err := s.ConfigManager.DisableAndRotateConfig(stats.Email); err != nil {
		log.Printf("❌ Ошибка DisableAndRotateConfig для %s: %v", stats.Email, err)
	} else {
		fmt.Printf("   ✅ Конфиг %s отключён, UUID обновлён\n", stats.Email)
	}
}

// handleNormalConfig обрабатывает нормальный конфиг
func (s *IPBanService) handleNormalConfig(stats *EmailIPStats) {
	fmt.Printf("✅ Нормальный конфиг: %s (IP адресов: %d)\n", stats.Email, stats.TotalIPs)

	// Проверяем, не забанен ли пользователь
	if s.BanManager.IsBanned(stats.Email) {
		// Если пользователь забанен, но активность нормализовалась, разблокируем IP
		fmt.Printf("   🔓 Разблокировка IP адресов для %s...\n", stats.Email)
		// Логируем в bot.log: начало разблокировки IP для забаненного пользователя
		LogIPBanInfo("Разблокировка IP адресов для забаненного пользователя %s (активность нормализовалась)", stats.Email)

		unblockedCount := 0
		for ip := range stats.IPs {
			if err := s.IPTables.UnblockIP(ip); err != nil {
				log.Printf("❌ Ошибка разблокировки IP %s: %v", ip, err)
				// Логируем в bot.log: ошибка разблокировки IP
				LogIPBanError("Ошибка разблокировки IP %s для пользователя %s: %v", ip, stats.Email, err)
			} else {
				unblockedCount++
				// Логируем в bot.log: успешная разблокировка IP
				LogIPBanInfo("IP %s успешно разблокирован для пользователя %s", ip, stats.Email)
			}
		}

		if unblockedCount > 0 {
			fmt.Printf("   ✅ Разблокировано %d IP адресов через iptables\n", unblockedCount)
		}
	} else {
		// ВАЖНО: Проверяем статус конфига в панели - если он отключен, включаем его
		currentStatus, err := s.ConfigManager.GetConfigStatus(stats.Email)
		if err != nil {
			log.Printf("❌ Ошибка получения статуса нормального конфига %s: %v", stats.Email, err)
		} else if !currentStatus {
			// Конфиг отключен в панели, но активность нормальная - включаем его
			fmt.Printf("   🔓 Нормальный конфиг %s отключен в панели - включаем!\n", stats.Email)
			// Логируем в bot.log: начало включения нормального конфига
			LogIPBanInfo("Включение нормального конфига %s (активность в пределах нормы)", stats.Email)
			if err := s.ConfigManager.EnableConfig(stats.Email); err != nil {
				log.Printf("❌ Ошибка включения нормального конфига %s: %v", stats.Email, err)
				// Логируем в bot.log: ошибка включения нормального конфига
				LogIPBanError("Ошибка включения нормального конфига %s: %v", stats.Email, err)
			} else {
				fmt.Printf("   ✅ Нормальный конфиг %s успешно включен в панели\n", stats.Email)
				// Логируем в bot.log: успешное включение нормального конфига с деталями IP
				LogIPBanAction("ВКЛЮЧЕН", stats.Email, stats.TotalIPs, getIPAddressesFromStats(stats))
			}
		} else {
			fmt.Printf("   ℹ️  Конфиг %s работает нормально и уже включен\n", stats.Email)
		}
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

// IPTablesManager управляет блокировкой IP через iptables
type IPTablesManager struct {
	BlockedIPs map[string]bool // Карта заблокированных IP
}

// NewIPTablesManager создает новый менеджер iptables
func NewIPTablesManager() *IPTablesManager {
	return &IPTablesManager{
		BlockedIPs: make(map[string]bool),
	}
}

// BlockIP блокирует IP адрес через iptables
func (i *IPTablesManager) BlockIP(ipAddress string) error {
	// Проверяем, не заблокирован ли уже IP
	if i.BlockedIPs[ipAddress] {
		fmt.Printf("ℹ️  IP %s уже заблокирован\n", ipAddress)
		// Логируем в bot.log: IP уже заблокирован
		LogIPBanInfo("IP %s уже заблокирован через iptables", ipAddress)
		return nil
	}

	// Логируем в bot.log: начало блокировки IP
	LogIPBanInfo("Блокировка IP %s через iptables", ipAddress)

	// Блокируем IP через iptables
	cmd := fmt.Sprintf("iptables -I INPUT -s %s -j DROP", ipAddress)
	if err := i.executeCommand(cmd); err != nil {
		// Логируем в bot.log: ошибка блокировки IP
		LogIPBanError("Ошибка блокировки IP %s через iptables: %v", ipAddress, err)
		return fmt.Errorf("ошибка блокировки IP %s: %v", ipAddress, err)
	}

	// Добавляем IP в список заблокированных
	i.BlockedIPs[ipAddress] = true
	fmt.Printf("✅ IP %s успешно заблокирован через iptables\n", ipAddress)

	// Логируем в bot.log: успешная блокировка IP
	LogIPBanAction("IP_ЗАБЛОКИРОВАН", ipAddress, 0, []string{})

	return nil
}

// UnblockIP разблокирует IP адрес через iptables
func (i *IPTablesManager) UnblockIP(ipAddress string) error {
	// Проверяем, заблокирован ли IP
	if !i.BlockedIPs[ipAddress] {
		fmt.Printf("ℹ️  IP %s не был заблокирован\n", ipAddress)
		// Логируем в bot.log: IP не был заблокирован
		LogIPBanInfo("IP %s не был заблокирован через iptables", ipAddress)
		return nil
	}

	// Логируем в bot.log: начало разблокировки IP
	LogIPBanInfo("Разблокировка IP %s через iptables", ipAddress)

	// Разблокируем IP через iptables
	cmd := fmt.Sprintf("iptables -D INPUT -s %s -j DROP", ipAddress)
	if err := i.executeCommand(cmd); err != nil {
		// Логируем в bot.log: ошибка разблокировки IP
		LogIPBanError("Ошибка разблокировки IP %s через iptables: %v", ipAddress, err)
		return fmt.Errorf("ошибка разблокировки IP %s: %v", ipAddress, err)
	}

	// Удаляем IP из списка заблокированных
	delete(i.BlockedIPs, ipAddress)
	fmt.Printf("✅ IP %s успешно разблокирован через iptables\n", ipAddress)

	// Логируем в bot.log: успешная разблокировка IP
	LogIPBanAction("IP_РАЗБЛОКИРОВАН", ipAddress, 0, []string{})

	return nil
}

// executeCommand выполняет команду в системе
func (i *IPTablesManager) executeCommand(cmd string) error {
	// Используем os/exec для выполнения команды
	parts := strings.Fields(cmd)
	if len(parts) < 2 {
		return fmt.Errorf("неверная команда: %s", cmd)
	}

	// Выполняем команду
	execCmd := exec.Command(parts[0], parts[1:]...)
	output, err := execCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка выполнения команды '%s': %v, output: %s", cmd, err, string(output))
	}

	return nil
}

// getIPAddressesFromStats извлекает IP адреса из статистики для логирования
func getIPAddressesFromStats(stats *EmailIPStats) []string {
	var ips []string
	for ip := range stats.IPs {
		ips = append(ips, ip)
	}
	return ips
}

// GetBlockedIPs возвращает список заблокированных IP
func (i *IPTablesManager) GetBlockedIPs() []string {
	var ips []string
	for ip := range i.BlockedIPs {
		ips = append(ips, ip)
	}
	return ips
}

// IsIPBlocked проверяет, заблокирован ли IP
func (i *IPTablesManager) IsIPBlocked(ipAddress string) bool {
	return i.BlockedIPs[ipAddress]
}
