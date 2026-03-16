<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { fetchChinaMap, fetchAnalyticsOptions, type MapPoint, type AnalyticsOptions } from '../api/client'
import { formatBytes } from '../utils/format'
import { createCityJoinKeyResolver, normalizeCityGeoFeatures } from './mapCity'

const CHINA_MAP_NAME = 'china-cities'
const CHINA_GEOJSON_URL = '/maps/china-cities.geojson'

const windowText = ref('1h')
const metric = ref('conn')
const limit = ref('100')
const ruleID = ref('')
const status = ref('')
const loading = ref(false)
const optionsLoading = ref(false)
const error = ref('')
const cityItems = ref<MapPoint[]>([])
const options = ref<AnalyticsOptions>({
  rules: [],
  provinces: [],
  cities: [],
  statuses: [],
})
const resolveCityJoinKey = ref<(item: MapPoint) => string>((item) => String(item.adcode || '').trim())
const mapCityNameByKey = ref<Map<string, string>>(new Map())

const chartEl = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null
let mapReady = false
let mapLoading: Promise<void> | null = null

const title = computed(() => (metric.value === 'bytes' ? '城市流量热度（市级边界）' : '城市连接热度（市级边界）'))
const emptyHint = computed(() => (!loading.value && !error.value && cityItems.value.length === 0 ? '当前窗口暂无城市指标数据' : ''))
const displayValue = (v: number) => (metric.value === 'bytes' ? formatBytes(v) : String(v))

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
    cityItems.value = await fetchChinaMap({
      window: windowText.value,
      metric: metric.value,
      limit: limit.value,
      rule_id: ruleID.value,
      status: status.value,
    })
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

  const cityData = cityItems.value
    .map((item) => ({
      name: resolveCityJoinKey.value(item),
      cityName: item.city,
      value: Number(item.value) || 0,
    }))
    .filter((item) => item.name && mapCityNameByKey.value.has(item.name))
  const cityNameByKey = new Map([
    ...mapCityNameByKey.value.entries(),
    ...cityData.map((it) => [it.name, it.cityName] as const),
  ])
  const values = cityData.map((item) => item.value)
  const min = values.length > 0 ? Math.min(...values) : 0
  const max = values.length > 0 ? Math.max(...values) : 1

  chart.setOption({
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item',
      formatter: (params: { data?: any; name?: string; value?: any }) => {
        const key = String(params.data?.name || params.name || '')
        const cityName = String(params.data?.cityName || cityNameByKey.get(key) || key || '未知城市').trim()
        const val = Number(params.data?.value ?? params.value ?? 0)
        return `${cityName}<br/>${title.value}: ${displayValue(val)}`
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
        nameProperty: 'city_key',
        roam: true,
        data: cityData,
        emphasis: {
          label: {
            show: true,
            formatter: (x: { data?: any; name?: string }) => String(x.data?.cityName || x.name || '').split('-').pop(),
          },
          itemStyle: { areaColor: '#8db5f2' },
        },
        itemStyle: {
          areaColor: '#f4f8ff',
          borderColor: '#99afc9',
          borderWidth: 0.7,
        },
      },
    ],
  })
}

watch(windowText, async () => {
  await loadOptions()
})

watch([metric, limit, ruleID, status], () => {
  void load()
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
            <option value="bytes">字节数</option>
          </select>
        </label>
        <label>
          Top
          <select v-model="limit">
            <option value="10">10</option>
            <option value="50">50</option>
            <option value="100">100</option>
            <option value="1000">1000</option>
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

    <div ref="chartEl" class="chart"></div>

    <ul class="city-list">
      <li v-for="item in cityItems.slice(0, 8)" :key="item.adcode + item.city">
        <span>{{ item.city }}</span>
        <strong>{{ displayValue(item.value) }}</strong>
      </li>
    </ul>
  </section>
</template>
