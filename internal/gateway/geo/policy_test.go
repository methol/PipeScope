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
			name: "empty allow and deny with require_allow_hit false allows all",
			policy: &rule.GeoPolicy{
				RequireAllowHit: false,
				Allow:           []rule.GeoRule{},
				Deny:            []rule.GeoRule{},
			},
			info:     GeoInfo{Country: "US"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "empty allow and deny with require_allow_hit true denies all",
			policy: &rule.GeoPolicy{
				RequireAllowHit: true,
				Allow:           []rule.GeoRule{},
				Deny:            []rule.GeoRule{},
			},
			info:     GeoInfo{Country: "US"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonNotInAllowlist},
		},
		{
			name: "allow rule - country match",
			policy: &rule.GeoPolicy{
				RequireAllowHit: true,
				Allow: []rule.GeoRule{
					{Country: "CN"},
				},
			},
			info:     GeoInfo{Country: "CN"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "allow rule - country mismatch with require_allow_hit true",
			policy: &rule.GeoPolicy{
				RequireAllowHit: true,
				Allow: []rule.GeoRule{
					{Country: "CN"},
				},
			},
			info:     GeoInfo{Country: "US"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonNotInAllowlist},
		},
		{
			name: "allow rule - country mismatch with require_allow_hit false",
			policy: &rule.GeoPolicy{
				RequireAllowHit: false,
				Allow: []rule.GeoRule{
					{Country: "CN"},
				},
			},
			info:     GeoInfo{Country: "US"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "deny rule - country match blocks",
			policy: &rule.GeoPolicy{
				Deny: []rule.GeoRule{
					{Country: "CN"},
				},
			},
			info:     GeoInfo{Country: "CN"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonDenied},
		},
		{
			name: "deny rule - country mismatch allows",
			policy: &rule.GeoPolicy{
				Deny: []rule.GeoRule{
					{Country: "CN"},
				},
			},
			info:     GeoInfo{Country: "US"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "allow rule - province match",
			policy: &rule.GeoPolicy{
				RequireAllowHit: true,
				Allow: []rule.GeoRule{
					{Country: "CN", Provinces: []string{"北京", "上海"}},
				},
			},
			info:     GeoInfo{Country: "CN", Province: "北京"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "allow rule - province mismatch",
			policy: &rule.GeoPolicy{
				RequireAllowHit: true,
				Allow: []rule.GeoRule{
					{Country: "CN", Provinces: []string{"北京", "上海"}},
				},
			},
			info:     GeoInfo{Country: "CN", Province: "广东"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonNotInAllowlist},
		},
		{
			name: "allow rule - adcode match",
			policy: &rule.GeoPolicy{
				RequireAllowHit: true,
				Allow: []rule.GeoRule{
					{Country: "CN", Adcodes: []string{"110000", "310000"}},
				},
			},
			info:     GeoInfo{Country: "CN", Adcode: "110000"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "deny takes precedence over allow",
			policy: &rule.GeoPolicy{
				RequireAllowHit: true,
				Allow: []rule.GeoRule{
					{Country: "CN"},
				},
				Deny: []rule.GeoRule{
					{Country: "CN", Provinces: []string{"福建"}},
				},
			},
			info:     GeoInfo{Country: "CN", Province: "福建"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonDenied},
		},
		{
			name: "allow CN but deny province - other provinces allowed",
			policy: &rule.GeoPolicy{
				RequireAllowHit: true,
				Allow: []rule.GeoRule{
					{Country: "CN"},
				},
				Deny: []rule.GeoRule{
					{Country: "CN", Provinces: []string{"福建"}},
				},
			},
			info:     GeoInfo{Country: "CN", Province: "广东"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "deny specific province but allow country - CN matched by allow",
			policy: &rule.GeoPolicy{
				RequireAllowHit: true,
				Allow: []rule.GeoRule{
					{Country: "CN"},
				},
				Deny: []rule.GeoRule{
					{Country: "CN", Provinces: []string{"福建"}},
				},
			},
			info:     GeoInfo{Country: "CN", Province: "北京"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "deny all CN - CN blocked even if in allow",
			policy: &rule.GeoPolicy{
				Allow: []rule.GeoRule{
					{Country: "CN"},
				},
				Deny: []rule.GeoRule{
					{Country: "CN"},
				},
			},
			info:     GeoInfo{Country: "CN"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonDenied},
		},
		{
			name: "deny province in CN - block that province",
			policy: &rule.GeoPolicy{
				Allow: []rule.GeoRule{
					{Country: "CN"},
				},
				Deny: []rule.GeoRule{
					{Country: "CN", Provinces: []string{"福建", "广东"}},
				},
			},
			info:     GeoInfo{Country: "CN", Province: "福建"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonDenied},
		},
		{
			name: "deny adcode in CN - block that city",
			policy: &rule.GeoPolicy{
				Allow: []rule.GeoRule{
					{Country: "CN"},
				},
				Deny: []rule.GeoRule{
					{Country: "CN", Adcodes: []string{"440300"}}, // 深圳
				},
			},
			info:     GeoInfo{Country: "CN", Adcode: "440300"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonDenied},
		},
		{
			name: "case insensitive country match in allow",
			policy: &rule.GeoPolicy{
				RequireAllowHit: true,
				Allow: []rule.GeoRule{
					{Country: "cn"},
				},
			},
			info:     GeoInfo{Country: "CN"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "case insensitive country match in deny",
			policy: &rule.GeoPolicy{
				Deny: []rule.GeoRule{
					{Country: "cn"},
				},
			},
			info:     GeoInfo{Country: "CN"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonDenied},
		},
		{
			name: "multiple deny rules",
			policy: &rule.GeoPolicy{
				Allow: []rule.GeoRule{
					{Country: "CN"},
				},
				Deny: []rule.GeoRule{
					{Country: "CN", Provinces: []string{"福建"}},
					{Country: "CN", Adcodes: []string{"440300"}},
				},
			},
			info:     GeoInfo{Country: "CN", Adcode: "440300"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonDenied},
		},
		{
			name: "multiple allow rules",
			policy: &rule.GeoPolicy{
				RequireAllowHit: true,
				Allow: []rule.GeoRule{
					{Country: "CN"},
					{Country: "US"},
				},
			},
			info:     GeoInfo{Country: "US"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "non-CN country denied when only CN allowed with require_allow_hit",
			policy: &rule.GeoPolicy{
				RequireAllowHit: true,
				Allow: []rule.GeoRule{
					{Country: "CN"},
				},
			},
			info:     GeoInfo{Country: "JP"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonNotInAllowlist},
		},
		{
			name: "allow rule - city match with province context",
			policy: &rule.GeoPolicy{
				RequireAllowHit: true,
				Allow: []rule.GeoRule{
					{Country: "CN", Provinces: []string{"北京"}, Cities: []string{"北京"}},
				},
			},
			info:     GeoInfo{Country: "CN", Province: "北京", City: "北京"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "allow rule - city match without province context in rule",
			policy: &rule.GeoPolicy{
				RequireAllowHit: true,
				Allow: []rule.GeoRule{
					{Country: "CN", Cities: []string{"北京"}},
				},
			},
			info:     GeoInfo{Country: "CN", Province: "北京", City: "北京"},
			expected: CheckResult{Allowed: true},
		},
		{
			name: "allow rule - city mismatch with province in rule",
			policy: &rule.GeoPolicy{
				RequireAllowHit: true,
				Allow: []rule.GeoRule{
					{Country: "CN", Provinces: []string{"北京"}, Cities: []string{"北京"}},
				},
			},
			info:     GeoInfo{Country: "CN", Province: "上海", City: "上海"},
			expected: CheckResult{Allowed: false, BlockedReason: BlockedReasonNotInAllowlist},
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
	t.Run("allow CN but deny specific provinces", func(t *testing.T) {
		policy := &rule.GeoPolicy{
			RequireAllowHit: true,
			Allow: []rule.GeoRule{
				{Country: "CN"},
			},
			Deny: []rule.GeoRule{
				{Country: "CN", Provinces: []string{"福建", "广东"}},
			},
		}
		m := NewMatcher(policy)

		// CN 福建应该被拒绝（命中 deny）
		if r := m.Check(GeoInfo{Country: "CN", Province: "福建"}); r.Allowed {
			t.Error("CN 福建应该被拒绝")
		}

		// CN 广东应该被拒绝（命中 deny）
		if r := m.Check(GeoInfo{Country: "CN", Province: "广东"}); r.Allowed {
			t.Error("CN 广东应该被拒绝")
		}

		// CN 北京应该通过（命中 allow，未命中 deny）
		if r := m.Check(GeoInfo{Country: "CN", Province: "北京"}); !r.Allowed {
			t.Error("CN 北京应该被允许")
		}

		// US 应该被拒绝（未命中任何规则，require_allow_hit=true）
		if r := m.Check(GeoInfo{Country: "US"}); r.Allowed {
			t.Error("US 应该被拒绝")
		}
	})

	t.Run("block foreign traffic - whitelist CN only", func(t *testing.T) {
		policy := &rule.GeoPolicy{
			RequireAllowHit: true,
			Allow: []rule.GeoRule{
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

	t.Run("block specific province - blacklist mode", func(t *testing.T) {
		policy := &rule.GeoPolicy{
			Deny: []rule.GeoRule{
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
			RequireAllowHit: true,
			Allow: []rule.GeoRule{
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

	t.Run("deny takes precedence - allow CN but deny CN 福建 and 深圳", func(t *testing.T) {
		policy := &rule.GeoPolicy{
			RequireAllowHit: true,
			Allow: []rule.GeoRule{
				{Country: "CN"},
			},
			Deny: []rule.GeoRule{
				{Country: "CN", Provinces: []string{"福建"}},
				{Country: "CN", Adcodes: []string{"440300"}}, // 深圳
			},
		}
		m := NewMatcher(policy)

		// 福建省被拒绝
		if r := m.Check(GeoInfo{Country: "CN", Province: "福建"}); r.Allowed {
			t.Error("福建 should be blocked")
		}
		if r := m.Check(GeoInfo{Country: "CN", Province: "福建", City: "福州"}); r.Allowed {
			t.Error("福州（福建） should be blocked")
		}

		// 深圳被拒绝（adcode 匹配）
		if r := m.Check(GeoInfo{Country: "CN", Province: "广东", City: "深圳", Adcode: "440300"}); r.Allowed {
			t.Error("深圳 should be blocked")
		}

		// 广东其他城市允许
		if r := m.Check(GeoInfo{Country: "CN", Province: "广东", City: "广州", Adcode: "440100"}); !r.Allowed {
			t.Error("广州 should be allowed")
		}

		// 北京允许
		if r := m.Check(GeoInfo{Country: "CN", Province: "北京"}); !r.Allowed {
			t.Error("北京 should be allowed")
		}

		// 国外拒绝
		if r := m.Check(GeoInfo{Country: "US"}); r.Allowed {
			t.Error("US should be blocked")
		}
	})
}
