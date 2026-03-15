package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlitestore "pipescope/internal/store/sqlite"
)

func TestAnalyticsAggregation(t *testing.T) {
	db := openTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	nowMS := now.UnixMilli()
	seedConnEvent(t, db, seedEvent{RuleID: "r1", Adcode: "440300", City: "深圳", Province: "广东", TotalBytes: 100, StartTS: nowMS - int64((5*time.Minute)/time.Millisecond)})
	seedConnEvent(t, db, seedEvent{RuleID: "r1", Adcode: "440300", City: "深圳", Province: "广东", TotalBytes: 50, StartTS: nowMS - int64((3*time.Minute)/time.Millisecond)})
	seedConnEvent(t, db, seedEvent{RuleID: "r2", Adcode: "440400", City: "珠海", Province: "广东", TotalBytes: 200, StartTS: nowMS - int64((2*time.Minute)/time.Millisecond)})

	svc := New(db)
	svc.SetNowFunc(func() time.Time { return now })

	res, err := svc.Analytics(context.Background(), AnalyticsQuery{Window: 15 * time.Minute, TopN: 10})
	if err != nil {
		t.Fatalf("Analytics: %v", err)
	}

	if res.Overview.ConnCount != 3 || res.Overview.TotalBytes != 350 {
		t.Fatalf("unexpected overview: %+v", res.Overview)
	}
	if res.Overview.ActiveRules != 2 || res.Overview.ActiveCities != 2 {
		t.Fatalf("unexpected active stats: %+v", res.Overview)
	}
	if len(res.TopCities) != 2 || res.TopCities[0].Name != "广东珠海" || res.TopCities[0].TotalBytes != 200 {
		t.Fatalf("unexpected top cities: %+v", res.TopCities)
	}
	if len(res.TopRules) != 2 || res.TopRules[0].Name != "r2" || res.TopRules[0].TotalBytes != 200 {
		t.Fatalf("unexpected top rules: %+v", res.TopRules)
	}
}

func TestAnalyticsOptions(t *testing.T) {
	db := openTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	nowMS := now.UnixMilli()
	seedConnEventWithStatus(t, db, seedEventWithStatus{seedEvent: seedEvent{RuleID: "r1", Province: "广东", City: "深圳", TotalBytes: 100, StartTS: nowMS - int64((5*time.Minute)/time.Millisecond)}, Status: "ok", DurationMS: 20})
	seedConnEventWithStatus(t, db, seedEventWithStatus{seedEvent: seedEvent{RuleID: "r2", Province: "广东", City: "珠海", TotalBytes: 80, StartTS: nowMS - int64((4*time.Minute)/time.Millisecond)}, Status: "err", DurationMS: 60})
	seedConnEventWithStatus(t, db, seedEventWithStatus{seedEvent: seedEvent{RuleID: "r3", Province: "浙江", City: "杭州", TotalBytes: 200, StartTS: nowMS - int64((3*time.Minute)/time.Millisecond)}, Status: "ok", DurationMS: 40})

	svc := New(db)
	svc.SetNowFunc(func() time.Time { return now })

	res, err := svc.AnalyticsOptions(context.Background(), AnalyticsOptionsQuery{Window: 15 * time.Minute})
	if err != nil {
		t.Fatalf("AnalyticsOptions: %v", err)
	}

	if len(res.Rules) != 3 || len(res.Provinces) != 2 || len(res.Cities) != 3 || len(res.Statuses) != 2 {
		t.Fatalf("unexpected options: %+v", res)
	}

	filtered, err := svc.AnalyticsOptions(context.Background(), AnalyticsOptionsQuery{Window: 15 * time.Minute, Province: "广东"})
	if err != nil {
		t.Fatalf("AnalyticsOptions with province filter: %v", err)
	}
	if len(filtered.Cities) != 2 {
		t.Fatalf("unexpected linked cities: %+v", filtered.Cities)
	}
}

func TestAnalyticsFilters(t *testing.T) {
	db := openTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	nowMS := now.UnixMilli()
	seedConnEventWithStatus(t, db, seedEventWithStatus{seedEvent: seedEvent{RuleID: "r1", Province: "广东", City: "深圳", TotalBytes: 100, StartTS: nowMS - int64((5*time.Minute)/time.Millisecond)}, Status: "ok", DurationMS: 20})
	seedConnEventWithStatus(t, db, seedEventWithStatus{seedEvent: seedEvent{RuleID: "r1", Province: "广东", City: "深圳", TotalBytes: 80, StartTS: nowMS - int64((4*time.Minute)/time.Millisecond)}, Status: "err", DurationMS: 60})
	seedConnEventWithStatus(t, db, seedEventWithStatus{seedEvent: seedEvent{RuleID: "r2", Province: "浙江", City: "杭州", TotalBytes: 200, StartTS: nowMS - int64((3*time.Minute)/time.Millisecond)}, Status: "ok", DurationMS: 40})

	svc := New(db)
	svc.SetNowFunc(func() time.Time { return now })

	res, err := svc.Analytics(context.Background(), AnalyticsQuery{
		Window:   15 * time.Minute,
		RuleID:   "r1",
		Province: "广",
		City:     "深",
		Status:   "ok",
		TopN:     5,
	})
	if err != nil {
		t.Fatalf("Analytics with filter: %v", err)
	}

	if res.Overview.ConnCount != 1 || res.Overview.TotalBytes != 100 {
		t.Fatalf("unexpected overview: %+v", res.Overview)
	}
	if len(res.TopCities) != 1 || res.TopCities[0].Name != "广东深圳" {
		t.Fatalf("unexpected top cities: %+v", res.TopCities)
	}
	if len(res.TopRules) != 1 || res.TopRules[0].Name != "r1" {
		t.Fatalf("unexpected top rules: %+v", res.TopRules)
	}
}

func TestAnalyticsAvgDurationPreservesDecimal(t *testing.T) {
	db := openTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	nowMS := now.UnixMilli()
	seedConnEventWithStatus(t, db, seedEventWithStatus{seedEvent: seedEvent{RuleID: "r1", Province: "湖北", City: "武汉", TotalBytes: 100, StartTS: nowMS - int64((5*time.Minute)/time.Millisecond)}, Status: "ok", DurationMS: 20681})
	seedConnEventWithStatus(t, db, seedEventWithStatus{seedEvent: seedEvent{RuleID: "r1", Province: "湖北", City: "武汉", TotalBytes: 80, StartTS: nowMS - int64((4*time.Minute)/time.Millisecond)}, Status: "ok", DurationMS: 20682})

	svc := New(db)
	svc.SetNowFunc(func() time.Time { return now })

	res, err := svc.Analytics(context.Background(), AnalyticsQuery{
		Window:   1 * time.Hour,
		Province: "湖北",
		TopN:     10,
	})
	if err != nil {
		t.Fatalf("Analytics with decimal avg duration should not fail: %v", err)
	}

	if res.Overview.ConnCount != 2 {
		t.Fatalf("unexpected conn count: %+v", res.Overview)
	}
	if got := float64(res.Overview.AvgDurationMS); got != 20681.5 {
		t.Fatalf("unexpected avg_duration_ms: got=%v want=20681.5", got)
	}
}

type seedEventWithStatus struct {
	seedEvent
	Status     string
	DurationMS int64
}

func seedConnEventWithStatus(t *testing.T, db *sql.DB, e seedEventWithStatus) {
	t.Helper()
	_, err := db.Exec(`
INSERT INTO conn_events(
  rule_id, listen_port, src_addr, src_ip, dst_addr, dst_host, dst_port,
  start_ts, end_ts, duration_ms, up_bytes, down_bytes, total_bytes,
  status, err_msg, province, city, adcode, lat, lng
) VALUES (?, 10001, '1.1.1.1:1', '1.1.1.1', '2.2.2.2:2', '2.2.2.2', 2, ?, ?, ?, 1, 1, ?, ?, '', ?, ?, '', 22.5, 114.0)
`, e.RuleID, e.StartTS, e.StartTS+1, e.DurationMS, e.TotalBytes, e.Status, e.Province, e.City)
	if err != nil {
		t.Fatalf("seed conn_events with status: %v", err)
	}
}
