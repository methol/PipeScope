package proxy

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"pipescope/internal/gateway/rule"
	"pipescope/internal/gateway/session"
)

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "dial timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestDialTimeoutStatus(t *testing.T) {
	events := make(chan session.Event, 1)
	runner := NewRunner([]rule.Rule{
		{
			ID:      "r-timeout",
			Listen:  "127.0.0.1:0",
			Forward: "127.0.0.1:65535",
		},
	}, events)

	runner.SetDialFunc(func(_ context.Context, _, _ string) (net.Conn, error) {
		return nil, timeoutErr{}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := runner.Start(ctx); err != nil {
		t.Fatalf("start runner: %v", err)
	}
	defer runner.Close()

	listenAddr, ok := runner.ListenAddr("r-timeout")
	if !ok {
		t.Fatalf("missing runtime listen addr")
	}

	conn, err := net.DialTimeout("tcp", listenAddr, time.Second)
	if err != nil {
		t.Fatalf("dial runner: %v", err)
	}
	_ = conn.Close()

	select {
	case evt := <-events:
		if evt.Status != "timeout" {
			t.Fatalf("status=%s, err=%s", evt.Status, evt.Error)
		}
		if evt.Error == "" {
			t.Fatalf("expected error message")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting event")
	}
}

func TestMarkIOStatusUsesIOErr(t *testing.T) {
	s := session.New("r-io", 10001, "1.1.1.1:1000", "2.2.2.2:80")
	markIOStatus(s, errors.New("copy failed"))
	evt := s.Finalize()
	if evt.Status != "io_err" {
		t.Fatalf("status=%s", evt.Status)
	}
	if evt.Error == "" {
		t.Fatalf("expected error message")
	}
}

func TestIsTimeoutError(t *testing.T) {
	if !isTimeoutError(timeoutErr{}) {
		t.Fatalf("expected timeout error")
	}
	if isTimeoutError(errors.New("plain")) {
		t.Fatalf("plain error should not be timeout")
	}
}

