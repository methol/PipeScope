package proxy

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"

	"pipescope/internal/gateway/rule"
	"pipescope/internal/gateway/session"
)

func TestProxyForwardsBytes(t *testing.T) {
	upstream := startEchoServer(t)
	events := make(chan session.Event, 1)
	runner := NewRunner([]rule.Rule{
		{
			ID:      "r1",
			Listen:  "127.0.0.1:0",
			Forward: upstream,
		},
	}, events)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := runner.Start(ctx); err != nil {
		t.Fatalf("start runner: %v", err)
	}
	defer runner.Close()

	listenAddr, ok := runner.ListenAddr("r1")
	if !ok {
		t.Fatalf("missing runtime listen addr")
	}

	conn, err := net.DialTimeout("tcp", listenAddr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial runner: %v", err)
	}

	payload := []byte("pipescope")
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("echo mismatch: got=%q want=%q", got, payload)
	}
	_ = conn.Close()

	select {
	case evt := <-events:
		if evt.UpBytes <= 0 || evt.DownBytes <= 0 || evt.TotalBytes <= 0 {
			t.Fatalf("unexpected event bytes: %+v", evt)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting event")
	}
}

func TestEmitDropPolicyDoesNotBlockWhenQueueFull(t *testing.T) {
	events := make(chan session.Event, 1)
	events <- session.Event{RuleID: "existing"}

	runner := NewRunner(nil, events)
	runner.SetQueuePolicy("drop", 1)

	done := make(chan struct{})
	go func() {
		runner.emit(session.Event{RuleID: "new"})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("emit blocked with drop policy")
	}

	if len(events) != 1 {
		t.Fatalf("expected channel to remain full, len=%d", len(events))
	}
}

func TestEmitBlockPolicyBlocksWhenQueueFull(t *testing.T) {
	events := make(chan session.Event, 1)
	events <- session.Event{RuleID: "existing"}

	runner := NewRunner(nil, events)
	runner.SetQueuePolicy("block", 1)

	done := make(chan struct{})
	go func() {
		runner.emit(session.Event{RuleID: "new"})
		close(done)
	}()

	select {
	case <-done:
		t.Fatalf("emit should block when queue is full")
	case <-time.After(100 * time.Millisecond):
	}

	<-events

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("emit did not unblock after draining channel")
	}
}

func TestEmitSamplePolicyDoesNotBlockWhenQueueFull(t *testing.T) {
	events := make(chan session.Event, 1)
	events <- session.Event{RuleID: "existing"}

	runner := NewRunner(nil, events)
	runner.SetQueuePolicy("sample", 1)

	done := make(chan struct{})
	go func() {
		runner.emit(session.Event{RuleID: "new"})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("emit blocked with sample policy")
	}
}

func TestEmitSamplePolicyEnqueuesWhenQueueHasCapacity(t *testing.T) {
	events := make(chan session.Event, 1)
	runner := NewRunner(nil, events)
	runner.SetQueuePolicy("sample", 0)

	runner.emit(session.Event{RuleID: "new"})

	if len(events) != 1 {
		t.Fatalf("expected event to be enqueued when queue has capacity, len=%d", len(events))
	}
}

func TestSamplePolicyUsesPersistentRNG(t *testing.T) {
	runner := NewRunner(nil, nil)
	before := runner.rng
	runner.SetQueuePolicy("sample", 0.5)

	for i := 0; i < 5; i++ {
		runner.emit(session.Event{RuleID: "r"})
	}

	if runner.rng == nil {
		t.Fatalf("rng should be initialized")
	}
	if before != runner.rng {
		t.Fatalf("rng should be reused instead of recreated")
	}
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

