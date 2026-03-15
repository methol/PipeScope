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
SELECT COALESCE(NULLIF(adcode, ''), 'unknown') AS adcode, province, city, MAX(lat) AS lat, MAX(lng) AS lng, `+metricExpr+` AS v
FROM conn_events
WHERE start_ts >= ?
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
SELECT COALESCE(NULLIF(adcode, ''), 'unknown') AS adcode, province, city, MAX(lat) AS lat, MAX(lng) AS lng, `+metricExpr+` AS v
FROM conn_events
WHERE start_ts >= ?
  AND province = ?
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

func (s *Service) ProvinceSummary(ctx context.Context, q MapQuery) ([]ProvinceSummaryPoint, error) {
	metricExpr := "COUNT(*)"
	if q.Metric == MetricBytes {
		metricExpr = "COALESCE(SUM(total_bytes), 0)"
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT COALESCE(NULLIF(province, ''), '未知') AS province, `+metricExpr+` AS v
FROM conn_events
WHERE start_ts >= ?
GROUP BY province
ORDER BY v DESC
`, s.windowStartMS(q.Window))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProvinceSummaryPoint
	for rows.Next() {
		var p ProvinceSummaryPoint
		if err := rows.Scan(&p.Province, &p.Value); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) AnalyticsOptions(ctx context.Context, q AnalyticsOptionsQuery) (AnalyticsOptions, error) {
	result := AnalyticsOptions{
		Rules:     []string{},
		Provinces: []string{},
		Cities:    []AnalyticsCityRef{},
		Statuses:  []string{},
	}
	windowStart := s.windowStartMS(q.Window)
	ruleID := q.RuleID
	province := q.Province
	city := q.City
	status := q.Status

	ruleRows, err := s.db.QueryContext(ctx, `
SELECT DISTINCT COALESCE(NULLIF(rule_id, ''), 'unknown') AS rule
FROM conn_events
WHERE start_ts >= ?
  AND (? = '' OR province LIKE '%' || ? || '%')
  AND (? = '' OR city LIKE '%' || ? || '%')
  AND (? = '' OR status = ?)
ORDER BY rule
`, windowStart, province, province, city, city, status, status)
	if err != nil {
		return result, err
	}
	defer ruleRows.Close()
	for ruleRows.Next() {
		var v string
		if err := ruleRows.Scan(&v); err != nil {
			return result, err
		}
		result.Rules = append(result.Rules, v)
	}
	if err := ruleRows.Err(); err != nil {
		return result, err
	}

	provinceRows, err := s.db.QueryContext(ctx, `
SELECT DISTINCT COALESCE(NULLIF(province, ''), '未知') AS province
FROM conn_events
WHERE start_ts >= ?
  AND (? = '' OR rule_id = ?)
  AND (? = '' OR city LIKE '%' || ? || '%')
  AND (? = '' OR status = ?)
ORDER BY province
`, windowStart, ruleID, ruleID, city, city, status, status)
	if err != nil {
		return result, err
	}
	defer provinceRows.Close()
	for provinceRows.Next() {
		var v string
		if err := provinceRows.Scan(&v); err != nil {
			return result, err
		}
		result.Provinces = append(result.Provinces, v)
	}
	if err := provinceRows.Err(); err != nil {
		return result, err
	}

	cityRows, err := s.db.QueryContext(ctx, `
SELECT DISTINCT
	COALESCE(NULLIF(province, ''), '未知') AS province,
	COALESCE(NULLIF(city, ''), '未知') AS city
FROM conn_events
WHERE start_ts >= ?
  AND (? = '' OR rule_id = ?)
  AND (? = '' OR COALESCE(NULLIF(province, ''), '未知') = ?)
  AND (? = '' OR status = ?)
ORDER BY province, city
`, windowStart, ruleID, ruleID, province, province, status, status)
	if err != nil {
		return result, err
	}
	defer cityRows.Close()
	for cityRows.Next() {
		var item AnalyticsCityRef
		if err := cityRows.Scan(&item.Province, &item.City); err != nil {
			return result, err
		}
		result.Cities = append(result.Cities, item)
	}
	if err := cityRows.Err(); err != nil {
		return result, err
	}

	statusRows, err := s.db.QueryContext(ctx, `
SELECT DISTINCT COALESCE(NULLIF(status, ''), 'unknown') AS status
FROM conn_events
WHERE start_ts >= ?
  AND (? = '' OR rule_id = ?)
  AND (? = '' OR province LIKE '%' || ? || '%')
  AND (? = '' OR city LIKE '%' || ? || '%')
ORDER BY status
`, windowStart, ruleID, ruleID, province, province, city, city)
	if err != nil {
		return result, err
	}
	defer statusRows.Close()
	for statusRows.Next() {
		var v string
		if err := statusRows.Scan(&v); err != nil {
			return result, err
		}
		result.Statuses = append(result.Statuses, v)
	}
	if err := statusRows.Err(); err != nil {
		return result, err
	}

	return result, nil
}

func (s *Service) Analytics(ctx context.Context, q AnalyticsQuery) (AnalyticsResult, error) {
	result := AnalyticsResult{}
	windowStart := s.windowStartMS(q.Window)
	ruleID := q.RuleID
	province := q.Province
	city := q.City
	status := q.Status
	topN := q.TopN
	if topN <= 0 {
		topN = 10
	}

	err := s.db.QueryRowContext(ctx, `
SELECT
	COUNT(*) AS conn_count,
	COALESCE(SUM(total_bytes), 0) AS total_bytes,
	COALESCE(AVG(duration_ms), 0) AS avg_duration_ms,
	COUNT(DISTINCT CASE WHEN rule_id != '' THEN rule_id END) AS active_rules,
	COUNT(DISTINCT CASE WHEN city != '' THEN COALESCE(NULLIF(province, ''), '未知') || '-' || city END) AS active_cities
FROM conn_events
WHERE start_ts >= ?
  AND (? = '' OR rule_id = ?)
  AND (? = '' OR province LIKE '%' || ? || '%')
  AND (? = '' OR city LIKE '%' || ? || '%')
  AND (? = '' OR status = ?)
`, windowStart, ruleID, ruleID, province, province, city, city, status, status).Scan(
		&result.Overview.ConnCount,
		&result.Overview.TotalBytes,
		&result.Overview.AvgDurationMS,
		&result.Overview.ActiveRules,
		&result.Overview.ActiveCities,
	)
	if err != nil {
		return result, err
	}

	cityRows, err := s.db.QueryContext(ctx, `
SELECT
	CASE
		WHEN COALESCE(NULLIF(province, ''), '') = '' AND COALESCE(NULLIF(city, ''), '') = '' THEN '未知城市'
		ELSE COALESCE(NULLIF(province, ''), '未知') || COALESCE(NULLIF(city, ''), '')
	END AS name,
	COUNT(*) AS conn_count,
	COALESCE(SUM(total_bytes), 0) AS total_bytes
FROM conn_events
WHERE start_ts >= ?
  AND (? = '' OR rule_id = ?)
  AND (? = '' OR province LIKE '%' || ? || '%')
  AND (? = '' OR city LIKE '%' || ? || '%')
  AND (? = '' OR status = ?)
GROUP BY COALESCE(NULLIF(province, ''), '未知'), COALESCE(NULLIF(city, ''), '')
ORDER BY total_bytes DESC, conn_count DESC
LIMIT ?
`, windowStart, ruleID, ruleID, province, province, city, city, status, status, topN)
	if err != nil {
		return result, err
	}
	defer cityRows.Close()

	for cityRows.Next() {
		var item AnalyticsBucket
		if err := cityRows.Scan(&item.Name, &item.ConnCount, &item.TotalBytes); err != nil {
			return result, err
		}
		result.TopCities = append(result.TopCities, item)
	}
	if err := cityRows.Err(); err != nil {
		return result, err
	}

	ruleRows, err := s.db.QueryContext(ctx, `
SELECT
	COALESCE(NULLIF(rule_id, ''), 'unknown') AS name,
	COUNT(*) AS conn_count,
	COALESCE(SUM(total_bytes), 0) AS total_bytes
FROM conn_events
WHERE start_ts >= ?
  AND (? = '' OR rule_id = ?)
  AND (? = '' OR province LIKE '%' || ? || '%')
  AND (? = '' OR city LIKE '%' || ? || '%')
  AND (? = '' OR status = ?)
GROUP BY COALESCE(NULLIF(rule_id, ''), 'unknown')
ORDER BY total_bytes DESC, conn_count DESC
LIMIT ?
`, windowStart, ruleID, ruleID, province, province, city, city, status, status, topN)
	if err != nil {
		return result, err
	}
	defer ruleRows.Close()

	for ruleRows.Next() {
		var item AnalyticsBucket
		if err := ruleRows.Scan(&item.Name, &item.ConnCount, &item.TotalBytes); err != nil {
			return result, err
		}
		result.TopRules = append(result.TopRules, item)
	}
	if err := ruleRows.Err(); err != nil {
		return result, err
	}

	return result, nil
}

func (s *Service) windowStartMS(window time.Duration) int64 {
	if window <= 0 {
		return 0
	}
	return s.now().Add(-window).UnixMilli()
}
