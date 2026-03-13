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

func TestOpenSQLiteAppliesBusyTimeoutToEachConnection(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pipescope-busy-timeout.db")
	db, err := openSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	conn1, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("conn1: %v", err)
	}
	defer conn1.Close()
	conn2, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("conn2: %v", err)
	}
	defer conn2.Close()

	for i, conn := range []*sql.Conn{conn1, conn2} {
		var timeoutMS int
		if err := conn.QueryRowContext(ctx, `PRAGMA busy_timeout;`).Scan(&timeoutMS); err != nil {
			t.Fatalf("query busy_timeout conn%d: %v", i+1, err)
		}
		if timeoutMS != sqliteBusyTimeoutMS {
			t.Fatalf("busy_timeout conn%d=%d want=%d", i+1, timeoutMS, sqliteBusyTimeoutMS)
		}
		var mode string
		if err := conn.QueryRowContext(ctx, `PRAGMA journal_mode;`).Scan(&mode); err != nil {
			t.Fatalf("query journal_mode conn%d: %v", i+1, err)
		}
		if strings.ToLower(mode) != "wal" {
			t.Fatalf("journal_mode conn%d=%q want=wal", i+1, mode)
		}
	}
}

func TestNewAdminServerSetsTimeouts(t *testing.T) {
	srv := newAdminServer("127.0.0.1:0", nethttp.NewServeMux())
	if srv.ReadHeaderTimeout != adminReadHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout=%s want=%s", srv.ReadHeaderTimeout, adminReadHeaderTimeout)
	}
	if srv.WriteTimeout != adminWriteTimeout {
		t.Fatalf("WriteTimeout=%s want=%s", srv.WriteTimeout, adminWriteTimeout)
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
