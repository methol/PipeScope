package http

import (
	"context"
	"encoding/json"
	"errors"
	nethttp "net/http"
	"strconv"
	"strings"
	"time"

	"pipescope/internal/admin/service"
)

type QueryService interface {
	ChinaMap(ctx context.Context, q service.MapQuery) ([]service.MapPoint, error)
	Rules(ctx context.Context, q service.RulesQuery) ([]service.RulePoint, error)
	Sessions(ctx context.Context, q service.SessionsQuery) ([]service.SessionItem, error)
	SessionsOptions(ctx context.Context, q service.SessionsOptionsQuery) (service.SessionsOptions, error)
	Overview(ctx context.Context, window time.Duration) (service.Overview, error)
	ProvinceMap(ctx context.Context, q service.ProvinceQuery) ([]service.MapPoint, error)
	ProvinceSummary(ctx context.Context, q service.MapQuery) ([]service.ProvinceSummaryPoint, error)
	Analytics(ctx context.Context, q service.AnalyticsQuery) (service.AnalyticsResult, error)
	AnalyticsOptions(ctx context.Context, q service.AnalyticsOptionsQuery) (service.AnalyticsOptions, error)
}

type handlers struct {
	svc     QueryService
	timeout time.Duration
}

func newHandlers(svc QueryService, timeout time.Duration) *handlers {
	return &handlers{svc: svc, timeout: timeout}
}

func (h *handlers) handleHealth(w nethttp.ResponseWriter, _ *nethttp.Request) {
	writeJSON(w, nethttp.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) handleMapChina(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx, cancel := h.queryContext(r.Context())
	defer cancel()
	points, err := h.svc.ChinaMap(ctx, service.MapQuery{
		Window: parseWindow(r.URL.Query().Get("window"), 15*time.Minute),
		Metric: parseMetric(r.URL.Query().Get("metric")),
		Limit:  parseBoundedInt(r.URL.Query().Get("limit"), 100, 1, 1000),
		RuleID: r.URL.Query().Get("rule_id"),
		Status: r.URL.Query().Get("status"),
	})
	if err != nil {
		writeQueryError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, map[string]any{"items": points})
}

func (h *handlers) handleMapProvince(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx, cancel := h.queryContext(r.Context())
	defer cancel()
	points, err := h.svc.ProvinceMap(ctx, service.ProvinceQuery{
		Window:   parseWindow(r.URL.Query().Get("window"), 15*time.Minute),
		Metric:   parseMetric(r.URL.Query().Get("metric")),
		Province: r.URL.Query().Get("province"),
	})
	if err != nil {
		writeQueryError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, map[string]any{"items": points})
}

func (h *handlers) handleMapProvinceSummary(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx, cancel := h.queryContext(r.Context())
	defer cancel()
	points, err := h.svc.ProvinceSummary(ctx, service.MapQuery{
		Window: parseWindow(r.URL.Query().Get("window"), 15*time.Minute),
		Metric: parseMetric(r.URL.Query().Get("metric")),
	})
	if err != nil {
		writeQueryError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, map[string]any{"items": points})
}

func (h *handlers) handleRules(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx, cancel := h.queryContext(r.Context())
	defer cancel()
	points, err := h.svc.Rules(ctx, service.RulesQuery{
		Window: parseWindow(r.URL.Query().Get("window"), 15*time.Minute),
	})
	if err != nil {
		writeQueryError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, map[string]any{"items": points})
}

func (h *handlers) handleSessions(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx, cancel := h.queryContext(r.Context())
	defer cancel()
	points, err := h.svc.Sessions(ctx, service.SessionsQuery{
		Window: parseWindow(r.URL.Query().Get("window"), 15*time.Minute),
		RuleID: r.URL.Query().Get("rule_id"),
		Limit:  parseBoundedInt(r.URL.Query().Get("limit"), 100, 1, 500),
		Offset: parseBoundedInt(r.URL.Query().Get("offset"), 0, 0, 1000000),
	})
	if err != nil {
		writeQueryError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, map[string]any{"items": points})
}

func (h *handlers) handleSessionsOptions(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx, cancel := h.queryContext(r.Context())
	defer cancel()
	result, err := h.svc.SessionsOptions(ctx, service.SessionsOptionsQuery{
		Window: parseWindow(r.URL.Query().Get("window"), 15*time.Minute),
	})
	if err != nil {
		writeQueryError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, result)
}

func (h *handlers) handleOverview(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx, cancel := h.queryContext(r.Context())
	defer cancel()
	o, err := h.svc.Overview(
		ctx,
		parseWindow(r.URL.Query().Get("window"), 15*time.Minute),
	)
	if err != nil {
		writeQueryError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, o)
}

func (h *handlers) handleAnalytics(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx, cancel := h.queryContext(r.Context())
	defer cancel()
	result, err := h.svc.Analytics(ctx, service.AnalyticsQuery{
		Window:   parseWindow(r.URL.Query().Get("window"), 15*time.Minute),
		RuleID:   r.URL.Query().Get("rule_id"),
		Province: r.URL.Query().Get("province"),
		City:     r.URL.Query().Get("city"),
		Status:   r.URL.Query().Get("status"),
		TopN:     parseBoundedInt(r.URL.Query().Get("top_n"), 10, 1, 1000),
	})
	if err != nil {
		writeQueryError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, result)
}

func (h *handlers) handleAnalyticsOptions(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx, cancel := h.queryContext(r.Context())
	defer cancel()
	result, err := h.svc.AnalyticsOptions(ctx, service.AnalyticsOptionsQuery{
		Window:   parseWindow(r.URL.Query().Get("window"), 15*time.Minute),
		RuleID:   r.URL.Query().Get("rule_id"),
		Province: r.URL.Query().Get("province"),
		City:     r.URL.Query().Get("city"),
		Status:   r.URL.Query().Get("status"),
	})
	if err != nil {
		writeQueryError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, result)
}

func parseWindow(raw string, fallback time.Duration) time.Duration {
	if raw == "" {
		return fallback
	}

	raw = strings.TrimSpace(strings.ToLower(raw))
	if d, ok := parseFriendlyWindow(raw); ok {
		return d
	}

	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func parseFriendlyWindow(raw string) (time.Duration, bool) {
	if len(raw) < 2 {
		return 0, false
	}

	parse := func(unit string, mul time.Duration) (time.Duration, bool) {
		if !strings.HasSuffix(raw, unit) {
			return 0, false
		}
		n, err := strconv.Atoi(strings.TrimSuffix(raw, unit))
		if err != nil || n <= 0 {
			return 0, false
		}
		return time.Duration(n) * mul, true
	}

	if d, ok := parse("d", 24*time.Hour); ok {
		return d, true
	}
	if d, ok := parse("w", 7*24*time.Hour); ok {
		return d, true
	}
	if d, ok := parse("mo", 30*24*time.Hour); ok {
		return d, true
	}
	return 0, false
}

func parseMetric(raw string) string {
	if raw == service.MetricBytes {
		return service.MetricBytes
	}
	return service.MetricConn
}

func parseBoundedInt(raw string, fallback int, min int, max int) int {
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if n < min {
		return fallback
	}
	if max > 0 && n > max {
		return max
	}
	return n
}

func (h *handlers) queryContext(parent context.Context) (context.Context, context.CancelFunc) {
	if h.timeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, h.timeout)
}

func writeQueryError(w nethttp.ResponseWriter, err error) {
	if err == nil {
		writeJSON(w, nethttp.StatusInternalServerError, map[string]any{"error": "unknown error"})
		return
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		writeJSON(w, nethttp.StatusGatewayTimeout, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, nethttp.StatusInternalServerError, map[string]any{
		"error": err.Error(),
	})
}

func writeJSON(w nethttp.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
