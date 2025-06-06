# 🔍 Benchmark — Local LLM Benchmarking UI + Simple CLI (would be upgraded to react)

This project provides a lightweight interface for benchmarking local LLMs (such as [Ollama](https://ollama.com)) using both a **web UI** and a **command-line interface**. It supports streaming responses, tracks token generation metrics, and displays memory usage and execution speed.

> 🧠 Built in Go (Golang) with clean modular structure, ready for expansion into Express/React.

---

## 📁 Project Structure

> Everything is initialized inside the `Benchmark/` folder (ignore any parent directory name).

```
Benchmark/
├── cli/                # CLI entrypoint (benchmark runner)
│   └── benchmark.go
├── cmd/                # Optional wrappers or future binaries
├── config/             # Model loader
│   └── models.go
├── core/               # Core execution logic and system metrics
│   ├── executor.go
│   ├── logger.go
│   └── metrics.go
├── ui/                 # CLI + Web UI helpers
│   ├── input.go
│   ├── metrics.go
│   └── progress.go
├── web/                # Web UI + API layer
│   ├── api.go
│   ├── handlers.go
│   ├── router.go
│   ├── templates/
│   │   └── index.html
│   └── types.go
├── models.txt          # List of installed models inside Ollama
├── go.mod              # Go module definition
├── main.go             # Starts the web server (http://localhost:8080)
```

---

## 🚀 Quick Start

### ✅ 1. Requirements
- Go 1.20+
- [Ollama](https://ollama.com) installed and running locally on `http://localhost:11434`

### ✅ 2. Setup

Initialize the Go module (if not already done):

```bash
cd Benchmark
go mod tidy
```

Ensure `models.txt` contains a list of models available in your local Ollama setup:

```
phi3:mini
llama3
mistral
...
```

### ✅ 3. Run Web UI

```bash
go run main.go
```

Open your browser to:  
👉 [http://localhost:8080](http://localhost:8080)

---

### ✅ 4. Run via CLI

```bash
go run cli/benchmark.go phi3:mini "Write a Go function that adds two numbers"
```

📦 Output:

```
📦 Running benchmark for model: phi3:mini
🧠 Prompt: Write a Go function that adds two numbers

✅ Output:
package main

...

📦 Tokens: 108
🚀 Tokens/sec: 26.52
🕒 First token: 3.41s
⏱️ Duration: 4.07s
✅ Done in 4.07s
```

---

## ⚙️ Configuration

- **Model list**: update `models.txt`
- **Ollama port**: defaults to `http://localhost:11434`
- **Prompt source**:
  - Web UI supports dropdown + free-form input
  - CLI takes the prompt as a second argument

---

## 🛠️ To Do

- [ ] Markdown rendering in UI
- [ ] Support for JSON export
- [ ] Multi-run benchmark comparison
- [ ] Express + React frontend (planned)

---

## 📄 License

MIT — use freely and contribute!

---

## 🤝 Contributing

Pull requests welcome!  
Open an issue for suggestions or bug reports.
