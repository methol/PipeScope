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
  if (!/^\d+$/.test(raw)) return ''
  if (raw.length >= 6) return raw.slice(0, 6)
  if (raw.length === 4) return `${raw}00`
  if (raw.length === 2) return `${raw}0000`
  return ''
}

function adcodePrefixes(rawAdcode?: string) {
  const raw = String(rawAdcode || '').trim()
  if (!/^\d+$/.test(raw)) return { adcode4: '', adcode2: '' }

  const adcode4 = raw.length >= 4 ? raw.slice(0, 4) : ''
  const adcode2 = raw.length >= 2 ? raw.slice(0, 2) : ''

  return { adcode4, adcode2 }
}

export function cityKey(item: CityLike) {
  const { adcode4 } = adcodePrefixes(item.adcode)
  return adcode4
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

  for (const feature of Array.isArray(features) ? features : []) {
    const p = feature?.properties || {}
    const key = String(p.city_key || cityKey({ province: p.province, city: p.city, adcode: p.adcode })).trim()
    if (key) keySet.add(key)
  }

  return (item: CityLike) => {
    const key = cityKey(item)
    if (!key) return ''
    return keySet.has(key) ? key : ''
  }
}
