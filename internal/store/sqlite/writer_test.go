package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"pipescope/internal/geo/areacity"
	"pipescope/internal/geo/ip2region"
	"pipescope/internal/gateway/session"
)

func TestWriterBatchInsert(t *testing.T) {
	db := openTempDB(t)
	s := New(db)
	if err := s.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	in := make(chan session.Event, 16)
	w := NewWriter(db, in, 3, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Run(ctx)
	}()

	total := 7
	for i := 0; i < total; i++ {
		in <- session.Event{
			RuleID:     "r1",
			ListenPort: 10001,
			SrcAddr:    "1.1.1.1:1000",
			DstAddr:    "2.2.2.2:80",
			StartTS:    time.Now().UnixMilli(),
			EndTS:      time.Now().UnixMilli(),
			Status:     "ok",
			UpBytes:    10,
			DownBytes:  20,
			TotalBytes: 30,
		}
	}
	close(in)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("writer run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("writer did not stop")
	}

	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conn_events`).Scan(&got); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if got != total {
		t.Fatalf("row count mismatch: got=%d want=%d", got, total)
	}
}

func TestWriterEnrichesGeoFields(t *testing.T) {
	db := openTempDB(t)
	s := New(db)
	if err := s.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	in := make(chan session.Event, 1)
	w := NewWriter(db, in, 1, time.Hour)
	w.SetGeoEnricher(
		fakeRegionLookup{
			region: ip2region.Region{
				Province: "广东",
				City:     "深圳",
			},
		},
		fakeMatcher{
			dim: areacity.DimAdcode{
				Adcode: "440300",
				Lat:    22.5431,
				Lng:    114.0579,
			},
			ok: true,
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Run(ctx)
	}()

	in <- session.Event{
		RuleID:     "r-geo",
		ListenPort: 10001,
		SrcAddr:    "1.1.1.1:1000",
		DstAddr:    "2.2.2.2:80",
		StartTS:    time.Now().UnixMilli(),
		EndTS:      time.Now().UnixMilli(),
		Status:     "ok",
		UpBytes:    10,
		DownBytes:  20,
		TotalBytes: 30,
	}
	close(in)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("writer run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("writer did not stop")
	}

	var province, city, adcode string
	var lat, lng float64
	if err := db.QueryRow(`
SELECT province, city, adcode, lat, lng
FROM conn_events
WHERE rule_id = 'r-geo'
LIMIT 1
`).Scan(&province, &city, &adcode, &lat, &lng); err != nil {
		t.Fatalf("query geo row: %v", err)
	}
	if province != "广东" || city != "深圳" || adcode != "440300" {
		t.Fatalf("unexpected geo fields: %s %s %s", province, city, adcode)
	}
	if lat == 0 || lng == 0 {
		t.Fatalf("unexpected coords: lat=%f lng=%f", lat, lng)
	}
}

type fakeRegionLookup struct {
	region ip2region.Region
	err    error
}

func (f fakeRegionLookup) Lookup(_ string) (ip2region.Region, error) {
	if f.err != nil {
		return ip2region.Region{}, f.err
	}
	return f.region, nil
}

type fakeMatcher struct {
	dim areacity.DimAdcode
	ok  bool
	err error
}

func (f fakeMatcher) Match(_, _ string) (areacity.DimAdcode, bool, error) {
	if f.err != nil {
		return areacity.DimAdcode{}, false, f.err
	}
	return f.dim, f.ok, nil
}

func TestWriterEnrichSkipsOnLookupError(t *testing.T) {
	db := openTempDB(t)
	s := New(db)
	if err := s.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	in := make(chan session.Event, 1)
	w := NewWriter(db, in, 1, time.Hour)
	w.SetGeoEnricher(
		fakeRegionLookup{err: errors.New("lookup failed")},
		fakeMatcher{},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Run(ctx)
	}()

	in <- session.Event{
		RuleID:     "r-geo-skip",
		ListenPort: 10001,
		SrcAddr:    "1.1.1.1:1000",
		DstAddr:    "2.2.2.2:80",
		StartTS:    time.Now().UnixMilli(),
		EndTS:      time.Now().UnixMilli(),
		Status:     "ok",
	}
	close(in)

	if err := <-errCh; err != nil {
		t.Fatalf("writer run: %v", err)
	}

	var province, city, adcode string
	if err := db.QueryRow(`
SELECT province, city, adcode
FROM conn_events
WHERE rule_id = 'r-geo-skip'
LIMIT 1
`).Scan(&province, &city, &adcode); err != nil {
		t.Fatalf("query row: %v", err)
	}
	if province != "" || city != "" || adcode != "" {
		t.Fatalf("expected empty geo fields, got %q %q %q", province, city, adcode)
	}
}
