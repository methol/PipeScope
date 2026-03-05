package ip2region

import (
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
