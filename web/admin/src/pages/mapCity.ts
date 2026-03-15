export type CityLike = { province?: string; city?: string; adcode?: string }

const MUNICIPALITIES = new Set(['北京市', '天津市', '上海市', '重庆市'])
const DIRECT_ADMIN_COUNTY_CODES = new Set([
  '429021', // 神农架林区
  '469021',
  '469022',
  '469023',
  '469024',
  '469025',
  '469026',
  '469027',
  '469028',
  '469029',
  '469030',
])

export function normalizeAdcode6(adcode?: string) {
  const raw = String(adcode || '').trim()
  return /^\d{6,}$/.test(raw) ? raw.slice(0, 6) : ''
}

function normalizedRegionName(raw?: string) {
  return String(raw || '')
    .trim()
    .replace(/(省|市|壮族自治区|回族自治区|维吾尔自治区|自治区|特别行政区)$/g, '')
}

function normalizedCityName(raw?: string) {
  return String(raw || '')
    .trim()
    .replace(/(市|地区|自治州|盟|县|区|林区)$/g, '')
}

function cityNameKey(item: CityLike) {
  const province = normalizedRegionName(item.province)
  const city = normalizedCityName(item.city)
  return `${province}-${city}`
}

export function cityKey(item: CityLike) {
  const adcode = normalizeAdcode6(item.adcode)
  if (adcode) return adcode
  return cityNameKey(item)
}

function isDirectAdminCounty(adcode?: string) {
  return DIRECT_ADMIN_COUNTY_CODES.has(normalizeAdcode6(adcode))
}

export function shouldKeepCityPolygon(city?: string, province?: string, adcode?: string) {
  const rawCity = String(city || '').trim()
  if (!rawCity) return false

  if (isDirectAdminCounty(adcode)) return true

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
    if (!shouldKeepCityPolygon(rawCity, String(p.province || '').trim(), String(p.adcode || '').trim())) continue

    p.city_key = cityKey({ province: p.province, city: p.city, adcode: p.adcode })
    p.city_name = rawCity
    feature.properties = p
    filteredFeatures.push(feature)
  }
  return filteredFeatures
}

export function createCityJoinKeyResolver(features: any[]) {
  const keySet = new Set<string>()
  const adcodeToKey = new Map<string, string>()
  const nameKeyToKey = new Map<string, string>()

  for (const feature of Array.isArray(features) ? features : []) {
    const p = feature?.properties || {}
    const key = String(p.city_key || cityKey({ province: p.province, city: p.city, adcode: p.adcode })).trim()
    if (!key) continue

    keySet.add(key)

    const adcode = normalizeAdcode6(p.adcode)
    if (adcode && !adcodeToKey.has(adcode)) adcodeToKey.set(adcode, key)

    const fallbackNameKey = cityNameKey({ province: p.province, city: p.city })
    if (fallbackNameKey && !nameKeyToKey.has(fallbackNameKey)) nameKeyToKey.set(fallbackNameKey, key)
  }

  return (item: CityLike) => {
    const adcode = normalizeAdcode6(item.adcode)
    const fallbackNameKey = cityNameKey(item)

    if (adcode && keySet.has(adcode)) return adcode
    if (adcode && adcodeToKey.has(adcode)) return adcodeToKey.get(adcode) as string

    if (fallbackNameKey && keySet.has(fallbackNameKey)) return fallbackNameKey
    if (fallbackNameKey && nameKeyToKey.has(fallbackNameKey)) return nameKeyToKey.get(fallbackNameKey) as string

    return adcode || fallbackNameKey
  }
}
