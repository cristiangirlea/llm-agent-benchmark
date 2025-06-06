package main

import (
	"benchmark/internal/agents"
	"benchmark/queue"
	"fmt"
	"os"
)

func main() {
	planner := &agents.PlannerAgent{}
	redisAddr := os.Getenv("REDIS_HOST")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	taskQueue := queue.NewRedisQueue(redisAddr, "agent_tasks")
	steps := planner.Run("Create a Chrome extension that summarizes YouTube videos")

	fmt.Println("📌 Planner generated steps:")
	for _, step := range steps {
		fmt.Println(" -", step)
		if err := taskQueue.Enqueue(step); err != nil {
			fmt.Println("❌ Failed to enqueue:", step, err)
		}
	}

	fmt.Println("✅ All tasks enqueued.")
}
