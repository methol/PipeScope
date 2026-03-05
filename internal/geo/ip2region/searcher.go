package ip2region

import (
	"errors"
	"fmt"
	"strings"

	ip2service "github.com/lionsoul2014/ip2region/binding/golang/service"

	"pipescope/internal/geo/normalize"
)

var ErrLookupNotImplemented = errors.New("ip2region lookup is not configured")

type Region struct {
	Country  string
	Province string
	City     string
	ISP      string
	Code     string
}

type Searcher struct {
	service  *ip2service.Ip2Region
	lookupFn func(ip string) (string, error)
}

type Config struct {
	V4XDBPath   string
	V6XDBPath   string
	CachePolicy string
	Searchers   int
}

func NewSearcher(xdbPath string) (*Searcher, error) {
	return NewSearcherWithConfig(Config{V4XDBPath: xdbPath})
}

func NewSearcherWithConfig(cfg Config) (*Searcher, error) {
	cachePolicy, err := parseCachePolicy(cfg.CachePolicy)
	if err != nil {
		return nil, err
	}

	searchers := cfg.Searchers
	if searchers <= 0 {
		searchers = 20
	}

	v4Path := strings.TrimSpace(cfg.V4XDBPath)
	v6Path := strings.TrimSpace(cfg.V6XDBPath)
	if v4Path == "" && v6Path == "" {
		return &Searcher{}, nil
	}

	var v4Config *ip2service.Config
	if v4Path != "" {
		v4Config, err = ip2service.NewV4Config(cachePolicy, v4Path, searchers)
		if err != nil {
			return nil, fmt.Errorf("init ip2region v4 config: %w", err)
		}
	}

	var v6Config *ip2service.Config
	if v6Path != "" {
		v6Config, err = ip2service.NewV6Config(cachePolicy, v6Path, searchers)
		if err != nil {
			return nil, fmt.Errorf("init ip2region v6 config: %w", err)
		}
	}

	svc, err := ip2service.NewIp2Region(v4Config, v6Config)
	if err != nil {
		return nil, fmt.Errorf("create ip2region service: %w", err)
	}
	return &Searcher{service: svc}, nil
}

func parseCachePolicy(policy string) (int, error) {
	p := strings.TrimSpace(policy)
	if p == "" {
		p = "vindex"
	}
	val, err := ip2service.CachePolicyFromName(p)
	if err != nil {
		return 0, fmt.Errorf("parse ip2region cache policy %q: %w", policy, err)
	}
	return val, nil
}

func (s *Searcher) Close() {
	if s == nil {
		return
	}
	if s.service != nil {
		s.service.Close()
	}
}

func (s *Searcher) SetLookupFn(fn func(ip string) (string, error)) {
	s.lookupFn = fn
}

func (s *Searcher) Lookup(ip string) (Region, error) {
	if s.lookupFn == nil {
		if s.service == nil {
			return Region{}, ErrLookupNotImplemented
		}
		raw, err := s.service.SearchByStr(ip)
		if err != nil {
			return Region{}, err
		}
		return ParseRegion(raw), nil
	}
	raw, err := s.lookupFn(ip)
	if err != nil {
		return Region{}, err
	}
	return ParseRegion(raw), nil
}

func ParseRegion(raw string) Region {
	parts := strings.Split(raw, "|")
	for len(parts) < 5 {
		parts = append(parts, "")
	}

	province := normalize.NormalizeProvince(cleanRegionPart(parts[1]))
	city := normalize.NormalizeCity(cleanRegionPart(parts[2]))
	if city == "" {
		city = province
	}

	return Region{
		Country:  cleanRegionPart(parts[0]),
		Province: province,
		City:     city,
		ISP:      cleanRegionPart(parts[3]),
		Code:     strings.ToUpper(cleanRegionPart(parts[4])),
	}
}

func cleanRegionPart(s string) string {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "", "0", "null", "nil", "n/a", "unknown", "reserved":
		return ""
	}
	switch s {
	case "保留地址", "内网IP", "内网ip", "未知":
		return ""
	}
	return s
}
