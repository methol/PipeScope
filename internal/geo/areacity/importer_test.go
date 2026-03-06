package areacity_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
	"pipescope/internal/geo/areacity"
	sqlitestore "pipescope/internal/store/sqlite"
)

func TestImportAndMatchByProvinceCity(t *testing.T) {
	db := openTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	imp := areacity.NewImporter(db)
	csvPath := filepath.Join("testdata", "ok_geo_sample.csv")
	if err := imp.ImportCSV(context.Background(), csvPath); err != nil {
		t.Fatalf("import csv: %v", err)
	}

	m := areacity.NewMatcher(db)
	got, ok, err := m.Match("广东", "深圳")
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if !ok {
		t.Fatalf("expected matched row")
	}
	if got.Adcode == "" || got.Lat == 0 || got.Lng == 0 {
		t.Fatalf("invalid result: %+v", got)
	}
	if got.Adcode != "4403" {
		t.Fatalf("unexpected adcode: %s", got.Adcode)
	}
	if got.Province != "广东省" || got.City != "深圳市" {
		t.Fatalf("unexpected row names: %+v", got)
	}

	bj, ok, err := m.Match("北京", "北京")
	if err != nil {
		t.Fatalf("match beijing: %v", err)
	}
	if !ok {
		t.Fatalf("expected beijing matched row")
	}
	if bj.Adcode != "1101" {
		t.Fatalf("expected beijing city adcode 1101, got %s", bj.Adcode)
	}
}

func TestMatchFallsBackToProvinceWhenCityMissing(t *testing.T) {
	db := openTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	imp := areacity.NewImporter(db)
	csvPath := filepath.Join("testdata", "ok_geo_sample.csv")
	if err := imp.ImportCSV(context.Background(), csvPath); err != nil {
		t.Fatalf("import csv: %v", err)
	}

	m := areacity.NewMatcher(db)
	got, ok, err := m.Match("北京市", "")
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if !ok {
		t.Fatalf("expected matched row")
	}
	if got.Adcode != "1101" {
		t.Fatalf("expected beijing city adcode 1101, got %s", got.Adcode)
	}
}

func TestReplaceCSVReaderReplacesExistingRows(t *testing.T) {
	db := openTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	imp := areacity.NewImporter(db)
	if err := imp.ImportCSV(context.Background(), filepath.Join("testdata", "ok_geo_sample.csv")); err != nil {
		t.Fatalf("import csv: %v", err)
	}

	seed := strings.NewReader("adcode,province,city,district,lat,lng\n9999,测试省,测试市,,22.1,114.1\n")
	if err := imp.ReplaceCSVReader(context.Background(), seed); err != nil {
		t.Fatalf("replace csv reader: %v", err)
	}

	m := areacity.NewMatcher(db)
	if _, ok, err := m.Match("广东", "深圳"); err != nil {
		t.Fatalf("match old row: %v", err)
	} else if ok {
		t.Fatalf("expected old row removed")
	}

	got, ok, err := m.Match("测试省", "测试市")
	if err != nil {
		t.Fatalf("match new row: %v", err)
	}
	if !ok {
		t.Fatalf("expected new row")
	}
	if got.Adcode != "9999" {
		t.Fatalf("unexpected adcode: %s", got.Adcode)
	}
}

func openTempDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "pipescope-areacity-test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
