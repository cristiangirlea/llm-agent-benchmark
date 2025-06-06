package web

import (
	"benchmark/internal/config"
	core2 "benchmark/internal/core"
	"benchmark/internal/persistence"
	"benchmark/queue"
	"fmt"
	"net/http"
)

func RenderIndex(w http.ResponseWriter, r *http.Request) {
	models := config.LoadModelsFromFile("models.txt")
	prompts := config.LoadPrompts()
	renderForm(w, models, prompts, "", core2.Metrics{}, false, "")
}

func HandleGenerateResponse(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	model := r.FormValue("model")
	prompt := r.FormValue("prompt")

	models := config.LoadModelsFromFile("models.txt")
	if !config.ValidateModel(model, models) {
		renderForm(w, models, config.LoadPrompts(), "", core2.Metrics{}, false, fmt.Sprintf("Model '%s' is not available.", model))
		return
	}

	output, metrics, err := core2.Execute(model, prompt) // ✅ call correct function
	if err != nil {
		http.Error(w, fmt.Sprintf("Request error: %v", err), http.StatusInternalServerError)
		return
	}

	renderForm(w, models, config.LoadPrompts(), output, metrics, true, "")
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	if err := queue.Ping(); err != nil {
		http.Error(w, "Redis unavailable", http.StatusServiceUnavailable)
		return
	}

	if err := persistence.DB.Ping(); err != nil {
		http.Error(w, "Postgres unavailable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
