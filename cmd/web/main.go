package main

import (
	"benchmark/internal/persistence"
	"benchmark/queue"
	"benchmark/web"
	"log"
	"net/http"
	"os"
)

func main() {
	log.Println("🛠 Initializing PostgreSQL...")

	// Initialize Postgres
	persistence.InitDB()
	log.Println("✅ PostgreSQL connected.")

	log.Println("🛠 Initializing Redis...")

	// Initialize Redis queue
	redisAddr := os.Getenv("REDIS_HOST")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	queue.InitDefaultQueue(redisAddr, "agent_tasks")
	if err := queue.Ping(); err != nil {
		log.Fatalf("❌ Redis unreachable: %v", err)
	}
	log.Println("✅ Redis connected.")

	log.Println("🔧 Registering HTTP routes...")
	web.RegisterRoutes()

	log.Println("🚀 Server running at http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
