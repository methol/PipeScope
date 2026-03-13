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
}

type AdminConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

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

	return &cfg, nil
}
