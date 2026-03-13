package embeddeddata

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"pipescope/internal/geo/areacity"
)

const areaCityVersionMetaKey = "embedded_areacity_version"

func EnsureAreaCitySeed(ctx context.Context, db *sql.DB, cacheDir string) error {
	manifest, err := LoadManifest()
	if err != nil {
		return err
	}
	current, err := loadMeta(ctx, db, areaCityVersionMetaKey)
	if err != nil {
		return err
	}
	if current == manifest.AreaCityVersion {
		return nil
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("mkdir cache dir: %w", err)
	}
	seedPath := filepath.Join(cacheDir, fmt.Sprintf("areacity_seed_%s.csv", manifest.AreaCityVersion))
	if _, err := os.Stat(seedPath); err != nil {
		if os.IsNotExist(err) {
			if err := writeGzipAsset("assets/areacity_seed.csv.gz", seedPath); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("stat areacity seed: %w", err)
		}
	}
	f, err := os.Open(seedPath)
	if err != nil {
		return fmt.Errorf("open areacity seed: %w", err)
	}
	defer f.Close()
	if err := areacity.NewImporter(db).ReplaceCSVReader(ctx, f); err != nil {
		return fmt.Errorf("import embedded areacity seed: %w", err)
	}
	return saveMeta(ctx, db, areaCityVersionMetaKey, manifest.AreaCityVersion)
}

func loadMeta(ctx context.Context, db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRowContext(ctx, `SELECT value FROM app_meta WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("load app_meta %s: %w", key, err)
	}
	return value, nil
}

func saveMeta(ctx context.Context, db *sql.DB, key, value string) error {
	_, err := db.ExecContext(ctx, `
INSERT INTO app_meta(key, value)
VALUES(?, ?)
ON CONFLICT(key) DO UPDATE SET value=excluded.value
`, key, value)
	if err != nil {
		return fmt.Errorf("save app_meta %s: %w", key, err)
	}
	return nil
}
