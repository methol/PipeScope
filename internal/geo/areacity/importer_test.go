package areacity

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	sqlitestore "pipescope/internal/store/sqlite"
	_ "modernc.org/sqlite"
)

func TestImportAndMatchByProvinceCity(t *testing.T) {
	db := openTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	imp := NewImporter(db)
	csvPath := filepath.Join("testdata", "ok_geo_sample.csv")
	if err := imp.ImportCSV(context.Background(), csvPath); err != nil {
		t.Fatalf("import csv: %v", err)
	}

	m := NewMatcher(db)
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
