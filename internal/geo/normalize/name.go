package normalize

import "strings"

func NormalizeProvince(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "特别行政区")
	s = strings.TrimSuffix(s, "维吾尔自治区")
	s = strings.TrimSuffix(s, "壮族自治区")
	s = strings.TrimSuffix(s, "回族自治区")
	s = strings.TrimSuffix(s, "自治区")
	s = strings.TrimSuffix(s, "省")
	s = strings.TrimSuffix(s, "市")
	return strings.TrimSpace(s)
}

func NormalizeCity(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "自治州")
	s = strings.TrimSuffix(s, "地区")
	s = strings.TrimSuffix(s, "盟")
	s = strings.TrimSuffix(s, "市")
	return strings.TrimSpace(s)
}

