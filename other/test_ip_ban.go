package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"bot/common"
)

func main() {
	fmt.Println("🧪 Тестирование новой системы IP банов")
	fmt.Println("=====================================")

	// Проверяем конфигурацию
	fmt.Printf("📋 Конфигурация:\n")
	fmt.Printf("  IP_BAN_ENABLED: %v\n", common.IP_BAN_ENABLED)
	fmt.Printf("  MAX_IPS_PER_CONFIG: %d\n", common.MAX_IPS_PER_CONFIG)
	fmt.Printf("  ACCESS_LOG_PATH: %s\n", common.ACCESS_LOG_PATH)
	fmt.Printf("  IP_ACCUMULATED_PATH: %s\n", common.IP_ACCUMULATED_PATH)
	fmt.Printf("  IP_SAVE_INTERVAL: %d минут\n", common.IP_SAVE_INTERVAL)
	fmt.Printf("  IP_CHECK_INTERVAL: %d минут\n", common.IP_CHECK_INTERVAL)
	fmt.Printf("  IP_COUNTER_RETENTION: %d минут\n", common.IP_COUNTER_RETENTION)
	fmt.Println()

	// Проверяем существование исходного файла
	if _, err := os.Stat(common.ACCESS_LOG_PATH); os.IsNotExist(err) {
		fmt.Printf("❌ Исходный файл %s не найден\n", common.ACCESS_LOG_PATH)
		return
	}
	fmt.Printf("✅ Исходный файл %s найден\n", common.ACCESS_LOG_PATH)

	// Создаем накопитель логов
	accumulator := common.NewLogAccumulator(common.ACCESS_LOG_PATH, common.IP_ACCUMULATED_PATH)

	// Запускаем накопитель логов
	fmt.Println("🚀 Запуск накопителя логов...")
	if err := accumulator.Start(); err != nil {
		log.Printf("❌ Ошибка запуска накопителя логов: %v", err)
		return
	}

	// Запускаем сервис очистки
	accumulator.StartCleanupService()
	fmt.Println("✅ Накопитель логов запущен")

	// Принудительно накапливаем данные для теста
	fmt.Printf("⏳ Принудительное накопление данных для теста...\n")
	accumulator.AccumulateNewLines()

	// Ждем немного для завершения операций
	time.Sleep(2 * time.Second)

	// Создаем анализатор
	analyzer := common.NewLogAnalyzer(common.IP_ACCUMULATED_PATH)

	// Анализируем накопленные данные
	fmt.Println("📊 Анализ накопленных данных...")
	stats, err := analyzer.AnalyzeLog()
	if err != nil {
		log.Printf("❌ Ошибка анализа: %v", err)
		return
	}

	// Выводим результаты
	fmt.Printf("📈 Результаты анализа:\n")
	fmt.Printf("  Всего email: %d\n", len(stats))

	suspiciousCount := 0
	normalCount := 0

	for email, emailStats := range stats {
		if emailStats.TotalIPs > common.MAX_IPS_PER_CONFIG {
			suspiciousCount++
			fmt.Printf("  🚨 %s: %d IP (ПОДОЗРИТЕЛЬНЫЙ)\n", email, emailStats.TotalIPs)
		} else {
			normalCount++
			fmt.Printf("  ✅ %s: %d IP (нормальный)\n", email, emailStats.TotalIPs)
		}
	}

	fmt.Printf("\n📊 Итоговая статистика:\n")
	fmt.Printf("  Подозрительных: %d\n", suspiciousCount)
	fmt.Printf("  Нормальных: %d\n", normalCount)

	// Останавливаем накопитель
	accumulator.Stop()
	fmt.Println("🛑 Накопитель логов остановлен")
}
