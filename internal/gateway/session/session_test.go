package session

import (
	"errors"
	"testing"
)

func TestFinalizeDialFail(t *testing.T) {
	s := New("r1", 10001, "1.1.1.1:1234", "2.2.2.2:80")
	s.MarkDialFail(errors.New("refused"))
	e := s.Finalize()
	if e.Status != "dial_fail" {
		t.Fatalf("status=%s", e.Status)
	}
}

func TestFinalizeIOErr(t *testing.T) {
	s := New("r1", 10001, "1.1.1.1:1234", "2.2.2.2:80")
	s.MarkIOErr(errors.New("io failed"))
	e := s.Finalize()
	if e.Status != "io_err" {
		t.Fatalf("status=%s", e.Status)
	}
	if e.Error == "" {
		t.Fatalf("expected error message")
	}
}

