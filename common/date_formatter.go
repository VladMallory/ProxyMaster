package common

import (
	"fmt"
	"time"
)

// Русские названия месяцев в родительном падеже
var russianMonths = map[time.Month]string{
	time.January:   "января",
	time.February:  "февраля",
	time.March:     "марта",
	time.April:     "апреля",
	time.May:       "мая",
	time.June:      "июня",
	time.July:      "июля",
	time.August:    "августа",
	time.September: "сентября",
	time.October:   "октября",
	time.November:  "ноября",
	time.December:  "декабря",
}

// FormatRussianDate форматирует дату в русском формате с правильными окончаниями
// Пример: "2025 13 сентября" или "2025 21 января"
func FormatRussianDate(t time.Time) string {
	day := t.Day()
	month := russianMonths[t.Month()]
	year := t.Year()

	// Определяем правильное окончание для числа
	dayStr := getDayWithEnding(day)

	return fmt.Sprintf("%d %s %s", year, dayStr, month)
}

// FormatRussianDateFromUnix форматирует дату из Unix timestamp в миллисекундах
func FormatRussianDateFromUnix(unixMilli int64) string {
	t := time.UnixMilli(unixMilli)
	return FormatRussianDate(t)
}

// getDayWithEnding возвращает число дня с правильным окончанием
func getDayWithEnding(day int) string {
	// Особые случаи для 11, 12, 13 (всегда "е")
	if day >= 11 && day <= 13 {
		return fmt.Sprintf("%dе", day)
	}

	// Проверяем последнюю цифру
	lastDigit := day % 10

	switch lastDigit {
	case 1:
		return fmt.Sprintf("%dе", day) // 1е, 21е, 31е
	case 2:
		return fmt.Sprintf("%dе", day) // 2е, 22е
	case 3:
		return fmt.Sprintf("%dе", day) // 3е, 23е
	default:
		return fmt.Sprintf("%dе", day) // все остальные с "е"
	}
}

// FormatRussianDateTime форматирует дату и время в русском формате
// Пример: "2025 13 сентября в 15:30"
func FormatRussianDateTime(t time.Time) string {
	dateStr := FormatRussianDate(t)
	timeStr := t.Format("15:04")
	return fmt.Sprintf("%s в %s", dateStr, timeStr)
}

// FormatRussianDateTimeFromUnix форматирует дату и время из Unix timestamp
func FormatRussianDateTimeFromUnix(unixMilli int64) string {
	t := time.UnixMilli(unixMilli)
	return FormatRussianDateTime(t)
}
