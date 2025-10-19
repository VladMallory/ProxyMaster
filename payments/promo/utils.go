package promo

import "bot/common"

// IsAdmin проверяет, является ли пользователь администратором
func IsAdmin(userID int64) bool {
	return userID == common.ADMIN_ID
}
