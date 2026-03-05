package areacity

import (
	"database/sql"

	"pipescope/internal/geo/normalize"
)

type Matcher struct {
	db *sql.DB
}

func NewMatcher(db *sql.DB) *Matcher {
	return &Matcher{db: db}
}

func (m *Matcher) Match(province, city string) (DimAdcode, bool, error) {
	nProvince := normalize.NormalizeProvince(province)
	nCity := normalize.NormalizeCity(city)

	var row DimAdcode
	err := m.db.QueryRow(`
SELECT adcode, province, city, district, lat, lng
FROM dim_adcode
WHERE normalized_province = ? AND normalized_city = ?
ORDER BY LENGTH(adcode) DESC, adcode DESC
LIMIT 1
`, nProvince, nCity).Scan(
		&row.Adcode,
		&row.Province,
		&row.City,
		&row.District,
		&row.Lat,
		&row.Lng,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return DimAdcode{}, false, nil
		}
		return DimAdcode{}, false, err
	}
	return row, true, nil
}
