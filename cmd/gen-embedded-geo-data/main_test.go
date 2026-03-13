package main

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAreaCitySeedFiltersToProvinceAndCity(t *testing.T) {
	src := strings.NewReader("id,deep,name,ext_path,geo\n11,0,北京市,北京市,116.4 39.9\n1101,1,北京市,北京市 北京市,116.4 39.9\n110101,2,东城区,北京市 北京市 东城区,116.4 39.9\n")
	var out bytes.Buffer
	rows, err := writeAreaCitySeed(src, &out)
	if err != nil {
		t.Fatalf("write area city seed: %v", err)
	}
	if rows != 2 {
		t.Fatalf("expected 2 rows, got %d", rows)
	}
	r := csv.NewReader(bytes.NewReader(out.Bytes()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read generated csv: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected header + 2 rows, got %d", len(records))
	}
	if records[1][0] != "11" || records[2][0] != "1101" {
		t.Fatalf("unexpected rows: %+v", records)
	}
}

func TestBuildManifestUsesAssetHashes(t *testing.T) {
	dir := t.TempDir()
	xdbPath := filepath.Join(dir, "ip2region_v4.xdb.gz")
	seedPath := filepath.Join(dir, "areacity_seed.csv.gz")
	if err := os.WriteFile(xdbPath, []byte("xdb-content"), 0o644); err != nil {
		t.Fatalf("write xdb asset: %v", err)
	}
	if err := os.WriteFile(seedPath, []byte("seed-content"), 0o644); err != nil {
		t.Fatalf("write seed asset: %v", err)
	}
	manifest, err := buildManifest(xdbPath, seedPath, "2023.240319.250114", 406)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	if !strings.HasPrefix(manifest.IP2RegionVersion, "sha256-") {
		t.Fatalf("unexpected ip2region version: %s", manifest.IP2RegionVersion)
	}
	if !strings.HasPrefix(manifest.AreaCityVersion, "2023.240319.250114+") {
		t.Fatalf("unexpected areacity version: %s", manifest.AreaCityVersion)
	}
	if manifest.GeneratedRows != 406 {
		t.Fatalf("unexpected rows: %d", manifest.GeneratedRows)
	}
}
