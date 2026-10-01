package sweep

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Stream is what one streamed completion measured.
type Stream struct {
	PromptTokens     int           `json:"prompt_tokens"`
	CompletionTokens int           `json:"completion_tokens"`
	FirstToken       time.Duration `json:"first_token_ns"` // request sent to first generated text
	Total            time.Duration `json:"total_ns"`
}

// DecodeRate is tokens per second after the first token, so it excludes queueing and prefill.
func (s Stream) DecodeRate() float64 {
	gen := s.Total - s.FirstToken
	if s.CompletionTokens < 2 || gen <= 0 {
		return 0
	}
	return float64(s.CompletionTokens-1) / gen.Seconds()
}

// Level is the result of n requests sent at once.
type Level struct {
	Concurrency int `json:"concurrency"`
	// Throughput is all completion tokens divided by the wall time of the whole level.
	Throughput float64 `json:"throughput_tps"`
	// The medians over the level's requests.
	FirstToken time.Duration `json:"median_first_token_ns"`
	Decode     float64       `json:"median_decode_tps"`
	Errors     []string      `json:"errors,omitempty"`
}

// measureSpeed sends c.Speed.Prompt at every concurrency level and returns one Level each.
// The OpenAI chat API is used because every local server speaks it.
func measureSpeed(ctx context.Context, sp Speed, e Endpoint, url string) []Level {
	// One request first, unmeasured: it loads weights into memory and warms caches.
	_, _ = stream(ctx, url, e, sp, -1)
	var levels []Level
	for _, n := range sp.Concurrency {
		var (
			wg      sync.WaitGroup
			mu      sync.Mutex
			streams []Stream
			errs    []string
		)
		start := time.Now()
		for i := range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				s, err := stream(ctx, url, e, sp, i)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					errs = append(errs, err.Error())
					return
				}
				streams = append(streams, s)
			}()
		}
		wg.Wait()
		levels = append(levels, summarise(n, streams, errs, time.Since(start)))
	}
	return levels
}

func summarise(n int, streams []Stream, errs []string, wall time.Duration) Level {
	l := Level{Concurrency: n, Errors: errs}
	if len(streams) == 0 {
		return l
	}
	tokens := 0
	firsts := make([]float64, 0, len(streams))
	rates := make([]float64, 0, len(streams))
	for _, s := range streams {
		tokens += s.CompletionTokens
		firsts = append(firsts, float64(s.FirstToken))
		rates = append(rates, s.DecodeRate())
	}
	l.Throughput = float64(tokens) / wall.Seconds()
	l.FirstToken = time.Duration(median(firsts))
	l.Decode = median(rates)
	return l
}

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	sort.Float64s(v)
	if len(v)%2 == 1 {
		return v[len(v)/2]
	}
	return (v[len(v)/2-1] + v[len(v)/2]) / 2
}

// stream sends one streamed chat completion and times it. seed varies per request so that
// concurrent requests are not identical; prompt caching is off so every request pays prefill.
func stream(ctx context.Context, url string, e Endpoint, sp Speed, seed int) (Stream, error) {
	body := map[string]any{
		"model":          e.ID(),
		"messages":       []map[string]string{{"role": "user", "content": sp.Prompt}},
		"max_tokens":     sp.MaxTokens,
		"temperature":    0.7,
		"seed":           seed,
		"stream":         true,
		"stream_options": map[string]bool{"include_usage": true},
		"cache_prompt":   false,
	}
	if !e.Thinking {
		body["chat_template_kwargs"] = map[string]bool{"enable_thinking": false}
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(url, "/")+"/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return Stream{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Stream{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var b bytes.Buffer
		b.ReadFrom(resp.Body)
		return Stream{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(b.String()))
	}
	return readStream(resp.Body, start)
}

// chunk is the part of an OpenAI streaming chunk the sweep reads.
type chunk struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			Reasoning        string `json:"reasoning"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// readStream reads server-sent events until [DONE]. The first token is the first chunk that
// carries any generated text, answer or reasoning. Token counts come from the server's usage
// report, never from counting chunks: one chunk can hold several tokens.
func readStream(r interface{ Read([]byte) (int, error) }, start time.Time) (Stream, error) {
	var s Stream
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		data, ok := strings.CutPrefix(line, "data:")
		if !ok {
			continue
		}
		data = strings.TrimSpace(data)
		if data == "[DONE]" {
			break
		}
		var c chunk
		if json.Unmarshal([]byte(data), &c) != nil {
			continue
		}
		if s.FirstToken == 0 {
			for _, ch := range c.Choices {
				if ch.Delta.Content != "" || ch.Delta.ReasoningContent != "" || ch.Delta.Reasoning != "" {
					s.FirstToken = time.Since(start)
					break
				}
			}
		}
		if c.Usage != nil {
			s.PromptTokens = c.Usage.PromptTokens
			s.CompletionTokens = c.Usage.CompletionTokens
		}
	}
	s.Total = time.Since(start)
	if err := sc.Err(); err != nil {
		return s, err
	}
	if s.CompletionTokens == 0 {
		return s, fmt.Errorf("the server reported no usage; it may not support stream_options.include_usage")
	}
	return s, nil
}
