package http

import (
	"context"
	"encoding/json"
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
	svc QueryService
}

func newHandlers(svc QueryService) *handlers {
	return &handlers{svc: svc}
}

func (h *handlers) handleHealth(w nethttp.ResponseWriter, _ *nethttp.Request) {
	writeJSON(w, nethttp.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) handleMapChina(w nethttp.ResponseWriter, r *nethttp.Request) {
	points, err := h.svc.ChinaMap(r.Context(), service.MapQuery{
		Window: parseWindow(r.URL.Query().Get("window"), 15*time.Minute),
		Metric: parseMetric(r.URL.Query().Get("metric")),
	})
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, map[string]any{"items": points})
}

func (h *handlers) handleMapProvince(w nethttp.ResponseWriter, r *nethttp.Request) {
	points, err := h.svc.ProvinceMap(r.Context(), service.ProvinceQuery{
		Window:   parseWindow(r.URL.Query().Get("window"), 15*time.Minute),
		Metric:   parseMetric(r.URL.Query().Get("metric")),
		Province: r.URL.Query().Get("province"),
	})
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, map[string]any{"items": points})
}

func (h *handlers) handleRules(w nethttp.ResponseWriter, r *nethttp.Request) {
	points, err := h.svc.Rules(r.Context(), service.RulesQuery{
		Window: parseWindow(r.URL.Query().Get("window"), 15*time.Minute),
	})
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, map[string]any{"items": points})
}

func (h *handlers) handleSessions(w nethttp.ResponseWriter, r *nethttp.Request) {
	points, err := h.svc.Sessions(r.Context(), service.SessionsQuery{
		Window: parseWindow(r.URL.Query().Get("window"), 15*time.Minute),
		RuleID: r.URL.Query().Get("rule_id"),
		Limit:  parseInt(r.URL.Query().Get("limit"), 100),
		Offset: parseInt(r.URL.Query().Get("offset"), 0),
	})
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, map[string]any{"items": points})
}

func (h *handlers) handleOverview(w nethttp.ResponseWriter, r *nethttp.Request) {
	o, err := h.svc.Overview(
		r.Context(),
		parseWindow(r.URL.Query().Get("window"), 15*time.Minute),
	)
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, err)
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

func parseInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func writeError(w nethttp.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]any{
		"error": err.Error(),
	})
}

func writeJSON(w nethttp.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
