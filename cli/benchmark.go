package main

import (
	"benchmark/config"
	"benchmark/core"
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("❌ Usage: go run main.go <model> <prompt>")
		return
	}

	model := os.Args[1]
	prompt := os.Args[2]

	models := config.LoadModelsFromFile("models.txt")
	valid := false
	for _, m := range models {
		if m == model {
			valid = true
			break
		}
	}
	if !valid {
		fmt.Printf("❌ Model \"%s\" not found in models.txt\n", model)
		return
	}

	fmt.Printf("📦 Running benchmark for model: %s\n🧠 Prompt: %s\n", model, prompt)

	start := time.Now()
	output, metrics, err := core.Execute(model, prompt)
	if err != nil {
		fmt.Printf("❌ Error during execution: %v\n", err)
		return
	}

	fmt.Printf("✅ Output: %s\n", output)
	fmt.Printf("   📦 Tokens: %d\n", metrics.TotalTokens)
	fmt.Printf("   🚀 Tokens/sec: %.2f\n", metrics.TokensPerSec)
	fmt.Printf("   🕒 First token: %v\n", metrics.FirstTokenTime)
	fmt.Printf("   ⏱️ Duration: %v\n", metrics.TotalDuration)
	fmt.Printf("✅ Done in %v\n", time.Since(start))
}
