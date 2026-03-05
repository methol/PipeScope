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

func newTestServer(t *testing.T) *Server {
	t.Helper()
	return NewServer(fakeService{})
}

type fakeService struct{}

func (fakeService) ChinaMap(context.Context, service.MapQuery) ([]service.MapPoint, error) {
	return []service.MapPoint{}, nil
}

func (fakeService) Rules(context.Context, service.RulesQuery) ([]service.RulePoint, error) {
	return []service.RulePoint{}, nil
}

func (fakeService) Sessions(context.Context, service.SessionsQuery) ([]service.SessionItem, error) {
	return []service.SessionItem{}, nil
}

func (fakeService) Overview(context.Context, time.Duration) (service.Overview, error) {
	return service.Overview{}, nil
}

func (fakeService) ProvinceMap(context.Context, service.ProvinceQuery) ([]service.MapPoint, error) {
	return []service.MapPoint{}, nil
}

