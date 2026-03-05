package sqlite

import (
	"context"
	"database/sql"
	"net"
	"strconv"
	"time"

	"pipescope/internal/gateway/session"
)

type Writer struct {
	db            *sql.DB
	in            <-chan session.Event
	batchSize     int
	flushInterval time.Duration
	region        RegionLookup
	matcher       AdcodeMatcher
}

func NewWriter(db *sql.DB, in <-chan session.Event, batchSize int, flushInterval time.Duration) *Writer {
	if batchSize <= 0 {
		batchSize = 1
	}
	if flushInterval <= 0 {
		flushInterval = time.Second
	}
	return &Writer{
		db:            db,
		in:            in,
		batchSize:     batchSize,
		flushInterval: flushInterval,
	}
}

func (w *Writer) SetGeoEnricher(region RegionLookup, matcher AdcodeMatcher) {
	w.region = region
	w.matcher = matcher
}

func (w *Writer) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	batch := make([]session.Event, 0, w.batchSize)

	flush := func(execCtx context.Context) error {
		if len(batch) == 0 {
			return nil
		}
		if err := w.insertBatch(execCtx, batch); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}

	flushWithGrace := func() error {
		flushCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return flush(flushCtx)
	}

	for {
		select {
		case <-ctx.Done():
			draining := true
			for draining {
				select {
				case evt, ok := <-w.in:
					if !ok {
						draining = false
						break
					}
					batch = append(batch, evt)
					if len(batch) >= w.batchSize {
						if err := flushWithGrace(); err != nil {
							return err
						}
					}
				default:
					draining = false
				}
			}
			if err := flushWithGrace(); err != nil {
				return err
			}
			return nil
		case evt, ok := <-w.in:
			if !ok {
				if err := flushWithGrace(); err != nil {
					return err
				}
				return nil
			}
			batch = append(batch, evt)
			if len(batch) >= w.batchSize {
				if err := flush(ctx); err != nil {
					return err
				}
			}
		case <-ticker.C:
			if err := flush(ctx); err != nil {
				return err
			}
		}
	}
}

func (w *Writer) insertBatch(ctx context.Context, batch []session.Event) error {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO conn_events(
  rule_id, listen_port, src_addr, src_ip, dst_addr, dst_host, dst_port,
  start_ts, end_ts, duration_ms, up_bytes, down_bytes, total_bytes,
  status, err_msg, province, city, adcode, lat, lng
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, evt := range batch {
		srcIP := extractHost(evt.SrcAddr)
		dstHost := extractHost(evt.DstAddr)
		dstPort := extractPort(evt.DstAddr)
		geo := enrichGeoFields(evt, w.region, w.matcher)
		if _, err := stmt.ExecContext(
			ctx,
			evt.RuleID,
			evt.ListenPort,
			evt.SrcAddr,
			srcIP,
			evt.DstAddr,
			dstHost,
			dstPort,
			evt.StartTS,
			evt.EndTS,
			evt.DurationMS,
			evt.UpBytes,
			evt.DownBytes,
			evt.TotalBytes,
			evt.Status,
			evt.Error,
			geo.Province,
			geo.City,
			geo.Adcode,
			geo.Lat,
			geo.Lng,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func extractHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return ""
	}
	return host
}

func extractPort(addr string) int {
	_, p, err := net.SplitHostPort(addr)
	if err != nil {
		return 0
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return 0
	}
	return port
}
