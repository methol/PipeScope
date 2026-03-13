package embeddeddata

import (
	"encoding/json"
	"fmt"
)

type Manifest struct {
	IP2RegionVersion string `json:"ip2region_version"`
	AreaCityVersion  string `json:"areacity_version"`
	GeneratedRows    int    `json:"generated_rows"`
}

func LoadManifest() (Manifest, error) {
	b, err := assetsFS.ReadFile("assets/manifest.json")
	if err != nil {
		return Manifest{}, fmt.Errorf("read embedded manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifest{}, fmt.Errorf("decode embedded manifest: %w", err)
	}
	return m, nil
}
