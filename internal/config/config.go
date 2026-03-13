package config

import (
	"fmt"
	"os"

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
	ID      string `yaml:"id"`
	Listen  string `yaml:"listen"`
	Forward string `yaml:"forward"`
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
}

func (t *TimeoutsConfig) UnmarshalYAML(value *yaml.Node) error {
	// Accept empty/missing map
	if value == nil || value.Kind == 0 {
		return nil
	}
	// Handle alias nodes and non-mapping nodes by decoding into a plain struct
	// and marking both fields as set (since the struct was populated).
	if value.Kind != yaml.MappingNode {
		type plain TimeoutsConfig
		if err := value.Decode((*plain)(t)); err != nil {
			return err
		}
		t.dialSet = true
		t.idleSet = true
		return nil
	}
	for i := 0; i < len(value.Content)-1; i += 2 {
		k := value.Content[i]
		v := value.Content[i+1]
		switch k.Value {
		case "dial_ms":
			t.dialSet = true
			if err := v.Decode(&t.DialMS); err != nil {
				return fmt.Errorf("invalid dial_ms: %w", err)
			}
		case "idle_ms":
			t.idleSet = true
			if err := v.Decode(&t.IdleMS); err != nil {
				return fmt.Errorf("invalid idle_ms: %w", err)
			}
		}
	}
	return nil
}

type AdminConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
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

	return &cfg, nil
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
	// If timeout fields are omitted, apply defaults. If explicitly set to 0, treat as 'disable timeout'.
	if !cfg.Timeouts.dialSet {
		cfg.Timeouts.DialMS = DefaultDialTimeoutMS
	}
	if !cfg.Timeouts.idleSet {
		cfg.Timeouts.IdleMS = DefaultIdleTimeoutMS
	}
	if cfg.Admin.Host == "" {
		cfg.Admin.Host = DefaultAdminHost
	}
	if cfg.Admin.Port <= 0 {
		cfg.Admin.Port = DefaultAdminPort
	}
}
