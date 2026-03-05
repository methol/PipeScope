package config

import "testing"

func TestLoadConfig(t *testing.T) {
	cfg, err := Load("testdata/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Admin.Host != "127.0.0.1" || cfg.Admin.Port != 9100 {
		t.Fatalf("admin config mismatch: %+v", cfg.Admin)
	}
}

