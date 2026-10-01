// Package sweep runs every target of a sweep file through the same measurements: speed for
// model endpoints, and hrn eval suites for every target, then writes one comparison.
package sweep

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// Config is a sweep file. Paths in it are taken from the file's directory when relative.
type Config struct {
	// LlamaServer is the llama-server binary used for endpoints that name a model file.
	LlamaServer string `json:"llama_server,omitempty"`
	// ModelsDir is where those model files live.
	ModelsDir string `json:"models_dir,omitempty"`
	// Port is where a launched llama-server listens; default 8081.
	Port int `json:"port,omitempty"`
	// ServerArgs go to every launched llama-server, before the endpoint's own args.
	ServerArgs []string `json:"server_args,omitempty"`
	// StartTimeout bounds how long a launched server may take to report healthy; default 3m.
	StartTimeout Duration `json:"start_timeout,omitempty"`

	Endpoints []Endpoint `json:"endpoints,omitempty"`
	Agents    []Agent    `json:"agents,omitempty"`
	Speed     Speed      `json:"speed"`
	Evals     Evals      `json:"evals"`

	// Out is the directory each sweep writes a timestamped results folder into; default results.
	Out string `json:"out,omitempty"`
}

// Endpoint is a model behind an HTTP API. With Model set, the sweep launches llama-server for
// it and stops it afterwards; with BaseURL set, the server is already running (Ollama,
// LM Studio, vLLM, a llama-server started by hand) and the sweep only connects to it.
type Endpoint struct {
	Name    string   `json:"name"`
	Model   string   `json:"model,omitempty"`    // a file in ModelsDir
	Args    []string `json:"args,omitempty"`     // extra llama-server flags for this model
	BaseURL string   `json:"base_url,omitempty"` // an already-running server
	// ModelID is sent as "model" in requests. llama-server serving one model ignores it;
	// Ollama and LM Studio route by it. Default: Name.
	ModelID string `json:"model_id,omitempty"`
	// Anthropic says whether the endpoint also serves the Anthropic Messages API at
	// /v1/messages, which hrn's api backend needs for the eval stage. Default true for launched
	// llama-server, false for an external endpoint until it is set.
	Anthropic *bool `json:"anthropic,omitempty"`
	// Thinking leaves the model's reasoning on during the speed stage. Off by default, so speed
	// measures answer tokens; it is a llama-server option and other servers ignore it.
	Thinking bool `json:"thinking,omitempty"`
}

// Agent is a coding agent hrn drives through its own backend: a CLI the user is logged in to
// (claude, codex, gemini, opencode) or the api backend with hosted credentials. Agents have no
// speed stage; they run the eval suites only.
type Agent struct {
	Name    string `json:"name"`
	Backend string `json:"backend"`
	Model   string `json:"model,omitempty"`
	Effort  string `json:"effort,omitempty"`
}

// Speed configures the throughput stage for endpoints.
type Speed struct {
	Concurrency []int  `json:"concurrency,omitempty"` // default [1, 4]
	MaxTokens   int    `json:"max_tokens,omitempty"`  // default 256
	Prompt      string `json:"prompt,omitempty"`
	Skip        bool   `json:"skip,omitempty"`
}

// Evals configures the quality stage: hrn eval suites whose matrix the sweep replaces with
// one target at a time.
type Evals struct {
	Hrn    string   `json:"hrn,omitempty"` // the hrn binary; default "hrn" on PATH
	Dir    string   `json:"dir,omitempty"` // the suites' repository, for relative suite paths
	Suites []string `json:"suites"`        // eval spec files
	// Repeats overrides every suite's repeats, so one number decides how many attempts each
	// task gets per target; 0 keeps what each suite says.
	Repeats int  `json:"repeats,omitempty"`
	Skip    bool `json:"skip,omitempty"` // speed only
}

var validName = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

const defaultPrompt = "Explain, step by step and in detail, how a hash map handles collisions, " +
	"then write a small Go implementation with comments."

// Load reads a sweep file and fills the defaults.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return Config{}, err
	}
	base := filepath.Dir(abs)
	resolve := func(p string) string {
		if p == "" || filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(base, p)
	}
	c.ModelsDir = resolve(c.ModelsDir)
	c.Evals.Dir = resolve(c.Evals.Dir)
	if c.LlamaServer != "" && filepath.Base(c.LlamaServer) != c.LlamaServer {
		c.LlamaServer = resolve(c.LlamaServer)
	}
	if c.Out == "" {
		c.Out = "results"
	}
	c.Out = resolve(c.Out)
	if c.Port == 0 {
		c.Port = 8081
	}
	if c.StartTimeout == 0 {
		c.StartTimeout = Duration(3 * time.Minute)
	}
	if len(c.Speed.Concurrency) == 0 {
		c.Speed.Concurrency = []int{1, 4}
	}
	if c.Speed.MaxTokens == 0 {
		c.Speed.MaxTokens = 256
	}
	if c.Speed.Prompt == "" {
		c.Speed.Prompt = defaultPrompt
	}
	if c.Evals.Hrn == "" {
		c.Evals.Hrn = "hrn"
	}
	for i, s := range c.Evals.Suites {
		if !filepath.IsAbs(s) {
			if c.Evals.Dir == "" {
				c.Evals.Suites[i] = filepath.Join(base, s)
			} else {
				c.Evals.Suites[i] = filepath.Join(c.Evals.Dir, s)
			}
		}
	}
	return c, c.validate()
}

func (c Config) validate() error {
	var errs []error
	seen := map[string]bool{}
	name := func(n string) {
		switch {
		case n == "":
			errs = append(errs, errors.New("every endpoint and agent needs a name"))
		case seen[n]:
			errs = append(errs, fmt.Errorf("name %q is used twice", n))
		case !validName.MatchString(n):
			// The name becomes an hrn eval variant, and hrn uses dots to separate a cell id.
			errs = append(errs, fmt.Errorf("name %q: use letters, digits, '_' and '-' only", n))
		}
		seen[n] = true
	}
	for _, e := range c.Endpoints {
		name(e.Name)
		switch {
		case e.Model == "" && e.BaseURL == "":
			errs = append(errs, fmt.Errorf("endpoint %q: set model (launched) or base_url (external)", e.Name))
		case e.Model != "" && e.BaseURL != "":
			errs = append(errs, fmt.Errorf("endpoint %q: model and base_url are exclusive", e.Name))
		case e.Model != "" && (c.LlamaServer == "" || c.ModelsDir == ""):
			errs = append(errs, fmt.Errorf("endpoint %q: a launched model needs llama_server and models_dir", e.Name))
		}
	}
	for _, a := range c.Agents {
		name(a.Name)
		if a.Backend == "" {
			errs = append(errs, fmt.Errorf("agent %q: backend is required", a.Name))
		}
	}
	if len(c.Endpoints)+len(c.Agents) == 0 {
		errs = append(errs, errors.New("nothing to sweep: no endpoints and no agents"))
	}
	if !c.Evals.Skip && len(c.Evals.Suites) == 0 {
		errs = append(errs, errors.New("evals.suites is empty (set evals.skip to measure speed only)"))
	}
	for _, n := range c.Speed.Concurrency {
		if n < 1 {
			errs = append(errs, fmt.Errorf("speed.concurrency: %d is not a positive number", n))
		}
	}
	return errors.Join(errs...)
}

// URL is where requests for the endpoint go.
func (e Endpoint) URL(port int) string {
	if e.BaseURL != "" {
		return e.BaseURL
	}
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

// ID is the model name sent in requests.
func (e Endpoint) ID() string {
	if e.ModelID != "" {
		return e.ModelID
	}
	return e.Name
}

// ServesAnthropic says whether hrn's api backend can reach this endpoint.
func (e Endpoint) ServesAnthropic() bool {
	if e.Anthropic != nil {
		return *e.Anthropic
	}
	return e.Model != ""
}

// Duration is a time.Duration written as "90s" or "3m" in JSON.
type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	v, err := time.ParseDuration(s)
	*d = Duration(v)
	return err
}

func (d Duration) MarshalJSON() ([]byte, error) { return json.Marshal(time.Duration(d).String()) }
