package areacity

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestHTTPMatcherMatchAndCache(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"c":0,"v":{"list":[{"id":"4403","ext_path":"广东省 深圳市","deep":"1","name":"深圳市","geo_wkt":"POINT (114.057939 22.543527)"}]},"m":""}`))
	}))
	defer srv.Close()

	m := NewHTTPMatcher(srv.URL, 0)
	row, ok, err := m.Match("广东", "深圳")
	if err != nil {
		t.Fatalf("match error: %v", err)
	}
	if !ok {
		t.Fatalf("expected match")
	}
	if row.Adcode != "4403" {
		t.Fatalf("unexpected adcode: %s", row.Adcode)
	}
	if row.Lat == 0 || row.Lng == 0 {
		t.Fatalf("unexpected geo: %+v", row)
	}

	row2, ok2, err2 := m.Match("广东", "深圳")
	if err2 != nil || !ok2 || row2.Adcode != "4403" {
		t.Fatalf("second match failed: ok=%v err=%v row=%+v", ok2, err2, row2)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("expected cached result, hits=%d", hits)
	}
}

func TestHTTPMatcherReturnsErrorWhenAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"c":1,"v":null,"m":"boom"}`))
	}))
	defer srv.Close()

	m := NewHTTPMatcher(srv.URL, 0)
	_, _, err := m.Match("广东", "深圳")
	if err == nil {
		t.Fatalf("expected error")
	}
}
