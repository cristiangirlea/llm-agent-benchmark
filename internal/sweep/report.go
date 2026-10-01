package sweep

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Report renders the results as Markdown: one summary row per target, then pass/total per
// task and target, then any errors.
func Report(results []Result, c Config) string {
	var b strings.Builder
	b.WriteString("## Summary\n\n")

	head := []string{"target", "kind"}
	for _, n := range c.Speed.Concurrency {
		if n == 1 {
			head = append(head, "first token", "decode t/s")
		} else {
			head = append(head, fmt.Sprintf("total t/s ×%d", n))
		}
	}
	head = append(head, "passed", "median turns", "median time", "cost")
	row(&b, head)
	row(&b, slices.Repeat([]string{"---"}, len(head)))

	for _, r := range results {
		cells := []string{r.Name, r.Kind}
		for _, n := range c.Speed.Concurrency {
			l, ok := level(r.Speed, n)
			switch {
			case n == 1 && ok:
				cells = append(cells, fmtDur(l.FirstToken), fmt.Sprintf("%.1f", l.Decode))
			case n == 1:
				cells = append(cells, "–", "–")
			case ok:
				cells = append(cells, fmt.Sprintf("%.1f", l.Throughput))
			default:
				cells = append(cells, "–")
			}
		}
		passed, attempts, cost := 0, 0, 0.0
		var turns []float64
		var times []float64
		for _, s := range r.Suites {
			passed += s.Passed
			attempts += s.Attempts
			cost += s.TotalCost
			if s.Attempts > 0 {
				turns = append(turns, s.MedianTurn)
				times = append(times, float64(s.MedianTime))
			}
		}
		if attempts == 0 {
			cells = append(cells, "–", "–", "–", "–")
		} else {
			costCell := "–"
			if cost > 0 {
				costCell = fmt.Sprintf("$%.2f", cost)
			}
			cells = append(cells, fmt.Sprintf("%d/%d", passed, attempts), fmt.Sprintf("%.0f", median(turns)),
				fmtDur(time.Duration(median(times))), costCell)
		}
		row(&b, cells)
	}

	tasks, names := taskMatrix(results)
	if len(tasks) > 0 {
		b.WriteString("\n## Tasks\n\n")
		row(&b, append([]string{"suite / task"}, names...))
		row(&b, slices.Repeat([]string{"---"}, len(names)+1))
		for _, t := range tasks {
			cells := []string{t.key}
			for _, n := range names {
				if v, ok := t.byTarget[n]; ok {
					cells = append(cells, v)
				} else {
					cells = append(cells, "–")
				}
			}
			row(&b, cells)
		}
	}

	var errs []string
	for _, r := range results {
		if r.Error != "" {
			errs = append(errs, fmt.Sprintf("- **%s**: %s", r.Name, r.Error))
		}
		for _, l := range r.Speed {
			for _, e := range l.Errors {
				errs = append(errs, fmt.Sprintf("- **%s** speed ×%d: %s", r.Name, l.Concurrency, e))
			}
		}
		for _, s := range r.Suites {
			if s.Error != "" {
				errs = append(errs, fmt.Sprintf("- **%s** %s: %s", r.Name, s.Suite, s.Error))
			}
		}
	}
	if len(errs) > 0 {
		b.WriteString("\n## Errors\n\n" + strings.Join(errs, "\n") + "\n")
	}
	return b.String()
}

type taskRow struct {
	key      string
	byTarget map[string]string
}

func taskMatrix(results []Result) ([]*taskRow, []string) {
	var rows []*taskRow
	index := map[string]*taskRow{}
	var names []string
	for _, r := range results {
		if len(r.Suites) == 0 {
			continue
		}
		names = append(names, r.Name)
		for _, s := range r.Suites {
			for _, t := range s.Tasks {
				key := strings.TrimSuffix(s.Suite, ".json") + " / " + t.Task
				tr, ok := index[key]
				if !ok {
					tr = &taskRow{key: key, byTarget: map[string]string{}}
					index[key] = tr
					rows = append(rows, tr)
				}
				tr.byTarget[r.Name] = fmt.Sprintf("%d/%d", t.Passed, t.Total)
			}
		}
	}
	return rows, names
}

func level(ls []Level, n int) (Level, bool) {
	for _, l := range ls {
		if l.Concurrency == n && l.Throughput > 0 {
			return l, true
		}
	}
	return Level{}, false
}

func fmtDur(d time.Duration) string {
	switch {
	case d <= 0:
		return "–"
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	default:
		return d.Round(100 * time.Millisecond).String()
	}
}

func row(b *strings.Builder, cells []string) {
	b.WriteString("| " + strings.Join(cells, " | ") + " |\n")
}

// ReadResults reads the results.jsonl of each run folder, in order. When a target appears in
// more than one folder the later result replaces the earlier one in place, so a re-run of one
// model (bench sweep --only) can be merged into a full sweep.
func ReadResults(dirs ...string) ([]Result, error) {
	var out []Result
	index := map[string]int{}
	for _, d := range dirs {
		data, err := os.ReadFile(filepath.Join(d, "results.jsonl"))
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if line == "" {
				continue
			}
			var r Result
			if err := json.Unmarshal([]byte(line), &r); err != nil {
				return nil, fmt.Errorf("%s: %w", d, err)
			}
			if i, ok := index[r.Name]; ok {
				out[i] = r
				continue
			}
			index[r.Name] = len(out)
			out = append(out, r)
		}
	}
	return out, nil
}

// Concurrencies lists the concurrency levels present in the results, in ascending order, for
// rendering a report without the sweep file.
func Concurrencies(results []Result) []int {
	seen := map[int]bool{}
	var out []int
	for _, r := range results {
		for _, l := range r.Speed {
			if !seen[l.Concurrency] {
				seen[l.Concurrency] = true
				out = append(out, l.Concurrency)
			}
		}
	}
	slices.Sort(out)
	return out
}
