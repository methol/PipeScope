package main

import (
	"bytes"
	"context"
	"database/sql"
	nethttp "net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
	"pipescope/internal/config"
	sqlitestore "pipescope/internal/store/sqlite"
)

func TestServeAdminIndex(t *testing.T) {
	db := openTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	handler := newAdminHandler(db)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(nethttp.MethodGet, "/", nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != nethttp.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	if !strings.Contains(strings.ToLower(rr.Body.String()), "pipescope admin") {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestInitAreaCityMatcherUsesEmbeddedSeed(t *testing.T) {
	db := openTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	matcher, err := initAreaCityMatcher(context.Background(), db, cfg)
	if err != nil {
		t.Fatalf("init matcher: %v", err)
	}
	if matcher == nil {
		t.Fatalf("expected matcher")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM dim_adcode`).Scan(&count); err != nil {
		t.Fatalf("count dim_adcode: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected embedded dim_adcode rows")
	}
}

func TestOpenSQLiteConfiguresConnectionPool(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pipescope-single-conn.db")
	db, err := openSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if max := db.Stats().MaxOpenConnections; max < 2 {
		t.Fatalf("MaxOpenConnections=%d want>=2", max)
	}

}

func TestUsageIncludesDefaultsAndFlags(t *testing.T) {
	buf := new(bytes.Buffer)
	writeUsage(buf)
	out := buf.String()
	for _, want := range []string{"PipeScope", "-config", "assets/config.example.yaml"} {
		if !strings.Contains(out, want) {
			t.Fatalf("usage missing %q in %q", want, out)
		}
	}
}

func openTempDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "pipescope-main-test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
