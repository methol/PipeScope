package config

import (
	"os"
	"testing"
)

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

func TestLoadConfigAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.yaml"
	content := []byte("data:\n  sqlite_path: ./data/test.db\nproxy_rules: []\nwriter: {}\ntimeouts: {}\nadmin: {}\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Writer.QueueSize != 1024 {
		t.Fatalf("QueueSize=%d want=1024", cfg.Writer.QueueSize)
	}
	if cfg.Writer.BatchSize != 200 {
		t.Fatalf("BatchSize=%d want=200", cfg.Writer.BatchSize)
	}
	if cfg.Writer.FlushInterval != 1000 {
		t.Fatalf("FlushInterval=%d want=1000", cfg.Writer.FlushInterval)
	}
	if cfg.Writer.FullQueuePolicy != "drop" {
		t.Fatalf("FullQueuePolicy=%q want=drop", cfg.Writer.FullQueuePolicy)
	}
	if cfg.Writer.SampleRate != 0.1 {
		t.Fatalf("SampleRate=%f want=0.1", cfg.Writer.SampleRate)
	}
	if cfg.Timeouts.DialMS != 1500 {
		t.Fatalf("DialMS=%d want=1500", cfg.Timeouts.DialMS)
	}
	if cfg.Timeouts.IdleMS != 60000 {
		t.Fatalf("IdleMS=%d want=60000", cfg.Timeouts.IdleMS)
	}
	if cfg.Admin.Host != "127.0.0.1" {
		t.Fatalf("Host=%q want=127.0.0.1", cfg.Admin.Host)
	}
	if cfg.Admin.Port != 9100 {
		t.Fatalf("Port=%d want=9100", cfg.Admin.Port)
	}
}
