package http

import (
	"context"
	"fmt"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pipescope/internal/admin/service"
)

func TestGetHealth(t *testing.T) {
	srv := newTestServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(nethttp.MethodGet, "/api/health", nil)
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != nethttp.StatusOK {
		t.Fatalf("code=%d", rr.Code)
	}
}

func TestSessionsEndpointClampsNegativePagination(t *testing.T) {
	svc := &capturingService{}
	srv := NewServer(svc, 50*time.Millisecond)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(nethttp.MethodGet, "/api/sessions?limit=-5&offset=-9", nil)

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != nethttp.StatusOK {
		t.Fatalf("code=%d", rr.Code)
	}
	if svc.sessionsQuery.Limit != 100 {
		t.Fatalf("Limit=%d want=100", svc.sessionsQuery.Limit)
	}
	if svc.sessionsQuery.Offset != 0 {
		t.Fatalf("Offset=%d want=0", svc.sessionsQuery.Offset)
	}
}

func TestSessionsEndpointClampsOversizedLimit(t *testing.T) {
	svc := &capturingService{}
	srv := NewServer(svc, 50*time.Millisecond)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(nethttp.MethodGet, "/api/sessions?limit=9999&offset=3", nil)

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != nethttp.StatusOK {
		t.Fatalf("code=%d", rr.Code)
	}
	if svc.sessionsQuery.Limit != 500 {
		t.Fatalf("Limit=%d want=500", svc.sessionsQuery.Limit)
	}
	if svc.sessionsQuery.Offset != 3 {
		t.Fatalf("Offset=%d want=3", svc.sessionsQuery.Offset)
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	return NewServer(fakeService{}, 50*time.Millisecond)
}

type fakeService struct{}

type capturingService struct {
	fakeService
	sessionsQuery service.SessionsQuery
}

func (fakeService) ChinaMap(context.Context, service.MapQuery) ([]service.MapPoint, error) {
	return []service.MapPoint{}, nil
}

func (fakeService) Rules(context.Context, service.RulesQuery) ([]service.RulePoint, error) {
	return []service.RulePoint{}, nil
}

func (fakeService) Sessions(context.Context, service.SessionsQuery) ([]service.SessionItem, error) {
	return []service.SessionItem{}, nil
}

func (c *capturingService) Sessions(_ context.Context, q service.SessionsQuery) ([]service.SessionItem, error) {
	c.sessionsQuery = q
	return []service.SessionItem{}, nil
}

func (fakeService) Overview(context.Context, time.Duration) (service.Overview, error) {
	return service.Overview{}, nil
}

func (fakeService) ProvinceMap(context.Context, service.ProvinceQuery) ([]service.MapPoint, error) {
	return []service.MapPoint{}, nil
}

func (fakeService) ProvinceSummary(context.Context, service.MapQuery) ([]service.ProvinceSummaryPoint, error) {
	return []service.ProvinceSummaryPoint{}, nil
}

func (fakeService) Analytics(context.Context, service.AnalyticsQuery) (service.AnalyticsResult, error) {
	return service.AnalyticsResult{}, nil
}

func (fakeService) AnalyticsOptions(context.Context, service.AnalyticsOptionsQuery) (service.AnalyticsOptions, error) {
	return service.AnalyticsOptions{}, nil
}

func (fakeService) SessionsOptions(context.Context, service.SessionsOptionsQuery) (service.SessionsOptions, error) {
	return service.SessionsOptions{}, nil
}

type timeoutService struct{}

type wrappedTimeoutService struct{}

func (timeoutService) ChinaMap(ctx context.Context, q service.MapQuery) ([]service.MapPoint, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (timeoutService) Rules(ctx context.Context, q service.RulesQuery) ([]service.RulePoint, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (timeoutService) Sessions(ctx context.Context, q service.SessionsQuery) ([]service.SessionItem, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (timeoutService) Overview(ctx context.Context, window time.Duration) (service.Overview, error) {
	<-ctx.Done()
	return service.Overview{}, ctx.Err()
}

func (timeoutService) ProvinceMap(ctx context.Context, q service.ProvinceQuery) ([]service.MapPoint, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (timeoutService) ProvinceSummary(ctx context.Context, q service.MapQuery) ([]service.ProvinceSummaryPoint, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (timeoutService) Analytics(ctx context.Context, q service.AnalyticsQuery) (service.AnalyticsResult, error) {
	<-ctx.Done()
	return service.AnalyticsResult{}, ctx.Err()
}

func (wrappedTimeoutService) ChinaMap(ctx context.Context, q service.MapQuery) ([]service.MapPoint, error) {
	<-ctx.Done()
	return nil, fmt.Errorf("wrapped: %w", ctx.Err())
}

func (wrappedTimeoutService) Rules(ctx context.Context, q service.RulesQuery) ([]service.RulePoint, error) {
	<-ctx.Done()
	return nil, fmt.Errorf("wrapped: %w", ctx.Err())
}

func (wrappedTimeoutService) Sessions(ctx context.Context, q service.SessionsQuery) ([]service.SessionItem, error) {
	<-ctx.Done()
	return nil, fmt.Errorf("wrapped: %w", ctx.Err())
}

func (wrappedTimeoutService) Overview(ctx context.Context, window time.Duration) (service.Overview, error) {
	<-ctx.Done()
	return service.Overview{}, fmt.Errorf("wrapped: %w", ctx.Err())
}

func (wrappedTimeoutService) ProvinceMap(ctx context.Context, q service.ProvinceQuery) ([]service.MapPoint, error) {
	<-ctx.Done()
	return nil, fmt.Errorf("wrapped: %w", ctx.Err())
}

func (wrappedTimeoutService) ProvinceSummary(ctx context.Context, q service.MapQuery) ([]service.ProvinceSummaryPoint, error) {
	<-ctx.Done()
	return nil, fmt.Errorf("wrapped: %w", ctx.Err())
}

func (wrappedTimeoutService) Analytics(ctx context.Context, q service.AnalyticsQuery) (service.AnalyticsResult, error) {
	<-ctx.Done()
	return service.AnalyticsResult{}, fmt.Errorf("wrapped: %w", ctx.Err())
}

func (timeoutService) AnalyticsOptions(ctx context.Context, q service.AnalyticsOptionsQuery) (service.AnalyticsOptions, error) {
	<-ctx.Done()
	return service.AnalyticsOptions{}, ctx.Err()
}

func (timeoutService) SessionsOptions(ctx context.Context, q service.SessionsOptionsQuery) (service.SessionsOptions, error) {
	<-ctx.Done()
	return service.SessionsOptions{}, ctx.Err()
}

func (wrappedTimeoutService) AnalyticsOptions(ctx context.Context, q service.AnalyticsOptionsQuery) (service.AnalyticsOptions, error) {
	<-ctx.Done()
	return service.AnalyticsOptions{}, fmt.Errorf("wrapped: %w", ctx.Err())
}

func (wrappedTimeoutService) SessionsOptions(ctx context.Context, q service.SessionsOptionsQuery) (service.SessionsOptions, error) {
	<-ctx.Done()
	return service.SessionsOptions{}, fmt.Errorf("wrapped: %w", ctx.Err())
}

func TestSessionsEndpointClampsOversizedOffset(t *testing.T) {
	svc := &capturingService{}
	srv := NewServer(svc, 50*time.Millisecond)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(nethttp.MethodGet, "/api/sessions?limit=5&offset=999999999", nil)

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != nethttp.StatusOK {
		t.Fatalf("code=%d", rr.Code)
	}
	if svc.sessionsQuery.Limit != 5 {
		t.Fatalf("Limit=%d want=5", svc.sessionsQuery.Limit)
	}
	if svc.sessionsQuery.Offset != 1000000 {
		t.Fatalf("Offset=%d want=1000000", svc.sessionsQuery.Offset)
	}
}

func TestRulesEndpointTimesOutSlowService(t *testing.T) {
	srv := NewServer(timeoutService{}, 10*time.Millisecond)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(nethttp.MethodGet, "/api/rules", nil)

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != nethttp.StatusGatewayTimeout {
		t.Fatalf("code=%d want=%d", rr.Code, nethttp.StatusGatewayTimeout)
	}
}

func TestRulesEndpointTimesOutWrappedServiceError(t *testing.T) {
	srv := NewServer(wrappedTimeoutService{}, 10*time.Millisecond)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(nethttp.MethodGet, "/api/rules", nil)

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != nethttp.StatusGatewayTimeout {
		t.Fatalf("code=%d want=%d", rr.Code, nethttp.StatusGatewayTimeout)
	}
}

func TestStaticGeoJSONServesBrotliWhenPreferred(t *testing.T) {
	srv := newTestServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(nethttp.MethodGet, "/maps/china-cities.geojson", nil)
	req.Header.Set("Accept-Encoding", "gzip;q=0.8, br;q=1")

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != nethttp.StatusOK {
		t.Fatalf("code=%d want=%d", rr.Code, nethttp.StatusOK)
	}
	if got := rr.Header().Get("Content-Encoding"); got != "br" {
		t.Fatalf("Content-Encoding=%q want=br", got)
	}
	if got := rr.Header().Get("Vary"); !strings.Contains(got, "Accept-Encoding") {
		t.Fatalf("Vary=%q want contains Accept-Encoding", got)
	}
	if got := rr.Header().Get("Content-Type"); !strings.Contains(got, "application/geo+json") {
		t.Fatalf("Content-Type=%q want geo+json", got)
	}
}

func TestStaticGeoJSONSkipsQZeroEncoding(t *testing.T) {
	srv := newTestServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(nethttp.MethodGet, "/maps/china-cities.geojson", nil)
	req.Header.Set("Accept-Encoding", "br;q=0, gzip;q=1")

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != nethttp.StatusOK {
		t.Fatalf("code=%d want=%d", rr.Code, nethttp.StatusOK)
	}
	if got := rr.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding=%q want=gzip", got)
	}
}
