package ip2region

import (
	"errors"
	"strings"

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
	lookupFn func(ip string) (string, error)
}

func NewSearcher(_ string) *Searcher {
	return &Searcher{}
}

func (s *Searcher) SetLookupFn(fn func(ip string) (string, error)) {
	s.lookupFn = fn
}

func (s *Searcher) Lookup(ip string) (Region, error) {
	if s.lookupFn == nil {
		return Region{}, ErrLookupNotImplemented
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
	return Region{
		Country:  strings.TrimSpace(parts[0]),
		Province: normalize.NormalizeProvince(parts[1]),
		City:     normalize.NormalizeCity(parts[2]),
		ISP:      strings.TrimSpace(parts[3]),
		Code:     strings.TrimSpace(parts[4]),
	}
}

