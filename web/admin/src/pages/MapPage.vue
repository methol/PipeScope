<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { fetchChinaMap, fetchAnalyticsOptions, type MapPoint, type AnalyticsOptions } from '../api/client'
import { formatBytes } from '../utils/format'
import { createCityJoinKeyResolver, extractProvinceBoundarySegments, normalizeCityGeoFeatures } from './mapCity'

const CHINA_MAP_NAME = 'china-cities'
const CHINA_GEOJSON_URL = '/maps/china-cities.geojson'

type CityMetricsItem = MapPoint & { conn: number; bytes: number }

const windowText = ref('1d')
const metric = ref('conn')
const limit = ref('1000')
const ruleID = ref('')
const status = ref('')
const loading = ref(false)
const optionsLoading = ref(false)
const error = ref('')
const cityItems = ref<CityMetricsItem[]>([])
const options = ref<AnalyticsOptions>({
  rules: [],
  provinces: [],
  cities: [],
  statuses: [],
})
const resolveCityJoinKey = ref<(item: MapPoint) => string>((item) => String(item.adcode || '').trim())
const mapCityNameByKey = ref<Map<string, string>>(new Map())
const mapProvinceNameByKey = ref<Map<string, string>>(new Map())
const provinceBoundarySegments = ref<number[][][]>([])

const chartEl = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null
let mapReady = false
let mapLoading: Promise<void> | null = null

const title = computed(() => (metric.value === 'bytes' ? '城市流量热度（市级边界）' : '城市连接热度（市级边界）'))
const sidebarTitle = computed(() => (metric.value === 'bytes' ? '按流量排序' : '按连接数排序'))
const emptyHint = computed(() => (!loading.value && !error.value && cityItems.value.length === 0 ? '当前窗口暂无城市指标数据' : ''))
const returnedCityCountText = computed(() => `已载入 ${cityItems.value.length} 城市 · Top ${resolveEffectiveLimit()} 上限`)
const returnedCityCountTitle = computed(
  () => `当前窗口实际返回 ${cityItems.value.length} 个城市；Top ${resolveEffectiveLimit()} 只是返回上限，不保证达到该数量。`,
)

function resolveEffectiveLimit(): string {
  const parsedPreset = Number(limit.value)
  if (Number.isFinite(parsedPreset) && parsedPreset > 0) {
    return String(Math.floor(parsedPreset))
  }
  return '1000'
}

function metricValue(item: CityMetricsItem, field: 'conn' | 'bytes'): number {
  return field === 'bytes' ? Number(item.bytes) || 0 : Number(item.conn) || 0
}

const sortedCityItems = computed<CityMetricsItem[]>(() => {
  return [...cityItems.value].sort((a, b) => {
    const delta = metricValue(b, metric.value) - metricValue(a, metric.value)
    if (delta !== 0) return delta
    return String(a.adcode || '').localeCompare(String(b.adcode || ''))
  })
})

async function loadOptions() {
  optionsLoading.value = true
  try {
    options.value = await fetchAnalyticsOptions({
      window: windowText.value,
      rule_id: ruleID.value,
      status: status.value,
    })
  } finally {
    optionsLoading.value = false
  }
}

async function ensureChinaMap() {
  if (mapReady) return
  if (mapLoading) return mapLoading

  mapLoading = (async () => {
    const rsp = await fetch(CHINA_GEOJSON_URL)
    if (!rsp.ok) throw new Error(`底图加载失败: ${rsp.status}`)
    const geoJSON = await rsp.json()
    geoJSON.features = normalizeCityGeoFeatures(Array.isArray(geoJSON?.features) ? geoJSON.features : [])
    resolveCityJoinKey.value = createCityJoinKeyResolver(geoJSON.features)
    mapCityNameByKey.value = new Map(
      geoJSON.features.map((feature: any) => {
        const p = feature?.properties || {}
        const key = String(p.city_key || '').trim()
        const name = String(p.city_name || p.city || '').trim()
        return [key, name] as const
      }),
    )
    mapProvinceNameByKey.value = new Map(
      geoJSON.features.map((feature: any) => {
        const p = feature?.properties || {}
        const key = String(p.city_key || '').trim()
        const province = String(p.province || '').trim()
        return [key, province] as const
      }),
    )
    provinceBoundarySegments.value = extractProvinceBoundarySegments(geoJSON.features)
    echarts.registerMap(CHINA_MAP_NAME, geoJSON)
    mapReady = true
  })()

  try {
    await mapLoading
  } finally {
    mapLoading = null
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    await ensureChinaMap()
    const effectiveLimit = resolveEffectiveLimit()
    const [connItems, bytesItems] = await Promise.all([
      fetchChinaMap({ window: windowText.value, metric: 'conn', limit: effectiveLimit, rule_id: ruleID.value, status: status.value }),
      fetchChinaMap({ window: windowText.value, metric: 'bytes', limit: effectiveLimit, rule_id: ruleID.value, status: status.value }),
    ])

    const itemByKey = new Map<string, CityMetricsItem>()
    for (const item of connItems) {
      const key = resolveCityJoinKey.value(item)
      if (!key) continue
      itemByKey.set(key, { ...item, conn: Number(item.value) || 0, bytes: 0 })
    }
    for (const item of bytesItems) {
      const key = resolveCityJoinKey.value(item)
      if (!key) continue
      const existing = itemByKey.get(key)
      if (existing) {
        existing.bytes = Number(item.value) || 0
        if (!existing.city && item.city) existing.city = item.city
        if (!existing.province && item.province) existing.province = item.province
        continue
      }
      itemByKey.set(key, { ...item, conn: 0, bytes: Number(item.value) || 0 })
    }
    cityItems.value = Array.from(itemByKey.values())
    render()
  } catch (e) {
    cityItems.value = []
    error.value = e instanceof Error ? e.message : 'unknown error'
    render()
  } finally {
    loading.value = false
  }
}

function render() {
  if (!chartEl.value) return
  if (typeof window !== 'undefined' && /jsdom/i.test(window.navigator.userAgent)) return
  if (!chart) chart = echarts.init(chartEl.value, undefined, { renderer: 'canvas' })

  const cityData = sortedCityItems.value
    .map((item) => {
      const key = resolveCityJoinKey.value(item)
      return {
        name: key,
        cityName: item.city,
        province: item.province,
        conn: Number(item.conn) || 0,
        bytes: Number(item.bytes) || 0,
        value: metric.value === 'bytes' ? Number(item.bytes) || 0 : Number(item.conn) || 0,
      }
    })
    .filter((item) => item.name && mapCityNameByKey.value.has(item.name))
  const cityNameByKey = new Map([
    ...mapCityNameByKey.value.entries(),
    ...cityData.map((it) => [it.name, it.cityName] as const),
  ])
  const provinceBoundaryData = provinceBoundarySegments.value.map((coords) => ({ coords }))
  const values = cityData.map((item) => item.value)
  const min = values.length > 0 ? Math.min(...values) : 0
  const max = values.length > 0 ? Math.max(...values) : 1

  chart.setOption({
    backgroundColor: 'transparent',
    geo: {
      map: CHINA_MAP_NAME,
      roam: true,
      silent: false,
      top: 16,
      bottom: 64,
      left: 12,
      right: 12,
      itemStyle: {
        areaColor: '#f4f8ff',
        borderColor: '#99afc9',
        borderWidth: 0.7,
      },
      emphasis: {
        itemStyle: {
          areaColor: '#f4f8ff',
          borderColor: '#99afc9',
          borderWidth: 0.7,
        },
      },
    },
    tooltip: {
      trigger: 'item',
      formatter: (params: { data?: any; name?: string; value?: any }) => {
        const key = String(params.data?.name || params.name || '')
        const cityName = String(params.data?.cityName || cityNameByKey.get(key) || key || '未知城市').trim()
        const province = String(params.data?.province || mapProvinceNameByKey.value.get(key) || '').trim()
        const conn = Number(params.data?.conn ?? 0)
        const bytes = Number(params.data?.bytes ?? 0)
        const header = province ? `${province} / ${cityName}` : cityName
        return `${header}<br/>连接数: ${conn}<br/>流量: ${formatBytes(bytes)}`
      },
    },
    visualMap: {
      min,
      max: max <= min ? min + 1 : max,
      calculable: true,
      orient: 'horizontal',
      left: 'center',
      bottom: 8,
      inRange: {
        color: ['#e8f1ff', '#9dc5f8', '#4f92ea', '#225db8'],
      },
      text: ['高', '低'],
      formatter: (value: number) => (metric.value === 'bytes' ? formatBytes(value) : String(value)),
    },
    series: [
      {
        name: title.value,
        type: 'map',
        map: CHINA_MAP_NAME,
        geoIndex: 0,
        nameProperty: 'city_key',
        data: cityData,
        emphasis: {
          label: {
            show: true,
            formatter: (x: { data?: any; name?: string }) => {
              const key = String(x.data?.name || x.name || '')
              const cityName = String(x.data?.cityName || cityNameByKey.get(key) || key || '').trim()
              return cityName.split('-').pop()
            },
          },
          itemStyle: { areaColor: '#8db5f2' },
        },
        itemStyle: {
          areaColor: '#f4f8ff',
          borderColor: '#99afc9',
          borderWidth: 0.7,
        },
      },
      {
        name: '省界',
        type: 'lines',
        coordinateSystem: 'geo',
        silent: true,
        zlevel: 2,
        z: 20,
        data: provinceBoundaryData,
        lineStyle: {
          color: '#16324f',
          width: 2.8,
          opacity: 1,
        },
      },
    ],
  })
}

watch(windowText, async () => {
  await loadOptions()
  void load()
})

watch([ruleID, status, limit], () => {
  void load()
})

watch(metric, () => {
  render()
})

onMounted(async () => {
  await nextTick()
  await loadOptions()
  await load()
  window.addEventListener('resize', onResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', onResize)
  if (chart) {
    chart.dispose()
    chart = null
  }
})

function onResize() {
  chart?.resize()
}
</script>

<template>
  <section class="panel">
    <div class="panel-header">
      <h2>中国地图视图（市级）</h2>
      <div class="filters">
        <label>
          窗口
          <select v-model="windowText">
            <option value="15m">15m</option>
            <option value="1h">1h</option>
            <option value="1d">1d</option>
            <option value="1w">1w</option>
            <option value="1mo">1mo</option>
          </select>
        </label>
        <label>
          Rule
          <select v-model="ruleID" :disabled="optionsLoading">
            <option value="">全部</option>
            <option v-for="item in options.rules" :key="item" :value="item">{{ item }}</option>
          </select>
        </label>
        <label>
          状态
          <select v-model="status" :disabled="optionsLoading">
            <option value="">全部</option>
            <option v-for="item in options.statuses" :key="item" :value="item">{{ item }}</option>
          </select>
        </label>
        <label>
          指标
          <select v-model="metric">
            <option value="conn">连接数</option>
            <option value="bytes">流量</option>
          </select>
        </label>
        <label>
          Top
          <select v-model="limit">
            <option value="100">100</option>
            <option value="1000">1000</option>
            <option value="5000">5000</option>
            <option value="10000">10000</option>
          </select>
        </label>
        <button class="btn" @click="load">手动刷新</button>
      </div>
    </div>

    <p class="meta">{{ title }} · 分析型页面（不自动刷新）</p>
    <p v-if="optionsLoading" class="meta">筛选项加载中...</p>
    <p v-if="loading" class="meta">加载中...</p>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="emptyHint" class="meta">{{ emptyHint }}</p>

    <div class="map-layout">
      <div class="map-main">
        <div class="map-main-shell">
          <div ref="chartEl" class="chart"></div>
        </div>
      </div>

      <aside class="map-sidebar">
        <div class="map-sidebar-body">
          <div class="sidebar-header">
            <p class="sidebar-eyebrow">城市统计</p>
            <h3>{{ sidebarTitle }}</h3>
            <p class="meta sidebar-meta" :title="returnedCityCountTitle">{{ returnedCityCountText }}</p>
          </div>

          <div class="city-list-scroll">
            <ul class="city-list">
              <li v-for="item in sortedCityItems" :key="item.adcode + item.city">
                <div class="city-copy">
                  <strong class="city-name">{{ item.city }}</strong>
                  <span class="city-province">{{ item.province }}</span>
                </div>
                <div class="city-stats">
                  <span class="city-stat city-stat-conn" :title="`连接数 ${item.conn}`">
                    <span class="city-stat-icon city-stat-icon--conn" aria-hidden="true">
                      <svg viewBox="0 0 16 16" focusable="false">
                        <path d="M4 4.5h8M4 11.5h8M4 4.5l8 7" />
                        <circle cx="4" cy="4.5" r="1.5" />
                        <circle cx="12" cy="4.5" r="1.5" />
                        <circle cx="12" cy="11.5" r="1.5" />
                      </svg>
                    </span>
                    <span class="sr-only">连接数</span>
                    <span class="city-stat-value">{{ item.conn }}</span>
                  </span>
                  <span class="city-stat city-stat-bytes" :title="`流量 ${formatBytes(item.bytes)}`">
                    <span class="city-stat-icon city-stat-icon--bytes" aria-hidden="true">
                      <svg viewBox="0 0 16 16" focusable="false">
                        <path d="M5 12V4M5 4 2.75 6.25M5 4l2.25 2.25M11 4v8M11 12 8.75 9.75M11 12l2.25-2.25" />
                      </svg>
                    </span>
                    <span class="sr-only">流量</span>
                    <span class="city-stat-value">{{ formatBytes(item.bytes) }}</span>
                  </span>
                </div>
              </li>
            </ul>
          </div>
        </div>
      </aside>
    </div>
  </section>
</template>

<style scoped>
.map-layout {
  --map-panel-height: clamp(500px, calc(100vh - 320px), 680px);
  --map-panel-height: clamp(500px, calc(100dvh - 320px), 680px);
  margin-top: 12px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 300px;
  gap: 16px;
  align-items: stretch;
  overflow: hidden;
}

.map-main {
  min-width: 0;
  min-height: 0;
  height: var(--map-panel-height);
}

.map-main-shell {
  height: 100%;
  min-height: 0;
}

.map-main-shell .chart {
  height: 100%;
  min-height: 100%;
}

.map-sidebar {
  min-width: 0;
  min-height: 0;
  height: var(--map-panel-height);
  display: flex;
  flex-direction: column;
  padding-left: 16px;
  border-left: 1px solid rgba(16, 36, 63, 0.1);
  overflow: hidden;
}

.map-sidebar-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow: hidden;
}

.sidebar-header {
  display: grid;
  gap: 4px;
}

.sidebar-eyebrow {
  margin: 0;
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--ink-soft);
}

.sidebar-header h3 {
  margin: 0;
  font-size: 16px;
}

.sidebar-meta {
  margin: 0;
}

.city-list-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding-right: 6px;
}

.city-list {
  margin-top: 0;
  grid-template-columns: 1fr;
  gap: 10px;
}

.city-list li {
  align-items: flex-start;
  gap: 10px;
}

.city-copy {
  display: grid;
  gap: 3px;
}

.city-name {
  font-size: 14px;
}

.city-province {
  color: var(--ink-soft);
  font-size: 12px;
}

.city-stats {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
}

.city-stat {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 4px 10px 4px 6px;
  border-radius: 999px;
  background: rgba(31, 122, 140, 0.12);
  color: var(--ink);
  font-size: 12px;
  line-height: 1;
  white-space: nowrap;
}

.city-stat-conn {
  background: linear-gradient(135deg, rgba(34, 93, 184, 0.14), rgba(34, 93, 184, 0.08));
}

.city-stat-bytes {
  background: linear-gradient(135deg, rgba(255, 140, 66, 0.2), rgba(255, 140, 66, 0.08));
}

.city-stat-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 999px;
  flex: 0 0 18px;
}

.city-stat-icon svg {
  width: 12px;
  height: 12px;
  fill: none;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 1.4;
}

.city-stat-icon--conn {
  background: rgba(34, 93, 184, 0.14);
  color: #225db8;
}

.city-stat-icon--conn svg circle {
  fill: currentColor;
  stroke-width: 0;
}

.city-stat-icon--bytes {
  background: rgba(255, 140, 66, 0.18);
  color: #d66724;
}

.city-stat-value {
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

@media (max-width: 980px) {
  .map-layout {
    grid-template-columns: 1fr;
  }

  .map-main,
  .map-sidebar {
    height: auto;
    padding-left: 0;
    padding-top: 14px;
    border-left: none;
    border-top: 1px solid rgba(16, 36, 63, 0.1);
  }

  .map-main {
    padding-top: 0;
    border-top: none;
  }

  .map-main-shell .chart {
    min-height: 320px;
    height: clamp(320px, 55vh, 440px);
  }

  .city-list-scroll {
    flex: initial;
    max-height: min(50vh, 420px);
  }
}
</style>
