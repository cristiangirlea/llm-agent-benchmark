package ui

import (
	"benchmark/internal/core"
	"encoding/json"
	"fmt"
	"os"
)

var OutputMode = "cli" // or "web"

func PrintMetrics(m core.Metrics) {
	switch OutputMode {
	case "cli":
		printMetricsCLI(m)
	case "web":
		printMetricsJSON(m)
	default:
		printMetricsCLI(m)
	}
}

func printMetricsCLI(m core.Metrics) {
	fmt.Printf("\n📊 Metrics for model: %s\n", m.Model)
	fmt.Printf("Prompt: %s\n", m.Prompt)
	fmt.Printf("🕐 First Token Time: %v\n", m.FirstTokenTime)
	fmt.Printf("🔢 Total Tokens: %d\n", m.TotalTokens)
	fmt.Printf("⚡ Tokens/sec: %.2f\n", m.TokensPerSec)
	fmt.Printf("⏱️ Total Duration: %v\n", m.TotalDuration)
	fmt.Printf("💾 Memory Before: %d KB\n", m.SystemBefore.Alloc/1024)
	fmt.Printf("💾 Memory After: %d KB\n", m.SystemAfter.Alloc/1024)
	fmt.Printf("♻️ GC Cycles: %d → %d\n", m.SystemBefore.NumGC, m.SystemAfter.NumGC)
}

func printMetricsJSON(m core.Metrics) {
	jsonData, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal metrics: %v\n", err)
		return
	}
	fmt.Println(string(jsonData)) // You can later replace this with `return string(jsonData)` if needed for your frontend.
}
