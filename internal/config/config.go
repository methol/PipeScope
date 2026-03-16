package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Data       DataConfig     `yaml:"data"`
	ProxyRules []ProxyRule    `yaml:"proxy_rules"`
	Writer     WriterConfig   `yaml:"writer"`
	Timeouts   TimeoutsConfig `yaml:"timeouts"`
	Admin      AdminConfig    `yaml:"admin"`
}

type DataConfig struct {
	SQLitePath            string `yaml:"sqlite_path"`
	IP2RegionXDB          string `yaml:"ip2region_xdb_path"`
	IP2RegionV6XDB        string `yaml:"ip2region_v6_xdb_path"`
	IP2RegionCachePolicy  string `yaml:"ip2region_cache_policy"`
	IP2RegionSearcherPool int    `yaml:"ip2region_searchers"`
	AreaCityCSVPath       string `yaml:"areacity_csv_path"`
	AreaCityAPIBaseURL    string `yaml:"areacity_api_base_url"`
	AreaCityAPIInstance   int    `yaml:"areacity_api_instance"`
}

type ProxyRule struct {
	ID        string     `yaml:"id"`
	Listen    string     `yaml:"listen"`
	Forward   string     `yaml:"forward"`
	GeoPolicy *GeoPolicy `yaml:"geo_policy"`
}

// GeoPolicy defines geo-based traffic filtering policy
type GeoPolicy struct {
	Mode           string    `yaml:"mode"`            // "allow" | "deny"
	RequireAllowHit bool     `yaml:"require_allow_hit"` // in allow mode, require explicit hit to pass
	Rules          []GeoRule `yaml:"rules"`
}

// GeoRule defines a single geo filter rule
type GeoRule struct {
	Country   string   `yaml:"country"`   // ISO 3166-1 alpha-2 country code
	Provinces []string `yaml:"provinces"` // province names (optional)
	Cities    []string `yaml:"cities"`    // city names (optional)
	Adcodes   []string `yaml:"adcodes"`   // administrative division codes (optional)
}

type WriterConfig struct {
	QueueSize       int     `yaml:"queue_size"`
	BatchSize       int     `yaml:"batch_size"`
	FlushInterval   int     `yaml:"flush_interval_ms"`
	FullQueuePolicy string  `yaml:"full_queue_policy"`
	SampleRate      float64 `yaml:"sample_rate"`
}

type TimeoutsConfig struct {
	DialMS int `yaml:"dial_ms"`
	IdleMS int `yaml:"idle_ms"`

	// presence flags: used to distinguish omitted fields vs explicitly set to 0
	dialSet bool `yaml:"-"`
	idleSet bool `yaml:"-"`

	// sectionSet indicates whether the `timeouts` section is present in YAML at all.
	// If absent, we keep timeouts disabled (0) for backward compatibility.
	sectionSet bool `yaml:"-"`
}

func (t *TimeoutsConfig) UnmarshalYAML(value *yaml.Node) error {
	// Accept empty/missing
	if value == nil || value.Kind == 0 || value.Tag == "!!null" {
		return nil
	}

	// Section exists in YAML (not null); enable defaults for omitted fields.
	t.sectionSet = true
	// Follow YAML alias nodes so we preserve per-field presence and allow merge keys.
	if value.Kind == yaml.AliasNode {
		if value.Alias == nil {
			return fmt.Errorf("invalid timeouts alias")
		}
		return t.UnmarshalYAML(value.Alias)
	}
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("invalid timeouts section: expected mapping")
	}

	// Let yaml.v3 resolve merge keys (<<: *anchor) while decoding.
	var aux struct {
		DialMS *int `yaml:"dial_ms"`
		IdleMS *int `yaml:"idle_ms"`
	}
	if err := value.Decode(&aux); err != nil {
		return err
	}
	if aux.DialMS != nil {
		t.DialMS = *aux.DialMS
		t.dialSet = true
	}
	if aux.IdleMS != nil {
		t.IdleMS = *aux.IdleMS
		t.idleSet = true
	}
	return nil
}

type AdminConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`

	// presence flags: used to distinguish omitted fields vs explicitly set to 0
	hostSet bool `yaml:"-"`
	portSet bool `yaml:"-"`
}

func (a *AdminConfig) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Kind == 0 || value.Tag == "!!null" {
		return nil
	}
	if value.Kind == yaml.AliasNode {
		if value.Alias == nil {
			return fmt.Errorf("invalid admin alias")
		}
		return a.UnmarshalYAML(value.Alias)
	}
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("invalid admin section: expected mapping")
	}

	var aux struct {
		Host *string `yaml:"host"`
		Port *int    `yaml:"port"`
	}
	if err := value.Decode(&aux); err != nil {
		return err
	}
	if aux.Host != nil {
		a.Host = *aux.Host
		a.hostSet = true
	}
	if aux.Port != nil {
		a.Port = *aux.Port
		a.portSet = true
	}
	return nil
}

const (
	DefaultWriterQueueSize     = 1024
	DefaultWriterBatchSize     = 200
	DefaultWriterFlushInterval = 1000
	DefaultWriterFullPolicy    = "drop"
	DefaultWriterSampleRate    = 0.1
	DefaultDialTimeoutMS       = 1500
	DefaultIdleTimeoutMS       = 60000
	DefaultAdminHost           = "127.0.0.1"
	DefaultAdminPort           = 9100
)

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if cfg.Data.IP2RegionXDB != "" || cfg.Data.IP2RegionV6XDB != "" || cfg.Data.AreaCityCSVPath != "" || cfg.Data.AreaCityAPIBaseURL != "" || cfg.Data.AreaCityAPIInstance != 0 {
		return nil, fmt.Errorf("legacy external geo/ip config is no longer supported; use embedded data assets")
	}
	applyDefaults(&cfg)
	normalizeGeoPolicy(&cfg)

	// Validate configuration
	if errs := cfg.Validate(); len(errs) > 0 {
		return nil, errs
	}

	return &cfg, nil
}

// normalizeGeoPolicy normalizes geo policy config values
func normalizeGeoPolicy(cfg *Config) {
	for i := range cfg.ProxyRules {
		if cfg.ProxyRules[i].GeoPolicy != nil {
			policy := cfg.ProxyRules[i].GeoPolicy
			policy.Mode = strings.ToLower(strings.TrimSpace(policy.Mode))
			for j := range policy.Rules {
				policy.Rules[j].Country = NormalizeCountryCode(policy.Rules[j].Country)
			}
		}
	}
}

func applyDefaults(cfg *Config) {
	if cfg.Writer.QueueSize <= 0 {
		cfg.Writer.QueueSize = DefaultWriterQueueSize
	}
	if cfg.Writer.BatchSize <= 0 {
		cfg.Writer.BatchSize = DefaultWriterBatchSize
	}
	if cfg.Writer.FlushInterval <= 0 {
		cfg.Writer.FlushInterval = DefaultWriterFlushInterval
	}
	if cfg.Writer.FullQueuePolicy == "" {
		cfg.Writer.FullQueuePolicy = DefaultWriterFullPolicy
	}
	if cfg.Writer.SampleRate <= 0 || cfg.Writer.SampleRate > 1 {
		cfg.Writer.SampleRate = DefaultWriterSampleRate
	}
	// Backward-compat: if `timeouts` section is omitted entirely, keep timeouts disabled (0).
	// If the section exists, apply defaults only for omitted fields; explicit 0 means "disable".
	if cfg.Timeouts.sectionSet {
		if !cfg.Timeouts.dialSet {
			cfg.Timeouts.DialMS = DefaultDialTimeoutMS
		}
		if !cfg.Timeouts.idleSet {
			cfg.Timeouts.IdleMS = DefaultIdleTimeoutMS
		}
	}
	// Preserve explicit admin.host: "" (bind all interfaces). Default only when omitted.
	if !cfg.Admin.hostSet {
		cfg.Admin.Host = DefaultAdminHost
	}
	// Preserve explicit admin.port: 0 (ephemeral port). Default only when omitted.
	if !cfg.Admin.portSet {
		cfg.Admin.Port = DefaultAdminPort
	}
}
