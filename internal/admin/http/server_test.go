package http

import (
	"context"
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
	srv := NewServer(svc)
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
	srv := NewServer(svc)
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
	return NewServer(fakeService{})
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

