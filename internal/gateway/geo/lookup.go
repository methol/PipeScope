package geo

import (
	"strings"

	"pipescope/internal/geo/areacity"
	"pipescope/internal/geo/ip2region"
	"pipescope/internal/geo/normalize"
)

// GeoInfo contains geographic information for a connection
type GeoInfo struct {
	Country  string
	Province string
	City     string
	Adcode   string
}

// GeoLookupFunc returns geo info for a given IP address.
// Used for geo-based traffic filtering before forwarding.
type GeoLookupFunc func(ip string) (GeoInfo, error)

// ErrLookupNotConfigured is returned when geo lookup is not configured
var ErrLookupNotConfigured = errorNotConfigured{}

type errorNotConfigured struct{}

func (errorNotConfigured) Error() string {
	return "geo lookup not configured"
}

// LookupFunc creates a GeoLookupFunc from ip2region searcher and areacity matcher.
// This is used by the proxy runner to check geo policy before forwarding connections.
func LookupFunc(region *ip2region.Searcher, matcher *areacity.Matcher) GeoLookupFunc {
	return func(ip string) (GeoInfo, error) {
		if region == nil {
			return GeoInfo{}, ErrLookupNotConfigured
		}

		regionResult, err := region.Lookup(ip)
		if err != nil {
			return GeoInfo{}, err
		}

		province := normalize.NormalizeProvince(regionResult.Province)
		city := normalize.NormalizeCity(regionResult.City)
		if city == "" {
			city = province
		}

		info := GeoInfo{
			Country:  resolveCountryCode(regionResult),
			Province: province,
			City:     city,
		}

		// Try to get adcode from matcher if available
		if matcher != nil && province != "" && city != "" {
			dim, ok, err := matcher.Match(province, city)
			if err == nil && ok {
				info.Adcode = dim.Adcode
			}
		}

		return info, nil
	}
}

func resolveCountryCode(region ip2region.Region) string {
	if code := strings.ToUpper(strings.TrimSpace(region.Code)); code != "" {
		return code
	}
	return strings.ToUpper(strings.TrimSpace(region.Country))
}
