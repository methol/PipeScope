package ip2region

import "testing"

func TestParseRegion(t *testing.T) {
	got := ParseRegion("中国|广东省|深圳市|电信|CN")
	if got.Province != "广东" || got.City != "深圳" {
		t.Fatalf("unexpected region: %+v", got)
	}
}

