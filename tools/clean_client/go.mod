module clean_client

go 1.25.1

require (
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
)

require balance_client v0.0.0-00010101000000-000000000000 // indirect

replace balance_client => ./balance_client
