package sqlite

import (
	"context"
	"testing"
	"time"

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

