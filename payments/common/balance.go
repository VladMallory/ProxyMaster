package common

import (
	"bot/common"
	"log"
)

// AddBalance добавляет средства к балансу пользователя
func AddBalance(userID int64, amount float64) error {
	// Получаем пользователя из базы данных
	user, err := common.GetUserByTelegramID(userID)
	if err != nil {
		return err
	}

	// Обновляем баланс
	user.Balance += amount

	// Сохраняем изменения
	err = common.UpdateUser(user)
	if err != nil {
		return err
	}

	log.Printf("BALANCE: Пользователь %d пополнил баланс на %.2f₽, новый баланс: %.2f₽", userID, amount, user.Balance)
	return nil
}

// GetUserByTelegramID получает пользователя по Telegram ID
func GetUserByTelegramID(userID int64) (*common.User, error) {
	return common.GetUserByTelegramID(userID)
}
