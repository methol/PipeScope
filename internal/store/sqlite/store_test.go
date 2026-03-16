package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"pipescope/internal/geo/areacity"
	"pipescope/internal/geo/normalize"

	_ "modernc.org/sqlite"
)

func TestInitSchemaCreatesTables(t *testing.T) {
	db := openTempDB(t)
	s := New(db)
	if err := s.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	requireTable(t, db, "conn_events")
	requireTable(t, db, "dim_adcode")
	requireTable(t, db, "app_meta")
}

func TestInitSchemaMigratesLegacyConnEventsColumns(t *testing.T) {
	db := openTempDB(t)
	_, err := db.Exec(`
CREATE TABLE conn_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ok'
);
`)
	if err != nil {
		t.Fatalf("create legacy conn_events: %v", err)
	}

	s := New(db)
	if err := s.InitSchema(context.Background()); err != nil {
		t.Fatalf("init schema with migration: %v", err)
	}

	requireColumn(t, db, "conn_events", "blocked_reason")
	requireColumn(t, db, "conn_events", "province")
	requireColumn(t, db, "conn_events", "city")
	requireColumn(t, db, "conn_events", "adcode")
	requireColumn(t, db, "conn_events", "lat")
	requireColumn(t, db, "conn_events", "lng")
}

func openTempDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "pipescope-test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func requireTable(t *testing.T, db *sql.DB, tableName string) {
	t.Helper()
	var exists int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?`,
		tableName,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	if exists != 1 {
		t.Fatalf("table %s not found", tableName)
	}
}

func requireColumn(t *testing.T, db *sql.DB, tableName, columnName string) {
	t.Helper()
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		t.Fatalf("query pragma_table_info: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			t.Fatalf("scan pragma_table_info: %v", err)
		}
		if name == columnName {
			return
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate pragma_table_info: %v", err)
	}
	t.Fatalf("column %s.%s not found", tableName, columnName)
}

func seedDimAdcode(t *testing.T, db *sql.DB, dim areacity.DimAdcode) {
	t.Helper()
	nProvince := normalize.NormalizeProvince(dim.Province)
	nCity := normalize.NormalizeCity(dim.City)
	if nCity == "" {
		nCity = nProvince
	}
	if _, err := db.Exec(`
INSERT INTO dim_adcode(adcode, province, city, district, lat, lng, normalized_province, normalized_city)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, dim.Adcode, dim.Province, dim.City, dim.District, dim.Lat, dim.Lng, nProvince, nCity); err != nil {
		t.Fatalf("seed dim_adcode: %v", err)
	}
}
