package embeddeddata

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func EnsureIP2RegionXDB(cacheDir string) (string, error) {
	manifest, err := LoadManifest()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir cache dir: %w", err)
	}
	path := filepath.Join(cacheDir, fmt.Sprintf("ip2region_v4_%s.xdb", manifest.IP2RegionVersion))
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return path, nil
	}
	if err := writeGzipAsset("assets/ip2region_v4.xdb.gz", path); err != nil {
		return "", err
	}
	return path, nil
}

func writeGzipAsset(assetPath, outputPath string) error {
	f, err := assetsFS.Open(assetPath)
	if err != nil {
		return fmt.Errorf("open embedded asset %s: %w", assetPath, err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("open gzip asset %s: %w", assetPath, err)
	}
	defer gz.Close()
	tmp := outputPath + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create output file %s: %w", outputPath, err)
	}
	if _, err := io.Copy(out, gz); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("write output file %s: %w", outputPath, err)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close output file %s: %w", outputPath, err)
	}
	if err := os.Rename(tmp, outputPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename output file %s: %w", outputPath, err)
	}
	return nil
}
