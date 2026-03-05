package areacity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"pipescope/internal/geo/normalize"
)

var rePointWKT = regexp.MustCompile(`(?i)POINT\s*\(\s*([+-]?\d+(?:\.\d+)?)\s+([+-]?\d+(?:\.\d+)?)\s*\)`)

type HTTPMatcher struct {
	baseURL  string
	instance int
	client   *http.Client

	mu    sync.RWMutex
	cache map[string]matchResult
}

type matchResult struct {
	row DimAdcode
	ok  bool
}

type apiItem struct {
	ID       json.Number `json:"id"`
	UniqueID json.Number `json:"unique_id"`
	ExtPath  string      `json:"ext_path"`
	Name     string      `json:"name"`
	GeoWKT   string      `json:"geo_wkt"`
	Geo      string      `json:"geo"`
}

func NewHTTPMatcher(baseURL string, instance int) *HTTPMatcher {
	return &HTTPMatcher{
		baseURL:  strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		instance: instance,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
		cache: make(map[string]matchResult),
	}
}

func (m *HTTPMatcher) Ping(ctx context.Context) error {
	if m.baseURL == "" {
		return fmt.Errorf("areacity api base url is empty")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.baseURL+"/", nil)
	if err != nil {
		return err
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("areacity api unhealthy: status=%d", resp.StatusCode)
	}
	return nil
}

func (m *HTTPMatcher) Match(province, city string) (DimAdcode, bool, error) {
	nProvince := normalize.NormalizeProvince(province)
	nCity := normalize.NormalizeCity(city)
	if nCity == "" {
		nCity = nProvince
	}
	if nProvince == "" || nCity == "" {
		return DimAdcode{}, false, nil
	}

	cacheKey := nProvince + "|" + nCity
	if hit, ok := m.getCache(cacheKey); ok {
		return hit.row, hit.ok, nil
	}

	row, ok, err := m.fetchMatch(nProvince, nCity)
	if err != nil {
		return DimAdcode{}, false, err
	}
	m.setCache(cacheKey, matchResult{row: row, ok: ok})
	return row, ok, nil
}

func (m *HTTPMatcher) getCache(key string) (matchResult, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.cache[key]
	return v, ok
}

func (m *HTTPMatcher) setCache(key string, res matchResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[key] = res
}

func (m *HTTPMatcher) fetchMatch(nProvince, nCity string) (DimAdcode, bool, error) {
	items, err := m.readCandidates(nCity)
	if err != nil {
		return DimAdcode{}, false, err
	}
	row, ok := pickBestByProvinceCity(items, nProvince, nCity)
	if ok {
		return row, true, nil
	}

	if nProvince != nCity {
		items, err = m.readCandidates(nProvince)
		if err != nil {
			return DimAdcode{}, false, err
		}
		row, ok = pickBestByProvinceCity(items, nProvince, nCity)
		if ok {
			return row, true, nil
		}
	}
	return DimAdcode{}, false, nil
}

func (m *HTTPMatcher) readCandidates(keyword string) ([]DimAdcode, error) {
	if m.baseURL == "" {
		return nil, fmt.Errorf("areacity api base url is empty")
	}

	apiURL, err := url.Parse(m.baseURL + "/readWKT")
	if err != nil {
		return nil, err
	}
	q := apiURL.Query()
	q.Set("deep", "1")
	q.Set("extPath", "*"+keyword+"*")
	q.Set("returnWKTKey", "0")
	q.Set("instance", strconv.Itoa(m.instance))
	apiURL.RawQuery = q.Encode()

	resp, err := m.client.Get(apiURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("areacity api status=%d", resp.StatusCode)
	}

	var body struct {
		C int `json:"c"`
		V struct {
			List []json.RawMessage `json:"list"`
		} `json:"v"`
		M string `json:"m"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.C != 0 {
		msg := strings.TrimSpace(body.M)
		if msg == "" {
			msg = "unknown areacity api error"
		}
		return nil, fmt.Errorf("areacity api error: %s", msg)
	}

	out := make([]DimAdcode, 0, len(body.V.List))
	for _, raw := range body.V.List {
		var item apiItem
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.UseNumber()
		if err := dec.Decode(&item); err != nil {
			continue
		}
		row, ok := toDimAdcode(item)
		if !ok {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func toDimAdcode(item apiItem) (DimAdcode, bool) {
	adcode := strings.TrimSpace(item.UniqueID.String())
	if adcode == "" || adcode == "0" {
		adcode = strings.TrimSpace(item.ID.String())
	}
	if adcode == "" || adcode == "0" {
		return DimAdcode{}, false
	}

	province, city, district := splitExtPathToNames(item.ExtPath, item.Name)
	if province == "" {
		return DimAdcode{}, false
	}
	if city == "" {
		city = province
	}

	lng, lat, ok := parseGeoFromAPI(item.GeoWKT, item.Geo)
	if !ok {
		return DimAdcode{}, false
	}

	return DimAdcode{
		Adcode:   adcode,
		Province: province,
		City:     city,
		District: district,
		Lat:      lat,
		Lng:      lng,
	}, true
}

func splitExtPathToNames(extPath, name string) (province, city, district string) {
	parts := strings.Fields(strings.TrimSpace(extPath))
	switch len(parts) {
	case 0:
		base := strings.TrimSpace(name)
		return base, base, ""
	case 1:
		return parts[0], parts[0], ""
	case 2:
		return parts[0], parts[1], ""
	default:
		return parts[0], parts[1], parts[2]
	}
}

func parseGeoFromAPI(geoWKT, geo string) (lng, lat float64, ok bool) {
	s := strings.TrimSpace(geoWKT)
	if s != "" {
		if m := rePointWKT.FindStringSubmatch(s); len(m) == 3 {
			lngV, err1 := strconv.ParseFloat(m[1], 64)
			latV, err2 := strconv.ParseFloat(m[2], 64)
			if err1 == nil && err2 == nil {
				return lngV, latV, true
			}
		}
	}

	g := strings.Fields(strings.TrimSpace(geo))
	if len(g) >= 2 {
		lngV, err1 := strconv.ParseFloat(g[0], 64)
		latV, err2 := strconv.ParseFloat(g[1], 64)
		if err1 == nil && err2 == nil {
			return lngV, latV, true
		}
	}
	return 0, 0, false
}

func pickBestByProvinceCity(rows []DimAdcode, nProvince, nCity string) (DimAdcode, bool) {
	bestScore := -1
	var best DimAdcode

	for _, row := range rows {
		rp := normalize.NormalizeProvince(row.Province)
		rc := normalize.NormalizeCity(row.City)
		if rc == "" {
			rc = rp
		}

		score := 0
		switch {
		case rp == nProvince && rc == nCity:
			score = 3
		case rc == nCity:
			score = 2
		case rp == nProvince && nCity == nProvince:
			score = 1
		default:
			continue
		}

		if score > bestScore || (score == bestScore && len(row.Adcode) > len(best.Adcode)) {
			bestScore = score
			best = row
		}
	}

	if bestScore < 0 {
		return DimAdcode{}, false
	}
	return best, true
}
