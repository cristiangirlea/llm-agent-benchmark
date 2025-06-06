// File: web/api.go
package web

import (
	core2 "benchmark/internal/core"
	"encoding/json"
	"net/http"
)

// APIRequest defines the expected input JSON structure.
type APIRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// APIResponse defines the output JSON structure including metrics and optional error.
type APIResponse struct {
	Output  string        `json:"output"`
	Metrics core2.Metrics `json:"metrics"`
	Error   string        `json:"error,omitempty"`
}

// Handler processes POST /api requests with a model and prompt.
func Handler(w http.ResponseWriter, r *http.Request) {
	var req APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	output, metrics, err := core2.Execute(req.Model, req.Prompt)

	resp := APIResponse{
		Output:  output,
		Metrics: metrics,
	}
	if err != nil {
		resp.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
