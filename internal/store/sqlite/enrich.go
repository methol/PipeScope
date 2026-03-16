package sqlite

import (
	"strings"

	"pipescope/internal/gateway/session"
	"pipescope/internal/geo/areacity"
	"pipescope/internal/geo/ip2region"
	"pipescope/internal/geo/normalize"
)

type RegionLookup interface {
	Lookup(ip string) (ip2region.Region, error)
}

type AdcodeMatcher interface {
	Match(province, city string) (areacity.DimAdcode, bool, error)
}

type enrichedFields struct {
	Country  string
	Province string
	City     string
	Adcode   string
	Lat      float64
	Lng      float64
}

func enrichGeoFields(evt session.Event, region RegionLookup, matcher AdcodeMatcher) enrichedFields {
	// If event already has geo info (e.g., from blocked connection), use it directly
	if evt.Country != "" || evt.Province != "" || evt.City != "" || evt.Adcode != "" {
		return enrichedFields{
			Country:  evt.Country,
			Province: evt.Province,
			City:     evt.City,
			Adcode:   evt.Adcode,
		}
	}

	if region == nil || matcher == nil {
		return enrichedFields{}
	}

	srcIP := extractHost(evt.SrcAddr)
	if srcIP == "" {
		return enrichedFields{}
	}

	geo, err := region.Lookup(srcIP)
	if err != nil {
		return enrichedFields{}
	}

	country := resolveCountryCode(geo)
	province := normalize.NormalizeProvince(geo.Province)
	city := normalize.NormalizeCity(geo.City)
	if province == "" || city == "" {
		return enrichedFields{Country: country}
	}

	dim, ok, err := matcher.Match(province, city)
	if err != nil {
		return enrichedFields{Country: country}
	}
	if !ok {
		return enrichedFields{
			Country:  country,
			Province: province,
			City:     city,
		}
	}
	return enrichedFields{
		Country:  country,
		Province: province,
		City:     city,
		Adcode:   dim.Adcode,
		Lat:      dim.Lat,
		Lng:      dim.Lng,
	}
}

func resolveCountryCode(region ip2region.Region) string {
	if code := strings.ToUpper(strings.TrimSpace(region.Code)); code != "" {
		return code
	}
	return strings.ToUpper(strings.TrimSpace(region.Country))
}
