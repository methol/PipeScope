package config

import (
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
	SQLitePath      string `yaml:"sqlite_path"`
	IP2RegionXDB    string `yaml:"ip2region_xdb_path"`
	AreaCityCSVPath string `yaml:"areacity_csv_path"`
}

type ProxyRule struct {
	ID      string `yaml:"id"`
	Listen  string `yaml:"listen"`
	Forward string `yaml:"forward"`
}

type WriterConfig struct {
	QueueSize     int `yaml:"queue_size"`
	BatchSize     int `yaml:"batch_size"`
	FlushInterval int `yaml:"flush_interval_ms"`
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

	return &cfg, nil
}

