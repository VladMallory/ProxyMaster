module reminder_manager

go 1.25.1

require (
	bot v0.0.0-00010101000000-000000000000
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
)

replace bot => ../..
