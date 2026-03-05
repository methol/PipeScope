package ip2region

import (
	"errors"
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

func NewSearcher(xdbPath string) (*Searcher, error) {
	path := strings.TrimSpace(xdbPath)
	if path == "" {
		return &Searcher{}, nil
	}

	svc, err := ip2service.NewIp2RegionWithPath(path, "")
	if err != nil {
		return nil, err
	}
	return &Searcher{service: svc}, nil
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
