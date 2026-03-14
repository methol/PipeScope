package http

import (
	"context"
	"fmt"
	nethttp "net/http"
	"net/http/httptest"
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
