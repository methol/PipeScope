package proxy

import (
	"errors"
	"io"
	"net"
)

type copyResult struct {
	direction string
	n         int64
	err       error
}

func proxyDuplex(client net.Conn, upstream net.Conn) (int64, int64, error) {
	results := make(chan copyResult, 2)

	go func() {
		n, err := io.Copy(upstream, client)
		_ = upstream.Close()
		results <- copyResult{
			direction: "up",
			n:         n,
			err:       normalizeCopyErr(err),
		}
	}()

	go func() {
		n, err := io.Copy(client, upstream)
		_ = client.Close()
		results <- copyResult{
			direction: "down",
			n:         n,
			err:       normalizeCopyErr(err),
		}
	}()

	var up int64
	var down int64
	var firstErr error
	for i := 0; i < 2; i++ {
		r := <-results
		if r.direction == "up" {
			up = r.n
		} else {
			down = r.n
		}
		if firstErr == nil && r.err != nil {
			firstErr = r.err
		}
	}
	return up, down, firstErr
}

func normalizeCopyErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

