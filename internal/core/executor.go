// File: core/executor.go
package core

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
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

// Execute performs a request to the model server and returns raw output and finalized metrics.
func Execute(model, prompt string) (string, Metrics, error) {
	live := &LiveMetrics{
		Model:        model,
		Prompt:       prompt,
		StartTime:    time.Now(),
		SystemBefore: CaptureSystemMetrics(),
	}

	req := GenerateRequest{Model: model, Prompt: prompt, Stream: true}
	jsonData, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", "http://localhost:11434/api/generate", bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", Metrics{}, err
	}
	defer resp.Body.Close()

	var output strings.Builder
	decoder := json.NewDecoder(resp.Body)

	for {
		var chunk GenerateResponse
		err := decoder.Decode(&chunk)
		if err == io.EOF || chunk.Done {
			break
		}
		if live.TotalTokens == 0 {
			live.RecordFirstToken()
		}
		live.IncrementToken()
		output.WriteString(chunk.Response)
	}

	live.SystemAfter = CaptureSystemMetrics()
	return output.String(), live.Finalize(), nil
}
