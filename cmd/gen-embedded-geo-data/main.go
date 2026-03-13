package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Manifest struct {
	IP2RegionVersion string `json:"ip2region_version"`
	AreaCityVersion  string `json:"areacity_version"`
	GeneratedRows    int    `json:"generated_rows"`
}

func main() {
	var (
		xdbPath      = flag.String("xdb", "data/ip2region_v4.xdb", "path to ip2region xdb")
		areaCityPath = flag.String("areacity", "data/ok_geo.csv", "path to ok_geo.csv")
		assetDir     = flag.String("out", "internal/embeddeddata/assets", "output asset directory")
		areaCityTag  = flag.String("areacity-tag", "manual-update", "source tag for AreaCity data")
	)
	flag.Parse()
	if err := run(*xdbPath, *areaCityPath, *assetDir, *areaCityTag); err != nil {
		fmt.Fprintf(os.Stderr, "generate embedded geo data: %v\n", err)
		os.Exit(1)
	}
}

func run(xdbPath, areaCityPath, assetDir, areaCityTag string) error {
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		return fmt.Errorf("mkdir asset dir: %w", err)
	}
	seedPath := filepath.Join(assetDir, "areacity_seed.csv.gz")
	rows, err := generateAreaCitySeed(areaCityPath, seedPath)
	if err != nil {
		return err
	}
	xdbAssetPath := filepath.Join(assetDir, "ip2region_v4.xdb.gz")
	if err := gzipFile(xdbPath, xdbAssetPath); err != nil {
		return err
	}
	manifest, err := buildManifest(xdbAssetPath, seedPath, areaCityTag, rows)
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(assetDir, "manifest.json")
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, body, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

func generateAreaCitySeed(srcPath, outPath string) (int, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return 0, fmt.Errorf("open area city csv: %w", err)
	}
	defer f.Close()
	out, err := os.Create(outPath)
	if err != nil {
		return 0, fmt.Errorf("create area city seed: %w", err)
	}
	defer out.Close()
	gz := gzip.NewWriter(out)
	defer gz.Close()
	return writeAreaCitySeed(f, gz)
}

func writeAreaCitySeed(src io.Reader, dst io.Writer) (int, error) {
	reader := csv.NewReader(src)
	reader.ReuseRecord = true
	header, err := reader.Read()
	if err != nil {
		return 0, fmt.Errorf("read area city header: %w", err)
	}
	idx := make(map[string]int, len(header))
	for i, col := range header {
		idx[strings.ToLower(strings.TrimSpace(strings.TrimPrefix(col, "\uFEFF")))] = i
	}
	writer := csv.NewWriter(dst)
	if err := writer.Write([]string{"adcode", "province", "city", "district", "lat", "lng"}); err != nil {
		return 0, fmt.Errorf("write seed header: %w", err)
	}
	rows := 0
	for {
		rec, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, fmt.Errorf("read area city row: %w", err)
		}
		deep, err := strconv.Atoi(strings.TrimSpace(valueAt(rec, idx["deep"])))
		if err != nil || deep > 1 {
			continue
		}
		adcode := strings.TrimSpace(valueAt(rec, idx["id"]))
		if adcode == "" {
			continue
		}
		extPath := strings.Fields(strings.TrimSpace(valueAt(rec, idx["ext_path"])))
		name := strings.TrimSpace(valueAt(rec, idx["name"]))
		province := choosePath(extPath, 0, name)
		if province == "" {
			continue
		}
		city := province
		if deep > 0 {
			city = choosePath(extPath, 1, name)
			if city == "" {
				city = province
			}
		}
		geo := strings.Fields(strings.TrimSpace(valueAt(rec, idx["geo"])))
		if len(geo) < 2 {
			continue
		}
		lng, lat := geo[0], geo[1]
		if err := writer.Write([]string{adcode, province, city, "", lat, lng}); err != nil {
			return 0, fmt.Errorf("write seed row: %w", err)
		}
		rows++
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return 0, fmt.Errorf("flush seed writer: %w", err)
	}
	return rows, nil
}

func valueAt(rec []string, idx int) string {
	if idx < 0 || idx >= len(rec) {
		return ""
	}
	return rec[idx]
}

func choosePath(parts []string, idx int, fallback string) string {
	if idx >= 0 && idx < len(parts) {
		return strings.TrimSpace(parts[idx])
	}
	return strings.TrimSpace(fallback)
}

func gzipFile(srcPath, dstPath string) error {
	in, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open xdb source: %w", err)
	}
	defer in.Close()
	out, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create xdb asset: %w", err)
	}
	defer out.Close()
	gz := gzip.NewWriter(out)
	if _, err := io.Copy(gz, in); err != nil {
		_ = gz.Close()
		return fmt.Errorf("gzip xdb asset: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("close xdb asset gzip: %w", err)
	}
	return nil
}

func buildManifest(xdbAssetPath, seedAssetPath, areaCityTag string, rows int) (Manifest, error) {
	xdbHash, err := fileSHA256(xdbAssetPath)
	if err != nil {
		return Manifest{}, err
	}
	seedHash, err := fileSHA256(seedAssetPath)
	if err != nil {
		return Manifest{}, err
	}
	return Manifest{
		IP2RegionVersion: "sha256-" + xdbHash[:16],
		AreaCityVersion:  areaCityTag + "+sha256-" + seedHash[:16],
		GeneratedRows:    rows,
	}, nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for hash %s: %w", path, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash file %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
