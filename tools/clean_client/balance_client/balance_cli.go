package balance_client

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// BalanceCLI предоставляет интерфейс командной строки для управления балансом
type BalanceCLI struct {
	manager *BalanceManager
}

// NewBalanceCLI создает новый экземпляр BalanceCLI
func NewBalanceCLI(manager *BalanceManager) *BalanceCLI {
	return &BalanceCLI{manager: manager}
}

// Run запускает интерактивный интерфейс управления балансом
func (cli *BalanceCLI) Run() {
	for {
		fmt.Println("\n=== УПРАВЛЕНИЕ БАЛАНСОМ КЛИЕНТОВ ===")
		fmt.Println("1. Показать баланс пользователя")
		fmt.Println("2. Показать полную информацию о пользователе")
		fmt.Println("3. Пополнить баланс")
		fmt.Println("4. Списать с баланса")
		fmt.Println("5. Установить точный баланс")
		fmt.Println("6. Показать историю изменений баланса")
		fmt.Println("7. Найти пользователей по имени/username")
		fmt.Println("0. Назад в главное меню")

		choice := cli.readUserInput("Ваш выбор: ")

		switch choice {
		case "1":
			cli.showBalance()

		case "2":
			cli.showUserInfo()

		case "3":
			cli.addBalance()

		case "4":
			cli.subtractBalance()

		case "5":
			cli.setBalance()

		case "6":
			cli.showBalanceHistory()

		case "7":
			cli.searchUsers()

		case "0":
			return

		default:
			fmt.Println("❌ Неверный выбор. Попробуйте снова.")
		}
	}
}

// showBalance показывает текущий баланс пользователя
func (cli *BalanceCLI) showBalance() {
	telegramID, err := cli.getTelegramID()
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	balance, err := cli.manager.GetUserBalance(telegramID)
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	fmt.Printf("💰 Баланс пользователя %d: %.2f₽\n", telegramID, balance)
}

// showUserInfo показывает полную информацию о пользователе
func (cli *BalanceCLI) showUserInfo() {
	telegramID, err := cli.getTelegramID()
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	userInfo, err := cli.manager.GetUserInfo(telegramID)
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	fmt.Println("\n=== ИНФОРМАЦИЯ О ПОЛЬЗОВАТЕЛЕ ===")
	fmt.Printf("🆔 Telegram ID: %d\n", userInfo.TelegramID)
	fmt.Printf("👤 Username: @%s\n", userInfo.Username)
	fmt.Printf("📝 Имя: %s %s\n", userInfo.FirstName, userInfo.LastName)
	fmt.Printf("💰 Баланс: %.2f₽\n", userInfo.Balance)
	fmt.Printf("💳 Всего оплачено: %.2f₽\n", userInfo.TotalPaid)
	
	hasConfig := "❌"
	if userInfo.HasActiveConfig {
		hasConfig = "✅"
	}
	fmt.Printf("🔧 Активный конфиг: %s\n", hasConfig)
	
	if userInfo.ClientID != "" {
		fmt.Printf("🆔 Client ID: %s\n", userInfo.ClientID)
	}
	if userInfo.SubID != "" {
		fmt.Printf("📋 Sub ID: %s\n", userInfo.SubID)
	}
	if userInfo.Email != "" {
		fmt.Printf("📧 Email: %s\n", userInfo.Email)
	}
	if userInfo.ExpiryTime > 0 {
		expiryTime := time.UnixMilli(userInfo.ExpiryTime).Format("2006-01-02 15:04:05")
		fmt.Printf("⏰ Истекает: %s\n", expiryTime)
	}
	
	fmt.Printf("📅 Регистрация: %s\n", userInfo.CreatedAt.Format("2006-01-02 15:04:05"))
}

// addBalance пополняет баланс пользователя
func (cli *BalanceCLI) addBalance() {
	telegramID, err := cli.getTelegramID()
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	amount, err := cli.getAmount("Введите сумму для пополнения: ")
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	description := cli.readUserInput("Введите описание операции (необязательно): ")
	if description == "" {
		description = "Пополнение баланса через админ-панель"
	}

	// Показываем текущий баланс
	currentBalance, err := cli.manager.GetUserBalance(telegramID)
	if err != nil {
		fmt.Printf("❌ Ошибка получения текущего баланса: %v\n", err)
		return
	}

	fmt.Printf("💰 Текущий баланс: %.2f₽\n", currentBalance)
	fmt.Printf("➕ Добавляем: %.2f₽\n", amount)
	fmt.Printf("💰 Новый баланс будет: %.2f₽\n", currentBalance+amount)

	confirm := cli.readUserInput("Подтвердить операцию? (yes/no): ")
	if strings.ToLower(confirm) != "yes" {
		fmt.Println("❌ Операция отменена")
		return
	}

	err = cli.manager.AddBalance(telegramID, amount)
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	// Логируем операцию
	cli.manager.LogBalanceChange(telegramID, amount, "add", description)

	fmt.Printf("✅ Баланс успешно пополнен на %.2f₽\n", amount)
}

// subtractBalance списывает средства с баланса пользователя
func (cli *BalanceCLI) subtractBalance() {
	telegramID, err := cli.getTelegramID()
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	amount, err := cli.getAmount("Введите сумму для списания: ")
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	description := cli.readUserInput("Введите описание операции (необязательно): ")
	if description == "" {
		description = "Списание с баланса через админ-панель"
	}

	// Показываем текущий баланс
	currentBalance, err := cli.manager.GetUserBalance(telegramID)
	if err != nil {
		fmt.Printf("❌ Ошибка получения текущего баланса: %v\n", err)
		return
	}

	fmt.Printf("💰 Текущий баланс: %.2f₽\n", currentBalance)
	fmt.Printf("➖ Списываем: %.2f₽\n", amount)
	fmt.Printf("💰 Новый баланс будет: %.2f₽\n", currentBalance-amount)

	if currentBalance < amount {
		fmt.Printf("❌ Недостаточно средств! Текущий баланс: %.2f₽, требуется: %.2f₽\n", currentBalance, amount)
		return
	}

	confirm := cli.readUserInput("Подтвердить операцию? (yes/no): ")
	if strings.ToLower(confirm) != "yes" {
		fmt.Println("❌ Операция отменена")
		return
	}

	err = cli.manager.SubtractBalance(telegramID, amount)
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	// Логируем операцию
	cli.manager.LogBalanceChange(telegramID, -amount, "subtract", description)

	fmt.Printf("✅ С баланса успешно списано %.2f₽\n", amount)
}

// setBalance устанавливает точный баланс пользователя
func (cli *BalanceCLI) setBalance() {
	telegramID, err := cli.getTelegramID()
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	amount, err := cli.getAmount("Введите новый баланс: ")
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	description := cli.readUserInput("Введите описание операции (необязательно): ")
	if description == "" {
		description = "Установка баланса через админ-панель"
	}

	// Показываем текущий баланс
	currentBalance, err := cli.manager.GetUserBalance(telegramID)
	if err != nil {
		fmt.Printf("❌ Ошибка получения текущего баланса: %v\n", err)
		return
	}

	fmt.Printf("💰 Текущий баланс: %.2f₽\n", currentBalance)
	fmt.Printf("🔄 Новый баланс: %.2f₽\n", amount)
	
	diff := amount - currentBalance
	if diff > 0 {
		fmt.Printf("➕ Изменение: +%.2f₽\n", diff)
	} else if diff < 0 {
		fmt.Printf("➖ Изменение: %.2f₽\n", diff)
	} else {
		fmt.Println("🔄 Изменений нет")
	}

	confirm := cli.readUserInput("Подтвердить операцию? (yes/no): ")
	if strings.ToLower(confirm) != "yes" {
		fmt.Println("❌ Операция отменена")
		return
	}

	err = cli.manager.SetBalance(telegramID, amount)
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	// Логируем операцию
	cli.manager.LogBalanceChange(telegramID, diff, "set", description)

	fmt.Printf("✅ Баланс успешно установлен: %.2f₽\n", amount)
}

// showBalanceHistory показывает историю изменений баланса
func (cli *BalanceCLI) showBalanceHistory() {
	telegramID, err := cli.getTelegramID()
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	limitStr := cli.readUserInput("Введите количество записей для показа (по умолчанию 20): ")
	limit := 20
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			fmt.Println("❌ Неверное количество записей, используем значение по умолчанию: 20")
			limit = 20
		}
	}

	history, err := cli.manager.GetBalanceHistory(telegramID, limit)
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	if len(history) == 0 {
		fmt.Println("📝 История изменений баланса пуста")
		return
	}

	fmt.Printf("\n=== ИСТОРИЯ ИЗМЕНЕНИЙ БАЛАНСА (пользователь %d) ===\n", telegramID)
	fmt.Printf("%-20s %-10s %-15s %-30s %-20s\n", "Дата", "Сумма", "Операция", "Описание", "Тип")
	fmt.Println(strings.Repeat("-", 100))

	for _, h := range history {
		amountStr := fmt.Sprintf("%.2f₽", h.Amount)
		if h.Amount > 0 {
			amountStr = "+" + amountStr
		}
		
		dateStr := h.CreatedAt.Format("2006-01-02 15:04")
		description := h.Description
		if len(description) > 30 {
			description = description[:27] + "..."
		}

		fmt.Printf("%-20s %-10s %-15s %-30s %-20s\n", 
			dateStr, amountStr, h.OperationType, description, "balance_change")
	}
}

// searchUsers ищет пользователей по имени или username
func (cli *BalanceCLI) searchUsers() {
	searchTerm := cli.readUserInput("Введите имя, фамилию или username для поиска: ")
	if searchTerm == "" {
		fmt.Println("❌ Поисковый запрос не может быть пустым")
		return
	}

	// Здесь можно добавить функцию поиска пользователей
	// Пока что просто показываем сообщение
	fmt.Println("🔍 Функция поиска пользователей будет добавлена в следующей версии")
	fmt.Printf("Поисковый запрос: '%s'\n", searchTerm)
}

// getTelegramID получает Telegram ID от пользователя
func (cli *BalanceCLI) getTelegramID() (int64, error) {
	telegramIDStr := cli.readUserInput("Введите Telegram ID: ")
	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("неверный формат Telegram ID: %v", err)
	}
	return telegramID, nil
}

// getAmount получает сумму от пользователя
func (cli *BalanceCLI) getAmount(prompt string) (float64, error) {
	amountStr := cli.readUserInput(prompt)
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return 0, fmt.Errorf("неверный формат суммы: %v", err)
	}
	if amount < 0 {
		return 0, fmt.Errorf("сумма не может быть отрицательной")
	}
	return amount, nil
}

// readUserInput читает ввод пользователя
func (cli *BalanceCLI) readUserInput(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
