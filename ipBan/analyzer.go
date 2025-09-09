package main

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
	LogPath string
	Stats   map[string]*EmailIPStats
}

// NewLogAnalyzer создает новый анализатор логов
func NewLogAnalyzer(logPath string) *LogAnalyzer {
	return &LogAnalyzer{
		LogPath: logPath,
		Stats:   make(map[string]*EmailIPStats),
	}
}

// AnalyzeLog анализирует access.log и возвращает статистику по email и IP
func (la *LogAnalyzer) AnalyzeLog() (map[string]*EmailIPStats, error) {
	file, err := os.Open(la.LogPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла %s: %v", la.LogPath, err)
	}
	defer file.Close()

	// Регулярное выражение для парсинга строк лога
	// Формат: 2025/09/04 10:17:03.008517 from 123.123.123.123:52624 accepted tcp:courier.push.apple.com:443 [inbound-443 >> direct] email: 123456789
	logRegex := regexp.MustCompile(`(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}\.\d+) from (\d+\.\d+\.\d+\.\d+):\d+ accepted.*email: (\d+)`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

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
