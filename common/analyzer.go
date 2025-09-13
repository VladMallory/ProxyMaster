package common

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// IPActivity представляет активность IP адреса для конкретного email
type IPActivity struct {
	Email     string
	IPAddress string
	LastSeen  time.Time
	Count     int
}

// EmailIPStats содержит статистику по IP адресам для email
type EmailIPStats struct {
	Email      string
	IPs        map[string]*IPActivity
	TotalIPs   int
	LastUpdate time.Time
}

// LogAnalyzer анализирует access.log файл
type LogAnalyzer struct {
	LogPath     string
	Stats       map[string]*EmailIPStats
	LastReadPos int64 // Позиция последнего прочитанного байта
}

// NewLogAnalyzer создает новый анализатор логов
func NewLogAnalyzer(logPath string) *LogAnalyzer {
	return &LogAnalyzer{
		LogPath:     logPath,
		Stats:       make(map[string]*EmailIPStats),
		LastReadPos: 0,
	}
}

// AnalyzeLog анализирует накопленный файл логов и возвращает статистику по email и IP
func (la *LogAnalyzer) AnalyzeLog() (map[string]*EmailIPStats, error) {
	// Сначала очищаем старые данные
	la.CleanupOldData(IP_COUNTER_RETENTION)

	// Используем накопленный файл вместо исходного access.log
	accumulatedPath := IP_ACCUMULATED_PATH
	file, err := os.Open(accumulatedPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия накопленного файла %s: %v", accumulatedPath, err)
	}
	defer file.Close()

	// Получаем размер файла
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения информации о файле: %v", err)
	}

	// Если файл пустой, возвращаем существующую статистику
	if fileInfo.Size() == 0 {
		fmt.Printf("📄 Накопленный файл пуст, возвращаем существующую статистику\n")
		return la.Stats, nil
	}

	fmt.Printf("📄 Анализируем накопленный файл размером %d байт\n", fileInfo.Size())

	// Регулярное выражение для парсинга строк лога
	// Формат: 2025/09/04 10:17:03.008517 from 123.123.123.123:52624 accepted tcp:courier.push.apple.com:443 [inbound-443 >> direct] email: 123456789
	logRegex := regexp.MustCompile(`(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}\.\d+) from (\d+\.\d+\.\d+\.\d+):\d+ accepted.*email: (\d+)`)

	scanner := bufio.NewScanner(file)
	processedLines := 0
	for scanner.Scan() {
		line := scanner.Text()

		// Пропускаем пустые строки
		if len(line) == 0 {
			continue
		}

		// Пропускаем строки с localhost (127.0.0.1) - это системные вызовы
		if strings.Contains(line, "127.0.0.1") {
			continue
		}

		matches := logRegex.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue
		}

		timestampStr := matches[1]
		ipAddress := matches[2]
		email := matches[3]

		// Парсим время
		timestamp, err := time.Parse("2006/01/02 15:04:05.000000", timestampStr)
		if err != nil {
			continue
		}

		// Проверяем, что запись не слишком старая (используем IP_COUNTER_RETENTION)
		now := time.Now()
		maxAge := time.Duration(IP_COUNTER_RETENTION) * time.Minute
		if maxAge > 0 && timestamp.Before(now.Add(-maxAge)) {
			continue
		}

		processedLines++

		// Инициализируем статистику для email если её нет
		if la.Stats[email] == nil {
			la.Stats[email] = &EmailIPStats{
				Email:      email,
				IPs:        make(map[string]*IPActivity),
				LastUpdate: timestamp,
			}
		}

		// Обновляем статистику для IP адреса
		if la.Stats[email].IPs[ipAddress] == nil {
			la.Stats[email].IPs[ipAddress] = &IPActivity{
				Email:     email,
				IPAddress: ipAddress,
				LastSeen:  timestamp,
				Count:     1,
			}
		} else {
			la.Stats[email].IPs[ipAddress].Count++
			if timestamp.After(la.Stats[email].IPs[ipAddress].LastSeen) {
				la.Stats[email].IPs[ipAddress].LastSeen = timestamp
			}
		}

		// Обновляем общее время последнего обновления
		if timestamp.After(la.Stats[email].LastUpdate) {
			la.Stats[email].LastUpdate = timestamp
		}
	}

	// Подсчитываем общее количество уникальных IP для каждого email
	for _, stats := range la.Stats {
		stats.TotalIPs = len(stats.IPs)
	}

	fmt.Printf("📊 Обработано строк: %d, найдено email: %d\n", processedLines, len(la.Stats))

	return la.Stats, scanner.Err()
}

// GetSuspiciousEmails возвращает email с подозрительной активностью (много IP)
func (la *LogAnalyzer) GetSuspiciousEmails(maxIPs int) []*EmailIPStats {
	var suspicious []*EmailIPStats

	for _, stats := range la.Stats {
		if stats.TotalIPs > maxIPs {
			suspicious = append(suspicious, stats)
		}
	}

	return suspicious
}

// GetNormalEmails возвращает email с нормальной активностью (мало IP)
func (la *LogAnalyzer) GetNormalEmails(maxIPs int) []*EmailIPStats {
	var normal []*EmailIPStats

	for _, stats := range la.Stats {
		if stats.TotalIPs <= maxIPs {
			normal = append(normal, stats)
		}
	}

	return normal
}

// PrintStats выводит статистику в консоль
func (la *LogAnalyzer) PrintStats() {
	fmt.Println("=== Статистика IP адресов по email ===")
	for _, stats := range la.Stats {
		fmt.Printf("Email: %s, IP адресов: %d\n", stats.Email, stats.TotalIPs)
		for ip, activity := range stats.IPs {
			fmt.Printf("  - %s (последний раз: %s, соединений: %d)\n",
				ip,
				activity.LastSeen.Format("2006-01-02 15:04:05"),
				activity.Count)
		}
		fmt.Println()
	}
}

// GetEmailIPs возвращает список IP адресов для конкретного email
func (la *LogAnalyzer) GetEmailIPs(email string) []string {
	if stats, exists := la.Stats[email]; exists {
		var ips []string
		for ip := range stats.IPs {
			ips = append(ips, ip)
		}
		return ips
	}
	return nil
}

// CleanupOldData очищает старые данные на основе времени хранения
func (la *LogAnalyzer) CleanupOldData(retentionMinutes int) {
	if retentionMinutes <= 0 {
		return // Если время хранения = 0, данные хранятся бесконечно
	}

	cutoffTime := time.Now().Add(-time.Duration(retentionMinutes) * time.Minute)
	fmt.Printf("🧹 Очистка старых данных: удаляются IP адреса старше %d минут\n", retentionMinutes)
	fmt.Printf("🧹 Текущее время: %s, время отсечения: %s\n", time.Now().Format("15:04:05"), cutoffTime.Format("15:04:05"))

	// Очищаем старые IP адреса для каждого email
	for email, stats := range la.Stats {
		ipsToRemove := make([]string, 0)

		for ip, activity := range stats.IPs {
			diffMinutes := int(time.Since(activity.LastSeen).Minutes())
			fmt.Printf("🧹 IP %s для %s: последний раз %s (%d мин назад), лимит %d мин\n",
				ip, email,
				activity.LastSeen.Format("15:04:05"),
				diffMinutes,
				retentionMinutes)

			if activity.LastSeen.Before(cutoffTime) {
				ipsToRemove = append(ipsToRemove, ip)
				fmt.Printf("🧹 ❌ УДАЛЯЕМ старый IP %s (возраст %d > %d мин)\n", ip, diffMinutes, retentionMinutes)
			} else {
				fmt.Printf("🧹 ✅ ОСТАВЛЯЕМ свежий IP %s (возраст %d < %d мин)\n", ip, diffMinutes, retentionMinutes)
			}
		}

		// Удаляем старые IP адреса
		for _, ip := range ipsToRemove {
			delete(stats.IPs, ip)
		}

		// Обновляем общее количество IP
		stats.TotalIPs = len(stats.IPs)

		// Если у email больше нет IP адресов, удаляем его полностью
		if stats.TotalIPs == 0 {
			delete(la.Stats, email)
		}
	}

	fmt.Printf("🧹 Очистка старых данных: удалены IP адреса старше %d минут\n", retentionMinutes)
}

// ResetStats полностью очищает все статистики
func (la *LogAnalyzer) ResetStats() {
	la.Stats = make(map[string]*EmailIPStats)
	fmt.Println("🔄 Все счетчики IP адресов сброшены")
}
