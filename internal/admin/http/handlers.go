package http

import (
	"context"
	"encoding/json"
	"errors"
	nethttp "net/http"
	"strconv"
	"time"

	"pipescope/internal/admin/service"
)

type QueryService interface {
	ChinaMap(ctx context.Context, q service.MapQuery) ([]service.MapPoint, error)
	Rules(ctx context.Context, q service.RulesQuery) ([]service.RulePoint, error)
	Sessions(ctx context.Context, q service.SessionsQuery) ([]service.SessionItem, error)
	Overview(ctx context.Context, window time.Duration) (service.Overview, error)
	ProvinceMap(ctx context.Context, q service.ProvinceQuery) ([]service.MapPoint, error)
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

func parseWindow(raw string, fallback time.Duration) time.Duration {
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
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
