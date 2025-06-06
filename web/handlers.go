package web

import (
	"benchmark/config"
	"benchmark/core"
	"fmt"
	"net/http"
)

func RenderIndex(w http.ResponseWriter, r *http.Request) {
	models := config.LoadModelsFromFile("models.txt")
	prompts := config.LoadPrompts()
	renderForm(w, models, prompts, "", core.Metrics{}, false, "")
}

func HandleGenerateResponse(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	model := r.FormValue("model")
	prompt := r.FormValue("prompt")

	models := config.LoadModelsFromFile("models.txt")
	if !config.ValidateModel(model, models) {
		renderForm(w, models, config.LoadPrompts(), "", core.Metrics{}, false, fmt.Sprintf("Model '%s' is not available.", model))
		return
	}

	output, metrics, err := core.Execute(model, prompt) // ✅ call correct function
	if err != nil {
		http.Error(w, fmt.Sprintf("Request error: %v", err), http.StatusInternalServerError)
		return
	}

	renderForm(w, models, config.LoadPrompts(), output, metrics, true, "")
}
