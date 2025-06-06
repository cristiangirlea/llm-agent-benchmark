package web

import (
	"benchmark/internal/core"
)

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

type PageData struct {
	Models      []string
	Prompts     []string
	Error       string
	Output      string
	Metrics     core.Metrics
	SystemAfter core.SystemMetrics // Add this for convenience
	ShowMetrics bool
}
