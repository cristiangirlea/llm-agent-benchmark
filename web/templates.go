package web

import (
	"benchmark/internal/core"
	"fmt"
	"html/template"
	"net/http"
)

// PageData is the struct passed to the HTML template

func renderForm(
	w http.ResponseWriter,
	models []string,
	prompts []string,
	output string,
	metrics core.Metrics,
	showMetrics bool,
	errMsg string,
) {
	funcMap := template.FuncMap{
		"toKB": func(val uint64) string {
			return fmt.Sprintf("%d KB", val/1024)
		},
		"toMB": func(val uint64) string {
			return fmt.Sprintf("%.2f MB", float64(val)/(1024*1024))
		},
		"formatFloat": func(f float64) string {
			return fmt.Sprintf("%.2f", f)
		},
	}

	tmpl := template.Must(template.New("index.html").Funcs(funcMap).ParseFiles("web/templates/index.html"))
	err := tmpl.Execute(w, PageData{
		Models:      models,
		Prompts:     prompts,
		Output:      output,
		Metrics:     metrics,
		SystemAfter: metrics.SystemAfter, // So templates don’t need to do .Metrics.SystemAfter.*
		ShowMetrics: showMetrics,
		Error:       errMsg,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
