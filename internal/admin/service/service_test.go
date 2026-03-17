package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	sqlitestore "pipescope/internal/store/sqlite"
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
		RuleID:     "r1",
		Adcode:     "440300",
		City:       "深圳",
		Province:   "广东",
		TotalBytes: 100,
		StartTS:    nowMS - int64((5*time.Minute)/time.Millisecond),
	})
	seedConnEvent(t, db, seedEvent{
		RuleID:     "r2",
		Adcode:     "440300",
		City:       "深圳",
		Province:   "广东",
		TotalBytes: 300,
		StartTS:    nowMS - int64((3*time.Minute)/time.Millisecond),
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
	SrcIP      string
	Country    string
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
  status, err_msg, country, province, city, adcode, lat, lng
) VALUES (?, 10001, ?, ?, '2.2.2.2:2', '2.2.2.2', 2, ?, ?, 0, 1, 1, ?, 'ok', '', ?, ?, ?, ?, 22.5, 114.0)
`, e.RuleID, firstNonEmpty(e.SrcIP, "1.1.1.1")+":1", firstNonEmpty(e.SrcIP, "1.1.1.1"), e.StartTS, e.StartTS+1, e.TotalBytes, e.Country, e.Province, e.City, e.Adcode)
	if err != nil {
		t.Fatalf("seed conn_events: %v", err)
	}
}

func firstNonEmpty(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func TestProvinceSummaryAggregation(t *testing.T) {
	db := openTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	nowMS := now.UnixMilli()
	seedConnEvent(t, db, seedEvent{
		RuleID:     "r1",
		Adcode:     "440300",
		City:       "深圳",
		Province:   "广东省",
		TotalBytes: 100,
		StartTS:    nowMS - int64((5*time.Minute)/time.Millisecond),
	})
	seedConnEvent(t, db, seedEvent{
		RuleID:     "r2",
		Adcode:     "330100",
		City:       "杭州",
		Province:   "浙江省",
		TotalBytes: 300,
		StartTS:    nowMS - int64((3*time.Minute)/time.Millisecond),
	})
	seedConnEvent(t, db, seedEvent{
		RuleID:     "r3",
		Adcode:     "330200",
		City:       "宁波",
		Province:   "浙江省",
		TotalBytes: 500,
		StartTS:    nowMS - int64((2*time.Minute)/time.Millisecond),
	})

	svc := New(db)
	svc.SetNowFunc(func() time.Time { return now })

	points, err := svc.ProvinceSummary(context.Background(), MapQuery{Window: 15 * time.Minute, Metric: MetricConn})
	if err != nil {
		t.Fatalf("ProvinceSummary conn: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("conn points len=%d", len(points))
	}
	if points[0].Province != "浙江省" || points[0].Value != 2 {
		t.Fatalf("unexpected conn top point: %+v", points[0])
	}

	points, err = svc.ProvinceSummary(context.Background(), MapQuery{Window: 15 * time.Minute, Metric: MetricBytes})
	if err != nil {
		t.Fatalf("ProvinceSummary bytes: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("bytes points len=%d", len(points))
	}
	if points[0].Province != "浙江省" || points[0].Value != 800 {
		t.Fatalf("unexpected bytes top point: %+v", points[0])
	}
}

func TestSessionsReturnsCountry(t *testing.T) {
	db := openTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	nowMS := now.UnixMilli()
	seedConnEvent(t, db, seedEvent{
		RuleID:     "r-country",
		Country:    "CN",
		Province:   "四川",
		City:       "成都",
		Adcode:     "510100",
		TotalBytes: 100,
		StartTS:    nowMS - int64((5*time.Minute)/time.Millisecond),
	})

	svc := New(db)
	svc.SetNowFunc(func() time.Time { return now })

	items, err := svc.Sessions(context.Background(), SessionsQuery{
		Window: 15 * time.Minute,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("Sessions: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len=%d want=1", len(items))
	}
	if items[0].Country != "CN" {
		t.Fatalf("country=%q want=CN", items[0].Country)
	}
	if items[0].Province != "四川" || items[0].City != "成都" || items[0].Adcode != "510100" {
		t.Fatalf("unexpected geo fields: %+v", items[0])
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
