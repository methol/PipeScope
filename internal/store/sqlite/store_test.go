package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

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

