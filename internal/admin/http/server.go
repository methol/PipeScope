package http

import (
	"io/fs"
	nethttp "net/http"
	"path"
	"strings"
)

type Server struct {
	mux *nethttp.ServeMux
}

func NewServer(svc QueryService) *Server {
	mux := nethttp.NewServeMux()
	h := newHandlers(svc)

	mux.HandleFunc("/api/health", h.handleHealth)
	mux.HandleFunc("/api/map/china", h.handleMapChina)
	mux.HandleFunc("/api/map/province", h.handleMapProvince)
	mux.HandleFunc("/api/rules", h.handleRules)
	mux.HandleFunc("/api/sessions", h.handleSessions)
	mux.HandleFunc("/api/overview", h.handleOverview)
	mux.Handle("/", staticHandler())

	return &Server{mux: mux}
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
