package ip2region

import (
	"strings"
	"testing"
)

func TestParseRegion(t *testing.T) {
	got := ParseRegion("中国|广东省|深圳市|电信|CN")
	if got.Province != "广东" || got.City != "深圳" {
		t.Fatalf("unexpected region: %+v", got)
	}
}

func TestParseRegionCityFallbackToProvince(t *testing.T) {
	got := ParseRegion("中国|北京市|0|联通|CN")
	if got.Province != "北京" || got.City != "北京" {
		t.Fatalf("unexpected fallback: %+v", got)
	}
}

func TestNewSearcherReturnsErrorForInvalidXDB(t *testing.T) {
	_, err := NewSearcher("/tmp/not-exists-ip2region.xdb")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseCachePolicy(t *testing.T) {
	p, err := parseCachePolicy("content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p <= 0 {
		t.Fatalf("unexpected policy: %d", p)
	}

	if _, err := parseCachePolicy("not-exists-policy"); err == nil {
		t.Fatalf("expected error for invalid cache policy")
	}
}

func TestNewSearcherWithConfigRejectsInvalidCachePolicy(t *testing.T) {
	_, err := NewSearcherWithConfig(Config{
		V4XDBPath:   "",
		V6XDBPath:   "",
		CachePolicy: "bad-policy",
		Searchers:   0,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "cache") {
		t.Fatalf("unexpected error: %v", err)
	}
}
