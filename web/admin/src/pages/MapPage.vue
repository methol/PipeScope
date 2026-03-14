<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { fetchChinaMap, type MapPoint } from '../api/client'
import { formatBytes } from '../utils/format'

const CHINA_MAP_NAME = 'china-cities'
const CHINA_GEOJSON_URL = '/maps/china-cities.geojson'

const windowText = ref('1h')
const metric = ref('conn')
const loading = ref(false)
const error = ref('')
const cityItems = ref<MapPoint[]>([])

const chartEl = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null
let mapReady = false
let mapLoading: Promise<void> | null = null

const title = computed(() => (metric.value === 'bytes' ? '城市流量热度（市级边界）' : '城市连接热度（市级边界）'))
const emptyHint = computed(() => (!loading.value && !error.value && cityItems.value.length === 0 ? '当前窗口暂无城市指标数据' : ''))
const displayValue = (v: number) => (metric.value === 'bytes' ? formatBytes(v) : String(v))

function cityKey(item: { province?: string; city?: string; adcode?: string }) {
  const adcode = String(item.adcode || '').trim()
  if (/^\d{4,}$/.test(adcode)) return adcode.slice(0, 4)

  const province = String(item.province || '').trim().replace(/(省|市|壮族自治区|回族自治区|维吾尔自治区|自治区|特别行政区)$/g, '')
  const city = String(item.city || '').trim().replace(/(市|地区|自治州|盟|县|区|林区)$/g, '')
  return `${province}-${city}`
}

async function ensureChinaMap() {
  if (mapReady) return
  if (mapLoading) return mapLoading

  mapLoading = (async () => {
    const rsp = await fetch(CHINA_GEOJSON_URL)
    if (!rsp.ok) throw new Error(`底图加载失败: ${rsp.status}`)
    const geoJSON = await rsp.json()
    const filteredFeatures: any[] = []
    for (const feature of Array.isArray(geoJSON?.features) ? geoJSON.features : []) {
      const p = feature?.properties || {}
      const rawCity = String(p.city || '').trim()
      if (!rawCity) continue
      // 过滤县区/林区（仅保留地级市/自治州/地区/盟 + 直辖市）
      if (/(县|区|林区)$/.test(rawCity) && !/(市|自治州|地区|盟)$/.test(rawCity)) continue
      p.city_key = cityKey({ province: p.province, city: p.city, adcode: p.adcode })
      p.city_name = rawCity
      feature.properties = p
      filteredFeatures.push(feature)
    }
    geoJSON.features = filteredFeatures
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
    cityItems.value = await fetchChinaMap({ window: windowText.value, metric: metric.value })
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

  const cityData = cityItems.value.map((item) => ({
    name: cityKey(item),
    cityName: item.city,
    value: Number(item.value) || 0,
  }))
  const cityNameByKey = new Map(cityData.map((it) => [it.name, it.cityName]))
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

watch([windowText, metric], () => {
  void load()
})

onMounted(async () => {
  await nextTick()
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
          指标
          <select v-model="metric">
            <option value="conn">连接数</option>
            <option value="bytes">字节数</option>
          </select>
        </label>
        <button class="btn" @click="load">手动刷新</button>
      </div>
    </div>

    <p class="meta">{{ title }} · 分析型页面（不自动刷新）</p>
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
