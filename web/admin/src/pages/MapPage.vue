<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { fetchChinaMap, fetchProvinceSummary, type MapPoint, type ProvinceSummaryPoint } from '../api/client'
import { formatBytes } from '../utils/format'

const CHINA_MAP_NAME = 'china-counties'
const CHINA_GEOJSON_URL = '/maps/china-counties.geojson'

const windowText = ref('15m')
const metric = ref('conn')
const loading = ref(false)
const error = ref('')
const cityItems = ref<MapPoint[]>([])
const provinceItems = ref<ProvinceSummaryPoint[]>([])
const mapProvinceNames = ref<Set<string>>(new Set())

const chartEl = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null
let timer: number | null = null
let mapReady = false
let mapLoading: Promise<void> | null = null

const title = computed(() => (metric.value === 'bytes' ? '城市流量热度（Grid）' : '城市连接热度（Grid）'))
const emptyHint = computed(() => (!loading.value && !error.value && cityItems.value.length === 0 ? '当前窗口暂无城市指标数据' : ''))
const displayValue = (v: number) => (metric.value === 'bytes' ? formatBytes(v) : String(v))

const PROVINCE_CANONICAL_MAP: Record<string, string> = {
  北京: '北京市',
  天津: '天津市',
  上海: '上海市',
  重庆: '重庆市',
  内蒙古: '内蒙古自治区',
  广西: '广西壮族自治区',
  宁夏: '宁夏回族自治区',
  新疆: '新疆维吾尔自治区',
  西藏: '西藏自治区',
  香港: '香港特别行政区',
  澳门: '澳门特别行政区',
  新疆生产建设兵团: '新疆维吾尔自治区',
}

function canonicalProvinceName(raw: string): string {
  const name = (raw || '').trim()
  if (!name) return '未知'
  if (PROVINCE_CANONICAL_MAP[name]) return PROVINCE_CANONICAL_MAP[name]

  const normalized = name
    .replace(/省$/, '')
    .replace(/市$/, '')
    .replace(/壮族自治区$/, '')
    .replace(/回族自治区$/, '')
    .replace(/维吾尔自治区$/, '')
    .replace(/特别行政区$/, '')
    .replace(/自治区$/, '')

  return PROVINCE_CANONICAL_MAP[normalized] ?? `${normalized}省`
}

function provinceColor(name: string): string {
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = (hash * 31 + name.charCodeAt(i)) >>> 0
  const hue = hash % 360
  return `hsl(${hue} 45% 80%)`
}

function provinceRegions(points: ProvinceSummaryPoint[]) {
  const dataByProvince = new Map<string, number>()
  for (const it of points) dataByProvince.set(canonicalProvinceName(it.province), it.value)
  return Array.from(dataByProvince.entries()).map(([name, value]) => ({
    name,
    value,
    itemStyle: { areaColor: provinceColor(name) },
  }))
}

const provinceCoverage = computed(() => {
  const regions = provinceRegions(provinceItems.value)
  if (regions.length === 0) return '省份命中率: 0/0'
  const total = regions.length
  const hit = regions.filter((item) => mapProvinceNames.value.has(item.name)).length
  return `省份命中率: ${hit}/${total}`
})

function cityHeatData(data: MapPoint[]) {
  return data
    .filter((item) => Number.isFinite(item.lng) && Number.isFinite(item.lat))
    .map((item) => [item.lng, item.lat, item.value] as [number, number, number])
}

async function ensureChinaMap() {
  if (mapReady) return
  if (mapLoading) return mapLoading

  mapLoading = (async () => {
    const rsp = await fetch(CHINA_GEOJSON_URL)
    if (!rsp.ok) throw new Error(`底图加载失败: ${rsp.status}`)
    const geoJSON = await rsp.json()
    const provinceSet = new Set<string>()
    for (const feature of Array.isArray(geoJSON?.features) ? geoJSON.features : []) {
      const province = String(feature?.properties?.province || '').trim()
      if (province) provinceSet.add(province)
    }
    mapProvinceNames.value = provinceSet
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
    const [cities, provinces] = await Promise.all([
      fetchChinaMap({ window: windowText.value, metric: metric.value }),
      fetchProvinceSummary({ window: windowText.value, metric: metric.value }),
    ])
    cityItems.value = cities
    provinceItems.value = provinces
    render()
  } catch (e) {
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

  const heatData = cityHeatData(cityItems.value)
  const values = heatData.map((item) => Number(item[2]) || 0)
  const min = values.length > 0 ? Math.min(...values) : 0
  const max = values.length > 0 ? Math.max(...values) : 1

  chart.setOption({
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item',
      formatter: (params: { seriesType?: string; data?: any; name?: string; value?: any }) => {
        if (params.seriesType === 'heatmap') {
          const val = Array.isArray(params.value) ? Number(params.value[2] ?? 0) : 0
          return `${title.value}: ${displayValue(val)}`
        }
        if (params.seriesType === 'map') {
          const val = Number(params.data?.value ?? 0)
          return `${params.name}<br/>${title.value}: ${displayValue(val)}`
        }
        return params.name || ''
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
    },
    geo: {
      map: CHINA_MAP_NAME,
      nameProperty: 'province',
      roam: true,
      silent: false,
      itemStyle: {
        areaColor: '#f4f8ff',
        borderColor: '#99afc9',
        borderWidth: 0.7,
      },
      emphasis: { itemStyle: { areaColor: '#d8e7fb' } },
      regions: provinceRegions(provinceItems.value),
    },
    series: [
      {
        name: '省份底色',
        type: 'map',
        map: CHINA_MAP_NAME,
        nameProperty: 'province',
        geoIndex: 0,
        data: provinceRegions(provinceItems.value),
        silent: true,
        zlevel: 0,
      },
      {
        name: title.value,
        type: 'heatmap',
        coordinateSystem: 'geo',
        data: heatData,
        pointSize: 8,
        blurSize: 18,
        zlevel: 1,
      },
    ],
  })
}

watch([windowText, metric], () => {
  void load()
})

onMounted(async () => {
  await nextTick()
  await load()
  timer = window.setInterval(() => {
    void load()
  }, 5000)
  window.addEventListener('resize', onResize)
})

onUnmounted(() => {
  if (timer !== null) window.clearInterval(timer)
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
      <h2>中国地图视图（城市级）</h2>
      <div class="filters">
        <label>
          窗口
          <select v-model="windowText">
            <option value="5m">5m</option>
            <option value="15m">15m</option>
            <option value="1h">1h</option>
            <option value="1d">1d</option>
            <option value="1w">1w</option>
            <option value="1mo">1mo</option>
          </select>
        </label>
        <label>
          指标
          <select v-model="metric">
            <option value="conn">连接数</option>
            <option value="bytes">字节数</option>
          </select>
        </label>
      </div>
    </div>

    <p class="meta">{{ title }} · 省份低饱和分色 · {{ provinceCoverage }} · 每 5 秒自动刷新</p>
    <p v-if="loading" class="meta">加载中...</p>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="emptyHint" class="meta">{{ emptyHint }}</p>

    <div ref="chartEl" class="chart"></div>

    <ul class="city-list">
      <li v-for="item in cityItems.slice(0, 8)" :key="item.adcode + item.city">
        <span>{{ item.province }}{{ item.city }}</span>
        <strong>{{ displayValue(item.value) }}</strong>
      </li>
    </ul>
  </section>
</template>
