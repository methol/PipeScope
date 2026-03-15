package service

import "time"

const (
	MetricConn  = "conn"
	MetricBytes = "bytes"
)

type MapQuery struct {
	Window time.Duration
	Metric string
}

type RulesQuery struct {
	Window time.Duration
}

type SessionsQuery struct {
	Window time.Duration
	RuleID string
	Limit  int
	Offset int
}

type AnalyticsQuery struct {
	Window   time.Duration
	RuleID   string
	Province string
	City     string
	Status   string
	TopN     int
}

type ProvinceQuery struct {
	Window   time.Duration
	Metric   string
	Province string
}

type MapPoint struct {
	Adcode   string  `json:"adcode"`
	Province string  `json:"province"`
	City     string  `json:"city"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Value    int64   `json:"value"`
}

type ProvinceSummaryPoint struct {
	Province string `json:"province"`
	Value    int64  `json:"value"`
}

type RulePoint struct {
	RuleID     string `json:"rule_id"`
	ConnCount  int64  `json:"conn_count"`
	TotalBytes int64  `json:"total_bytes"`
}

type SessionItem struct {
	ID         int64  `json:"id"`
	RuleID     string `json:"rule_id"`
	SrcAddr    string `json:"src_addr"`
	DstAddr    string `json:"dst_addr"`
	Status     string `json:"status"`
	UpBytes    int64  `json:"up_bytes"`
	DownBytes  int64  `json:"down_bytes"`
	TotalBytes int64  `json:"total_bytes"`
	StartTS    int64  `json:"start_ts"`
	EndTS      int64  `json:"end_ts"`
	DurationMS int64  `json:"duration_ms"`
	Province   string `json:"province"`
	City       string `json:"city"`
	Adcode     string `json:"adcode"`
}

type Overview struct {
	ConnCount  int64 `json:"conn_count"`
	TotalBytes int64 `json:"total_bytes"`
}

type AnalyticsOverview struct {
	ConnCount     int64 `json:"conn_count"`
	TotalBytes    int64 `json:"total_bytes"`
	AvgDurationMS int64 `json:"avg_duration_ms"`
	ActiveRules   int64 `json:"active_rules"`
	ActiveCities  int64 `json:"active_cities"`
}

type AnalyticsBucket struct {
	Name       string `json:"name"`
	ConnCount  int64  `json:"conn_count"`
	TotalBytes int64  `json:"total_bytes"`
}

type AnalyticsResult struct {
	Overview  AnalyticsOverview `json:"overview"`
	TopCities []AnalyticsBucket `json:"top_cities"`
	TopRules  []AnalyticsBucket `json:"top_rules"`
}
