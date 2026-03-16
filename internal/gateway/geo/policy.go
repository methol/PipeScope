package geo

import (
	"strings"

	"pipescope/internal/gateway/rule"
)

const (
	BlockedReasonDenied         = "geo_denied"
	BlockedReasonNotInAllowlist = "geo_not_in_allowlist"
)

// Matcher checks if a connection should be blocked based on geo policy
type Matcher struct {
	policy *rule.GeoPolicy
}

// NewMatcher creates a new geo policy matcher
func NewMatcher(policy *rule.GeoPolicy) *Matcher {
	return &Matcher{policy: policy}
}

// CheckResult contains the result of a geo policy check
type CheckResult struct {
	Allowed       bool
	BlockedReason string
}

// Check evaluates the geo policy against the given geo info
// Returns whether the connection is allowed and the blocked reason if not
func (m *Matcher) Check(info GeoInfo) CheckResult {
	if m.policy == nil || len(m.policy.Rules) == 0 {
		return CheckResult{Allowed: true}
	}

	hit := m.matchRule(info)

	switch m.policy.Mode {
	case "allow":
		if hit {
			return CheckResult{Allowed: true}
		}
		if m.policy.RequireAllowHit {
			return CheckResult{
				Allowed:       false,
				BlockedReason: BlockedReasonNotInAllowlist,
			}
		}
		return CheckResult{Allowed: true}

	case "deny":
		if hit {
			return CheckResult{
				Allowed:       false,
				BlockedReason: BlockedReasonDenied,
			}
		}
		return CheckResult{Allowed: true}

	default:
		// Unknown mode, allow by default
		return CheckResult{Allowed: true}
	}
}

// matchRule checks if the geo info matches any rule in the policy
// Priority: adcode > city+province > province > country
func (m *Matcher) matchRule(info GeoInfo) bool {
	for _, r := range m.policy.Rules {
		if m.matchSingleRule(r, info) {
			return true
		}
	}
	return false
}

// matchSingleRule checks if the geo info matches a single rule
func (m *Matcher) matchSingleRule(r rule.GeoRule, info GeoInfo) bool {
	// Country must match first
	if !matchCountry(r.Country, info.Country) {
		return false
	}

	// If only country is specified, match
	if len(r.Provinces) == 0 && len(r.Cities) == 0 && len(r.Adcodes) == 0 {
		return true
	}

	// Check adcode match (highest priority)
	if len(r.Adcodes) > 0 && info.Adcode != "" {
		for _, code := range r.Adcodes {
			if normalizeAdcode(code) == normalizeAdcode(info.Adcode) {
				return true
			}
		}
	}

	// Check city match with province context
	if len(r.Cities) > 0 && info.City != "" {
		for _, city := range r.Cities {
			if normalizeString(city) == normalizeString(info.City) {
				// If provinces are also specified, must match both
				if len(r.Provinces) > 0 {
					for _, prov := range r.Provinces {
						if normalizeString(prov) == normalizeString(info.Province) {
							return true
						}
					}
					continue
				}
				return true
			}
		}
	}

	// Check province match
	if len(r.Provinces) > 0 && info.Province != "" {
		for _, prov := range r.Provinces {
			if normalizeString(prov) == normalizeString(info.Province) {
				// If no cities specified, match province only
				if len(r.Cities) == 0 {
					return true
				}
			}
		}
	}

	return false
}

// matchCountry compares country codes (case-insensitive)
func matchCountry(ruleCountry, infoCountry string) bool {
	if ruleCountry == "" {
		return true
	}
	return strings.EqualFold(ruleCountry, infoCountry)
}

// normalizeString normalizes a string for comparison
func normalizeString(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// normalizeAdcode normalizes an administrative code for comparison
func normalizeAdcode(s string) string {
	return strings.TrimSpace(s)
}
