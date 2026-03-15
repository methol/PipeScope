package http

import (
	"io/fs"
	"log"
	nethttp "net/http"
	"path"
	"sort"
	"strconv"
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
	mux.HandleFunc("/api/analytics/options", h.handleAnalyticsOptions)
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
			if tryServePrecompressed(w, r, staticFS, trimmed) {
				return
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		r2 := r.Clone(r.Context())
		r2.URL.Path = "/index.html"
		fileServer.ServeHTTP(w, r2)
	})
}

func tryServePrecompressed(w nethttp.ResponseWriter, r *nethttp.Request, staticFS fs.FS, trimmed string) bool {
	encodings := parseAcceptedEncodings(r.Header.Get("Accept-Encoding"))
	for _, enc := range encodings {
		var suffix string
		switch enc {
		case "br":
			suffix = ".br"
		case "gzip":
			suffix = ".gz"
		default:
			continue
		}
		if _, err := fs.Stat(staticFS, trimmed+suffix); err != nil {
			continue
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/" + trimmed + suffix
		w.Header().Set("Content-Encoding", enc)
		w.Header().Set("Vary", "Accept-Encoding")
		if strings.HasSuffix(trimmed, ".geojson") {
			w.Header().Set("Content-Type", "application/geo+json; charset=utf-8")
		}
		nethttp.FileServer(nethttp.FS(staticFS)).ServeHTTP(w, r2)
		return true
	}
	return false
}

func parseAcceptedEncodings(header string) []string {
	if header == "" {
		return nil
	}
	type pref struct {
		name string
		q    float64
		ord  int
	}
	parts := strings.Split(header, ",")
	prefs := make([]pref, 0, len(parts))
	for i, raw := range parts {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		name := item
		q := 1.0
		if semi := strings.Index(item, ";"); semi >= 0 {
			name = strings.TrimSpace(item[:semi])
			params := strings.Split(item[semi+1:], ";")
			for _, p := range params {
				p = strings.TrimSpace(p)
				if strings.HasPrefix(strings.ToLower(p), "q=") {
					if v, err := strconv.ParseFloat(strings.TrimSpace(p[2:]), 64); err == nil {
						q = v
					}
				}
			}
		}
		name = strings.ToLower(name)
		if q <= 0 {
			continue
		}
		switch name {
		case "br", "gzip":
			prefs = append(prefs, pref{name: name, q: q, ord: i})
		case "*":
			prefs = append(prefs, pref{name: "br", q: q, ord: i})
			prefs = append(prefs, pref{name: "gzip", q: q, ord: i})
		}
	}
	if len(prefs) == 0 {
		return nil
	}
	sort.SliceStable(prefs, func(i, j int) bool {
		if prefs[i].q == prefs[j].q {
			return prefs[i].ord < prefs[j].ord
		}
		return prefs[i].q > prefs[j].q
	})
	result := make([]string, 0, len(prefs))
	seen := map[string]bool{}
	for _, p := range prefs {
		if seen[p.name] {
			continue
		}
		seen[p.name] = true
		result = append(result, p.name)
	}
	return result
}
