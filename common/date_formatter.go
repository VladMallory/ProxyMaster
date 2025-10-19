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

// FormatRussianDate форматирует дату в русском формате
// Пример: "2025 13 сентября" или "2025 21 января"
func FormatRussianDate(t time.Time) string {
	day := t.Day()
	month := russianMonths[t.Month()]
	year := t.Year()

	return fmt.Sprintf("%d %d %s", year, day, month)
}

// FormatRussianDateFromUnix форматирует дату из Unix timestamp в миллисекундах
func FormatRussianDateFromUnix(unixMilli int64) string {
	t := time.UnixMilli(unixMilli)
	return FormatRussianDate(t)
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
