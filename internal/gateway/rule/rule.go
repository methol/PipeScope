package rule

// Rule defines a TCP proxy rule
type Rule struct {
	ID        string     `yaml:"id"`
	Listen    string     `yaml:"listen"`
	Forward   string     `yaml:"forward"`
	GeoPolicy *GeoPolicy `yaml:"geo_policy"`
}

// GeoPolicy defines geo-based traffic filtering policy
// Matching order: deny rules first, then allow rules, finally fallback to require_allow_hit
type GeoPolicy struct {
	RequireAllowHit bool      `yaml:"require_allow_hit"` // when no rules match: true=deny, false=allow
	Allow           []GeoRule `yaml:"allow"`             // allow rules (checked after deny)
	Deny            []GeoRule `yaml:"deny"`              // deny rules (checked first)
}

// GeoRule defines a single geo filter rule
type GeoRule struct {
	Country   string   `yaml:"country"`   // ISO 3166-1 alpha-2 country code
	Provinces []string `yaml:"provinces"` // province names (optional)
	Cities    []string `yaml:"cities"`    // city names (optional)
	Adcodes   []string `yaml:"adcodes"`   // administrative division codes (optional)
}

