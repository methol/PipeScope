package sqlite

import (
	"pipescope/internal/geo/areacity"
	"pipescope/internal/geo/ip2region"
	"pipescope/internal/geo/normalize"
	"pipescope/internal/gateway/session"
)

type RegionLookup interface {
	Lookup(ip string) (ip2region.Region, error)
}

type AdcodeMatcher interface {
	Match(province, city string) (areacity.DimAdcode, bool, error)
}

type enrichedFields struct {
	Province string
	City     string
	Adcode   string
	Lat      float64
	Lng      float64
}

func enrichGeoFields(evt session.Event, region RegionLookup, matcher AdcodeMatcher) enrichedFields {
	// If event already has geo info (e.g., from blocked connection), use it directly
	if evt.Province != "" || evt.City != "" || evt.Adcode != "" {
		return enrichedFields{
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

	province := normalize.NormalizeProvince(geo.Province)
	city := normalize.NormalizeCity(geo.City)
	if province == "" || city == "" {
		return enrichedFields{}
	}

	dim, ok, err := matcher.Match(province, city)
	if err != nil {
		return enrichedFields{}
	}
	if !ok {
		return enrichedFields{
			Province: province,
			City:     city,
		}
	}
	return enrichedFields{
		Province: province,
		City:     city,
		Adcode:   dim.Adcode,
		Lat:      dim.Lat,
		Lng:      dim.Lng,
	}
}

