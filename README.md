# llm-agent-benchmark

Compare local models and coding agents on the two things that decide whether one is worth
using: **how fast it is** on this hardware, and **whether it gets coding tasks right**.

`bench sweep` takes a list of targets and puts each one through the same measurements:

- **Endpoints**, meaning models behind an HTTP API. A model file is launched in llama-server and stopped
  afterwards; an already-running server (Ollama, LM Studio, vLLM, a llama-server started by hand)
  is only connected to. Each gets a **speed** stage (time to first token, decode tokens per
  second, total throughput with several requests at once) and the **eval** stage.
- **Agents**, meaning coding agents that [hrn](https://github.com/cristiangirlea/harness) drives:
  Claude Code, Codex, Gemini or opencode through their CLIs, or hosted models through hrn's API
  loop. They get the eval stage only.

The eval stage runs [hrn-evals](https://github.com/cristiangirlea/hrn-evals) suites through
`hrn eval`, one target at a time: real agent tasks in git worktrees, decided by gate commands
(tests pass, a mutant fails, a trap was not followed), not by a model's opinion. Every target
runs the same tasks, so a 4B local model and Claude Code end up in one table.

```
bench sweep sweeps/local-7900xt.json

══ qwen35-4b (Qwen3.5-4B-Q4_K_M.gguf)
speed: concurrency [1 4], 256 tokens each
  c=1  first token 91ms  decode 123.5 t/s  total 118.8 t/s
  c=4  first token 267ms  decode 82.3 t/s  total 304.3 t/s
eval suites/golden/greeting.json… 1/1 passed
eval suites/golden/gobench.json… 3/4 passed
...
```

Each sweep writes `results/<timestamp>/`: `results.jsonl` (one line per target, written as soon
as it finishes, so an interrupted sweep keeps what it measured), `report.md` (summary and a
task-by-target table), `manifest.json` (the sweep as loaded, hrn and llama-server versions) and
`logs/` (server output, hrn output).

## Install

Go 1.26, standard library only:

```bash
go build -o bench ./cmd/bench
```

The eval stage needs `hrn` on PATH and a checkout of hrn-evals. Launching models needs a
llama.cpp build (`llama-server`); see [inference](https://github.com/cristiangirlea/inference)
for the one measured on an RX 7900 XT.

## The sweep file

[`sweeps/local-7900xt.json`](sweeps/local-7900xt.json) is a complete example. Relative paths are
taken from the file's directory.

| Field | Meaning |
|---|---|
| `llama_server`, `models_dir`, `port`, `server_args`, `start_timeout` | How launched endpoints start. `server_args` go to every model, an endpoint's own `args` after them. |
| `endpoints[]` | `name`, plus either `model` (a file in `models_dir`, launched) or `base_url` (already running). `model_id` is sent as the request's model (Ollama routes by it). `anthropic` says whether the server answers `/v1/messages`; it defaults to true for launched llama-server. |
| `agents[]` | `name`, `backend` (an hrn backend: `claude`, `codex`, `gemini`, `opencode`, `api`), optional `model`, `effort`. |
| `speed` | `concurrency` (default `[1, 4]`), `max_tokens` (256), `prompt`, `skip`. |
| `evals` | `dir` (the hrn-evals checkout), `suites`, `repeats` (attempts per task, overriding each suite), `hrn`, `skip`. |

Names may contain letters, digits, `_` and `-` only: they become hrn eval variant names.
`bench check <file>` validates a sweep file and prints what it would run.

Flags: `--only a,b` runs only the named targets, `--no-agents` skips the CLI baselines,
`--no-evals` measures speed only, `--no-speed` runs evals only.

## How the pieces connect

An endpoint's eval stage runs hrn's own agent loop (`backend: api`) with
`ANTHROPIC_BASE_URL` pointing at the endpoint. Current llama-server builds serve the Anthropic
Messages API at `/v1/messages`, tool calls included, so hrn needs no change and no extra
program to drive a local model. An external endpoint is assumed to serve only the OpenAI API
until its entry says `"anthropic": true`; check your server's version for `/v1/messages`
before setting it, or the eval stage is skipped for that endpoint (speed still runs).

Agents run with `ANTHROPIC_BASE_URL` removed from their environment. Claude Code reads that
variable too, so a value left over from an endpoint would send the "Claude" baseline to a local
model with no error. Endpoints never receive a real API key: the sweep replaces it with a
placeholder.

## Choosing a local server

| Server | Use it when |
|---|---|
| **llama.cpp `llama-server`** | The default here. One binary, every GPU vendor (Vulkan, CUDA, ROCm, Metal) and CPU, GGUF quantizations, OpenAI and Anthropic APIs, grammar-constrained JSON. Gives full control of offload (`--n-cpu-moe`), context and slots. |
| **Ollama** | Convenience: `ollama pull` and it runs, built on llama.cpp. Less control over offload and slots, and it trails llama.cpp's newest models and flags. Good for trying a model quickly. |
| **LM Studio** | A desktop UI over llama.cpp (and MLX on Macs). Good for browsing and chatting; the same engine underneath. |
| **vLLM / SGLang** | Serving many users on Linux with NVIDIA (or supported AMD) GPUs: continuous batching and paged attention for throughput. Not for a Windows desktop. |

All of them speak the OpenAI chat API, which is why the speed stage uses it. The practice that
lasts is to depend on that protocol, not on one server.

## What LangChain, LangGraph and Hugging Face are, and why this repo uses none of them

- **Hugging Face** is the hub where open models and datasets are published (every GGUF file
  measured here came from it), plus Python libraries (`transformers`, `datasets`) and servers
  (TGI) for running models from Python. The hub is used here as the model registry; the
  libraries are not needed, because llama.cpp runs the models.
- **LangChain** is a Python/JS library of adapters: one interface over many model providers,
  vector stores and document loaders, plus helpers for chaining calls. Useful to prototype
  against many providers quickly; the cost is a thick abstraction that changes often.
- **LangGraph** (from the LangChain team) builds stateful agents as graphs of steps, with
  checkpoints, retries and human approval between steps. It is the framework answer to "an agent
  loop that survives restarts".

In this ecosystem those jobs are already done by smaller pieces: hrn is the agent loop (backends,
tools, budgets, records), hrn's durable workers are the restartable orchestration, and
[php-python-ai-bridge](https://github.com/cristiangirlea/php-python-ai-bridge) is retrieval. Adding
LangChain would wrap them, not replace anything. They are worth knowing because many teams use
them; they are not needed here.

## Limits

- One attempt per task is noise, especially for small models: set `evals.repeats` (3 or more)
  before comparing.
- The gates check what can be checked by a command. Prose quality (the goroutines explanation)
  is checked for length and topics only; a judge model would score it.
- Agents' costs are what hrn records: Claude Code reports cost, the Codex CLI reports tokens
  only, and local models cost nothing per token.

## Licence

AGPL-3.0, see [LICENSE.txt](LICENSE.txt).
