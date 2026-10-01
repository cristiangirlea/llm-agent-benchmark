package sweep

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Result is everything measured for one target.
type Result struct {
	Name     string        `json:"name"`
	Kind     string        `json:"kind"` // endpoint | agent
	Model    string        `json:"model,omitempty"`
	Backend  string        `json:"backend,omitempty"`
	Started  time.Time     `json:"started"`
	Duration time.Duration `json:"duration_ns"`
	Speed    []Level       `json:"speed,omitempty"`
	Suites   []SuiteResult `json:"suites,omitempty"`
	Error    string        `json:"error,omitempty"`
}

// Options narrow a sweep without editing the file.
type Options struct {
	Only       []string // run only these targets
	SkipAgents bool
	SkipEvals  bool
	SkipSpeed  bool
}

func (o Options) wants(name string) bool { return len(o.Only) == 0 || slices.Contains(o.Only, name) }

// Run sweeps every selected target in order, appends each result to results.jsonl as soon as it
// is known (an interrupted sweep keeps what it measured), and writes report.md at the end.
// It returns the run's folder.
func Run(ctx context.Context, c Config, opt Options, log io.Writer) (string, error) {
	dir := filepath.Join(c.Out, time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(filepath.Join(dir, "logs"), 0o755); err != nil {
		return "", err
	}
	if err := writeManifest(dir, c); err != nil {
		return dir, err
	}
	jsonl, err := os.Create(filepath.Join(dir, "results.jsonl"))
	if err != nil {
		return dir, err
	}
	defer jsonl.Close()

	var results []Result
	keep := func(r Result) {
		results = append(results, r)
		line, _ := json.Marshal(r)
		jsonl.Write(append(line, '\n'))
		fmt.Fprintf(log, "── %s done in %s%s\n", r.Name, r.Duration.Round(time.Second), errSuffix(r.Error))
	}

	for _, e := range c.Endpoints {
		if !opt.wants(e.Name) {
			continue
		}
		if ctx.Err() != nil {
			break
		}
		keep(runEndpoint(ctx, c, opt, e, dir, log))
	}
	if !opt.SkipAgents && !opt.SkipEvals && !c.Evals.Skip {
		for _, a := range c.Agents {
			if !opt.wants(a.Name) {
				continue
			}
			if ctx.Err() != nil {
				break
			}
			keep(runAgent(ctx, c, a, dir, log))
		}
	}
	report := Report(results, c)
	if err := os.WriteFile(filepath.Join(dir, "report.md"), []byte(report), 0o644); err != nil {
		return dir, err
	}
	fmt.Fprint(log, "\n"+report)
	return dir, ctx.Err()
}

func errSuffix(e string) string {
	if e == "" {
		return ""
	}
	return " · error: " + e
}

func runEndpoint(ctx context.Context, c Config, opt Options, e Endpoint, dir string, log io.Writer) Result {
	r := Result{Name: e.Name, Kind: "endpoint", Model: e.Model, Started: time.Now()}
	if r.Model == "" {
		r.Model = e.ID() + " @ " + e.BaseURL
	}
	defer func() { r.Duration = time.Since(r.Started) }()
	fmt.Fprintf(log, "\n══ %s (%s)\n", e.Name, r.Model)

	if e.Model != "" {
		fmt.Fprintf(log, "starting llama-server…\n")
		srv, err := launch(ctx, c, e, filepath.Join(dir, "logs", e.Name+".server.log"))
		if err != nil {
			r.Error = err.Error()
			r.Duration = time.Since(r.Started)
			return r
		}
		defer srv.stop()
	} else if !healthy(strings.TrimRight(e.URL(c.Port), "/")+"/health") && !healthy(strings.TrimRight(e.URL(c.Port), "/")+"/v1/models") {
		r.Error = "external endpoint is not answering at " + e.BaseURL
		r.Duration = time.Since(r.Started)
		return r
	}
	url := e.URL(c.Port)

	if !opt.SkipSpeed && !c.Speed.Skip {
		fmt.Fprintf(log, "speed: concurrency %v, %d tokens each\n", c.Speed.Concurrency, c.Speed.MaxTokens)
		r.Speed = measureSpeed(ctx, c.Speed, e, url)
		for _, l := range r.Speed {
			fmt.Fprintf(log, "  c=%d  first token %s  decode %.1f t/s  total %.1f t/s%s\n", l.Concurrency,
				l.FirstToken.Round(time.Millisecond), l.Decode, l.Throughput, errSuffix(strings.Join(l.Errors, "; ")))
		}
	}
	if !opt.SkipEvals && !c.Evals.Skip {
		if !e.ServesAnthropic() {
			r.Error = "evals skipped: the endpoint does not serve /v1/messages (set \"anthropic\": true if it does)"
			r.Duration = time.Since(r.Started)
			return r
		}
		v := variant{Name: e.Name, Backend: "api", Model: e.ID()}
		r.Suites = runSuites(ctx, c, v, url, dir, log)
	}
	r.Duration = time.Since(r.Started)
	return r
}

func runAgent(ctx context.Context, c Config, a Agent, dir string, log io.Writer) Result {
	r := Result{Name: a.Name, Kind: "agent", Model: a.Model, Backend: a.Backend, Started: time.Now()}
	fmt.Fprintf(log, "\n══ %s (hrn backend %s)\n", a.Name, a.Backend)
	v := variant{Name: a.Name, Backend: a.Backend, Model: a.Model, Effort: a.Effort}
	r.Suites = runSuites(ctx, c, v, "", dir, log)
	r.Duration = time.Since(r.Started)
	return r
}

func runSuites(ctx context.Context, c Config, v variant, url, dir string, log io.Writer) []SuiteResult {
	logf, err := os.Create(filepath.Join(dir, "logs", v.Name+".evals.log"))
	if err != nil {
		return []SuiteResult{{Error: err.Error()}}
	}
	defer logf.Close()
	var out []SuiteResult
	for _, s := range c.Evals.Suites {
		if ctx.Err() != nil {
			break
		}
		fmt.Fprintf(log, "eval %s…", relSuite(c, s))
		res := runSuite(ctx, c.Evals.Hrn, s, dir, v, c.Evals.Repeats, url, logf)
		res.Suite = relSuite(c, s)
		fmt.Fprintf(log, " %d/%d passed%s\n", res.Passed, res.Attempts, errSuffix(res.Error))
		out = append(out, res)
	}
	return out
}

func relSuite(c Config, s string) string {
	if c.Evals.Dir != "" {
		if rel, err := filepath.Rel(c.Evals.Dir, s); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.Base(s)
}

// writeManifest records what produced the numbers: the sweep as loaded, and the versions of
// the tools that ran it.
func writeManifest(dir string, c Config) error {
	m := map[string]any{"started": time.Now(), "config": c}
	if out, err := exec.Command(c.Evals.Hrn, "version").Output(); err == nil {
		m["hrn_version"] = strings.TrimSpace(string(out))
	}
	if c.LlamaServer != "" {
		if out, err := exec.Command(c.LlamaServer, "--version").CombinedOutput(); err == nil {
			m["llama_server_version"] = lastNonEmpty(string(out), 2)
		}
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644)
}

func lastNonEmpty(s string, n int) string {
	var keep []string
	for _, l := range strings.Split(strings.TrimSpace(s), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			keep = append(keep, l)
		}
	}
	if len(keep) > n {
		keep = keep[len(keep)-n:]
	}
	return strings.Join(keep, " | ")
}
