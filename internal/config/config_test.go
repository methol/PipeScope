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
	if cfg.Writer.FullQueuePolicy != "drop" {
		t.Fatalf("writer full_queue_policy mismatch: %s", cfg.Writer.FullQueuePolicy)
	}
	if cfg.Writer.SampleRate != 0.1 {
		t.Fatalf("writer sample_rate mismatch: %f", cfg.Writer.SampleRate)
	}
}

func TestLoadConfigRejectsLegacyGeoFields(t *testing.T) {
	_, err := Load("testdata/config_legacy_geo.yaml")
	if err == nil {
		t.Fatalf("expected legacy geo config to be rejected")
	}
}
