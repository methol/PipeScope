package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	nethttp "net/http"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestE2EForwardRecordAndQuery(t *testing.T) {
	upstreamAddr := startEchoServer(t)
	proxyPort := getFreePort(t)
	adminPort := getFreePort(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "pipescope-e2e.db")
	cfgPath := filepath.Join(tempDir, "config.yaml")

	cfg := fmt.Sprintf(`
data:
  sqlite_path: %q
  ip2region_xdb_path: ""
  areacity_csv_path: ""
proxy_rules:
  - id: "e2e-rule"
    listen: "127.0.0.1:%d"
    forward: %q
writer:
  queue_size: 1024
  batch_size: 1
  flush_interval_ms: 100
timeouts:
  dial_ms: 1000
  idle_ms: 30000
admin:
  host: "127.0.0.1"
  port: %d
`, dbPath, proxyPort, upstreamAddr, adminPort)
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/pipescope", "-config", cfgPath)
	cmd.Dir = repoRoot(t)
	var logs bytes.Buffer
	cmd.Stdout = &logs
	cmd.Stderr = &logs
	if err := cmd.Start(); err != nil {
		t.Fatalf("start pipescope: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan struct{})
		go func() {
			_, _ = cmd.Process.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = cmd.Process.Kill()
		}
	})

	baseURL := "http://127.0.0.1:" + strconv.Itoa(adminPort)
	waitFor(t, 8*time.Second, 200*time.Millisecond, func() bool {
		rsp, err := nethttp.Get(baseURL + "/api/health")
		if err != nil {
			return false
		}
		defer rsp.Body.Close()
		return rsp.StatusCode == nethttp.StatusOK
	}, "admin health not ready; logs:\n"+logs.String())

	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(proxyPort), 2*time.Second)
	if err != nil {
		t.Fatalf("dial proxy: %v\nlogs:\n%s", err, logs.String())
	}
	payload := []byte("pipescope-e2e")
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("write proxy payload: %v", err)
	}
	buf := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("read proxy echo: %v", err)
	}
	_ = conn.Close()

	waitFor(t, 8*time.Second, 250*time.Millisecond, func() bool {
		items, err := fetchItems(baseURL + "/api/sessions?window=15m")
		return err == nil && len(items) > 0
	}, "session list still empty; logs:\n"+logs.String())

	waitFor(t, 8*time.Second, 250*time.Millisecond, func() bool {
		items, err := fetchItems(baseURL + "/api/map/china?window=15m&metric=conn")
		return err == nil && len(items) > 0
	}, "china map still empty; logs:\n"+logs.String())
}

func fetchItems(url string) ([]map[string]any, error) {
	rsp, err := nethttp.Get(url)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != nethttp.StatusOK {
		return nil, fmt.Errorf("status %d", rsp.StatusCode)
	}
	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(rsp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body.Items, nil
}

func startEchoServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen echo: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}(c)
		}
	}()

	return ln.Addr().String()
}

func waitFor(t *testing.T, timeout time.Duration, interval time.Duration, check func() bool, failMsg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatalf("%s", failMsg)
}

func getFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("get free port: %v", err)
	}
	defer ln.Close()

	_, p, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("parse free port: %v", err)
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		t.Fatalf("atoi port: %v", err)
	}
	return port
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}

