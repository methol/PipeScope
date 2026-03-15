package http

import (
	"io/fs"
	"log"
	nethttp "net/http"
	"path"
	"strings"
	"time"
)

type Server struct {
	mux nethttp.Handler
}

func NewServer(svc QueryService, timeout time.Duration) *Server {
	mux := nethttp.NewServeMux()
	h := newHandlers(svc, timeout)

	mux.HandleFunc("/api/health", h.handleHealth)
	mux.HandleFunc("/api/map/china", h.handleMapChina)
	mux.HandleFunc("/api/map/province", h.handleMapProvince)
	mux.HandleFunc("/api/map/province-summary", h.handleMapProvinceSummary)
	mux.HandleFunc("/api/rules", h.handleRules)
	mux.HandleFunc("/api/sessions", h.handleSessions)
	mux.HandleFunc("/api/overview", h.handleOverview)
	mux.HandleFunc("/api/analytics", h.handleAnalytics)
	mux.Handle("/", staticHandler())

	return &Server{mux: loggingMiddleware(mux)}
}

func loggingMiddleware(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		start := time.Now()
		log.Printf("admin http start method=%s path=%s remote=%s", r.Method, r.URL.Path, r.RemoteAddr)
		defer func() {
			log.Printf("admin http done method=%s path=%s dur=%s", r.Method, r.URL.Path, time.Since(start))
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Handler() nethttp.Handler {
	return s.mux
}

func staticHandler() nethttp.Handler {
	staticFS := EmbeddedStaticFS()
	fileServer := nethttp.FileServer(nethttp.FS(staticFS))

	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		cleanPath := path.Clean(r.URL.Path)
		trimmed := strings.TrimPrefix(cleanPath, "/")
		if trimmed == "" {
			trimmed = "index.html"
		}
		if _, err := fs.Stat(staticFS, trimmed); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		r2 := r.Clone(r.Context())
		r2.URL.Path = "/index.html"
		fileServer.ServeHTTP(w, r2)
	})
}
