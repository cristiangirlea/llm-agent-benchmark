package web

import (
	"benchmark/internal/agents"
	"encoding/json"
	"log"
	"net/http"
)

type PlanRequest struct {
	Objective string `json:"objective"`
}

type PlanResponse struct {
	Steps []string `json:"steps"`
	Error string   `json:"error,omitempty"`
}

func PlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("🔍 PlanHandler triggered")

	var req struct {
		Objective string `json:"objective"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	agent := agents.PlannerAgent{}
	steps := agent.Run(req.Objective)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"steps": steps,
	})
}
