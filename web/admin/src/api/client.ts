export interface MapPoint {
  adcode: string
  province: string
  city: string
  lat: number
  lng: number
  value: number
}

export interface RulePoint {
  rule_id: string
  conn_count: number
  total_bytes: number
}

export interface SessionItem {
  id: number
  rule_id: string
  src_addr: string
  dst_addr: string
  status: string
  up_bytes: number
  down_bytes: number
  total_bytes: number
  start_ts: number
  end_ts: number
  duration_ms: number
  province: string
  city: string
  adcode: string
}

export interface Overview {
  conn_count: number
  total_bytes: number
}

export interface AnalyticsOverview {
  conn_count: number
  total_bytes: number
  avg_duration_ms: number
  active_rules: number
  active_cities: number
}

export interface AnalyticsBucket {
  name: string
  conn_count: number
  total_bytes: number
}

export interface AnalyticsResult {
  overview: AnalyticsOverview
  top_cities: AnalyticsBucket[]
  top_rules: AnalyticsBucket[]
}

export interface AnalyticsCityOption {
  province: string
  city: string
}

export interface AnalyticsOptions {
  rules: string[]
  provinces: string[]
  cities: AnalyticsCityOption[]
  statuses: string[]
}

async function fetchJSON<T>(url: string): Promise<T> {
  const rsp = await fetch(url)
  if (!rsp.ok) {
    throw new Error(`request failed: ${rsp.status}`)
  }
  return rsp.json() as Promise<T>
}

export async function fetchChinaMap(params: { window: string; metric: string }): Promise<MapPoint[]> {
  const q = new URLSearchParams(params)
  const rsp = await fetchJSON<{ items: MapPoint[] }>(`/api/map/china?${q.toString()}`)
  return rsp.items ?? []
}

export interface ProvinceSummaryPoint {
  province: string
  value: number
}

export async function fetchProvinceMap(params: {
  window: string
  metric: string
  province: string
}): Promise<MapPoint[]> {
  const q = new URLSearchParams(params)
  const rsp = await fetchJSON<{ items: MapPoint[] }>(`/api/map/province?${q.toString()}`)
  return rsp.items ?? []
}

export async function fetchProvinceSummary(params: { window: string; metric: string }): Promise<ProvinceSummaryPoint[]> {
  const q = new URLSearchParams(params)
  const rsp = await fetchJSON<{ items: ProvinceSummaryPoint[] }>(`/api/map/province-summary?${q.toString()}`)
  return rsp.items ?? []
}

export async function fetchRules(params: { window: string }): Promise<RulePoint[]> {
  const q = new URLSearchParams(params)
  const rsp = await fetchJSON<{ items: RulePoint[] }>(`/api/rules?${q.toString()}`)
  return rsp.items ?? []
}

export async function fetchSessions(params: {
  window: string
  rule_id?: string
  limit?: string
  offset?: string
}): Promise<SessionItem[]> {
  const q = new URLSearchParams()
  q.set('window', params.window)
  if (params.rule_id) q.set('rule_id', params.rule_id)
  if (params.limit) q.set('limit', params.limit)
  if (params.offset) q.set('offset', params.offset)
  const rsp = await fetchJSON<{ items: SessionItem[] }>(`/api/sessions?${q.toString()}`)
  return rsp.items ?? []
}

export async function fetchOverview(params: { window: string }): Promise<Overview> {
  const q = new URLSearchParams(params)
  return fetchJSON<Overview>(`/api/overview?${q.toString()}`)
}

export async function fetchAnalytics(params: {
  window: string
  rule_id?: string
  province?: string
  city?: string
  status?: string
  top_n?: string
}): Promise<AnalyticsResult> {
  const q = new URLSearchParams()
  q.set('window', params.window)
  if (params.rule_id) q.set('rule_id', params.rule_id)
  if (params.province) q.set('province', params.province)
  if (params.city) q.set('city', params.city)
  if (params.status) q.set('status', params.status)
  if (params.top_n) q.set('top_n', params.top_n)
  return fetchJSON<AnalyticsResult>(`/api/analytics?${q.toString()}`)
}

export async function fetchAnalyticsOptions(params: {
  window: string
  rule_id?: string
  province?: string
  city?: string
  status?: string
}): Promise<AnalyticsOptions> {
  const q = new URLSearchParams()
  q.set('window', params.window)
  if (params.rule_id) q.set('rule_id', params.rule_id)
  if (params.province) q.set('province', params.province)
  if (params.city) q.set('city', params.city)
  if (params.status) q.set('status', params.status)
  return fetchJSON<AnalyticsOptions>(`/api/analytics/options?${q.toString()}`)
}
