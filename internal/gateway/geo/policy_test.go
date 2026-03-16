package geo

import (
	"testing"

	"pipescope/internal/gateway/rule"
)

func TestMatcher_Check(t *testing.T) {
	tests := []struct {
		name     string
		policy   *rule.GeoPolicy
		info     GeoInfo
		expected CheckResult
	}{
		{
			name:   "nil policy allows all",
			policy: nil,
			info:   GeoInfo{Country: "US"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "empty rules allows all",
			policy: &rule.GeoPolicy{
				Mode:  "allow",
				Rules: []rule.GeoRule{},
			},
			info:     GeoInfo{Country: "US"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "allow mode - country match",
			policy: &rule.GeoPolicy{
				Mode: "allow",
				Rules: []rule.GeoRule{
					{Country: "CN"},
				},
			},
			info:     GeoInfo{Country: "CN"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "allow mode - country mismatch without require hit",
			policy: &rule.GeoPolicy{
				Mode: "allow",
				Rules: []rule.GeoRule{
					{Country: "CN"},
				},
			},
			info:     GeoInfo{Country: "US"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "allow mode - country mismatch with require hit",
			policy: &rule.GeoPolicy{
				Mode:            "allow",
				RequireAllowHit: true,
				Rules: []rule.GeoRule{
					{Country: "CN"},
				},
			},
			info:     GeoInfo{Country: "US"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonNotInAllowlist},
		},
		{
			name: "deny mode - country match blocks",
			policy: &rule.GeoPolicy{
				Mode: "deny",
				Rules: []rule.GeoRule{
					{Country: "CN"},
				},
			},
			info:     GeoInfo{Country: "CN"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonDenied},
		},
		{
			name: "deny mode - country mismatch allows",
			policy: &rule.GeoPolicy{
				Mode: "deny",
				Rules: []rule.GeoRule{
					{Country: "CN"},
				},
			},
			info:     GeoInfo{Country: "US"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "allow mode - province match",
			policy: &rule.GeoPolicy{
				Mode: "allow",
				Rules: []rule.GeoRule{
					{Country: "CN", Provinces: []string{"北京", "上海"}},
				},
			},
			info:     GeoInfo{Country: "CN", Province: "北京"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "allow mode - province mismatch",
			policy: &rule.GeoPolicy{
				Mode:            "allow",
				RequireAllowHit: true,
				Rules: []rule.GeoRule{
					{Country: "CN", Provinces: []string{"北京", "上海"}},
				},
			},
			info:     GeoInfo{Country: "CN", Province: "广东"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonNotInAllowlist},
		},
		{
			name: "allow mode - adcode match",
			policy: &rule.GeoPolicy{
				Mode: "allow",
				Rules: []rule.GeoRule{
					{Country: "CN", Adcodes: []string{"110000", "310000"}},
				},
			},
			info:     GeoInfo{Country: "CN", Adcode: "110000"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "allow mode - city match with province context",
			policy: &rule.GeoPolicy{
				Mode: "allow",
				Rules: []rule.GeoRule{
					{Country: "CN", Provinces: []string{"北京"}, Cities: []string{"北京"}},
				},
			},
			info:     GeoInfo{Country: "CN", Province: "北京", City: "北京"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "allow mode - city match without province context in rule",
			policy: &rule.GeoPolicy{
				Mode: "allow",
				Rules: []rule.GeoRule{
					{Country: "CN", Cities: []string{"北京"}},
				},
			},
			info:     GeoInfo{Country: "CN", Province: "北京", City: "北京"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "allow mode - city mismatch with province in rule",
			policy: &rule.GeoPolicy{
				Mode:            "allow",
				RequireAllowHit: true,
				Rules: []rule.GeoRule{
					{Country: "CN", Provinces: []string{"北京"}, Cities: []string{"北京"}},
				},
			},
			info:     GeoInfo{Country: "CN", Province: "上海", City: "上海"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonNotInAllowlist},
		},
		{
			name: "case insensitive country match",
			policy: &rule.GeoPolicy{
				Mode: "allow",
				Rules: []rule.GeoRule{
					{Country: "cn"},
				},
			},
			info:     GeoInfo{Country: "CN"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "deny multiple countries",
			policy: &rule.GeoPolicy{
				Mode: "deny",
				Rules: []rule.GeoRule{
					{Country: "CN"},
					{Country: "RU"},
				},
			},
			info:     GeoInfo{Country: "RU"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonDenied},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMatcher(tt.policy)
			result := m.Check(tt.info)
			if result.Allowed != tt.expected.Allowed {
				t.Errorf("Allowed = %v, want %v", result.Allowed, tt.expected.Allowed)
			}
			if result.BlockedReason != tt.expected.BlockedReason {
				t.Errorf("BlockedReason = %v, want %v", result.BlockedReason, tt.expected.BlockedReason)
			}
		})
	}
}

func TestMatcher_TypicalScenarios(t *testing.T) {
	t.Run("block foreign traffic", func(t *testing.T) {
		policy := &rule.GeoPolicy{
			Mode:            "allow",
			RequireAllowHit: true,
			Rules: []rule.GeoRule{
				{Country: "CN"},
			},
		}
		m := NewMatcher(policy)

		// CN traffic should pass
		if r := m.Check(GeoInfo{Country: "CN"}); !r.Allowed {
			t.Error("CN traffic should be allowed")
		}

		// US traffic should be blocked
		if r := m.Check(GeoInfo{Country: "US"}); r.Allowed {
			t.Error("US traffic should be blocked")
		}
	})

	t.Run("block specific province", func(t *testing.T) {
		policy := &rule.GeoPolicy{
			Mode: "deny",
			Rules: []rule.GeoRule{
				{Country: "CN", Provinces: []string{"某省"}},
			},
		}
		m := NewMatcher(policy)

		// Blocked province
		if r := m.Check(GeoInfo{Country: "CN", Province: "某省"}); r.Allowed {
			t.Error("某省 traffic should be blocked")
		}

		// Other provinces should pass
		if r := m.Check(GeoInfo{Country: "CN", Province: "北京"}); !r.Allowed {
			t.Error("北京 traffic should be allowed")
		}
	})

	t.Run("whitelist specific cities by adcode", func(t *testing.T) {
		policy := &rule.GeoPolicy{
			Mode:            "allow",
			RequireAllowHit: true,
			Rules: []rule.GeoRule{
				{Country: "CN", Adcodes: []string{"110000", "310000"}}, // 北京, 上海
			},
		}
		m := NewMatcher(policy)

		// 北京 should pass
		if r := m.Check(GeoInfo{Country: "CN", Adcode: "110000"}); !r.Allowed {
			t.Error("北京 should be allowed")
		}

		// 上海 should pass
		if r := m.Check(GeoInfo{Country: "CN", Adcode: "310000"}); !r.Allowed {
			t.Error("上海 should be allowed")
		}

		// Other cities should be blocked
		if r := m.Check(GeoInfo{Country: "CN", Adcode: "440100"}); r.Allowed {
			t.Error("广州 should be blocked")
		}
	})
}
