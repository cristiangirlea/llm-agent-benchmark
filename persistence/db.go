package persistence

import (
	"database/sql"
	_ "github.com/lib/pq"
	"log"
	"os"
)

var DB *sql.DB

func InitDB() {
	dsn := os.Getenv("POSTGRES_DSN")
	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatalf("DB unreachable: %v", err)
	}

	log.Println("✅ Connected to Postgres")
}
