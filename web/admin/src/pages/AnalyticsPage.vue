<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  fetchAnalytics,
  fetchAnalyticsOptions,
  type AnalyticsBucket,
  type AnalyticsCityOption,
  type AnalyticsOptions,
  type AnalyticsResult,
} from '../api/client'
import { formatBytes } from '../utils/format'

const windowText = ref('1d')
const ruleID = ref('')
const province = ref('')
const city = ref('')
const status = ref('')
const topN = ref('10')
const loading = ref(false)
const optionsLoading = ref(false)
const error = ref('')

const analytics = ref<AnalyticsResult>({
  overview: {
    conn_count: 0,
    total_bytes: 0,
    avg_duration_ms: 0,
    active_rules: 0,
    active_cities: 0,
  },
  top_cities: [],
  top_rules: [],
})

const options = ref<AnalyticsOptions>({
  rules: [],
  provinces: [],
  cities: [],
  statuses: [],
})

const filteredCities = computed<AnalyticsCityOption[]>(() => {
  if (!province.value) return options.value.cities
  return options.value.cities.filter((item) => item.province === province.value)
})

function formatBucket(item: AnalyticsBucket): string {
  return `${item.name} - ${formatBytes(item.total_bytes)}`
}

async function loadOptions() {
  optionsLoading.value = true
  try {
    options.value = await fetchAnalyticsOptions({
      window: windowText.value,
      rule_id: ruleID.value,
      province: province.value,
      city: city.value,
      status: status.value,
    })
  } finally {
    optionsLoading.value = false
  }
}

watch(province, () => {
  const available = new Set(filteredCities.value.map((item) => item.city))
  if (city.value && !available.has(city.value)) {
    city.value = ''
  }
})

watch(windowText, async () => {
  await loadOptions()
})

async function search() {
  try {
    loading.value = true
    error.value = ''
    analytics.value = await fetchAnalytics({
      window: windowText.value,
      rule_id: ruleID.value,
      province: province.value,
      city: city.value,
      status: status.value,
      top_n: topN.value,
    })
  } catch (e) {
    analytics.value = {
      overview: {
        conn_count: 0,
        total_bytes: 0,
        avg_duration_ms: 0,
        active_rules: 0,
        active_cities: 0,
      },
      top_cities: [],
      top_rules: [],
    }
    error.value = e instanceof Error ? e.message : 'unknown error'
  } finally {
    loading.value = false
  }
}

void loadOptions()
</script>

<template>
  <section class="panel">
    <div class="panel-header">
      <h2>统计/分析</h2>
      <div class="filters">
        <label>
          时间范围
          <select v-model="windowText">
            <option value="1h">1h</option>
            <option value="6h">6h</option>
            <option value="1d">1d</option>
            <option value="1w">1w</option>
            <option value="1mo">1mo</option>
          </select>
        </label>
        <label>
          Rule
          <select v-model="ruleID">
            <option value="">全部</option>
            <option v-for="item in options.rules" :key="item" :value="item">{{ item }}</option>
          </select>
        </label>
        <label>
          省
          <select v-model="province">
            <option value="">全部</option>
            <option v-for="item in options.provinces" :key="item" :value="item">{{ item }}</option>
          </select>
        </label>
        <label>
          市
          <select v-model="city">
            <option value="">全部</option>
            <option v-for="item in filteredCities" :key="`${item.province}-${item.city}`" :value="item.city">
              {{ item.city }}
            </option>
          </select>
        </label>
        <label>
          状态
          <select v-model="status">
            <option value="">全部</option>
            <option v-for="item in options.statuses" :key="item" :value="item">{{ item }}</option>
          </select>
        </label>
        <label>
          Top
          <select v-model="topN">
            <option value="10">10</option>
            <option value="50">50</option>
            <option value="100">100</option>
            <option value="1000">1000</option>
          </select>
        </label>
        <button class="btn" @click="search">检索</button>
      </div>
    </div>

    <p class="meta">分析型页面：不自动刷新（手动检索）</p>
    <p v-if="optionsLoading" class="meta">筛选项加载中...</p>
    <p v-if="loading" class="meta">加载中...</p>
    <p v-if="error" class="error">{{ error }}</p>

    <div class="analytics-grid">
      <article class="analytics-card">
        <h3>总览</h3>
        <p>连接数：{{ analytics.overview.conn_count }}</p>
        <p>总字节：{{ formatBytes(analytics.overview.total_bytes) }}</p>
        <p>平均时长：{{ analytics.overview.avg_duration_ms }} ms</p>
        <p>活跃规则：{{ analytics.overview.active_rules }}</p>
        <p>活跃城市：{{ analytics.overview.active_cities }}</p>
      </article>

      <article class="analytics-card">
        <h3>Top 城市（按字节）</h3>
        <ol>
          <li v-for="item in analytics.top_cities" :key="item.name">{{ formatBucket(item) }}</li>
        </ol>
      </article>

      <article class="analytics-card">
        <h3>Top 规则（按字节）</h3>
        <ol>
          <li v-for="item in analytics.top_rules" :key="item.name">{{ formatBucket(item) }}</li>
        </ol>
      </article>
    </div>
  </section>
</template>
