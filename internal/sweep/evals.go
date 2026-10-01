package sweep

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SuiteResult is one hrn eval suite run for one target, read back from hrn's saved record.
type SuiteResult struct {
	Suite      string        `json:"suite"`
	EvalID     string        `json:"eval_id,omitempty"`
	Attempts   int           `json:"attempts"`
	Passed     int           `json:"passed"`
	MedianTime time.Duration `json:"median_duration_ns"`
	MedianTurn float64       `json:"median_turns"`
	MedianTool float64       `json:"median_tool_calls"`
	TotalCost  float64       `json:"total_cost_usd"`
	Tasks      []TaskResult  `json:"tasks,omitempty"`
	Error      string        `json:"error,omitempty"`
}

type TaskResult struct {
	Task   string `json:"task"`
	Passed int    `json:"passed"`
	Total  int    `json:"total"`
}

// variant is one cell of an hrn eval matrix.
type variant struct {
	Name    string `json:"name"`
	Backend string `json:"backend"`
	Model   string `json:"model,omitempty"`
	Effort  string `json:"effort,omitempty"`
}

// rewriteSuite returns the suite with its matrix replaced by v alone, its dir made absolute so
// the copy can be written anywhere, and repeats overridden when set. Every other field is kept.
func rewriteSuite(path string, v variant, repeats int) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var spec map[string]any
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	dir, _ := spec["dir"].(string)
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(filepath.Dir(path), dir)
	}
	spec["dir"] = filepath.Clean(dir)
	spec["matrix"] = []variant{v}
	if repeats > 0 {
		spec["repeats"] = repeats
	}
	return json.MarshalIndent(spec, "", "  ")
}

// evalEnv is the environment hrn runs with. For an endpoint, hrn's api backend is pointed at
// it. For an agent, ANTHROPIC_BASE_URL is removed: Claude Code reads it too, and a value left
// over from an endpoint would send a "claude" baseline to a local model without any error.
func evalEnv(base []string, endpointURL string) []string {
	env := make([]string, 0, len(base)+2)
	for _, kv := range base {
		key, _, _ := strings.Cut(kv, "=")
		switch strings.ToUpper(key) {
		case "ANTHROPIC_BASE_URL":
			continue
		case "ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN":
			if endpointURL != "" {
				continue // never send a real key to a local server
			}
		}
		env = append(env, kv)
	}
	if endpointURL != "" {
		env = append(env, "ANTHROPIC_BASE_URL="+endpointURL, "ANTHROPIC_API_KEY=local")
	}
	return env
}

var evalIDLine = regexp.MustCompile(`── eval ([0-9a-z]+) ·`)

// runSuite runs one suite for one target and reads the saved record back.
func runSuite(ctx context.Context, hrn, suite, workDir string, v variant, repeats int, endpointURL string, log io.Writer) SuiteResult {
	res := SuiteResult{Suite: suite}
	spec, err := rewriteSuite(suite, v, repeats)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	specPath := filepath.Join(workDir, fmt.Sprintf("%s.%s", v.Name, filepath.Base(suite)))
	if err := os.WriteFile(specPath, spec, 0o644); err != nil {
		res.Error = err.Error()
		return res
	}
	var out strings.Builder
	cmd := exec.CommandContext(ctx, hrn, "eval", specPath)
	cmd.Env = evalEnv(os.Environ(), endpointURL)
	cmd.Stdout = io.MultiWriter(&out, log)
	cmd.Stderr = io.MultiWriter(&out, log)
	runErr := cmd.Run()

	m := evalIDLine.FindStringSubmatch(out.String())
	if m == nil {
		res.Error = fmt.Sprintf("hrn eval did not report an eval id (%v)", runErr)
		return res
	}
	res.EvalID = m[1]
	if err := readRecord(res.EvalID, &res); err != nil {
		res.Error = err.Error()
	}
	return res
}

// record is the part of hrn's saved eval record the sweep reads.
type record struct {
	Table struct {
		Variants []struct {
			Attempts       int           `json:"attempts"`
			Passed         int           `json:"passed"`
			TotalCostUSD   float64       `json:"total_cost_usd"`
			MedianDuration time.Duration `json:"median_duration_ns"`
			MedianTurns    float64       `json:"median_turns"`
			MedianTools    float64       `json:"median_tool_calls"`
		} `json:"variants"`
		Tasks []struct {
			Task   string `json:"task"`
			Passed int    `json:"passed"`
			Total  int    `json:"total"`
		} `json:"tasks"`
	} `json:"table"`
}

func hrnHome() string {
	if h := os.Getenv("HRN_HOME"); h != "" {
		return h
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".hrn")
}

func readRecord(id string, res *SuiteResult) error {
	data, err := os.ReadFile(filepath.Join(hrnHome(), "evals", id+".json"))
	if err != nil {
		return fmt.Errorf("read eval record %s: %w", id, err)
	}
	return fillFromRecord(data, res)
}

func fillFromRecord(data []byte, res *SuiteResult) error {
	var rec record
	if err := json.Unmarshal(data, &rec); err != nil {
		return err
	}
	if len(rec.Table.Variants) != 1 {
		return fmt.Errorf("eval record has %d variants, want 1", len(rec.Table.Variants))
	}
	v := rec.Table.Variants[0]
	res.Attempts, res.Passed = v.Attempts, v.Passed
	res.MedianTime, res.MedianTurn, res.MedianTool = v.MedianDuration, v.MedianTurns, v.MedianTools
	res.TotalCost = v.TotalCostUSD
	for _, t := range rec.Table.Tasks {
		res.Tasks = append(res.Tasks, TaskResult{Task: t.Task, Passed: t.Passed, Total: t.Total})
	}
	return nil
}
