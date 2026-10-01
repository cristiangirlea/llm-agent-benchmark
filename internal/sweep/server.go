package sweep

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// server is a llama-server the sweep launched for one endpoint.
type server struct {
	cmd  *exec.Cmd
	log  *os.File
	done chan error
}

// launch starts llama-server for e, writes its output to logPath and waits until /health
// answers 200. It refuses to start when something already listens on the port, so a server
// left over from an earlier run is never measured by mistake.
func launch(ctx context.Context, c Config, e Endpoint, logPath string) (*server, error) {
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(c.Port))
	if conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond); err == nil {
		conn.Close()
		return nil, fmt.Errorf("port %d is already in use; stop whatever listens there first", c.Port)
	}
	args := []string{"-m", filepath.Join(c.ModelsDir, e.Model)}
	args = append(args, c.ServerArgs...)
	args = append(args, e.Args...)
	args = append(args, "--host", "127.0.0.1", "--port", strconv.Itoa(c.Port))

	logf, err := os.Create(logPath)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(c.LlamaServer, args...)
	cmd.Stdout, cmd.Stderr = logf, logf
	if err := cmd.Start(); err != nil {
		logf.Close()
		return nil, fmt.Errorf("start llama-server: %w", err)
	}
	s := &server{cmd: cmd, log: logf, done: make(chan error, 1)}
	go func() { s.done <- cmd.Wait() }()

	deadline := time.Now().Add(time.Duration(c.StartTimeout))
	url := e.URL(c.Port) + "/health"
	for {
		select {
		case err := <-s.done:
			logf.Close()
			return nil, fmt.Errorf("llama-server exited before it was ready (%v); see %s", err, logPath)
		case <-ctx.Done():
			s.stop()
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
		if healthy(url) {
			return s, nil
		}
		if time.Now().After(deadline) {
			s.stop()
			return nil, fmt.Errorf("llama-server not healthy after %s; see %s", time.Duration(c.StartTimeout), logPath)
		}
	}
}

func healthy(url string) bool {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// stop kills the server and waits for it, so the next endpoint finds the port and the VRAM free.
func (s *server) stop() {
	if s == nil {
		return
	}
	_ = s.cmd.Process.Kill()
	select {
	case <-s.done:
	case <-time.After(30 * time.Second):
	}
	s.log.Close()
}
