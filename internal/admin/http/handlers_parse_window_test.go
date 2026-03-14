package http

import (
	"testing"
	"time"
)

func TestParseWindowFriendly(t *testing.T) {
	fallback := 15 * time.Minute
	cases := []struct {
		raw  string
		want time.Duration
	}{
		{"1d", 24 * time.Hour},
		{"2w", 14 * 24 * time.Hour},
		{"1mo", 30 * 24 * time.Hour},
		{" 1D ", 24 * time.Hour},
		{"3MO", 90 * 24 * time.Hour},
	}

	for _, tc := range cases {
		got := parseWindow(tc.raw, fallback)
		if got != tc.want {
			t.Fatalf("raw=%q got=%s want=%s", tc.raw, got, tc.want)
		}
	}
}

func TestParseWindowFallback(t *testing.T) {
	fallback := 15 * time.Minute
	cases := []string{"", "abc", "0d", "-1h", "7x"}
	for _, raw := range cases {
		if got := parseWindow(raw, fallback); got != fallback {
			t.Fatalf("raw=%q got=%s want=%s", raw, got, fallback)
		}
	}
}
