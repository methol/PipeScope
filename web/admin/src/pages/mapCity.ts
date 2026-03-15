export type CityLike = { province?: string; city?: string; adcode?: string }

const MUNICIPALITIES = new Set(['北京市', '天津市', '上海市', '重庆市'])

export function cityKey(item: CityLike) {
  const adcode = String(item.adcode || '').trim()
  if (/^\d{6,}$/.test(adcode)) return adcode.slice(0, 6)

  const province = String(item.province || '')
    .trim()
    .replace(/(省|市|壮族自治区|回族自治区|维吾尔自治区|自治区|特别行政区)$/g, '')
  const city = String(item.city || '')
    .trim()
    .replace(/(市|地区|自治州|盟|县|区|林区)$/g, '')
  return `${province}-${city}`
}

export function shouldKeepCityPolygon(city?: string, province?: string) {
  const rawCity = String(city || '').trim()
  if (!rawCity) return false

  const rawProvince = String(province || '').trim()
  if (MUNICIPALITIES.has(rawProvince)) return true

  // 过滤县区/林区（仅保留地级市/自治州/地区/盟）
  if (/(县|区|林区)$/.test(rawCity) && !/(市|自治州|地区|盟)$/.test(rawCity)) return false
  return true
}

export function normalizeCityGeoFeatures(features: any[]) {
  const filteredFeatures: any[] = []
  for (const feature of Array.isArray(features) ? features : []) {
    const p = feature?.properties || {}
    const rawCity = String(p.city || '').trim()
    if (!shouldKeepCityPolygon(rawCity, String(p.province || '').trim())) continue

    p.city_key = cityKey({ province: p.province, city: p.city, adcode: p.adcode })
    p.city_name = rawCity
    feature.properties = p
    filteredFeatures.push(feature)
  }
  return filteredFeatures
}
