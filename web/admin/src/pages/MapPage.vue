<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { fetchChinaMap, type MapPoint } from '../api/client'

const CHINA_MAP_NAME = 'china-counties'
const CHINA_GEOJSON_URL = '/maps/china-counties.geojson'

const windowText = ref('15m')
const metric = ref('conn')
const loading = ref(false)
const error = ref('')
const items = ref<MapPoint[]>([])

const chartEl = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null
let timer: number | null = null
let mapReady = false
let mapLoading: Promise<void> | null = null

const title = computed(() => (metric.value === 'bytes' ? '流量热度' : '连接热度'))
const emptyHint = computed(() => (!loading.value && !error.value && items.value.length === 0 ? '当前窗口暂无城市指标数据' : ''))

async function ensureChinaMap() {
  if (mapReady) return
  if (mapLoading) return mapLoading

  mapLoading = (async () => {
    const rsp = await fetch(CHINA_GEOJSON_URL)
    if (!rsp.ok) {
      throw new Error(`底图加载失败: ${rsp.status}`)
    }
    const geoJSON = await rsp.json()
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
    items.value = await fetchChinaMap({ window: windowText.value, metric: metric.value })
    render()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'unknown error'
    render()
  } finally {
    loading.value = false
  }
}

function toScatterData(data: MapPoint[]) {
  return data
    .filter((item) => Number.isFinite(item.lng) && Number.isFinite(item.lat))
    .map((item) => ({
      name: item.city || item.adcode,
      value: [item.lng, item.lat, item.value],
      province: item.province,
      city: item.city,
      adcode: item.adcode,
    }))
}

function render() {
  if (!chartEl.value) return
  if (typeof window !== 'undefined' && /jsdom/i.test(window.navigator.userAgent)) return
  if (!chart) {
    chart = echarts.init(chartEl.value, undefined, { renderer: 'canvas' })
  }

  const scatterData = toScatterData(items.value)
  const values = scatterData.map((item) => Number(item.value[2]) || 0)
  const min = values.length > 0 ? Math.min(...values) : 0
  const max = values.length > 0 ? Math.max(...values) : 1

  chart.setOption({
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item',
      formatter: (params: { seriesType?: string; data?: any; name?: string; value?: any }) => {
        if (params.seriesType === 'scatter' && params.data) {
          const v = Array.isArray(params.value) ? params.value[2] : params.data.value?.[2]
          return `${params.data.province || ''}${params.data.city || params.name}<br/>${title.value}: ${v ?? 0}`
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
        color: ['#e8f1ff', '#7ab6f9', '#2c7be5', '#1b4f9a'],
      },
      text: ['高', '低'],
    },
    geo: {
      map: CHINA_MAP_NAME,
      roam: true,
      silent: true,
      emphasis: { disabled: true },
      itemStyle: {
        areaColor: '#f7faff',
        borderColor: '#a8bfd8',
        borderWidth: 0.7,
      },
    },
    series: [
      {
        name: title.value,
        type: 'scatter',
        coordinateSystem: 'geo',
        data: scatterData,
        symbolSize: (val: unknown) => {
          const v = Array.isArray(val) ? Number(val[2]) || 0 : 0
          return Math.max(4, Math.min(22, Math.sqrt(v) * 1.3))
        },
        encode: { value: 2 },
        emphasis: {
          scale: true,
          itemStyle: { borderColor: '#fff', borderWidth: 1 },
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
      <h2>中国地图视图</h2>
      <div class="filters">
        <label>
          窗口
          <select v-model="windowText">
            <option value="5m">5m</option>
            <option value="15m">15m</option>
            <option value="1h">1h</option>
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

    <p class="meta">{{ title }} · 每 5 秒自动刷新</p>
    <p v-if="loading" class="meta">加载中...</p>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="emptyHint" class="meta">{{ emptyHint }}</p>

    <div ref="chartEl" class="chart"></div>

    <ul class="city-list">
      <li v-for="item in items.slice(0, 8)" :key="item.adcode + item.city">
        <span>{{ item.province }}{{ item.city }}</span>
        <strong>{{ item.value }}</strong>
      </li>
    </ul>
  </section>
</template>
