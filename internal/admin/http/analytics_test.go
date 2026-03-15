package http

import (
	"context"
	"database/sql"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"pipescope/internal/admin/service"
	sqlitestore "pipescope/internal/store/sqlite"
)

func TestAnalyticsEndpointPreservesDecimalAvgDuration(t *testing.T) {
	db := openAnalyticsTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	nowMS := now.UnixMilli()
	seedAnalyticsConnEvent(t, db, analyticsSeedEvent{
		RuleID:     "r1",
		Province:   "湖北",
		City:       "武汉",
		TotalBytes: 100,
		StartTS:    nowMS - int64((5*time.Minute)/time.Millisecond),
		DurationMS: 20681,
	})
	seedAnalyticsConnEvent(t, db, analyticsSeedEvent{
		RuleID:     "r1",
		Province:   "湖北",
		City:       "武汉",
		TotalBytes: 80,
		StartTS:    nowMS - int64((4*time.Minute)/time.Millisecond),
		DurationMS: 20682,
	})

	svc := service.New(db)
	svc.SetNowFunc(func() time.Time { return now })

	srv := NewServer(svc, 50*time.Millisecond)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(nethttp.MethodGet, "/api/analytics?window=1h&province=%E6%B9%96%E5%8C%97", nil)

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != nethttp.StatusOK {
		t.Fatalf("code=%d want=%d body=%s", rr.Code, nethttp.StatusOK, rr.Body.String())
	}

	var rsp struct {
		Overview struct {
			ConnCount     int64   `json:"conn_count"`
			AvgDurationMS float64 `json:"avg_duration_ms"`
		} `json:"overview"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&rsp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if rsp.Overview.ConnCount != 2 {
		t.Fatalf("unexpected conn count: %+v", rsp.Overview)
	}
	if rsp.Overview.AvgDurationMS != 20681.5 {
		t.Fatalf("unexpected avg_duration_ms: got=%v want=20681.5", rsp.Overview.AvgDurationMS)
	}
}

type analyticsSeedEvent struct {
	RuleID     string
	Province   string
	City       string
	TotalBytes int64
	StartTS    int64
	DurationMS int64
}

func openAnalyticsTempDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "pipescope-admin-http-test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func seedAnalyticsConnEvent(t *testing.T, db *sql.DB, e analyticsSeedEvent) {
	t.Helper()
	_, err := db.Exec(`
INSERT INTO conn_events(
  rule_id, listen_port, src_addr, src_ip, dst_addr, dst_host, dst_port,
  start_ts, end_ts, duration_ms, up_bytes, down_bytes, total_bytes,
  status, err_msg, province, city, adcode, lat, lng
) VALUES (?, 10001, '1.1.1.1:1', '1.1.1.1', '2.2.2.2:2', '2.2.2.2', 2, ?, ?, ?, 1, 1, ?, 'ok', '', ?, ?, '', 22.5, 114.0)
`, e.RuleID, e.StartTS, e.StartTS+1, e.DurationMS, e.TotalBytes, e.Province, e.City)
	if err != nil {
		t.Fatalf("seed conn_events: %v", err)
	}
}
