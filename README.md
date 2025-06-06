# 🔍 Benchmark — LLM Benchmarking UI + CLI + Agents (Live Reload + Queue-ready)

This project provides a modular system to **benchmark local LLMs (Ollama)** via a web UI or CLI, and lays the groundwork for agent-based tasks and project planning.

> 🧠 Built in Go (Golang), with Air-based hot reload in Docker and a clean Domain-Driven folder structure. Includes Redis queue + Postgres, and can evolve into a full SaaS backend.

---

## 📁 Project Structure

All code lives under `Benchmark/` (ignore parent dirs if any):

```
Benchmark/
├── cmd/                # Entrypoints for binaries
│   ├── web/            # HTTP API server
│   ├── cli/            # CLI benchmarking
│   └── worker/         # Agent/queue processor
├── docker/             # Docker build targets
│   ├── Dockerfile.dev
│   └── Dockerfile.prod
├── internal/           # App logic (domain-driven)
│   ├── agents/         # Autonomous planner logic
│   ├── config/         # Global config or constants
│   ├── core/           # Benchmark logic, system metrics
│   ├── migrations/     # DB migrations (SQL)
│   ├── models/         # App-level structs (e.g., Project)
│   └── persistence/    # Database connection (Postgres)
├── queue/              # Redis queue logic
├── results/            # Benchmark result processors
├── tests/              # Unit tests (example included)
├── ui/                 # Helpers for CLI or Web UI display
├── web/                # Web interface layer
│   ├── templates/      # HTML templates
│   └── routes, handlers, types, etc.
├── .air.toml           # Hot reload config (Air)
├── docker-compose.yml  # Full dev environment
├── models.txt          # Ollama model list
├── go.mod / go.sum     # Go module setup
```

---

## 🚀 Quick Start (Development)

### ✅ 1. Requirements

- [Docker](https://www.docker.com/)
- [Ollama](https://ollama.com) running locally at `http://localhost:11434`

### ✅ 2. Start Dev Mode

```bash
docker compose up --build
```

This will:

- Start Redis + Postgres
- Start the app with **Air** (live reloading on file change)
- Expose the app on [http://localhost:8080](http://localhost:8080)

⏱ Changes to `.go` or `.html` files will automatically rebuild + restart the server.

> ❗ Note: Browser will not auto-refresh. You’ll need to reload the page manually.

---

### ✅ 3. Try CLI Benchmark

```bash
go run ./cmd/cli phi3:mini "What is a goroutine in Go?"
```

It prints:

```
📦 Running benchmark for model: phi3:mini
✅ Output:
...

📦 Tokens: 108
🚀 Tokens/sec: 26.52
⏱ First token: 3.41s
⏱ Total duration: 4.07s
```

---

## 🧠 Tech Highlights

- **Air**-based hot reload in dev container (`Dockerfile.dev`)
- **Multistage Docker build** for production (`Dockerfile.prod`)
- **Redis-backed queue system** with `Enqueue` / `Dequeue`
- **Postgres DB connection** with env-configurable DSN
- **Ollama integration** (via HTTP on port 11434)
- Clean Go module layout & separation of concerns
- Easily extendable to run agents or multi-step tasks

---

## ⚙️ Config

- `models.txt`: list of model IDs used by CLI/Web UI
- `.env`: optional, used for overriding:
  - `REDIS_HOST`
  - `POSTGRES_DSN`
  - `OLLAMA_HOST`
- `.air.toml`: defines hot reload behavior for `air`

---

## 🛠 Planned Features

- [ ] Agent worker to pick tasks from queue
- [ ] Agent planner to split prompts into subtasks
- [ ] React-based frontend (currently HTML + Go templating)
- [ ] Markdown rendering in UI
- [ ] JSON export & comparison of runs
- [ ] SQLite support (optional fallback)
- [ ] GitHub Action for benchmarking models on PR

---

## 📄 License

Licensed under **GNU Affero General Public License v3 (AGPLv3)**

✅ You can:
- Use it personally
- Fork & contribute
- Benchmark local models

🚫 You **may not**:
- Use it in paid/hosted products without a commercial license
- Rehost without releasing source

💼 For commercial use → [cristiangirlea@gmail.com](mailto:cristiangirlea@gmail.com)

---

## 🤝 Contributing

PRs welcome. Issues encouraged. Ideas loved.
