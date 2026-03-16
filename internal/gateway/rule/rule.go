package rule

// Rule defines a TCP proxy rule
type Rule struct {
	ID        string     `yaml:"id"`
	Listen    string     `yaml:"listen"`
	Forward   string     `yaml:"forward"`
	GeoPolicy *GeoPolicy `yaml:"geo_policy"`
}

// GeoPolicy defines geo-based traffic filtering policy
type GeoPolicy struct {
	Mode            string    `yaml:"mode"`             // "allow" | "deny"
	RequireAllowHit bool      `yaml:"require_allow_hit"` // in allow mode, require explicit hit to pass
	Rules           []GeoRule `yaml:"rules"`
}

// GeoRule defines a single geo filter rule
type GeoRule struct {
	Country   string   `yaml:"country"`   // ISO 3166-1 alpha-2 country code
	Provinces []string `yaml:"provinces"` // province names (optional)
	Cities    []string `yaml:"cities"`    // city names (optional)
	Adcodes   []string `yaml:"adcodes"`   // administrative division codes (optional)
}

