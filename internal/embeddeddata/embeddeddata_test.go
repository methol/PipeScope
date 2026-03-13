package embeddeddata

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pipescope/internal/geo/ip2region"
	sqlitestore "pipescope/internal/store/sqlite"

	_ "modernc.org/sqlite"
)

func TestEnsureAreaCitySeedImportsOncePerVersion(t *testing.T) {
	db := openTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	cacheDir := t.TempDir()
	if err := EnsureAreaCitySeed(context.Background(), db, cacheDir); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	count1 := countRows(t, db, "dim_adcode")
	if count1 == 0 {
		t.Fatalf("expected imported rows")
	}

	if err := EnsureAreaCitySeed(context.Background(), db, cacheDir); err != nil {
		t.Fatalf("second ensure: %v", err)
	}
	count2 := countRows(t, db, "dim_adcode")
	if count2 != count1 {
		t.Fatalf("expected stable row count, got %d -> %d", count1, count2)
	}
}

func TestEnsureIP2RegionXDBMaterializesFile(t *testing.T) {
	cacheDir := t.TempDir()
	path, err := EnsureIP2RegionXDB(cacheDir)
	if err != nil {
		t.Fatalf("ensure ip2region xdb: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat xdb: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("expected non-empty xdb file")
	}
	if filepath.Ext(path) != ".xdb" {
		t.Fatalf("expected .xdb file, got %s", path)
	}
	searcher, err := ip2region.NewSearcher(path)
	if err != nil {
		t.Fatalf("open embedded xdb: %v", err)
	}
	searcher.Close()
}

func TestDefaultCacheDirFallsBackWhenUserCacheDirUnavailable(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	path, err := DefaultCacheDir()
	if err != nil {
		t.Fatalf("default cache dir: %v", err)
	}
	if path == "" {
		t.Fatalf("expected fallback cache dir")
	}
}

func TestResolveWritableCacheDirFallsBackWhenPreferredUnwritable(t *testing.T) {
	preferredRoot := filepath.Join(t.TempDir(), "preferred")
	if err := os.MkdirAll(preferredRoot, 0o755); err != nil {
		t.Fatalf("mkdir preferred root: %v", err)
	}
	if err := os.Chmod(preferredRoot, 0o555); err != nil {
		t.Fatalf("chmod preferred root: %v", err)
	}
	defer os.Chmod(preferredRoot, 0o755)

	fallbackRoot := filepath.Join(t.TempDir(), "fallback")
	path, err := resolveWritableCacheDir(filepath.Join(preferredRoot, "pipescope", "embeddeddata"), filepath.Join(fallbackRoot, "pipescope", "embeddeddata"))
	if err != nil {
		t.Fatalf("resolve writable cache dir: %v", err)
	}
	if !strings.HasPrefix(path, fallbackRoot) {
		t.Fatalf("expected fallback root, got %s", path)
	}
}

func openTempDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "embeddeddata-test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count rows from %s: %v", table, err)
	}
	return count
}
