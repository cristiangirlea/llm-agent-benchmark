package web

import (
	"benchmark/internal/persistence"
	"benchmark/queue"
	"benchmark/web"
	"log"
	"net/http"
	"os"
)

func main() {
	// Initialize Postgres
	persistence.InitDB()

	// Initialize Redis queue
	redisAddr := os.Getenv("REDIS_HOST")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	queue.InitDefaultQueue(redisAddr, "agent_tasks")

	web.RegisterRoutes()
	log.Println("🚀 Server running at http://localhost:8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
