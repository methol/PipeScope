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
		{"JP", true},
		{"cn", false}, // lowercase not allowed
		{"Cn", false}, // mixed case not allowed
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
			name: "valid allow policy",
			policy: GeoPolicy{
				Mode: "allow",
				Rules: []GeoRule{
					{Country: "CN"},
					{Country: "US", Provinces: []string{"California"}},
				},
			},
			expectError: false,
		},
		{
			name: "valid deny policy",
			policy: GeoPolicy{
				Mode: "deny",
				Rules: []GeoRule{
					{Country: "CN", Provinces: []string{"福建"}},
				},
			},
			expectError: false,
		},
		{
			name: "valid policy with adcodes",
			policy: GeoPolicy{
				Mode:            "allow",
				RequireAllowHit: true,
				Rules: []GeoRule{
					{Country: "CN", Adcodes: []string{"110000", "310000"}},
				},
			},
			expectError: false,
		},
		{
			name: "invalid mode",
			policy: GeoPolicy{
				Mode: "invalid",
				Rules: []GeoRule{
					{Country: "CN"},
				},
			},
			expectError: true,
			errorField:  "mode",
		},
		{
			name: "empty mode",
			policy: GeoPolicy{
				Mode: "",
				Rules: []GeoRule{
					{Country: "CN"},
				},
			},
			expectError: true,
			errorField:  "mode",
		},
		{
			name: "require_allow_hit with deny mode",
			policy: GeoPolicy{
				Mode:            "deny",
				RequireAllowHit: true,
				Rules: []GeoRule{
					{Country: "CN"},
				},
			},
			expectError: true,
			errorField:  "require_allow_hit",
		},
		{
			name: "missing country in rule",
			policy: GeoPolicy{
				Mode: "allow",
				Rules: []GeoRule{
					{Provinces: []string{"北京"}},
				},
			},
			expectError: true,
			errorField:  "rules[0].country",
		},
		{
			name: "invalid country code format",
			policy: GeoPolicy{
				Mode: "allow",
				Rules: []GeoRule{
					{Country: "china"}, // should be CN
				},
			},
			expectError: true,
			errorField:  "rules[0].country",
		},
		{
			name: "lowercase country code should fail validation",
			policy: GeoPolicy{
				Mode: "allow",
				Rules: []GeoRule{
					{Country: "cn"}, // lowercase not valid ISO alpha-2
				},
			},
			expectError: true,
			errorField:  "rules[0].country",
		},
		{
			name: "invalid adcode format",
			policy: GeoPolicy{
				Mode: "allow",
				Rules: []GeoRule{
					{Country: "CN", Adcodes: []string{"11000"}}, // 5 digits
				},
			},
			expectError: true,
			errorField:  "rules[0].adcodes[0]",
		},
		{
			name: "adcode with letters",
			policy: GeoPolicy{
				Mode: "allow",
				Rules: []GeoRule{
					{Country: "CN", Adcodes: []string{"11000a"}},
				},
			},
			expectError: true,
			errorField:  "rules[0].adcodes[0]",
		},
		{
			name: "empty province string",
			policy: GeoPolicy{
				Mode: "allow",
				Rules: []GeoRule{
					{Country: "CN", Provinces: []string{""}},
				},
			},
			expectError: true,
			errorField:  "rules[0].provinces[0]",
		},
		{
			name: "whitespace-only province",
			policy: GeoPolicy{
				Mode: "allow",
				Rules: []GeoRule{
					{Country: "CN", Provinces: []string{"   "}},
				},
			},
			expectError: true,
			errorField:  "rules[0].provinces[0]",
		},
		{
			name: "empty city string",
			policy: GeoPolicy{
				Mode: "allow",
				Rules: []GeoRule{
					{Country: "CN", Cities: []string{""}},
				},
			},
			expectError: true,
			errorField:  "rules[0].cities[0]",
		},
		{
			name: "empty rules list is valid",
			policy: GeoPolicy{
				Mode:  "allow",
				Rules: []GeoRule{},
			},
			expectError: false,
		},
		{
			name: "nil rules is valid",
			policy: GeoPolicy{
				Mode:  "allow",
				Rules: nil,
			},
			expectError: false,
		},
		{
			name: "multiple rules with one invalid",
			policy: GeoPolicy{
				Mode: "allow",
				Rules: []GeoRule{
					{Country: "CN"},
					{Country: "invalid"}, // invalid
					{Country: "US"},
				},
			},
			expectError: true,
			errorField:  "rules[1].country",
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
					Mode: "allow",
					Rules: []GeoRule{
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
			name: "invalid geo policy in rule",
			rule: ProxyRule{
				ID:      "test",
				Listen:  "0.0.0.0:8080",
				Forward: "127.0.0.1:8081",
				GeoPolicy: &GeoPolicy{
					Mode: "invalid",
					Rules: []GeoRule{
						{Country: "CN"},
					},
				},
			},
			expectError: true,
			errorFields: []string{"mode"},
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
	t.Run("valid config with geo policy", func(t *testing.T) {
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
      mode: "allow"
      require_allow_hit: true
      rules:
        - country: "CN"
        - country: "US"
          provinces: ["California"]
        - country: "CN"
          adcodes: ["110000", "310000"]
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
		if cfg.ProxyRules[0].GeoPolicy.Mode != "allow" {
			t.Errorf("expected mode 'allow', got %q", cfg.ProxyRules[0].GeoPolicy.Mode)
		}
		// Check country codes are normalized
		for _, rule := range cfg.ProxyRules[0].GeoPolicy.Rules {
			if rule.Country != "CN" && rule.Country != "US" {
				t.Errorf("unexpected country code: %q", rule.Country)
			}
		}
	})

	t.Run("country code normalization", func(t *testing.T) {
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
      mode: "ALLOW"
      rules:
        - country: "cn"
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
		// Mode should be normalized to lowercase
		if cfg.ProxyRules[0].GeoPolicy.Mode != "allow" {
			t.Errorf("expected mode 'allow', got %q", cfg.ProxyRules[0].GeoPolicy.Mode)
		}
		// Country codes should be normalized to uppercase
		for _, rule := range cfg.ProxyRules[0].GeoPolicy.Rules {
			if rule.Country != "CN" && rule.Country != "US" {
				t.Errorf("expected normalized country code, got %q", rule.Country)
			}
		}
	})

	t.Run("invalid mode rejected", func(t *testing.T) {
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
      mode: "whitelist"
      rules:
        - country: "CN"
writer: {}
timeouts: {}
admin: {}
`)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		_, err := Load(path)
		if err == nil {
			t.Fatal("expected error for invalid mode")
		}
	})

	t.Run("invalid country code rejected", func(t *testing.T) {
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
      mode: "allow"
      rules:
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
      mode: "allow"
      rules:
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

	t.Run("require_allow_hit with deny mode rejected", func(t *testing.T) {
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
      mode: "deny"
      require_allow_hit: true
      rules:
        - country: "CN"
writer: {}
timeouts: {}
admin: {}
`)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		_, err := Load(path)
		if err == nil {
			t.Fatal("expected error for require_allow_hit with deny mode")
		}
	})

	t.Run("empty geo policy rules is valid", func(t *testing.T) {
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
      mode: "allow"
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
		if cfg.ProxyRules[0].GeoPolicy.Mode != "allow" {
			t.Errorf("expected mode 'allow', got %q", cfg.ProxyRules[0].GeoPolicy.Mode)
		}
	})

	t.Run("missing country in rule rejected", func(t *testing.T) {
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
      mode: "allow"
      rules:
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
}

func TestValidationErrorsFormatting(t *testing.T) {
	t.Run("single error", func(t *testing.T) {
		errs := ValidationErrors{
			{Field: "mode", Message: "invalid mode"},
		}
		expected := "mode: invalid mode"
		if errs.Error() != expected {
			t.Errorf("Error() = %q, want %q", errs.Error(), expected)
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		errs := ValidationErrors{
			{Field: "mode", Message: "invalid mode"},
			{Field: "country", Message: "invalid country code"},
		}
		expected := "mode: invalid mode; country: invalid country code"
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
