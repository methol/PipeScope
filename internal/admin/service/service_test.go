package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	sqlitestore "pipescope/internal/store/sqlite"
	_ "modernc.org/sqlite"
)

func TestChinaMapAggregation(t *testing.T) {
	db := openTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	nowMS := now.UnixMilli()
	seedConnEvent(t, db, seedEvent{
		RuleID:    "r1",
		Adcode:    "440300",
		City:      "深圳",
		Province:  "广东",
		TotalBytes: 100,
		StartTS:   nowMS - int64((5 * time.Minute) / time.Millisecond),
	})
	seedConnEvent(t, db, seedEvent{
		RuleID:    "r2",
		Adcode:    "440300",
		City:      "深圳",
		Province:  "广东",
		TotalBytes: 300,
		StartTS:   nowMS - int64((3 * time.Minute) / time.Millisecond),
	})

	svc := New(db)
	svc.SetNowFunc(func() time.Time { return now })

	points, err := svc.ChinaMap(context.Background(), MapQuery{
		Window: 15 * time.Minute,
		Metric: MetricConn,
	})
	if err != nil {
		t.Fatalf("ChinaMap: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("points len=%d", len(points))
	}
	if points[0].Adcode != "440300" || points[0].Value != 2 {
		t.Fatalf("unexpected point: %+v", points[0])
	}
}

type seedEvent struct {
	RuleID     string
	Adcode     string
	City       string
	Province   string
	TotalBytes int64
	StartTS    int64
}

func seedConnEvent(t *testing.T, db *sql.DB, e seedEvent) {
	t.Helper()
	_, err := db.Exec(`
INSERT INTO conn_events(
  rule_id, listen_port, src_addr, src_ip, dst_addr, dst_host, dst_port,
  start_ts, end_ts, duration_ms, up_bytes, down_bytes, total_bytes,
  status, err_msg, province, city, adcode, lat, lng
) VALUES (?, 10001, '1.1.1.1:1', '1.1.1.1', '2.2.2.2:2', '2.2.2.2', 2, ?, ?, 0, 1, 1, ?, 'ok', '', ?, ?, ?, 22.5, 114.0)
`, e.RuleID, e.StartTS, e.StartTS+1, e.TotalBytes, e.Province, e.City, e.Adcode)
	if err != nil {
		t.Fatalf("seed conn_events: %v", err)
	}
}

func openTempDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "pipescope-admin-service-test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

