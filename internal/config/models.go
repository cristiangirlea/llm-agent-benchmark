package config

import (
	"fmt"
	"os"
	"strings"
)

func LoadModelsFromFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("❌ Failed to read models file: %v\n", err)
		return []string{}
	}
	lines := strings.Split(string(data), "\n")
	var models []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			models = append(models, trimmed)
		}
	}
	return models
}

func LoadPrompts() []string {
	return []string{
		"Write a function in Go that calculates the Fibonacci sequence using memoization.",
		"Explain the difference between goroutines and threads.",
		"Generate a REST API endpoint in Go using net/http.",
		"Write a unit test for a function that adds two numbers in Go.",
	}
}

func ValidateModel(model string, available []string) bool {
	for _, m := range available {
		if m == model {
			return true
		}
	}
	return false
}
