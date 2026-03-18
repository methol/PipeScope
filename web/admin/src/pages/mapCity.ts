export type CityLike = { province?: string; city?: string; adcode?: string }
export type BoundarySegment = [[number, number], [number, number]]

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

export function cityKey(item: CityLike) {
  const raw = String(item.adcode || '').trim()
  if (!/^\d+$/.test(raw) || raw.length < 4) return ''
  return normalizeAdcode6(raw)
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
    // ECharts map region matching is safest with canonical `name` set explicitly.
    // Keep it aligned with city_key to avoid runtime nameProperty drift/caching mismatch.
    p.name = p.city_key
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

function normalizePoint(raw: any): [number, number] | null {
  if (!Array.isArray(raw) || raw.length < 2) return null
  const lng = Number(raw[0])
  const lat = Number(raw[1])
  if (!Number.isFinite(lng) || !Number.isFinite(lat)) return null
  return [Number(lng.toFixed(6)), Number(lat.toFixed(6))]
}

function pointKey(point: [number, number]) {
  return `${point[0].toFixed(6)},${point[1].toFixed(6)}`
}

function segmentKey(a: [number, number], b: [number, number]) {
  const [p1, p2] = [pointKey(a), pointKey(b)].sort()
  return `${p1}|${p2}`
}

function forEachRing(feature: any, visit: (ring: any[]) => void) {
  const geometry = feature?.geometry
  if (!geometry) return

  if (geometry.type === 'Polygon') {
    for (const ring of Array.isArray(geometry.coordinates) ? geometry.coordinates : []) {
      visit(ring)
    }
    return
  }

  if (geometry.type === 'MultiPolygon') {
    for (const polygon of Array.isArray(geometry.coordinates) ? geometry.coordinates : []) {
      for (const ring of Array.isArray(polygon) ? polygon : []) {
        visit(ring)
      }
    }
  }
}

export function extractProvinceBoundarySegments(features: any[]): BoundarySegment[] {
  const segmentOwners = new Map<string, { segment: BoundarySegment; provinces: Set<string> }>()

  for (const feature of Array.isArray(features) ? features : []) {
    const province = String(feature?.properties?.province || '').trim()
    if (!province) continue

    forEachRing(feature, (ring) => {
      const points = Array.isArray(ring)
        ? ring.map((point) => normalizePoint(point)).filter((point): point is [number, number] => point !== null)
        : []
      if (points.length < 2) return

      for (let i = 1; i < points.length; i += 1) {
        const start = points[i - 1]
        const end = points[i]
        if (pointKey(start) === pointKey(end)) continue

        const key = segmentKey(start, end)
        const existing = segmentOwners.get(key)
        if (existing) {
          existing.provinces.add(province)
          continue
        }
        segmentOwners.set(key, { segment: [start, end], provinces: new Set([province]) })
      }

      const first = points[0]
      const last = points[points.length - 1]
      if (pointKey(first) === pointKey(last)) return

      const key = segmentKey(last, first)
      const existing = segmentOwners.get(key)
      if (existing) {
        existing.provinces.add(province)
        return
      }
      segmentOwners.set(key, { segment: [last, first], provinces: new Set([province]) })
    })
  }

  const out: BoundarySegment[] = []
  for (const item of segmentOwners.values()) {
    if (item.provinces.size < 2) continue
    out.push(item.segment)
  }
  return out
}
