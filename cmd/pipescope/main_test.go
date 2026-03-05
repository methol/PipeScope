package main

import (
	"context"
	"database/sql"
	nethttp "net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	sqlitestore "pipescope/internal/store/sqlite"
	_ "modernc.org/sqlite"
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

