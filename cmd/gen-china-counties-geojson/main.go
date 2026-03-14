package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type options struct {
	SimplifyEpsilon float64
	Precision       int
}

type stats struct {
	TotalRows    int
	Features     int
	FilteredRows int
	EmptyRows    int
	InvalidRows  int
	Polygons     int
	PointsBefore int
	PointsAfter  int
}

type featureCollection struct {
	Type     string    `json:"type"`
	Features []feature `json:"features"`
}

type feature struct {
	Type       string            `json:"type"`
	Properties featureProperties `json:"properties"`
	Geometry   featureGeometry   `json:"geometry"`
}

type featureProperties struct {
	Adcode    string    `json:"adcode"`
	Name      string    `json:"name"`
	ShortName string    `json:"short_name"`
	Province  string    `json:"province"`
	City      string    `json:"city"`
	District  string    `json:"district"`
	CP        []float64 `json:"cp,omitempty"`
}

type featureGeometry struct {
	Type        string          `json:"type"`
	Coordinates [][][][]float64 `json:"coordinates"`
}

func main() {
	var (
		inputPath       = flag.String("input", "data/ok_geo.csv", "path to ok_geo.csv")
		outputPath      = flag.String("output", "web/admin/public/maps/china-counties.geojson", "output GeoJSON path")
		simplifyEpsilon = flag.Float64("simplify-epsilon", 0.00035, "Douglas-Peucker epsilon in lon/lat degrees; 0 disables simplification")
		precision       = flag.Int("precision", 5, "decimal places to keep for coordinates after simplification")
	)
	flag.Parse()

	if err := run(*inputPath, *outputPath, options{
		SimplifyEpsilon: *simplifyEpsilon,
		Precision:       *precision,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "generate china counties geojson: %v\n", err)
		os.Exit(1)
	}
}

func run(inputPath, outputPath string, opts options) error {
	in, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input csv: %w", err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("mkdir output dir: %w", err)
	}
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output geojson: %w", err)
	}
	defer out.Close()

	stats, err := writeFeatureCollection(in, out, opts)
	if err != nil {
		return err
	}

	fmt.Printf(
		"generated %d county features (%d polygons, %d -> %d points), skipped filtered=%d empty=%d invalid=%d, simplify_epsilon=%g precision=%d\n",
		stats.Features,
		stats.Polygons,
		stats.PointsBefore,
		stats.PointsAfter,
		stats.FilteredRows,
		stats.EmptyRows,
		stats.InvalidRows,
		opts.SimplifyEpsilon,
		opts.Precision,
	)
	return nil
}

func writeFeatureCollection(src io.Reader, dst io.Writer, opts options) (stats, error) {
	reader := csv.NewReader(src)
	reader.TrimLeadingSpace = true
	reader.ReuseRecord = true

	header, err := reader.Read()
	if err != nil {
		return stats{}, fmt.Errorf("read header: %w", err)
	}
	idx, err := detectColumns(header, "id", "deep", "name", "ext_path", "geo", "polygon")
	if err != nil {
		return stats{}, err
	}

	fc := featureCollection{Type: "FeatureCollection"}
	var st stats

	for {
		rec, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return stats{}, fmt.Errorf("read row: %w", err)
		}
		st.TotalRows++

		feature, rowStats, include := buildFeature(rec, idx, opts)
		mergeStats(&st, &rowStats)
		if !include {
			continue
		}
		fc.Features = append(fc.Features, feature)
	}

	enc := json.NewEncoder(dst)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(fc); err != nil {
		return stats{}, fmt.Errorf("encode geojson: %w", err)
	}
	return st, nil
}

func buildFeature(rec []string, idx map[string]int, opts options) (feature, stats, bool) {
	var st stats

	adcode := getCSV(rec, idx["id"])
	if adcode == "" {
		st.InvalidRows++
		return feature{}, st, false
	}

	deep, err := strconv.Atoi(getCSV(rec, idx["deep"]))
	if err != nil {
		st.InvalidRows++
		return feature{}, st, false
	}
	if deep != 2 || isNonMainlandAdcode(adcode) {
		st.FilteredRows++
		return feature{}, st, false
	}

	polygonText := getCSV(rec, idx["polygon"])
	if polygonText == "" || strings.EqualFold(polygonText, "empty") {
		st.EmptyRows++
		return feature{}, st, false
	}

	province, city, district := splitExtPath(deep, getCSV(rec, idx["ext_path"]), getCSV(rec, idx["name"]))
	if province == "" || city == "" || district == "" {
		st.InvalidRows++
		return feature{}, st, false
	}

	center, ok, err := parseCenterPoint(getCSV(rec, idx["geo"]), opts.Precision)
	if err != nil {
		st.InvalidRows++
		return feature{}, st, false
	}
	if !ok {
		center = nil
	}

	coordinates, polyCount, before, after, err := parseMultiPolygon(polygonText, opts)
	if err != nil {
		st.InvalidRows++
		return feature{}, st, false
	}

	st.Features = 1
	st.Polygons = polyCount
	st.PointsBefore = before
	st.PointsAfter = after

	propsName := strings.TrimSpace(getCSV(rec, idx["ext_path"]))
	if propsName == "" {
		propsName = district
	}

	return feature{
		Type: "Feature",
		Properties: featureProperties{
			Adcode:    adcode,
			Name:      propsName,
			ShortName: district,
			Province:  province,
			City:      city,
			District:  district,
			CP:        center,
		},
		Geometry: featureGeometry{
			Type:        "MultiPolygon",
			Coordinates: coordinates,
		},
	}, st, true
}

func parseMultiPolygon(raw string, opts options) ([][][][]float64, int, int, int, error) {
	segments := strings.Split(raw, ";")
	coords := make([][][][]float64, 0, len(segments))
	polygons := 0
	pointsBefore := 0
	pointsAfter := 0

	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		ring, err := parseRing(segment)
		if err != nil {
			return nil, 0, 0, 0, err
		}
		before := len(ring)
		ring = simplifyClosedRing(ring, opts.SimplifyEpsilon, opts.Precision)
		if len(ring) < 4 {
			return nil, 0, 0, 0, fmt.Errorf("ring too short after simplification")
		}
		coords = append(coords, [][][]float64{ring})
		polygons++
		pointsBefore += before
		pointsAfter += len(ring)
	}

	if len(coords) == 0 {
		return nil, 0, 0, 0, fmt.Errorf("no polygon segments")
	}
	return coords, polygons, pointsBefore, pointsAfter, nil
}

func parseRing(raw string) ([][]float64, error) {
	parts := strings.Split(raw, ",")
	ring := make([][]float64, 0, len(parts)+1)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Fields(part)
		if len(fields) < 2 {
			return nil, fmt.Errorf("invalid point %q", part)
		}
		lng, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid lng %q: %w", fields[0], err)
		}
		lat, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid lat %q: %w", fields[1], err)
		}
		ring = append(ring, []float64{lng, lat})
	}
	if len(ring) < 3 {
		return nil, fmt.Errorf("ring needs at least 3 points")
	}
	return closeRing(dedupeConsecutive(ring)), nil
}

func simplifyClosedRing(ring [][]float64, epsilon float64, precision int) [][]float64 {
	original := normalizeRing(ring, precision)
	if len(original) < 4 || epsilon <= 0 {
		return original
	}

	open := clonePoints(original[:len(original)-1])
	simplified := simplifyOpenLine(open, epsilon)
	candidate := normalizeRing(closeRing(simplified), precision)
	if len(candidate) < 4 {
		return original
	}
	return candidate
}

func simplifyOpenLine(points [][]float64, epsilon float64) [][]float64 {
	if len(points) <= 2 {
		return clonePoints(points)
	}

	maxDistance := 0.0
	index := -1
	start := points[0]
	end := points[len(points)-1]
	for i := 1; i < len(points)-1; i++ {
		d := perpendicularDistance(points[i], start, end)
		if d > maxDistance {
			maxDistance = d
			index = i
		}
	}
	if index >= 0 && maxDistance > epsilon {
		left := simplifyOpenLine(points[:index+1], epsilon)
		right := simplifyOpenLine(points[index:], epsilon)
		return append(left[:len(left)-1], right...)
	}
	return [][]float64{clonePoint(start), clonePoint(end)}
}

func perpendicularDistance(point, start, end []float64) float64 {
	dx := end[0] - start[0]
	dy := end[1] - start[1]
	if dx == 0 && dy == 0 {
		return math.Hypot(point[0]-start[0], point[1]-start[1])
	}
	num := math.Abs(dy*point[0] - dx*point[1] + end[0]*start[1] - end[1]*start[0])
	den := math.Hypot(dx, dy)
	return num / den
}

func normalizeRing(ring [][]float64, precision int) [][]float64 {
	rounded := make([][]float64, 0, len(ring)+1)
	for _, point := range ring {
		if len(point) < 2 {
			continue
		}
		rounded = append(rounded, []float64{roundTo(point[0], precision), roundTo(point[1], precision)})
	}
	rounded = dedupeConsecutive(rounded)
	if len(rounded) == 0 {
		return nil
	}
	return closeRing(rounded)
}

func closeRing(ring [][]float64) [][]float64 {
	if len(ring) == 0 {
		return nil
	}
	closed := clonePoints(ring)
	first := closed[0]
	last := closed[len(closed)-1]
	if !samePoint(first, last) {
		closed = append(closed, clonePoint(first))
	}
	return closed
}

func dedupeConsecutive(points [][]float64) [][]float64 {
	if len(points) == 0 {
		return nil
	}
	out := make([][]float64, 0, len(points))
	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		if len(out) == 0 || !samePoint(out[len(out)-1], point) {
			out = append(out, clonePoint(point))
		}
	}
	return out
}

func parseCenterPoint(raw string, precision int) ([]float64, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "empty") {
		return nil, false, nil
	}
	parts := strings.Fields(raw)
	if len(parts) < 2 {
		return nil, false, fmt.Errorf("invalid geo center %q", raw)
	}
	lng, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, false, fmt.Errorf("invalid geo lng %q: %w", parts[0], err)
	}
	lat, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return nil, false, fmt.Errorf("invalid geo lat %q: %w", parts[1], err)
	}
	return []float64{roundTo(lng, precision), roundTo(lat, precision)}, true, nil
}

func detectColumns(header []string, required ...string) (map[string]int, error) {
	idx := make(map[string]int, len(header))
	for i, col := range header {
		idx[normalizeHeader(col)] = i
	}
	for _, name := range required {
		if _, ok := idx[name]; !ok {
			return nil, fmt.Errorf("missing required column %q", name)
		}
	}
	return idx, nil
}

func normalizeHeader(s string) string {
	s = strings.TrimSpace(strings.TrimPrefix(s, "\uFEFF"))
	return strings.ToLower(s)
}

func getCSV(rec []string, idx int) string {
	if idx < 0 || idx >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[idx])
}

func splitExtPath(deep int, extPath, name string) (province, city, district string) {
	parts := strings.Fields(strings.TrimSpace(extPath))
	name = strings.TrimSpace(name)

	switch deep {
	case 0:
		province = choosePath(parts, 0, name)
		city = province
	case 1:
		province = choosePath(parts, 0, "")
		city = choosePath(parts, 1, name)
		if city == "" {
			city = province
		}
	default:
		province = choosePath(parts, 0, "")
		city = choosePath(parts, 1, province)
		district = choosePath(parts, 2, name)
	}
	return province, city, district
}

func choosePath(parts []string, idx int, fallback string) string {
	if idx >= 0 && idx < len(parts) {
		return strings.TrimSpace(parts[idx])
	}
	return strings.TrimSpace(fallback)
}

func isNonMainlandAdcode(adcode string) bool {
	return strings.HasPrefix(adcode, "71") || strings.HasPrefix(adcode, "81") || strings.HasPrefix(adcode, "82")
}

func roundTo(v float64, precision int) float64 {
	if precision < 0 {
		return v
	}
	factor := math.Pow10(precision)
	return math.Round(v*factor) / factor
}

func samePoint(a, b []float64) bool {
	return len(a) >= 2 && len(b) >= 2 && a[0] == b[0] && a[1] == b[1]
}

func clonePoint(point []float64) []float64 {
	return []float64{point[0], point[1]}
}

func clonePoints(points [][]float64) [][]float64 {
	out := make([][]float64, 0, len(points))
	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		out = append(out, clonePoint(point))
	}
	return out
}

func mergeStats(dst, src *stats) {
	dst.TotalRows += src.TotalRows
	dst.Features += src.Features
	dst.FilteredRows += src.FilteredRows
	dst.EmptyRows += src.EmptyRows
	dst.InvalidRows += src.InvalidRows
	dst.Polygons += src.Polygons
	dst.PointsBefore += src.PointsBefore
	dst.PointsAfter += src.PointsAfter
}
