// Command bench sweeps local model endpoints and coding agents through the same measurements:
// speed (first token, decode, throughput under concurrency) and hrn eval suites.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/cristiangirlea/llm-agent-benchmark/internal/sweep"
)

const usage = `bench: compare local models and coding agents on speed and on hrn eval suites

Usage:
  bench sweep [flags] [sweep.json]   run the sweep (default file: sweep.json)
  bench check [sweep.json]           load and validate a sweep file, print what it would run
  bench report <run-dir>...          one report from several runs; a later run of a target
                                     replaces the earlier one (merge a --only re-run)

Sweep flags:
  --only a,b      run only the named endpoints and agents
  --no-agents     skip the agents (CLI and hosted baselines)
  --no-evals      speed only
  --no-speed      evals only
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "sweep":
		os.Exit(runSweep(os.Args[2:]))
	case "check":
		os.Exit(runCheck(os.Args[2:]))
	case "report":
		os.Exit(runReport(os.Args[2:]))
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "bench: unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}

func runSweep(args []string) int {
	fs := flag.NewFlagSet("sweep", flag.ContinueOnError)
	only := fs.String("only", "", "comma-separated target names")
	noAgents := fs.Bool("no-agents", false, "skip agents")
	noEvals := fs.Bool("no-evals", false, "speed only")
	noSpeed := fs.Bool("no-speed", false, "evals only")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	c, err := sweep.Load(fileArg(fs.Args()))
	if err != nil {
		fmt.Fprintln(os.Stderr, "bench:", err)
		return 1
	}
	opt := sweep.Options{SkipAgents: *noAgents, SkipEvals: *noEvals, SkipSpeed: *noSpeed}
	if *only != "" {
		opt.Only = strings.Split(*only, ",")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	dir, err := sweep.Run(ctx, c, opt, os.Stdout)
	if dir != "" {
		fmt.Println("\nresults:", dir)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "bench:", err)
		return 1
	}
	return 0
}

func runCheck(args []string) int {
	c, err := sweep.Load(fileArg(args))
	if err != nil {
		fmt.Fprintln(os.Stderr, "bench:", err)
		return 1
	}
	for _, e := range c.Endpoints {
		where := e.BaseURL
		if e.Model != "" {
			where = "launch " + e.Model + " " + strings.Join(e.Args, " ")
		}
		fmt.Printf("endpoint  %-18s %s (evals: %v)\n", e.Name, where, e.ServesAnthropic())
	}
	for _, a := range c.Agents {
		fmt.Printf("agent     %-18s hrn backend %s %s\n", a.Name, a.Backend, a.Model)
	}
	for _, s := range c.Evals.Suites {
		fmt.Printf("suite     %s\n", s)
	}
	return 0
}

func runReport(dirs []string) int {
	if len(dirs) == 0 {
		fmt.Fprintln(os.Stderr, "bench report: name one or more run folders")
		return 2
	}
	results, err := sweep.ReadResults(dirs...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bench:", err)
		return 1
	}
	c := sweep.Config{Speed: sweep.Speed{Concurrency: sweep.Concurrencies(results)}}
	fmt.Print(sweep.Report(results, c))
	return 0
}

func fileArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "sweep.json"
}
