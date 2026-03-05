package proxy

import (
	"context"
	"net"
	"strconv"
	"sync"

	"pipescope/internal/gateway/rule"
	"pipescope/internal/gateway/session"
)

type Runner struct {
	rules []rule.Rule
	out   chan<- session.Event

	mu        sync.RWMutex
	listeners map[string]net.Listener
}

func NewRunner(rules []rule.Rule, out chan<- session.Event) *Runner {
	return &Runner{
		rules:     rules,
		out:       out,
		listeners: make(map[string]net.Listener, len(rules)),
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
	defer r.mu.Unlock()

	var closeErr error
	for id, ln := range r.listeners {
		if err := ln.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
		delete(r.listeners, id)
	}
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
		go r.proxyConn(ctx, conn, rl)
	}
}

func (r *Runner) proxyConn(ctx context.Context, client net.Conn, rl rule.Rule) {
	defer client.Close()

	sess := session.New(
		rl.ID,
		listenPort(client.LocalAddr()),
		client.RemoteAddr().String(),
		rl.Forward,
	)

	upstream, err := (&net.Dialer{}).DialContext(ctx, "tcp", rl.Forward)
	if err != nil {
		sess.MarkDialFail(err)
		r.emit(sess.Finalize())
		return
	}
	defer upstream.Close()

	upBytes, downBytes, copyErr := proxyDuplex(client, upstream)
	sess.AddUpBytes(upBytes)
	sess.AddDownBytes(downBytes)
	if copyErr != nil {
		sess.MarkDialFail(copyErr)
	}

	r.emit(sess.Finalize())
}

func (r *Runner) emit(evt session.Event) {
	if r.out == nil {
		return
	}
	r.out <- evt
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

