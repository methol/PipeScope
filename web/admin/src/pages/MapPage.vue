<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { fetchChinaMap, type MapPoint } from '../api/client'

const windowText = ref('15m')
const metric = ref('conn')
const loading = ref(false)
const error = ref('')
const items = ref<MapPoint[]>([])

const chartEl = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null
let timer: number | null = null

const title = computed(() => (metric.value === 'bytes' ? '流量热度' : '连接热度'))

async function load() {
  loading.value = true
  error.value = ''
  try {
    items.value = await fetchChinaMap({ window: windowText.value, metric: metric.value })
    render()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'unknown error'
  } finally {
    loading.value = false
  }
}

function render() {
  if (!chartEl.value) return
  if (typeof window !== 'undefined' && /jsdom/i.test(window.navigator.userAgent)) return
  if (!chart) {
    chart = echarts.init(chartEl.value, undefined, { renderer: 'svg' })
  }
  const top = [...items.value]
    .sort((a, b) => b.value - a.value)
    .slice(0, 12)

  chart.setOption({
    backgroundColor: 'transparent',
    grid: { left: 18, right: 24, top: 30, bottom: 36, containLabel: true },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: {
      type: 'category',
      axisLabel: { rotate: 30, color: '#183153' },
      data: top.map((v) => v.city || v.adcode),
    },
    yAxis: { type: 'value', axisLabel: { color: '#183153' } },
    series: [
      {
        type: 'bar',
        barWidth: 22,
        data: top.map((v) => v.value),
        itemStyle: {
          borderRadius: [8, 8, 0, 0],
          color: '#1f7a8c',
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

    <div ref="chartEl" class="chart"></div>

    <ul class="city-list">
      <li v-for="item in items.slice(0, 8)" :key="item.adcode + item.city">
        <span>{{ item.province }}{{ item.city }}</span>
        <strong>{{ item.value }}</strong>
      </li>
    </ul>
  </section>
</template>
