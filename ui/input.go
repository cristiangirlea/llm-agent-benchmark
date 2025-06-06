package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// PromptModelSelection displays available models and lets user select one or more
func PromptModelSelection(models []string) []string {
	fmt.Println("📦 Available models:")
	for i, model := range models {
		fmt.Printf("[%d] %s\n", i, model)
	}
	fmt.Print("\n👉 Enter model numbers to benchmark (e.g., 0,2,3 or 'all'): ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())

	if input == "all" {
		return models
	}

	indices := strings.Split(input, ",")
	var selected []string
	for _, idxStr := range indices {
		idxStr = strings.TrimSpace(idxStr)
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 0 || idx >= len(models) {
			fmt.Printf("⚠️  Invalid index: %s (skipped)\n", idxStr)
			continue
		}
		selected = append(selected, models[idx])
	}

	return selected
}
