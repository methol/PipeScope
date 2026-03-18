<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { fetchChinaMap, fetchAnalyticsOptions, type MapPoint, type AnalyticsOptions } from '../api/client'
import { formatBytes } from '../utils/format'
import { createCityJoinKeyResolver, extractProvinceBoundarySegments, normalizeCityGeoFeatures } from './mapCity'

const CHINA_MAP_NAME = 'china-cities'
const CHINA_GEOJSON_URL = '/maps/china-cities.geojson'

type CityMetricsItem = MapPoint & { conn: number; bytes: number }

const windowText = ref('1h')
const metric = ref('conn')
const sortBy = ref<'conn' | 'bytes'>('conn')
const sortOrder = ref<'desc' | 'asc'>('desc')
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
const emptyHint = computed(() => (!loading.value && !error.value && cityItems.value.length === 0 ? '当前窗口暂无城市指标数据' : ''))
const returnedCityCountText = computed(
  () => `当前窗口返回 ${cityItems.value.length} 城市（Top ${resolveEffectiveLimit()} 为上限，不是保底）`,
)
const displayValue = (v: number) => (metric.value === 'bytes' ? formatBytes(v) : String(v))

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
  const order = sortOrder.value === 'asc' ? 1 : -1
  return [...cityItems.value].sort((a, b) => {
    const delta = metricValue(a, sortBy.value) - metricValue(b, sortBy.value)
    if (delta !== 0) return delta * order
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

watch([metric, sortBy, sortOrder], () => {
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
          地图着色
          <select v-model="metric">
            <option value="conn">按连接数</option>
            <option value="bytes">按流量</option>
          </select>
        </label>
        <label>
          排序
          <select v-model="sortBy">
            <option value="conn">连接数</option>
            <option value="bytes">流量</option>
          </select>
        </label>
        <label>
          顺序
          <select v-model="sortOrder">
            <option value="desc">降序</option>
            <option value="asc">升序</option>
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
    <p class="meta">{{ returnedCityCountText }}</p>
    <p v-if="optionsLoading" class="meta">筛选项加载中...</p>
    <p v-if="loading" class="meta">加载中...</p>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="emptyHint" class="meta">{{ emptyHint }}</p>

    <div ref="chartEl" class="chart"></div>

    <ul class="city-list">
      <li v-for="item in sortedCityItems" :key="item.adcode + item.city">
        <span>{{ item.province }} / {{ item.city }}</span>
        <strong>连接 {{ item.conn }} · 流量 {{ formatBytes(item.bytes) }}</strong>
      </li>
    </ul>
  </section>
</template>
