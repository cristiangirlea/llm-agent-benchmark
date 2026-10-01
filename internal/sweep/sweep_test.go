package sweep

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadResolvesPathsAndDefaults(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "sweep.json", `{
	  "llama_server": "bin/llama-server.exe", "models_dir": "models",
	  "endpoints": [{"name": "a", "model": "a.gguf"}, {"name": "o", "base_url": "http://127.0.0.1:11434", "model_id": "qwen"}],
	  "agents": [{"name": "claude", "backend": "claude", "model": "sonnet"}],
	  "evals": {"dir": "evals", "suites": ["suites/golden/x.json"]}
	}`)
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.LlamaServer != filepath.Join(dir, "bin", "llama-server.exe") || c.ModelsDir != filepath.Join(dir, "models") {
		t.Errorf("paths not resolved from the file's directory: %q %q", c.LlamaServer, c.ModelsDir)
	}
	if c.Evals.Suites[0] != filepath.Join(dir, "evals", "suites", "golden", "x.json") {
		t.Errorf("suite = %q, want it under evals.dir", c.Evals.Suites[0])
	}
	if c.Port != 8081 || c.Speed.MaxTokens != 256 || !slices.Equal(c.Speed.Concurrency, []int{1, 4}) || c.Evals.Hrn != "hrn" {
		t.Errorf("defaults not filled: %+v", c)
	}
	if !c.Endpoints[0].ServesAnthropic() || c.Endpoints[1].ServesAnthropic() {
		t.Error("a launched llama-server serves /v1/messages by default, an external endpoint does not")
	}
	if c.Endpoints[1].ID() != "qwen" || c.Endpoints[0].ID() != "a" {
		t.Error("ID is model_id when set, else the name")
	}
}

func TestLoadRejectsBadFiles(t *testing.T) {
	for name, body := range map[string]string{
		"no targets":       `{"evals": {"suites": ["x.json"]}}`,
		"both model+url":   `{"llama_server": "l", "models_dir": "m", "endpoints": [{"name": "a", "model": "a", "base_url": "http://x"}], "evals": {"skip": true}}`,
		"neither":          `{"endpoints": [{"name": "a"}], "evals": {"skip": true}}`,
		"model, no server": `{"endpoints": [{"name": "a", "model": "a.gguf"}], "evals": {"skip": true}}`,
		"duplicate name":   `{"agents": [{"name": "a", "backend": "claude"}, {"name": "a", "backend": "codex"}], "evals": {"skip": true}}`,
		"no suites":        `{"agents": [{"name": "a", "backend": "claude"}]}`,
		"dot in name":      `{"agents": [{"name": "qwen3.5", "backend": "api"}], "evals": {"skip": true}}`,
		"zero concurrency": `{"agents": [{"name": "a", "backend": "claude"}], "speed": {"concurrency": [0]}, "evals": {"skip": true}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(write(t, t.TempDir(), "s.json", body)); err == nil {
				t.Error("Load accepted an invalid sweep file")
			}
		})
	}
}

func TestEvalEnvIsolatesEndpointsFromAgents(t *testing.T) {
	base := []string{"PATH=/bin", "ANTHROPIC_BASE_URL=http://stale", "ANTHROPIC_API_KEY=sk-real", "anthropic_auth_token=tok"}

	agent := evalEnv(base, "")
	if slices.ContainsFunc(agent, func(kv string) bool { return strings.HasPrefix(strings.ToUpper(kv), "ANTHROPIC_BASE_URL=") }) {
		t.Errorf("an agent must not inherit ANTHROPIC_BASE_URL (Claude Code reads it): %v", agent)
	}
	if !slices.Contains(agent, "ANTHROPIC_API_KEY=sk-real") {
		t.Error("an agent keeps the user's real credentials")
	}

	ep := evalEnv(base, "http://127.0.0.1:8081")
	want := []string{"PATH=/bin", "ANTHROPIC_BASE_URL=http://127.0.0.1:8081", "ANTHROPIC_API_KEY=local"}
	if !slices.Equal(ep, want) {
		t.Errorf("endpoint env = %v, want %v (no real key may reach a local server)", ep, want)
	}
}

func TestRewriteSuiteReplacesMatrixAndPinsDir(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "suites", "golden"), 0o755)
	p := write(t, filepath.Join(dir, "suites", "golden"), "s.json",
		`{"dir": "../..", "mode": "yolo", "matrix": [{"backend": "codex"}], "tasks": [{"id": "t", "prompt": "p", "gate": ["true"]}]}`)
	data, err := rewriteSuite(p, variant{Name: "local", Backend: "api", Model: "m"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	json.Unmarshal(data, &got)
	if got["dir"] != dir {
		t.Errorf("dir = %v, want %v", got["dir"], dir)
	}
	m := got["matrix"].([]any)
	if len(m) != 1 || m[0].(map[string]any)["name"] != "local" || m[0].(map[string]any)["backend"] != "api" {
		t.Errorf("matrix = %v", m)
	}
	if got["repeats"] != float64(3) {
		t.Errorf("repeats = %v, want the sweep's 3", got["repeats"])
	}
	if got["mode"] != "yolo" || len(got["tasks"].([]any)) != 1 {
		t.Error("fields other than dir and matrix must be kept")
	}
}

func TestFillFromRecord(t *testing.T) {
	rec := `{"table": {"variants": [{"name": "x", "attempts": 4, "passed": 3, "total_cost_usd": 0.5,
	  "median_duration_ns": 18000000000, "median_turns": 5, "median_tool_calls": 4}],
	  "tasks": [{"task": "a", "variant": "x", "passed": 1, "total": 1}, {"task": "b", "variant": "x", "passed": 0, "total": 1}]}}`
	var r SuiteResult
	if err := fillFromRecord([]byte(rec), &r); err != nil {
		t.Fatal(err)
	}
	if r.Attempts != 4 || r.Passed != 3 || r.MedianTime != 18*time.Second || r.MedianTurn != 5 || len(r.Tasks) != 2 {
		t.Errorf("got %+v", r)
	}
	if m := evalIDLine.FindStringSubmatch("x\n── eval a22ac · batch 2026 · 1 cells · 18s ──\n"); m == nil || m[1] != "a22ac" {
		t.Errorf("eval id not found: %v", m)
	}
}

// sse serves a fake streaming chat completion: three content chunks after a delay, then usage.
func sse(delay time.Duration, usage bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["stream"] != true {
			http.Error(w, "want stream", 400)
			return
		}
		f := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		time.Sleep(delay)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\n") // no text: not the first token
		f.Flush()
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"hm\"}}]}\n\n")
		f.Flush()
		for range 2 {
			time.Sleep(20 * time.Millisecond)
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"tok tok\"}}]}\n\n")
			f.Flush()
		}
		if usage {
			fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":11}}\n\n")
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}
}

func TestStreamTimesFirstTokenAndUsesUsage(t *testing.T) {
	srv := httptest.NewServer(sse(50*time.Millisecond, true))
	defer srv.Close()
	s, err := stream(context.Background(), srv.URL, Endpoint{Name: "x"}, Speed{Prompt: "p", MaxTokens: 8}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if s.CompletionTokens != 11 || s.PromptTokens != 12 {
		t.Errorf("tokens = %d/%d, want the usage report (11/12), not a chunk count", s.CompletionTokens, s.PromptTokens)
	}
	if s.FirstToken < 50*time.Millisecond || s.FirstToken >= s.Total {
		t.Errorf("first token %s, total %s: first token must follow the delay and precede the end", s.FirstToken, s.Total)
	}
	if s.DecodeRate() <= 0 {
		t.Error("decode rate must be positive")
	}
}

func TestStreamWithoutUsageIsAnError(t *testing.T) {
	srv := httptest.NewServer(sse(0, false))
	defer srv.Close()
	if _, err := stream(context.Background(), srv.URL, Endpoint{Name: "x"}, Speed{Prompt: "p"}, 0); err == nil {
		t.Error("a stream with no usage report must fail rather than report 0 tokens")
	}
}

func TestMeasureSpeedLevels(t *testing.T) {
	srv := httptest.NewServer(sse(10*time.Millisecond, true))
	defer srv.Close()
	levels := measureSpeed(context.Background(), Speed{Concurrency: []int{1, 3}, Prompt: "p", MaxTokens: 8}, Endpoint{Name: "x"}, srv.URL)
	if len(levels) != 2 || levels[1].Concurrency != 3 {
		t.Fatalf("levels = %+v", levels)
	}
	for _, l := range levels {
		if l.Throughput <= 0 || l.FirstToken <= 0 || len(l.Errors) > 0 {
			t.Errorf("level %+v", l)
		}
	}
}

func TestReport(t *testing.T) {
	c := Config{Speed: Speed{Concurrency: []int{1, 4}}}
	results := []Result{
		{Name: "local", Kind: "endpoint", Speed: []Level{
			{Concurrency: 1, Throughput: 40, FirstToken: 250 * time.Millisecond, Decode: 42.5},
			{Concurrency: 4, Throughput: 101.4},
		}, Suites: []SuiteResult{{Suite: "golden/gobench.json", Attempts: 2, Passed: 1, MedianTurn: 5, MedianTime: 18 * time.Second,
			Tasks: []TaskResult{{Task: "fib", Passed: 1, Total: 1}, {Task: "rest", Passed: 0, Total: 1}}}}},
		{Name: "claude", Kind: "agent", Suites: []SuiteResult{{Suite: "golden/gobench.json", Attempts: 2, Passed: 2, TotalCost: 0.31,
			Tasks: []TaskResult{{Task: "fib", Passed: 1, Total: 1}, {Task: "rest", Passed: 1, Total: 1}}}}},
		{Name: "broken", Kind: "endpoint", Error: "llama-server exited"},
	}
	got := Report(results, c)
	for _, want := range []string{
		"| target | kind | first token | decode t/s | total t/s ×4 | passed | median turns | median time | cost |",
		"| local | endpoint | 250ms | 42.5 | 101.4 | 1/2 | 5 | 18s | – |",
		"| claude | agent | – | – | – | 2/2 |",
		"$0.31",
		"| golden/gobench / rest | 0/1 | 1/1 |",
		"- **broken**: llama-server exited",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("report lacks %q:\n%s", want, got)
		}
	}
}
