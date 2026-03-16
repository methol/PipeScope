package geo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"pipescope/internal/gateway/rule"
	"pipescope/internal/geo/areacity"
	"pipescope/internal/geo/ip2region"
	"pipescope/internal/geo/normalize"
	sqlitestore "pipescope/internal/store/sqlite"

	_ "modernc.org/sqlite"
)

func TestLookupResolveCountryCode(t *testing.T) {
	tests := []struct {
		name   string
		region ip2region.Region
		want   string
	}{
		{
			name:   "prefer code when present",
			region: ip2region.Region{Country: "中国", Code: "CN"},
			want:   "CN",
		},
		{
			name:   "fallback to country uppercase when code missing",
			region: ip2region.Region{Country: "cn", Code: ""},
			want:   "CN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveCountryCode(tt.region); got != tt.want {
				t.Fatalf("resolveCountryCode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMatcher_AllowCNWithDenyXJXZ_RequireAllowHitFalse(t *testing.T) {
	policy := &rule.GeoPolicy{
		RequireAllowHit: false,
		Allow: []rule.GeoRule{
			{Country: "CN"},
		},
		Deny: []rule.GeoRule{
			{Country: "CN", Provinces: []string{"新疆", "西藏"}},
		},
	}

	m := NewMatcher(policy)

	if got := m.Check(GeoInfo{Country: "CN", Province: "四川"}); !got.Allowed {
		t.Fatalf("sichuan should be allowed, got %+v", got)
	}

	if got := m.Check(GeoInfo{Country: "CN", Province: "湖北"}); !got.Allowed {
		t.Fatalf("hubei should be allowed, got %+v", got)
	}

	if got := m.Check(GeoInfo{Country: "CN", Province: "贵州"}); !got.Allowed {
		t.Fatalf("guizhou should be allowed, got %+v", got)
	}

	if got := m.Check(GeoInfo{Country: "CN", Province: "新疆"}); got.Allowed || got.BlockedReason != BlockedReasonDenied {
		t.Fatalf("xinjiang should be denied, got %+v", got)
	}

	if got := m.Check(GeoInfo{Country: "CN", Province: "西藏"}); got.Allowed || got.BlockedReason != BlockedReasonDenied {
		t.Fatalf("xizang should be denied, got %+v", got)
	}
}

func TestLookupFunc_MapsAuditableGeoSamples(t *testing.T) {
	db := openLookupTempDB(t)
	store := sqlitestore.New(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	seedLookupAdcode(t, db, areacity.DimAdcode{Adcode: "5101", Province: "四川省", City: "成都市"})
	seedLookupAdcode(t, db, areacity.DimAdcode{Adcode: "4201", Province: "湖北省", City: "武汉市"})
	seedLookupAdcode(t, db, areacity.DimAdcode{Adcode: "5203", Province: "贵州省", City: "遵义市"})
	seedLookupAdcode(t, db, areacity.DimAdcode{Adcode: "6501", Province: "新疆维吾尔自治区", City: "乌鲁木齐市"})
	seedLookupAdcode(t, db, areacity.DimAdcode{Adcode: "5401", Province: "西藏自治区", City: "拉萨市"})

	samples := []struct {
		name string
		ip   string
		raw  string
		want GeoInfo
	}{
		// Source note (verified for this PR on 2026-03-16): 61.139.2.69 is commonly
		// published as a Sichuan/Chengdu Telecom DNS IP. We inject the ip2region raw
		// response so the mapping stays auditable even if the bundled offline DB drifts.
		{
			name: "sichuan chengdu",
			ip:   "61.139.2.69",
			raw:  "中国|四川省|成都市|电信|CN",
			want: GeoInfo{Country: "CN", Province: "四川", City: "成都", Adcode: "5101"},
		},
		// Source note: 202.103.24.68 is widely listed as a Hubei/Wuhan Telecom DNS IP.
		// The fixture-injected raw payload keeps this test stable and reviewable.
		{
			name: "hubei wuhan",
			ip:   "202.103.24.68",
			raw:  "中国|湖北省|武汉市|电信|CN",
			want: GeoInfo{Country: "CN", Province: "湖北", City: "武汉", Adcode: "4201"},
		},
		// Source note: 119.0.110.67 appears on public 17CE resolve-IP pages as
		// "中国贵州遵义电信". Raw ip2region output is injected here for deterministic review.
		{
			name: "guizhou zunyi",
			ip:   "119.0.110.67",
			raw:  "中国|贵州省|遵义市|电信|CN",
			want: GeoInfo{Country: "CN", Province: "贵州", City: "遵义", Adcode: "5203"},
		},
		// Source note: 61.128.114.166 is commonly listed as a Xinjiang Telecom DNS IP.
		// We fixture the raw lookup result to audit the Xinjiang -> Urumqi mapping path.
		{
			name: "xinjiang urumqi",
			ip:   "61.128.114.166",
			raw:  "中国|新疆维吾尔自治区|乌鲁木齐市|电信|CN",
			want: GeoInfo{Country: "CN", Province: "新疆", City: "乌鲁木齐", Adcode: "6501"},
		},
		// Source note: 202.98.224.68 is commonly published as a Tibet/Lhasa Telecom DNS IP.
		// The test asserts stable field mapping without trusting live/offline DB behavior.
		{
			name: "tibet lhasa",
			ip:   "202.98.224.68",
			raw:  "中国|西藏自治区|拉萨市|电信|CN",
			want: GeoInfo{Country: "CN", Province: "西藏", City: "拉萨", Adcode: "5401"},
		},
	}

	rawByIP := make(map[string]string, len(samples))
	for _, sample := range samples {
		rawByIP[sample.ip] = sample.raw
	}

	searcher := &ip2region.Searcher{}
	searcher.SetLookupFn(func(ip string) (string, error) {
		raw, ok := rawByIP[ip]
		if !ok {
			return "", fmt.Errorf("unexpected sample ip %s", ip)
		}
		return raw, nil
	})

	lookup := LookupFunc(searcher, areacity.NewMatcher(db))
	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			got, err := lookup(sample.ip)
			if err != nil {
				t.Fatalf("lookup(%s): %v", sample.ip, err)
			}
			if !reflect.DeepEqual(got, sample.want) {
				t.Fatalf("lookup(%s) = %+v, want %+v", sample.ip, got, sample.want)
			}
		})
	}
}

func TestLookupFunc_UsesCountryCodeForGeoPolicyMatching(t *testing.T) {
	searcher := &ip2region.Searcher{}
	searcher.SetLookupFn(func(ip string) (string, error) {
		return "中国|四川省|成都市|电信|CN", nil
	})

	lookup := LookupFunc(searcher, nil)
	info, err := lookup("1.1.1.1")
	if err != nil {
		t.Fatalf("lookup error: %v", err)
	}
	if info.Country != "CN" {
		t.Fatalf("country=%q, want CN", info.Country)
	}

	policy := &rule.GeoPolicy{
		RequireAllowHit: false,
		Allow:           []rule.GeoRule{{Country: "CN"}},
		Deny:            []rule.GeoRule{{Country: "CN", Provinces: []string{"新疆", "西藏"}}},
	}
	if got := NewMatcher(policy).Check(info); !got.Allowed {
		t.Fatalf("sichuan should be allowed after country-code normalization, got %+v", got)
	}
}

func TestLookupFunc_PropagatesLookupError(t *testing.T) {
	wantErr := errors.New("lookup failed")
	searcher := &ip2region.Searcher{}
	searcher.SetLookupFn(func(ip string) (string, error) {
		return "", wantErr
	})

	lookup := LookupFunc(searcher, nil)
	_, err := lookup("8.8.8.8")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func openLookupTempDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "geo-lookup-test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func seedLookupAdcode(t *testing.T, db *sql.DB, dim areacity.DimAdcode) {
	t.Helper()

	nProvince := normalize.NormalizeProvince(dim.Province)
	nCity := normalize.NormalizeCity(dim.City)
	if nCity == "" {
		nCity = nProvince
	}

	if _, err := db.Exec(`
INSERT INTO dim_adcode(adcode, province, city, district, lat, lng, normalized_province, normalized_city)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, dim.Adcode, dim.Province, dim.City, dim.District, dim.Lat, dim.Lng, nProvince, nCity); err != nil {
		t.Fatalf("seed dim_adcode: %v", err)
	}
}
