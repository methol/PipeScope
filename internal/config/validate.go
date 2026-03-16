package config

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	msgs := make([]string, len(e))
	for i, err := range e {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "; ")
}

// ValidModes are the allowed geo policy modes
// Deprecated: no longer used, kept for reference
var ValidModes = map[string]bool{
	"allow": true,
	"deny":  true,
}

// isoAlpha2Regex matches ISO 3166-1 alpha-2 country codes (2 uppercase letters)
var isoAlpha2Regex = regexp.MustCompile(`^[A-Z]{2}$`)

// adcodeRegex matches 6-digit administrative division codes
var adcodeRegex = regexp.MustCompile(`^\d{6}$`)

// Validate validates the entire configuration
func (c *Config) Validate() ValidationErrors {
	var errs ValidationErrors

	// Validate proxy rules
	for i, rule := range c.ProxyRules {
		ruleErrs := rule.Validate()
		for _, err := range ruleErrs {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("proxy_rules[%d].%s", i, err.Field),
				Message: err.Message,
			})
		}
	}

	return errs
}

// Validate validates a ProxyRule
func (r *ProxyRule) Validate() ValidationErrors {
	var errs ValidationErrors

	if r.ID == "" {
		errs = append(errs, ValidationError{
			Field:   "id",
			Message: "id is required",
		})
	}

	if r.Listen == "" {
		errs = append(errs, ValidationError{
			Field:   "listen",
			Message: "listen address is required",
		})
	}

	if r.Forward == "" {
		errs = append(errs, ValidationError{
			Field:   "forward",
			Message: "forward address is required",
		})
	}

	if r.GeoPolicy != nil {
		policyErrs := r.GeoPolicy.Validate()
		errs = append(errs, policyErrs...)
	}

	return errs
}

// Validate validates a GeoPolicy
func (p *GeoPolicy) Validate() ValidationErrors {
	var errs ValidationErrors

	// Validate allow rules
	for i, rule := range p.Allow {
		ruleErrs := rule.Validate()
		for _, err := range ruleErrs {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("allow[%d].%s", i, err.Field),
				Message: err.Message,
			})
		}
	}

	// Validate deny rules
	for i, rule := range p.Deny {
		ruleErrs := rule.Validate()
		for _, err := range ruleErrs {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("deny[%d].%s", i, err.Field),
				Message: err.Message,
			})
		}
	}

	return errs
}

// Validate validates a GeoRule
func (r *GeoRule) Validate() ValidationErrors {
	var errs ValidationErrors

	// Country is required for each rule
	if r.Country == "" {
		errs = append(errs, ValidationError{
			Field:   "country",
			Message: "country is required for geo rule",
		})
	} else if !ValidateCountryCode(r.Country) {
		errs = append(errs, ValidationError{
			Field:   "country",
			Message: fmt.Sprintf("invalid country code %q, must be ISO 3166-1 alpha-2 format (2 uppercase letters)", r.Country),
		})
	}

	// Validate adcodes
	for j, code := range r.Adcodes {
		if !ValidateAdcode(code) {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("adcodes[%d]", j),
				Message: fmt.Sprintf("invalid adcode %q, must be 6 digits", code),
			})
		}
	}

	// Validate provinces are not empty strings
	for j, prov := range r.Provinces {
		if strings.TrimSpace(prov) == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("provinces[%d]", j),
				Message: "province cannot be empty",
			})
		}
	}

	// Validate cities are not empty strings
	for j, city := range r.Cities {
		if strings.TrimSpace(city) == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("cities[%d]", j),
				Message: "city cannot be empty",
			})
		}
	}

	return errs
}

// ValidateCountryCode validates ISO 3166-1 alpha-2 country code format
// Note: This only validates the format (2 uppercase letters), not whether
// the code is an officially assigned country code in ISO 3166-1.
func ValidateCountryCode(code string) bool {
	return isoAlpha2Regex.MatchString(code)
}

// ValidateAdcode validates Chinese administrative division code format (6 digits)
func ValidateAdcode(code string) bool {
	return adcodeRegex.MatchString(code)
}

// NormalizeCountryCode normalizes a country code to uppercase
func NormalizeCountryCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}
