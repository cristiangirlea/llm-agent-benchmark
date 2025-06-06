package core

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func LogResultCSV(model, prompt string, firstTokenTime time.Duration, totalTokens int, tokensPerSec float64, elapsed time.Duration) {
	_ = os.MkdirAll("results", 0755)
	file, err := os.OpenFile("results/benchmark_results.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("❌ Failed to write CSV:", err)
		return
	}
	defer file.Close()

	line := fmt.Sprintf("%s,%q,%v,%d,%.2f,%v\n", model, prompt, firstTokenTime, totalTokens, tokensPerSec, elapsed)
	if _, err := file.WriteString(line); err != nil {
		fmt.Println("❌ Failed to write line:", err)
	}
}

func LogOutputToFile(model, prompt, content string) {
	dir := filepath.Join("results", "logs")
	_ = os.MkdirAll(dir, 0755)

	safeName := sanitizeFileName(model + "_" + time.Now().Format("20060102_150405") + ".txt")
	path := filepath.Join(dir, safeName)

	log := fmt.Sprintf("🔍 Model: %s\n🧠 Prompt: %s\n\n📤 Output:\n%s\n", model, prompt, content)
	if err := os.WriteFile(path, []byte(log), 0644); err != nil {
		fmt.Println("❌ Failed to write log file:", err)
	}
}

func sanitizeFileName(name string) string {
	// Remove or replace characters that could be unsafe for filenames
	return filepath.Clean(name)
}
