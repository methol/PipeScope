<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
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
const srcIP = ref('')
const topN = ref('10')
const loading = ref(false)
const optionsLoading = ref(false)
const error = ref('')
const citySelectionProvince = ref('')

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
const optionsSrcIP = computed(() => resolveOptionsSrcIP(srcIP.value))

let optionsRequestID = 0
let suppressedOptionsReloads = 0

function formatBucket(item: AnalyticsBucket): string {
  return `${item.name} - ${formatBytes(item.total_bytes)}`
}

function pickValidSelection(current: string, available: string[]): string {
  return current && !available.includes(current) ? '' : current
}

function isCompleteIPv4(value: string): boolean {
  const octets = value.split('.')
  return octets.length === 4 && octets.every((part) => /^\d{1,3}$/.test(part) && Number(part) <= 255)
}

function isCompleteIPv6(value: string): boolean {
  if (!value.includes(':') || !/^[0-9a-f:.]+$/i.test(value)) {
    return false
  }

  const compressed = value.split('::')
  if (compressed.length > 2) {
    return false
  }

  const parseGroups = (segment: string): string[] => (segment ? segment.split(':') : [])
  const left = parseGroups(compressed[0])
  const right = compressed.length === 2 ? parseGroups(compressed[1]) : []
  const hasValidGroups = [...left, ...right].every((group) => /^[0-9a-f]{1,4}$/i.test(group))
  if (!hasValidGroups) {
    return false
  }

  if (compressed.length === 1) {
    return left.length === 8
  }
  return left.length + right.length < 8
}

function resolveOptionsSrcIP(value: string): string | null {
  if (!value) return ''
  return isCompleteIPv4(value) || isCompleteIPv6(value) ? value : null
}

function citySelectionScope(nextProvince = province.value): string {
  return nextProvince || citySelectionProvince.value
}

function pickValidCitySelection(current: string, available: AnalyticsCityOption[], provinceScope: string): string {
  if (!current) return ''
  return available.some((item) => item.city === current && (!provinceScope || item.province === provinceScope)) ? current : ''
}

async function reconcileSelectedFilters(next: AnalyticsOptions): Promise<boolean> {
  const nextRuleID = pickValidSelection(ruleID.value, next.rules)
  const nextProvince = pickValidSelection(province.value, next.provinces)
  const nextStatus = pickValidSelection(status.value, next.statuses)
  const nextCity = pickValidCitySelection(city.value, next.cities, citySelectionScope(nextProvince))

  const changed =
    nextRuleID !== ruleID.value ||
    nextProvince !== province.value ||
    nextStatus !== status.value ||
    nextCity !== city.value
  if (!changed) return false

  suppressedOptionsReloads += 1
  try {
    ruleID.value = nextRuleID
    province.value = nextProvince
    city.value = nextCity
    status.value = nextStatus
    await nextTick()
  } finally {
    suppressedOptionsReloads -= 1
  }
  return true
}

async function loadOptions() {
  const requestID = ++optionsRequestID
  optionsLoading.value = true
  try {
    // Selected filters can only collapse toward empty, so a few passes reach a stable query quickly.
    for (let pass = 0; pass < 5; pass += 1) {
      const next = await fetchAnalyticsOptions({
        window: windowText.value,
        rule_id: ruleID.value,
        province: province.value,
        city: city.value,
        status: status.value,
        src_ip: optionsSrcIP.value ?? undefined,
      })
      if (requestID !== optionsRequestID) return
      options.value = next
      if (!(await reconcileSelectedFilters(next))) {
        return
      }
    }
  } finally {
    if (requestID === optionsRequestID) {
      optionsLoading.value = false
    }
  }
}

watch(city, (next, prev) => {
  if (!next) {
    citySelectionProvince.value = ''
    return
  }
  if (next !== prev) {
    citySelectionProvince.value = province.value
  }
})

watch(province, (next) => {
  if (next && city.value) {
    citySelectionProvince.value = next
  }
})

watch([province, filteredCities], () => {
  if (city.value && !pickValidCitySelection(city.value, filteredCities.value, citySelectionScope())) {
    city.value = ''
    citySelectionProvince.value = ''
  }
})

watch([windowText, ruleID, province, city, status, optionsSrcIP], () => {
  if (suppressedOptionsReloads > 0) return
  void loadOptions()
}, { immediate: true })

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
      src_ip: srcIP.value,
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
</script>

<template>
  <section class="panel">
    <div class="panel-header">
      <h2>统计</h2>
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
          源 IP
          <input v-model.trim="srcIP" type="text" placeholder="例如 10.0.0.8" />
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
