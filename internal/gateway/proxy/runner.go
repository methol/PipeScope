package proxy

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"pipescope/internal/gateway/rule"
	"pipescope/internal/gateway/session"
)

type dialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

type Runner struct {
	rules []rule.Rule
	out   chan<- session.Event

	dial        dialFunc
	dialTimeout time.Duration
	idleTimeout time.Duration

	queuePolicy string
	sampleRate  float64

	mu          sync.RWMutex
	listeners   map[string]net.Listener
	activeConns map[net.Conn]struct{}
	connWG      sync.WaitGroup
	closing     bool
}

func NewRunner(rules []rule.Rule, out chan<- session.Event) *Runner {
	defaultDialer := &net.Dialer{}
	return &Runner{
		rules:       rules,
		out:         out,
		dial:        defaultDialer.DialContext,
		queuePolicy: "drop",
		sampleRate:  0.1,
		listeners:   make(map[string]net.Listener, len(rules)),
		activeConns: make(map[net.Conn]struct{}),
	}
}

func (r *Runner) SetDialFunc(fn dialFunc) {
	if fn != nil {
		r.dial = fn
	}
}

func (r *Runner) SetTimeouts(dialTimeout, idleTimeout time.Duration) {
	r.dialTimeout = dialTimeout
	r.idleTimeout = idleTimeout
}

func (r *Runner) SetQueuePolicy(policy string, sampleRate float64) {
	switch policy {
	case "drop", "sample", "block":
		r.queuePolicy = policy
	default:
		r.queuePolicy = "drop"
	}
	if sampleRate > 0 && sampleRate <= 1 {
		r.sampleRate = sampleRate
	}
}

func (r *Runner) Start(ctx context.Context) error {
	for _, rl := range r.rules {
		ln, err := net.Listen("tcp", rl.Listen)
		if err != nil {
			_ = r.Close()
			return err
		}

		r.mu.Lock()
		r.listeners[rl.ID] = ln
		r.mu.Unlock()

		go r.acceptLoop(ctx, ln, rl)
	}

	go func() {
		<-ctx.Done()
		_ = r.Close()
	}()
	return nil
}

func (r *Runner) Close() error {
	r.mu.Lock()
	if r.closing {
		r.mu.Unlock()
		r.connWG.Wait()
		return nil
	}
	r.closing = true

	var closeErr error
	for id, ln := range r.listeners {
		if err := ln.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
		delete(r.listeners, id)
	}
	for conn := range r.activeConns {
		if err := conn.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
		delete(r.activeConns, conn)
	}
	r.mu.Unlock()

	r.connWG.Wait()
	return closeErr
}

func (r *Runner) ListenAddr(ruleID string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ln, ok := r.listeners[ruleID]
	if !ok {
		return "", false
	}
	return ln.Addr().String(), true
}

func (r *Runner) acceptLoop(ctx context.Context, ln net.Listener, rl rule.Rule) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				return
			}
		}

		r.mu.Lock()
		if r.closing {
			r.mu.Unlock()
			_ = conn.Close()
			return
		}
		r.activeConns[conn] = struct{}{}
		r.connWG.Add(1)
		r.mu.Unlock()

		go r.proxyConn(ctx, conn, rl)
	}
}

func (r *Runner) proxyConn(ctx context.Context, client net.Conn, rl rule.Rule) {
	defer r.connWG.Done()
	defer func() {
		r.mu.Lock()
		delete(r.activeConns, client)
		r.mu.Unlock()
	}()
	defer client.Close()

	sess := session.New(
		rl.ID,
		listenPort(client.LocalAddr()),
		client.RemoteAddr().String(),
		rl.Forward,
	)

	dialCtx := ctx
	cancel := func() {}
	if r.dialTimeout > 0 {
		dialCtx, cancel = context.WithTimeout(ctx, r.dialTimeout)
	}
	defer cancel()

	upstream, err := r.dial(dialCtx, "tcp", rl.Forward)
	if err != nil {
		markDialStatus(sess, err)
		r.emit(sess.Finalize())
		return
	}
	defer upstream.Close()

	if r.idleTimeout > 0 {
		deadline := time.Now().Add(r.idleTimeout)
		_ = client.SetDeadline(deadline)
		_ = upstream.SetDeadline(deadline)
	}

	upBytes, downBytes, copyErr := proxyDuplex(client, upstream)
	sess.AddUpBytes(upBytes)
	sess.AddDownBytes(downBytes)
	if copyErr != nil {
		markIOStatus(sess, copyErr)
	}

	r.emit(sess.Finalize())
}

func (r *Runner) emit(evt session.Event) {
	if r.out == nil {
		return
	}

	switch r.queuePolicy {
	case "drop":
		select {
		case r.out <- evt:
		default:
		}
	case "sample":
		if r.sampleRate <= 0 {
			return
		}
		if r.sampleRate < 1 {
			rng := rand.New(rand.NewSource(time.Now().UnixNano()))
			if rng.Float64() > r.sampleRate {
				return
			}
		}
		select {
		case r.out <- evt:
		default:
		}
	default:
		r.out <- evt
	}
}

func markDialStatus(sess *session.ConnSession, err error) {
	if isTimeoutError(err) {
		sess.MarkTimeout(err)
		return
	}
	sess.MarkDialFail(err)
}

func markIOStatus(sess *session.ConnSession, err error) {
	if isTimeoutError(err) {
		sess.MarkTimeout(err)
		return
	}
	sess.MarkIOErr(err)
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

func listenPort(addr net.Addr) int {
	if addr == nil {
		return 0
	}
	_, p, err := net.SplitHostPort(addr.String())
	if err != nil {
		return 0
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return 0
	}
	return port
}
