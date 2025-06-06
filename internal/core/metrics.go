// File: core/metrics.go
package core

import (
	"runtime"
	"time"
)

// SystemMetrics captures memory and GC info from Go runtime
type SystemMetrics struct {
	Alloc      uint64
	TotalAlloc uint64
	Sys        uint64
	NumGC      uint32
}

func CaptureSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemMetrics{
		Alloc:      m.Alloc,
		TotalAlloc: m.TotalAlloc,
		Sys:        m.Sys,
		NumGC:      m.NumGC,
	}
}

// LiveMetrics tracks ongoing execution data to be finalized later
type LiveMetrics struct {
	Model          string
	Prompt         string
	FirstTokenTime time.Duration
	TotalTokens    int
	StartTime      time.Time
	SystemBefore   SystemMetrics
	SystemAfter    SystemMetrics
}

func (m *LiveMetrics) RecordFirstToken() {
	m.FirstTokenTime = time.Since(m.StartTime)
}

func (m *LiveMetrics) IncrementToken() {
	m.TotalTokens++
}

func (m *LiveMetrics) Finalize() Metrics {
	totalDuration := time.Since(m.StartTime)
	tokensPerSec := float64(m.TotalTokens) / totalDuration.Seconds()

	return Metrics{
		Model:          m.Model,
		Prompt:         m.Prompt,
		FirstTokenTime: m.FirstTokenTime,
		TotalTokens:    m.TotalTokens,
		TokensPerSec:   tokensPerSec,
		TotalDuration:  totalDuration,
		SystemBefore:   m.SystemBefore,
		SystemAfter:    m.SystemAfter,
	}
}

func RunBenchmarkWithMetrics(model string, prompt string, metrics *LiveMetrics) {
	// Simulate token generation or processing
	for i := 0; i < 100; i++ {
		if i == 0 {
			metrics.RecordFirstToken()
		}
		time.Sleep(5 * time.Millisecond) // fake token processing delay
		metrics.IncrementToken()
	}
}

// Finalized metrics result
type Metrics struct {
	Model          string
	Prompt         string
	FirstTokenTime time.Duration
	TotalTokens    int
	TokensPerSec   float64
	TotalDuration  time.Duration
	SystemBefore   SystemMetrics
	SystemAfter    SystemMetrics
}
