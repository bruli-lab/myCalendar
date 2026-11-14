//go:build functional

package harness

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	startTimeout = 15 * time.Second
	stopTimeout  = 10 * time.Second
)

var binaryPath string

func BuildApp(ctx context.Context) (cleanup func(), err error) {
	dir, err := os.MkdirTemp("", "mycalendar-functional-*")
	if err != nil {
		return nil, err
	}
	root, err := moduleRoot(ctx)
	if err != nil {
		return nil, err
	}
	binaryPath = filepath.Join(dir, "myCalendar")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "./cmd/myCalendar")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to build app: %w\n%s", err, out)
	}
	return func() { _ = os.RemoveAll(dir) }, nil
}

func moduleRoot(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "go", "env", "GOMOD").Output()
	if err != nil {
		return "", err
	}
	return filepath.Dir(strings.TrimSpace(string(out))), nil
}

type Options struct {
	NtfyURL    string
	Topic      string
	GoogleURL  string
	TokensFile string
}

type App struct {
	cmd         *exec.Cmd
	logs        *syncBuffer
	done        chan struct{}
	HealthURL   string
	CallbackURL string
}

func Start(t *testing.T, opts Options) *App {
	t.Helper()
	serverHost := FreeHost(t)
	callbackHost := FreeHost(t)
	cmd := exec.CommandContext(context.WithoutCancel(t.Context()), binaryPath)
	cmd.Env = append(os.Environ(),
		"CLIENT_ID=functional-client",
		"CLIENT_SECRET=functional-secret",
		"CALLBACK_HOST="+callbackHost,
		"CALLBACK_URL=http://"+callbackHost+"/callback",
		"TOKENS_FILE="+opts.TokensFile,
		"SERVER_HOST="+serverHost,
		"LOG_LEVEL=debug",
		"NTFY_URL="+opts.NtfyURL,
		"NTFY_TOPIC="+opts.Topic,
		"NTFY_USER=functional",
		"NTFY_PASSWORD=functional",
		"TIMEZONE=Europe/Madrid",
		"DAILY_SCHEDULE=@every 1s",
		"WEEKLY_SCHEDULE=0 0 1 1 *",
		"JOB_TIMEOUT=5s",
		"GOOGLE_API_URL="+opts.GoogleURL,
		"OTEL_TRACES_EXPORTER=none",
	)
	logs := &syncBuffer{}
	cmd.Stdout = logs
	cmd.Stderr = logs
	require.NoError(t, cmd.Start())
	app := &App{
		cmd:         cmd,
		logs:        logs,
		done:        make(chan struct{}),
		HealthURL:   "http://" + serverHost + "/healthz",
		CallbackURL: "http://" + callbackHost + "/callback",
	}
	go func() {
		_ = cmd.Wait()
		close(app.done)
	}()
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("app logs:\n%s", app.Logs())
		}
		_ = app.Stop()
	})
	app.waitHealthy(t)
	return app
}

func (a *App) waitHealthy(t *testing.T) {
	t.Helper()
	require.Eventually(t, func() bool {
		return a.Healthy(t.Context())
	}, startTimeout, 100*time.Millisecond, "app is not healthy:\n%s", a.Logs())
}

func (a *App) Healthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.HealthURL, http.NoBody)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (a *App) Stop() int {
	select {
	case <-a.done:
	default:
		_ = a.cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-a.done:
		case <-time.After(stopTimeout):
			_ = a.cmd.Process.Kill()
			<-a.done
		}
	}
	return a.cmd.ProcessState.ExitCode()
}

func (a *App) Logs() string {
	return a.logs.String()
}

func (a *App) Authorize(t *testing.T, state, code string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, a.CallbackURL+"?state="+state+"&code="+code, http.NoBody)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	return resp.StatusCode
}

func FreeHost(t *testing.T) string {
	t.Helper()
	ln, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host := ln.Addr().String()
	require.NoError(t, ln.Close())
	return host
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
