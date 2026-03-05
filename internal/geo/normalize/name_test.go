package normalize

import "testing"

func TestNormalizeProvince(t *testing.T) {
	if got := NormalizeProvince("广东省"); got != "广东" {
		t.Fatalf("got=%q", got)
	}
}

func TestNormalizeCity(t *testing.T) {
	if got := NormalizeCity("深圳市"); got != "深圳" {
		t.Fatalf("got=%q", got)
	}
}

