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

func TestMarkBlockedGeo(t *testing.T) {
	s := New("r1", 10001, "1.1.1.1:1234", "2.2.2.2:80")
	geo := GeoInfo{
		Country:  "CN",
		Province: "北京",
		City:     "北京",
		Adcode:   "110000",
	}
	s.MarkBlockedGeo("geo_denied", geo)

	if s.Status != "blocked" {
		t.Fatalf("status=%s, want blocked", s.Status)
	}
	if s.BlockedReason != "geo_denied" {
		t.Fatalf("blocked_reason=%s, want geo_denied", s.BlockedReason)
	}
	if s.Country != "CN" {
		t.Fatalf("country=%s, want CN", s.Country)
	}
	if s.Province != "北京" {
		t.Fatalf("province=%s, want 北京", s.Province)
	}
	if s.City != "北京" {
		t.Fatalf("city=%s, want 北京", s.City)
	}
	if s.Adcode != "110000" {
		t.Fatalf("adcode=%s, want 110000", s.Adcode)
	}
}

func TestFinalizeIncludesBlockedGeoInfo(t *testing.T) {
	s := New("r1", 10001, "1.1.1.1:1234", "2.2.2.2:80")
	geo := GeoInfo{
		Country:  "US",
		Province: "California",
		City:     "San Francisco",
		Adcode:   "06075",
	}
	s.MarkBlockedGeo("geo_not_in_allowlist", geo)

	e := s.Finalize()
	if e.Status != "blocked" {
		t.Fatalf("status=%s, want blocked", e.Status)
	}
	if e.BlockedReason != "geo_not_in_allowlist" {
		t.Fatalf("blocked_reason=%s, want geo_not_in_allowlist", e.BlockedReason)
	}
	if e.Country != "US" {
		t.Fatalf("country=%s, want US", e.Country)
	}
	if e.Province != "California" {
		t.Fatalf("province=%s, want California", e.Province)
	}
	if e.City != "San Francisco" {
		t.Fatalf("city=%s, want San Francisco", e.City)
	}
	if e.Adcode != "06075" {
		t.Fatalf("adcode=%s, want 06075", e.Adcode)
	}
}

