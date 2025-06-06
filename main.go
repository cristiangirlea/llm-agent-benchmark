package main

import (
	"benchmark/web"
	"log"
	"net/http"
)

func main() {
	web.RegisterRoutes()
	log.Println("🚀 Server running at http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
