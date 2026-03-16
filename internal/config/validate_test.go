package config

import (
	"os"
	"testing"
)

func TestValidateCountryCode(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		{"CN", true},
		{"US", true},
		{"GB", true},
		{"jp", false}, // lowercase not allowed
		{"Jp", false}, // mixed case not allowed
		{"USA", false}, // 3 letters
		{"C", false},   // 1 letter
		{"", false},    // empty
		{"12", false},  // digits
		{"C1", false},  // letter + digit
		{"AA", true},   // valid format (even if not assigned)
		{"ZZ", true},   // valid format (even if not assigned)
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := ValidateCountryCode(tt.code)
			if result != tt.expected {
				t.Errorf("ValidateCountryCode(%q) = %v, want %v", tt.code, result, tt.expected)
			}
		})
	}
}

func TestValidateAdcode(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		{"110000", true},  // Beijing
		{"310000", true},  // Shanghai
		{"440300", true},  // Shenzhen
		{"000000", true},  // valid format
		{"123456", true},  // valid format
		{"11000", false},  // 5 digits
		{"1100000", false}, // 7 digits
		{"", false},        // empty
		{"abcdef", false},  // letters
		{"11000a", false},  // 5 digits + letter
		{" 110000", false}, // leading space
		{"110000 ", false}, // trailing space
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := ValidateAdcode(tt.code)
			if result != tt.expected {
				t.Errorf("ValidateAdcode(%q) = %v, want %v", tt.code, result, tt.expected)
			}
		})
	}
}

func TestNormalizeCountryCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"cn", "CN"},
		{"CN", "CN"},
		{"cN", "CN"},
		{"  us  ", "US"},
		{"us", "US"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeCountryCode(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeCountryCode(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGeoPolicyValidation(t *testing.T) {
	tests := []struct {
		name        string
		policy      GeoPolicy
		expectError bool
		errorField  string
	}{
		{
			name: "valid allow-only policy",
			policy: GeoPolicy{
				RequireAllowHit: true,
				Allow: []GeoRule{
					{Country: "CN"},
					{Country: "US", Provinces: []string{"California"}},
				},
			},
			expectError: false,
		},
		{
			name: "valid deny-only policy",
			policy: GeoPolicy{
				Deny: []GeoRule{
					{Country: "CN", Provinces: []string{"福建"}},
				},
			},
			expectError: false,
		},
		{
			name: "valid allow and deny combination",
			policy: GeoPolicy{
				RequireAllowHit: true,
				Allow: []GeoRule{
					{Country: "CN"},
				},
				Deny: []GeoRule{
					{Country: "CN", Provinces: []string{"福建"}},
				},
			},
			expectError: false,
		},
		{
			name: "valid policy with adcodes",
			policy: GeoPolicy{
				RequireAllowHit: true,
				Allow: []GeoRule{
					{Country: "CN", Adcodes: []string{"110000", "310000"}},
				},
			},
			expectError: false,
		},
		{
			name: "empty policy is valid",
			policy: GeoPolicy{
				RequireAllowHit: false,
			},
			expectError: false,
		},
		{
			name: "missing country in allow rule",
			policy: GeoPolicy{
				Allow: []GeoRule{
					{Provinces: []string{"北京"}},
				},
			},
			expectError: true,
			errorField:  "allow[0].country",
		},
		{
			name: "missing country in deny rule",
			policy: GeoPolicy{
				Deny: []GeoRule{
					{Provinces: []string{"福建"}},
				},
			},
			expectError: true,
			errorField:  "deny[0].country",
		},
		{
			name: "invalid country code format in allow",
			policy: GeoPolicy{
				Allow: []GeoRule{
					{Country: "china"}, // should be CN
				},
			},
			expectError: true,
			errorField:  "allow[0].country",
		},
		{
			name: "lowercase country code should fail validation",
			policy: GeoPolicy{
				Allow: []GeoRule{
					{Country: "cn"}, // lowercase not valid ISO alpha-2
				},
			},
			expectError: true,
			errorField:  "allow[0].country",
		},
		{
			name: "invalid adcode format in allow",
			policy: GeoPolicy{
				Allow: []GeoRule{
					{Country: "CN", Adcodes: []string{"11000"}}, // 5 digits
				},
			},
			expectError: true,
			errorField:  "allow[0].adcodes[0]",
		},
		{
			name: "invalid adcode format in deny",
			policy: GeoPolicy{
				Deny: []GeoRule{
					{Country: "CN", Adcodes: []string{"11000a"}}, // has letter
				},
			},
			expectError: true,
			errorField:  "deny[0].adcodes[0]",
		},
		{
			name: "empty province string in allow",
			policy: GeoPolicy{
				Allow: []GeoRule{
					{Country: "CN", Provinces: []string{""}},
				},
			},
			expectError: true,
			errorField:  "allow[0].provinces[0]",
		},
		{
			name: "whitespace-only province in deny",
			policy: GeoPolicy{
				Deny: []GeoRule{
					{Country: "CN", Provinces: []string{"   "}},
				},
			},
			expectError: true,
			errorField:  "deny[0].provinces[0]",
		},
		{
			name: "empty city string in allow",
			policy: GeoPolicy{
				Allow: []GeoRule{
					{Country: "CN", Cities: []string{""}},
				},
			},
			expectError: true,
			errorField:  "allow[0].cities[0]",
		},
		{
			name: "multiple allow rules with one invalid",
			policy: GeoPolicy{
				Allow: []GeoRule{
					{Country: "CN"},
					{Country: "invalid"}, // invalid
					{Country: "US"},
				},
			},
			expectError: true,
			errorField:  "allow[1].country",
		},
		{
			name: "multiple deny rules with one invalid adcode",
			policy: GeoPolicy{
				Deny: []GeoRule{
					{Country: "CN", Provinces: []string{"福建"}},
					{Country: "CN", Adcodes: []string{"12345"}}, // invalid 5-digit
				},
			},
			expectError: true,
			errorField:  "deny[1].adcodes[0]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.policy.Validate()
			hasError := len(errs) > 0

			if hasError != tt.expectError {
				t.Errorf("expected error = %v, got %v (errors: %v)", tt.expectError, hasError, errs)
				return
			}

			if tt.expectError && tt.errorField != "" {
				found := false
				for _, err := range errs {
					if err.Field == tt.errorField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error in field %q, got errors: %v", tt.errorField, errs)
				}
			}
		})
	}
}

func TestProxyRuleValidation(t *testing.T) {
	tests := []struct {
		name        string
		rule        ProxyRule
		expectError bool
		errorFields []string
	}{
		{
			name: "valid rule without geo policy",
			rule: ProxyRule{
				ID:      "test",
				Listen:  "0.0.0.0:8080",
				Forward: "127.0.0.1:8081",
			},
			expectError: false,
		},
		{
			name: "valid rule with geo policy",
			rule: ProxyRule{
				ID:      "test",
				Listen:  "0.0.0.0:8080",
				Forward: "127.0.0.1:8081",
				GeoPolicy: &GeoPolicy{
					RequireAllowHit: true,
					Allow: []GeoRule{
						{Country: "CN"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing id",
			rule: ProxyRule{
				Listen:  "0.0.0.0:8080",
				Forward: "127.0.0.1:8081",
			},
			expectError: true,
			errorFields: []string{"id"},
		},
		{
			name: "missing listen",
			rule: ProxyRule{
				ID:      "test",
				Forward: "127.0.0.1:8081",
			},
			expectError: true,
			errorFields: []string{"listen"},
		},
		{
			name: "missing forward",
			rule: ProxyRule{
				ID:     "test",
				Listen: "0.0.0.0:8080",
			},
			expectError: true,
			errorFields: []string{"forward"},
		},
		{
			name: "invalid geo policy in rule - missing country in allow",
			rule: ProxyRule{
				ID:      "test",
				Listen:  "0.0.0.0:8080",
				Forward: "127.0.0.1:8081",
				GeoPolicy: &GeoPolicy{
					Allow: []GeoRule{
						{Provinces: []string{"北京"}}, // missing country
					},
				},
			},
			expectError: true,
			errorFields: []string{"allow[0].country"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.rule.Validate()
			hasError := len(errs) > 0

			if hasError != tt.expectError {
				t.Errorf("expected error = %v, got %v (errors: %v)", tt.expectError, hasError, errs)
				return
			}

			if tt.expectError && len(tt.errorFields) > 0 {
				for _, field := range tt.errorFields {
					found := false
					for _, err := range errs {
						if err.Field == field {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error in field %q, got errors: %v", field, errs)
					}
				}
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	t.Run("valid config with allow and deny geo policy", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/config.yaml"
		content := []byte(`
data:
  sqlite_path: ./data/test.db
proxy_rules:
  - id: "test"
    listen: "0.0.0.0:10001"
    forward: "127.0.0.1:10002"
    geo_policy:
      require_allow_hit: true
      allow:
        - country: "CN"
        - country: "US"
          provinces: ["California"]
        - country: "CN"
          adcodes: ["110000", "310000"]
      deny:
        - country: "CN"
          provinces: ["福建"]
writer: {}
timeouts: {}
admin: {}
`)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.ProxyRules) != 1 {
			t.Fatalf("expected 1 proxy rule, got %d", len(cfg.ProxyRules))
		}
		if cfg.ProxyRules[0].GeoPolicy == nil {
			t.Fatal("expected geo policy to be set")
		}
		if cfg.ProxyRules[0].GeoPolicy.RequireAllowHit != true {
			t.Errorf("expected RequireAllowHit=true")
		}
		if len(cfg.ProxyRules[0].GeoPolicy.Allow) != 3 {
			t.Errorf("expected 3 allow rules, got %d", len(cfg.ProxyRules[0].GeoPolicy.Allow))
		}
		if len(cfg.ProxyRules[0].GeoPolicy.Deny) != 1 {
			t.Errorf("expected 1 deny rule, got %d", len(cfg.ProxyRules[0].GeoPolicy.Deny))
		}
		// Check country codes are normalized
		for _, rule := range cfg.ProxyRules[0].GeoPolicy.Allow {
			if rule.Country != "CN" && rule.Country != "US" {
				t.Errorf("unexpected country code: %q", rule.Country)
			}
		}
	})

	t.Run("country code normalization in allow and deny", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/config.yaml"
		content := []byte(`
data:
  sqlite_path: ./data/test.db
proxy_rules:
  - id: "test"
    listen: "0.0.0.0:10001"
    forward: "127.0.0.1:10002"
    geo_policy:
      allow:
        - country: "cn"
      deny:
        - country: "us"
writer: {}
timeouts: {}
admin: {}
`)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Country codes should be normalized to uppercase
		for _, rule := range cfg.ProxyRules[0].GeoPolicy.Allow {
			if rule.Country != "CN" {
				t.Errorf("expected normalized country code CN, got %q", rule.Country)
			}
		}
		for _, rule := range cfg.ProxyRules[0].GeoPolicy.Deny {
			if rule.Country != "US" {
				t.Errorf("expected normalized country code US, got %q", rule.Country)
			}
		}
	})

	t.Run("invalid country code rejected in allow", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/config.yaml"
		content := []byte(`
data:
  sqlite_path: ./data/test.db
proxy_rules:
  - id: "test"
    listen: "0.0.0.0:10001"
    forward: "127.0.0.1:10002"
    geo_policy:
      allow:
        - country: "CHINA"
writer: {}
timeouts: {}
admin: {}
`)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		_, err := Load(path)
		if err == nil {
			t.Fatal("expected error for invalid country code")
		}
	})

	t.Run("invalid country code rejected in deny", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/config.yaml"
		content := []byte(`
data:
  sqlite_path: ./data/test.db
proxy_rules:
  - id: "test"
    listen: "0.0.0.0:10001"
    forward: "127.0.0.1:10002"
    geo_policy:
      deny:
        - country: "invalid"
writer: {}
timeouts: {}
admin: {}
`)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		_, err := Load(path)
		if err == nil {
			t.Fatal("expected error for invalid country code")
		}
	})

	t.Run("invalid adcode rejected", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/config.yaml"
		content := []byte(`
data:
  sqlite_path: ./data/test.db
proxy_rules:
  - id: "test"
    listen: "0.0.0.0:10001"
    forward: "127.0.0.1:10002"
    geo_policy:
      allow:
        - country: "CN"
          adcodes: ["11000"]
writer: {}
timeouts: {}
admin: {}
`)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		_, err := Load(path)
		if err == nil {
			t.Fatal("expected error for invalid adcode")
		}
	})

	t.Run("empty geo policy is valid", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/config.yaml"
		content := []byte(`
data:
  sqlite_path: ./data/test.db
proxy_rules:
  - id: "test"
    listen: "0.0.0.0:10001"
    forward: "127.0.0.1:10002"
    geo_policy:
      require_allow_hit: false
writer: {}
timeouts: {}
admin: {}
`)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.ProxyRules[0].GeoPolicy.RequireAllowHit != false {
			t.Errorf("expected RequireAllowHit=false")
		}
	})

	t.Run("missing country in allow rule rejected", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/config.yaml"
		content := []byte(`
data:
  sqlite_path: ./data/test.db
proxy_rules:
  - id: "test"
    listen: "0.0.0.0:10001"
    forward: "127.0.0.1:10002"
    geo_policy:
      allow:
        - provinces: ["北京"]
writer: {}
timeouts: {}
admin: {}
`)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		_, err := Load(path)
		if err == nil {
			t.Fatal("expected error for missing country in rule")
		}
	})

	t.Run("missing country in deny rule rejected", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/config.yaml"
		content := []byte(`
data:
  sqlite_path: ./data/test.db
proxy_rules:
  - id: "test"
    listen: "0.0.0.0:10001"
    forward: "127.0.0.1:10002"
    geo_policy:
      deny:
        - provinces: ["福建"]
writer: {}
timeouts: {}
admin: {}
`)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		_, err := Load(path)
		if err == nil {
			t.Fatal("expected error for missing country in deny rule")
		}
	})
}

func TestValidationErrorsFormatting(t *testing.T) {
	t.Run("single error", func(t *testing.T) {
		errs := ValidationErrors{
			{Field: "allow[0].country", Message: "country is required"},
		}
		expected := "allow[0].country: country is required"
		if errs.Error() != expected {
			t.Errorf("Error() = %q, want %q", errs.Error(), expected)
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		errs := ValidationErrors{
			{Field: "allow[0].country", Message: "invalid country code"},
			{Field: "deny[0].country", Message: "invalid country code"},
		}
		expected := "allow[0].country: invalid country code; deny[0].country: invalid country code"
		if errs.Error() != expected {
			t.Errorf("Error() = %q, want %q", errs.Error(), expected)
		}
	})

	t.Run("empty field", func(t *testing.T) {
		errs := ValidationErrors{
			{Field: "", Message: "general error"},
		}
		expected := "general error"
		if errs.Error() != expected {
			t.Errorf("Error() = %q, want %q", errs.Error(), expected)
		}
	})
}
