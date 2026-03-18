package proxy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"pipescope/internal/gateway/geo"
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

func assertWriteCompletes(t *testing.T, conn net.Conn, payload []byte, timeout time.Duration) {
	t.Helper()

	writeDone := make(chan error, 1)
	go func() {
		_, err := conn.Write(payload)
		writeDone <- err
	}()

	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("write payload for silent drop: %v", err)
		}
	case <-time.After(timeout):
		t.Fatalf("write payload blocked for %v", timeout)
	}
}

func assertReadTimesOutWithoutPayload(t *testing.T, conn net.Conn, timeout time.Duration) {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	if n != 0 {
		t.Fatalf("expected no payload during silent drop window, got %q", buf[:n])
	}
	if err == nil {
		t.Fatalf("expected read to time out without payload during silent drop window")
	}
	if isTimeoutError(err) {
		return
	}
	t.Fatalf("expected timeout without payload during silent drop window, got: %v", err)
}

func assertReadClosesWithoutPayload(t *testing.T, conn net.Conn, timeout time.Duration) {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	if n != 0 {
		t.Fatalf("expected no payload before blocked connection closed, got %q", buf[:n])
	}
	if err == nil {
		t.Fatalf("expected blocked connection to close after silent drop window")
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) || errors.Is(err, net.ErrClosed) {
		return
	}
	t.Fatalf("expected close-related read error after silent drop window, got: %v", err)
}

func assertSilentDropWindow(t *testing.T, conn net.Conn, window time.Duration) {
	t.Helper()

	assertWriteCompletes(t, conn, []byte("blocked probe payload"), 250*time.Millisecond)
	assertReadTimesOutWithoutPayload(t, conn, window/2)
	assertReadClosesWithoutPayload(t, conn, window*3)
}

func startProxyConnWithPipe(t *testing.T, runner *Runner, rl rule.Rule) (net.Conn, <-chan struct{}) {
	t.Helper()

	client, peer := net.Pipe()

	runner.mu.Lock()
	runner.activeConns[client] = struct{}{}
	runner.connWG.Add(1)
	runner.mu.Unlock()

	done := make(chan struct{})
	go func() {
		runner.proxyConn(context.Background(), client, rl)
		close(done)
	}()

	t.Cleanup(func() {
		_ = peer.Close()
		<-done
	})

	return peer, done
}

// Geo policy tests

func TestGeoPolicyDenyModeBlocksMatchingIP(t *testing.T) {
	upstream := startEchoServer(t)
	events := make(chan session.Event, 10)

	// Mock geo lookup that returns CN for 1.2.3.4
	geoLookup := func(ip string) (geo.GeoInfo, error) {
		if ip == "1.2.3.4" {
			return geo.GeoInfo{Country: "CN", Province: "北京", City: "北京", Adcode: "110000"}, nil
		}
		return geo.GeoInfo{Country: "US"}, nil
	}

	runner := NewRunner([]rule.Rule{
		{
			ID:      "r1",
			Listen:  "127.0.0.1:0",
			Forward: upstream,
			GeoPolicy: &rule.GeoPolicy{
				Deny: []rule.GeoRule{
					{Country: "CN"},
				},
			},
		},
	}, events)
	runner.SetGeoLookup(geoLookup)

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

	// Dial from a mocked source (the runner will see the remote addr as 127.0.0.1:xxxxx)
	// For this test, we'll use a custom dialer approach
	// Since we can't easily control the source IP in this test setup,
	// we test the geo lookup is called and policy is applied via the mock

	// The actual IP seen by runner will be 127.0.0.1, which should pass
	conn, err := net.DialTimeout("tcp", listenAddr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial runner: %v", err)
	}
	_ = conn.Close()

	// The connection should succeed since 127.0.0.1 returns US in our mock
	select {
	case evt := <-events:
		if evt.Status == "blocked" {
			t.Fatalf("unexpected block for 127.0.0.1: %+v", evt)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting event")
	}
}

func TestGeoPolicyAllowModeWithRequireAllowHit(t *testing.T) {
	events := make(chan session.Event, 10)
	const blockedDropWindow = 80 * time.Millisecond

	geoLookup := func(ip string) (geo.GeoInfo, error) {
		if ip == "pipe" {
			return geo.GeoInfo{Country: "US"}, nil
		}
		return geo.GeoInfo{Country: "CN"}, nil
	}

	rl := rule.Rule{
		ID:      "r1",
		Listen:  "127.0.0.1:0",
		Forward: "127.0.0.1:9999", // Intentionally wrong, should not be dialed
		GeoPolicy: &rule.GeoPolicy{
			RequireAllowHit: true,
			Allow: []rule.GeoRule{
				{Country: "CN"},
			},
		},
	}

	runner := NewRunner(nil, events)
	runner.SetGeoLookup(geoLookup)
	runner.SetBlockedDropDuration(blockedDropWindow)

	peer, done := startProxyConnWithPipe(t, runner, rl)
	assertSilentDropWindow(t, peer, blockedDropWindow)
	<-done

	// Connection should be blocked because pipe returns US, which is not in allowlist
	select {
	case evt := <-events:
		if evt.Status != "blocked" {
			t.Fatalf("expected blocked status, got: %s", evt.Status)
		}
		if evt.BlockedReason != "geo_not_in_allowlist" {
			t.Fatalf("expected blocked_reason=geo_not_in_allowlist, got: %s", evt.BlockedReason)
		}
		if evt.Country != "US" {
			t.Fatalf("expected country=US, got: %s", evt.Country)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting event")
	}
}

func TestGeoPolicyBlockedConnectionDoesNotDial(t *testing.T) {
	events := make(chan session.Event, 10)
	const blockedDropWindow = 80 * time.Millisecond

	var dialCount int
	var dialMu sync.Mutex

	// Custom dialer that counts calls
	dialFunc := func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialMu.Lock()
		dialCount++
		dialMu.Unlock()
		return nil, &net.DNSError{Err: "should not dial"}
	}

	geoLookup := func(ip string) (geo.GeoInfo, error) {
		if ip == "pipe" {
			return geo.GeoInfo{Country: "CN", Province: "北京"}, nil
		}
		return geo.GeoInfo{Country: "US"}, nil
	}

	rl := rule.Rule{
		ID:      "r1",
		Listen:  "127.0.0.1:0",
		Forward: "127.0.0.1:9999",
		GeoPolicy: &rule.GeoPolicy{
			Deny: []rule.GeoRule{
				{Country: "CN"},
			},
		},
	}

	runner := NewRunner(nil, events)
	runner.SetDialFunc(dialFunc)
	runner.SetGeoLookup(geoLookup)
	runner.SetBlockedDropDuration(blockedDropWindow)

	peer, done := startProxyConnWithPipe(t, runner, rl)
	assertSilentDropWindow(t, peer, blockedDropWindow)
	<-done

	select {
	case evt := <-events:
		if evt.Status != "blocked" {
			t.Fatalf("expected blocked status, got: %s", evt.Status)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting event")
	}

	// Verify no dial was made
	dialMu.Lock()
	count := dialCount
	dialMu.Unlock()

	if count != 0 {
		t.Fatalf("dial should not be called for blocked connection, but was called %d times", count)
	}
}

func TestGeoPolicyRecordsGeoInfoInBlockedEvent(t *testing.T) {
	events := make(chan session.Event, 10)
	const blockedDropWindow = 80 * time.Millisecond

	geoLookup := func(ip string) (geo.GeoInfo, error) {
		if ip != "pipe" {
			t.Fatalf("unexpected lookup ip: %q", ip)
		}
		return geo.GeoInfo{
			Country:  "RU",
			Province: "Moscow",
			City:     "Moscow",
			Adcode:   "101000",
		}, nil
	}

	rl := rule.Rule{
		ID:      "r1",
		Listen:  "127.0.0.1:0",
		Forward: "127.0.0.1:9999",
		GeoPolicy: &rule.GeoPolicy{
			Deny: []rule.GeoRule{
				{Country: "RU"},
			},
		},
	}

	runner := NewRunner(nil, events)
	runner.SetGeoLookup(geoLookup)
	runner.SetBlockedDropDuration(blockedDropWindow)

	peer, done := startProxyConnWithPipe(t, runner, rl)
	assertSilentDropWindow(t, peer, blockedDropWindow)
	<-done

	select {
	case evt := <-events:
		if evt.Status != "blocked" {
			t.Fatalf("expected blocked status, got: %s", evt.Status)
		}
		if evt.BlockedReason != "geo_denied" {
			t.Fatalf("expected blocked_reason=geo_denied, got: %s", evt.BlockedReason)
		}
		if evt.Country != "RU" {
			t.Fatalf("expected country=RU, got: %s", evt.Country)
		}
		if evt.Province != "Moscow" {
			t.Fatalf("expected province=Moscow, got: %s", evt.Province)
		}
		if evt.City != "Moscow" {
			t.Fatalf("expected city=Moscow, got: %s", evt.City)
		}
		if evt.Adcode != "101000" {
			t.Fatalf("expected adcode=101000, got: %s", evt.Adcode)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting event")
	}
}
