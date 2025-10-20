package main

import (
	"fmt"
	"ipBanSystem/common"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	serviceName        = "ipBanService"
	serviceDescription = "IP Ban System Service"
	serviceFilePath    = "/etc/systemd/system/ipBanService.service"
	binaryPath         = "/usr/local/bin/ipBanService"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--install":
			installService()
			return
		case "--uninstall":
			uninstallService()
			return
		}
	}

	// Инициализируем логгер
	if err := common.InitIPBanLogger(); err != nil {
		log.Fatalf("Ошибка инициализации логгера: %v", err)
	}

	common.LogIPBanInfo("Запуск IP Ban сервиса...")

	// Создаем компоненты
	accumulator := common.NewLogAccumulator(common.ACCESS_LOG_PATH, common.IP_ACCUMULATED_PATH)
	if err := accumulator.Start(); err != nil {
		common.LogIPBanError("Ошибка запуска накопителя логов: %v", err)
		return
	}
	accumulator.StartCleanupService()
	common.LogIPBanInfo("Накопитель логов запущен")

	analyzer := common.NewLogAnalyzer(common.IP_ACCUMULATED_PATH)

	configManager := common.NewConfigManager(
		common.PANEL_URL,
		common.PANEL_USER,
		common.PANEL_PASS,
		common.INBOUND_ID,
	)

	banManager := common.NewBanManager("/var/log/ip_bans.json")
	iptablesManager := common.NewIPTablesManager()

	// Создаем и запускаем сервис
	service := common.NewIPBanService(
		analyzer,
		configManager,
		banManager,
		iptablesManager,
		common.MAX_IPS_PER_CONFIG,
		time.Duration(common.IP_CHECK_INTERVAL)*time.Minute,
		time.Duration(common.IP_BAN_GRACE_PERIOD)*time.Minute,
	)

	if err := service.Start(); err != nil {
		common.LogIPBanError("Ошибка запуска IP Ban сервиса: %v", err)
		return
	}

	common.LogIPBanInfo("IP Ban сервис успешно запущен. Нажмите CTRL+C для выхода.")

	// Ожидаем сигнала для завершения работы
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Останавливаем сервис
	service.Stop()
	common.LogIPBanInfo("IP Ban сервис остановлен.")
}

func installService() {
	fmt.Println("Установка сервиса ipBanService...")

	// 1. Сборка бинарного файла
	fmt.Println("Шаг 1: Сборка бинарного файла...")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "/root/bot/tools/ipBanSystem"
	if err := runCommand(cmd); err != nil {
		log.Fatalf("Ошибка сборки бинарного файла: %v", err)
	}
	fmt.Println("Бинарный файл успешно собран в", binaryPath)

	// 2. Создание файла сервиса systemd
	fmt.Println("Шаг 2: Создание файла сервиса systemd...")
	workingDir, _ := os.Getwd()
	serviceFileContent := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
Type=simple
User=root
ExecStart=%s
WorkingDirectory=%s
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
`, serviceDescription, binaryPath, workingDir)

	if err := os.WriteFile(serviceFilePath, []byte(serviceFileContent), 0644); err != nil {
		log.Fatalf("Ошибка создания файла сервиса: %v", err)
	}
	fmt.Println("Файл сервиса успешно создан в", serviceFilePath)

	// 3. Перезагрузка, включение и запуск сервиса
	fmt.Println("Шаг 3: Перезагрузка, включение и запуск сервиса...")
	if err := runCommand(exec.Command("systemctl", "daemon-reload")); err != nil {
		log.Fatalf("Ошибка перезагрузки демона systemd: %v", err)
	}
	if err := runCommand(exec.Command("systemctl", "enable", serviceName)); err != nil {
		log.Fatalf("Ошибка включения сервиса: %v", err)
	}
	if err := runCommand(exec.Command("systemctl", "start", serviceName)); err != nil {
		log.Fatalf("Ошибка запуска сервиса: %v", err)
	}

	fmt.Println("\n✅ Сервис ipBanService успешно установлен и запущен!")
	fmt.Println("Для проверки статуса используйте: systemctl status", serviceName)
}

func uninstallService() {
	fmt.Println("Удаление сервиса ipBanService...")

	// 1. Остановка и отключение сервиса
	fmt.Println("Шаг 1: Остановка и отключение сервиса...")
	runCommand(exec.Command("systemctl", "stop", serviceName)) // Игнорируем ошибку, если сервис не запущен
	runCommand(exec.Command("systemctl", "disable", serviceName)) // Игнорируем ошибку, если сервис не включен

	// 2. Удаление файла сервиса
	fmt.Println("Шаг 2: Удаление файла сервиса...")
	if err := os.Remove(serviceFilePath); err != nil {
		log.Printf("Ошибка удаления файла сервиса (возможно, он уже удален): %v", err)
	} else {
		fmt.Println("Файл сервиса удален.")
	}

	// 3. Перезагрузка демона systemd
	fmt.Println("Шаг 3: Перезагрузка демона systemd...")
	if err := runCommand(exec.Command("systemctl", "daemon-reload")); err != nil {
		log.Printf("Ошибка перезагрузки демона systemd: %v", err)
	}

	// 4. Удаление бинарного файла
	fmt.Println("Шаг 4: Удаление бинарного файла...")
	if err := os.Remove(binaryPath); err != nil {
		log.Printf("Ошибка удаления бинарного файла (возможно, он уже удален): %v", err)
	} else {
		fmt.Println("Бинарный файл удален.")
	}

	fmt.Println("\n✅ Сервис ipBanService успешно удален.")
}

func runCommand(cmd *exec.Cmd) error {
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("команда '%s' завершилась с ошибкой: %v\nВывод: %s", cmd.String(), err, out.String())
	}
	return nil
}

