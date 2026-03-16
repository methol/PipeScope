package geo

import (
	"errors"
	"testing"

	"pipescope/internal/gateway/rule"
	"pipescope/internal/geo/ip2region"
)

func TestLookupResolveCountryCode(t *testing.T) {
	tests := []struct {
		name   string
		region ip2region.Region
		want   string
	}{
		{
			name:   "prefer code when present",
			region: ip2region.Region{Country: "中国", Code: "CN"},
			want:   "CN",
		},
		{
			name:   "fallback to country uppercase when code missing",
			region: ip2region.Region{Country: "cn", Code: ""},
			want:   "CN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveCountryCode(tt.region); got != tt.want {
				t.Fatalf("resolveCountryCode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMatcher_AllowCNWithDenyXJXZ_RequireAllowHitFalse(t *testing.T) {
	policy := &rule.GeoPolicy{
		RequireAllowHit: false,
		Allow: []rule.GeoRule{
			{Country: "CN"},
		},
		Deny: []rule.GeoRule{
			{Country: "CN", Provinces: []string{"新疆", "西藏"}},
		},
	}

	m := NewMatcher(policy)

	if got := m.Check(GeoInfo{Country: "CN", Province: "四川"}); !got.Allowed {
		t.Fatalf("sichuan should be allowed, got %+v", got)
	}

	if got := m.Check(GeoInfo{Country: "CN", Province: "湖北"}); !got.Allowed {
		t.Fatalf("hubei should be allowed, got %+v", got)
	}

	if got := m.Check(GeoInfo{Country: "CN", Province: "贵州"}); !got.Allowed {
		t.Fatalf("guizhou should be allowed, got %+v", got)
	}

	if got := m.Check(GeoInfo{Country: "CN", Province: "新疆"}); got.Allowed || got.BlockedReason != BlockedReasonDenied {
		t.Fatalf("xinjiang should be denied, got %+v", got)
	}

	if got := m.Check(GeoInfo{Country: "CN", Province: "西藏"}); got.Allowed || got.BlockedReason != BlockedReasonDenied {
		t.Fatalf("xizang should be denied, got %+v", got)
	}
}

func TestLookupFunc_UsesCountryCodeForGeoPolicyMatching(t *testing.T) {
	searcher := &ip2region.Searcher{}
	searcher.SetLookupFn(func(ip string) (string, error) {
		return "中国|四川省|成都市|电信|CN", nil
	})

	lookup := LookupFunc(searcher, nil)
	info, err := lookup("1.1.1.1")
	if err != nil {
		t.Fatalf("lookup error: %v", err)
	}
	if info.Country != "CN" {
		t.Fatalf("country=%q, want CN", info.Country)
	}

	policy := &rule.GeoPolicy{
		RequireAllowHit: false,
		Allow:           []rule.GeoRule{{Country: "CN"}},
		Deny:            []rule.GeoRule{{Country: "CN", Provinces: []string{"新疆", "西藏"}}},
	}
	if got := NewMatcher(policy).Check(info); !got.Allowed {
		t.Fatalf("sichuan should be allowed after country-code normalization, got %+v", got)
	}
}

func TestLookupFunc_PropagatesLookupError(t *testing.T) {
	wantErr := errors.New("lookup failed")
	searcher := &ip2region.Searcher{}
	searcher.SetLookupFn(func(ip string) (string, error) {
		return "", wantErr
	})

	lookup := LookupFunc(searcher, nil)
	_, err := lookup("8.8.8.8")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}
