package areacity

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"pipescope/internal/geo/normalize"
)

type DimAdcode struct {
	Adcode   string
	Province string
	City     string
	District string
	Lat      float64
	Lng      float64
}

type Importer struct {
	db *sql.DB
}

func NewImporter(db *sql.DB) *Importer {
	return &Importer{db: db}
}

type csvFormat int

const (
	formatUnknown csvFormat = iota
	formatLegacyAdcode
	formatAreaCityOKGeo
)

type csvLayout struct {
	format csvFormat
	index  map[string]int
}

func (i *Importer) ImportCSV(ctx context.Context, csvPath string) error {
	f, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return i.ImportCSVReader(ctx, f)
}

func (i *Importer) ImportCSVReader(ctx context.Context, r io.Reader) error {
	return i.importCSV(ctx, r, false)
}

func (i *Importer) ReplaceCSVReader(ctx context.Context, r io.Reader) error {
	return i.importCSV(ctx, r, true)
}

func (i *Importer) importCSV(ctx context.Context, r io.Reader, replace bool) error {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.ReuseRecord = true

	header, err := reader.Read()
	if err != nil {
		return err
	}
	layout, err := detectCSVLayout(header)
	if err != nil {
		return fmt.Errorf("detect csv layout: %w", err)
	}

	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if replace {
		if _, err := tx.ExecContext(ctx, `DELETE FROM dim_adcode`); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO dim_adcode(adcode, province, city, district, lat, lng, normalized_province, normalized_city)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(adcode) DO UPDATE SET
  province=excluded.province,
  city=excluded.city,
  district=excluded.district,
  lat=excluded.lat,
  lng=excluded.lng,
  normalized_province=excluded.normalized_province,
  normalized_city=excluded.normalized_city
`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	lineNo := 1
	for {
		rec, err := reader.Read()
		lineNo++
		if err != nil {
			if err == io.EOF {
				break
			}
			_ = tx.Rollback()
			return fmt.Errorf("read line %d: %w", lineNo, err)
		}
		row, ok, err := parseRecord(layout, rec)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("parse line %d: %w", lineNo, err)
		}
		if !ok {
			continue
		}
		nProvince := normalize.NormalizeProvince(row.Province)
		nCity := normalize.NormalizeCity(row.City)
		if nCity == "" {
			nCity = nProvince
		}

		if _, err := stmt.ExecContext(
			ctx,
			row.Adcode,
			row.Province,
			row.City,
			row.District,
			row.Lat,
			row.Lng,
			nProvince,
			nCity,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func parseRecord(layout csvLayout, rec []string) (DimAdcode, bool, error) {
	switch layout.format {
	case formatLegacyAdcode:
		return parseLegacyRecord(layout.index, rec)
	case formatAreaCityOKGeo:
		return parseAreaCityRecord(layout.index, rec)
	default:
		return DimAdcode{}, false, errors.New("unsupported csv format")
	}
}

func parseLegacyRecord(idx map[string]int, rec []string) (DimAdcode, bool, error) {
	adcode := getCSV(rec, idx["adcode"])
	if adcode == "" {
		return DimAdcode{}, false, nil
	}

	lat, err := parseFloatField(getCSV(rec, idx["lat"]))
	if err != nil {
		return DimAdcode{}, false, fmt.Errorf("invalid lat: %w", err)
	}
	lng, err := parseFloatField(getCSV(rec, idx["lng"]))
	if err != nil {
		return DimAdcode{}, false, fmt.Errorf("invalid lng: %w", err)
	}

	row := DimAdcode{
		Adcode:   adcode,
		Province: getCSV(rec, idx["province"]),
		City:     getCSV(rec, idx["city"]),
		District: getCSV(rec, idx["district"]),
		Lat:      lat,
		Lng:      lng,
	}
	if row.City == "" {
		row.City = row.Province
	}
	return row, true, nil
}

func parseAreaCityRecord(idx map[string]int, rec []string) (DimAdcode, bool, error) {
	adcode := getCSV(rec, idx["id"])
	if adcode == "" {
		return DimAdcode{}, false, nil
	}

	deep, err := strconv.Atoi(getCSV(rec, idx["deep"]))
	if err != nil {
		return DimAdcode{}, false, fmt.Errorf("invalid deep: %w", err)
	}

	// PipeScope 当前仅做省/市级聚合，区级会带来同省同市多 adcode 冲突。
	if deep > 1 {
		return DimAdcode{}, false, nil
	}

	province, city, district := splitExtPath(deep, getCSV(rec, idx["ext_path"]), getCSV(rec, idx["name"]))
	if province == "" {
		return DimAdcode{}, false, nil
	}
	if city == "" {
		city = province
	}

	lng, lat, ok, err := parseGeoPoint(getCSV(rec, idx["geo"]))
	if err != nil {
		return DimAdcode{}, false, err
	}
	if !ok {
		return DimAdcode{}, false, nil
	}

	return DimAdcode{
		Adcode:   adcode,
		Province: province,
		City:     city,
		District: district,
		Lat:      lat,
		Lng:      lng,
	}, true, nil
}

func detectCSVLayout(header []string) (csvLayout, error) {
	idx := make(map[string]int, len(header))
	for i, col := range header {
		name := normalizeHeader(col)
		if name == "" {
			continue
		}
		idx[name] = i
	}

	if hasColumns(idx, "adcode", "province", "city", "lat", "lng") {
		if _, ok := idx["district"]; !ok {
			idx["district"] = -1
		}
		return csvLayout{format: formatLegacyAdcode, index: idx}, nil
	}

	if hasColumns(idx, "id", "deep", "name", "ext_path", "geo") {
		return csvLayout{format: formatAreaCityOKGeo, index: idx}, nil
	}

	return csvLayout{}, fmt.Errorf("unknown header: %v", header)
}

func normalizeHeader(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "\uFEFF")
	return strings.ToLower(s)
}

func hasColumns(idx map[string]int, cols ...string) bool {
	for _, col := range cols {
		if _, ok := idx[col]; !ok {
			return false
		}
	}
	return true
}

func getCSV(rec []string, idx int) string {
	if idx < 0 || idx >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[idx])
}

func parseFloatField(raw string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(raw), 64)
}

func parseGeoPoint(raw string) (lng float64, lat float64, ok bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "empty") {
		return 0, 0, false, nil
	}
	parts := strings.Fields(raw)
	if len(parts) < 2 {
		return 0, 0, false, fmt.Errorf("invalid geo point: %q", raw)
	}
	lng, err = parseFloatField(parts[0])
	if err != nil {
		return 0, 0, false, fmt.Errorf("invalid geo lng: %w", err)
	}
	lat, err = parseFloatField(parts[1])
	if err != nil {
		return 0, 0, false, fmt.Errorf("invalid geo lat: %w", err)
	}
	return lng, lat, true, nil
}

func splitExtPath(deep int, extPath, name string) (province, city, district string) {
	path := strings.Fields(strings.TrimSpace(extPath))
	name = strings.TrimSpace(name)

	switch deep {
	case 0:
		province = choosePath(path, 0, name)
		city = province
	case 1:
		province = choosePath(path, 0, "")
		city = choosePath(path, 1, name)
		if city == "" {
			city = province
		}
	default:
		province = choosePath(path, 0, "")
		city = choosePath(path, 1, province)
		district = choosePath(path, 2, name)
	}
	return province, city, district
}

func choosePath(path []string, idx int, fallback string) string {
	if idx >= 0 && idx < len(path) {
		return strings.TrimSpace(path[idx])
	}
	return strings.TrimSpace(fallback)
}
