package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

type featureCollectionFixture struct {
	Type     string `json:"type"`
	Features []struct {
		Type       string `json:"type"`
		Properties struct {
			Adcode   string    `json:"adcode"`
			Name     string    `json:"name"`
			Province string    `json:"province"`
			City     string    `json:"city"`
			District string    `json:"district"`
			CP       []float64 `json:"cp"`
		} `json:"properties"`
		Geometry struct {
			Type        string          `json:"type"`
			Coordinates [][][][]float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"features"`
}

func TestWriteFeatureCollectionFiltersMainlandCountiesAndBuildsMultiPolygon(t *testing.T) {
	src := strings.NewReader(`id,pid,deep,name,ext_path,geo,polygon
110102,1101,2,西城区,北京市 北京市 西城区,116.36585 39.9126,"116.1 39.9,116.2 39.9,116.2 40.0,116.1 40.0;116.3 39.9,116.4 39.9,116.4 40.0,116.3 40.0"
4403,44,1,深圳市,广东省 深圳市,114.057939 22.543527,"EMPTY"
810001,81,2,中西区,香港特别行政区 香港特别行政区 中西区,114.154334 22.281931,"114.1 22.2,114.2 22.2,114.2 22.3,114.1 22.3"
110105,1101,2,朝阳区,北京市 北京市 朝阳区,116.44355 39.9219,"EMPTY"
`)

	var out bytes.Buffer
	stats, err := writeFeatureCollection(src, &out, options{SimplifyEpsilon: 0, Precision: 6})
	if err != nil {
		t.Fatalf("write feature collection: %v", err)
	}
	if stats.TotalRows != 4 {
		t.Fatalf("expected 4 total rows, got %d", stats.TotalRows)
	}
	if stats.Features != 1 {
		t.Fatalf("expected 1 feature, got %d", stats.Features)
	}
	if stats.FilteredRows != 2 {
		t.Fatalf("expected 2 filtered rows, got %d", stats.FilteredRows)
	}
	if stats.EmptyRows != 1 {
		t.Fatalf("expected 1 empty row, got %d", stats.EmptyRows)
	}

	var fc featureCollectionFixture
	if err := json.Unmarshal(out.Bytes(), &fc); err != nil {
		t.Fatalf("unmarshal feature collection: %v", err)
	}
	if fc.Type != "FeatureCollection" {
		t.Fatalf("unexpected feature collection type: %s", fc.Type)
	}
	if len(fc.Features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(fc.Features))
	}
	feature := fc.Features[0]
	if feature.Geometry.Type != "MultiPolygon" {
		t.Fatalf("unexpected geometry type: %s", feature.Geometry.Type)
	}
	if got := len(feature.Geometry.Coordinates); got != 2 {
		t.Fatalf("expected 2 polygons, got %d", got)
	}
	if feature.Properties.Adcode != "110102" {
		t.Fatalf("unexpected adcode: %s", feature.Properties.Adcode)
	}
	if feature.Properties.Province != "北京市" || feature.Properties.City != "北京市" || feature.Properties.District != "西城区" {
		t.Fatalf("unexpected properties: %+v", feature.Properties)
	}
	if got := len(feature.Properties.CP); got != 2 {
		t.Fatalf("expected cp with 2 numbers, got %d", got)
	}
	for i, polygon := range feature.Geometry.Coordinates {
		if len(polygon) != 1 {
			t.Fatalf("polygon %d expected 1 ring, got %d", i, len(polygon))
		}
		ring := polygon[0]
		if len(ring) < 4 {
			t.Fatalf("polygon %d ring too short: %d", i, len(ring))
		}
		first := ring[0]
		last := ring[len(ring)-1]
		if len(first) != 2 || len(last) != 2 || first[0] != last[0] || first[1] != last[1] {
			t.Fatalf("polygon %d ring is not closed: first=%v last=%v", i, first, last)
		}
	}
}

func TestWriteFeatureCollectionSkipsInvalidPolygonRowsAndCountsThem(t *testing.T) {
	src := strings.NewReader(`id,pid,deep,name,ext_path,geo,polygon
110102,1101,2,西城区,北京市 北京市 西城区,116.36585 39.9126,"broken"
110105,1101,2,朝阳区,北京市 北京市 朝阳区,116.44355 39.9219,"116.1 39.9,116.2 39.9,116.2 40.0,116.1 40.0"
`)

	var out bytes.Buffer
	stats, err := writeFeatureCollection(src, &out, options{SimplifyEpsilon: 0, Precision: 6})
	if err != nil {
		t.Fatalf("write feature collection: %v", err)
	}
	if stats.InvalidRows != 1 {
		t.Fatalf("expected 1 invalid row, got %d", stats.InvalidRows)
	}
	if stats.Features != 1 {
		t.Fatalf("expected 1 feature, got %d", stats.Features)
	}
}

func TestSimplifyClosedRingKeepsClosure(t *testing.T) {
	ring := [][]float64{
		{0, 0},
		{1, 0},
		{2, 0},
		{3, 0},
		{3, 1},
		{3, 2},
		{2, 2},
		{1, 2},
		{0, 2},
		{0, 1},
		{0, 0},
	}

	got := simplifyClosedRing(ring, 0.25, 6)
	if len(got) >= len(ring) {
		t.Fatalf("expected simplified ring to have fewer points: before=%d after=%d", len(ring), len(got))
	}
	if len(got) < 4 {
		t.Fatalf("expected simplified ring to stay valid, got %d points", len(got))
	}
	first := got[0]
	last := got[len(got)-1]
	if first[0] != last[0] || first[1] != last[1] {
		t.Fatalf("expected simplified ring to stay closed: first=%v last=%v", first, last)
	}
}
