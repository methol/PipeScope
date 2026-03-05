package areacity

import (
	"context"
	"database/sql"
	"encoding/csv"
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

func (i *Importer) ImportCSV(ctx context.Context, csvPath string) error {
	f, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true

	_, err = reader.Read() // header
	if err != nil {
		return err
	}

	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return err
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

	for {
		rec, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			_ = tx.Rollback()
			return err
		}
		if len(rec) < 6 {
			continue
		}

		lat, err := strconv.ParseFloat(strings.TrimSpace(rec[4]), 64)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		lng, err := strconv.ParseFloat(strings.TrimSpace(rec[5]), 64)
		if err != nil {
			_ = tx.Rollback()
			return err
		}

		province := strings.TrimSpace(rec[1])
		city := strings.TrimSpace(rec[2])
		district := strings.TrimSpace(rec[3])

		if _, err := stmt.ExecContext(
			ctx,
			strings.TrimSpace(rec[0]),
			province,
			city,
			district,
			lat,
			lng,
			normalize.NormalizeProvince(province),
			normalize.NormalizeCity(city),
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

