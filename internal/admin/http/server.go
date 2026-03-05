package http

import (
	nethttp "net/http"
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

	return &Server{mux: mux}
}

func (s *Server) Handler() nethttp.Handler {
	return s.mux
}

