package service

import (
	"context"
	"database/sql"
	"time"
)

type Service struct {
	db  *sql.DB
	now func() time.Time
}

func New(db *sql.DB) *Service {
	return &Service{
		db:  db,
		now: time.Now,
	}
}

func (s *Service) SetNowFunc(fn func() time.Time) {
	if fn != nil {
		s.now = fn
	}
}

func (s *Service) ChinaMap(ctx context.Context, q MapQuery) ([]MapPoint, error) {
	metricExpr := "COUNT(*)"
	if q.Metric == MetricBytes {
		metricExpr = "COALESCE(SUM(total_bytes), 0)"
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT adcode, province, city, MAX(lat) AS lat, MAX(lng) AS lng, `+metricExpr+` AS v
FROM conn_events
WHERE start_ts >= ?
  AND adcode <> ''
GROUP BY adcode, province, city
ORDER BY v DESC
`, s.windowStartMS(q.Window))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MapPoint
	for rows.Next() {
		var p MapPoint
		if err := rows.Scan(&p.Adcode, &p.Province, &p.City, &p.Lat, &p.Lng, &p.Value); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) Rules(ctx context.Context, q RulesQuery) ([]RulePoint, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT rule_id, COUNT(*) AS conn_count, COALESCE(SUM(total_bytes), 0) AS total_bytes
FROM conn_events
WHERE start_ts >= ?
GROUP BY rule_id
ORDER BY total_bytes DESC
`, s.windowStartMS(q.Window))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RulePoint
	for rows.Next() {
		var p RulePoint
		if err := rows.Scan(&p.RuleID, &p.ConnCount, &p.TotalBytes); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) Sessions(ctx context.Context, q SessionsQuery) ([]SessionItem, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, rule_id, src_addr, dst_addr, status, up_bytes, down_bytes, total_bytes,
       start_ts, end_ts, duration_ms, province, city, adcode
FROM conn_events
WHERE start_ts >= ?
  AND (? = '' OR rule_id = ?)
ORDER BY start_ts DESC
LIMIT ? OFFSET ?
`, s.windowStartMS(q.Window), q.RuleID, q.RuleID, limit, q.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SessionItem
	for rows.Next() {
		var item SessionItem
		if err := rows.Scan(
			&item.ID,
			&item.RuleID,
			&item.SrcAddr,
			&item.DstAddr,
			&item.Status,
			&item.UpBytes,
			&item.DownBytes,
			&item.TotalBytes,
			&item.StartTS,
			&item.EndTS,
			&item.DurationMS,
			&item.Province,
			&item.City,
			&item.Adcode,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) Overview(ctx context.Context, window time.Duration) (Overview, error) {
	var out Overview
	err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*) AS conn_count, COALESCE(SUM(total_bytes), 0) AS total_bytes
FROM conn_events
WHERE start_ts >= ?
`, s.windowStartMS(window)).Scan(&out.ConnCount, &out.TotalBytes)
	return out, err
}

func (s *Service) ProvinceMap(ctx context.Context, q ProvinceQuery) ([]MapPoint, error) {
	metricExpr := "COUNT(*)"
	if q.Metric == MetricBytes {
		metricExpr = "COALESCE(SUM(total_bytes), 0)"
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT adcode, province, city, MAX(lat) AS lat, MAX(lng) AS lng, `+metricExpr+` AS v
FROM conn_events
WHERE start_ts >= ?
  AND province = ?
  AND adcode <> ''
GROUP BY adcode, province, city
ORDER BY v DESC
`, s.windowStartMS(q.Window), q.Province)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MapPoint
	for rows.Next() {
		var p MapPoint
		if err := rows.Scan(&p.Adcode, &p.Province, &p.City, &p.Lat, &p.Lng, &p.Value); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) windowStartMS(window time.Duration) int64 {
	if window <= 0 {
		return 0
	}
	return s.now().Add(-window).UnixMilli()
}

