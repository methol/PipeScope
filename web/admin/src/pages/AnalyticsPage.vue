<script setup lang="ts">
import { computed, ref } from 'vue'
import { fetchSessions, type SessionItem } from '../api/client'
import { formatBytes } from '../utils/format'

const windowText = ref('1d')
const ruleID = ref('')
const province = ref('')
const city = ref('')
const status = ref('')
const loading = ref(false)
const error = ref('')
const items = ref<SessionItem[]>([])

const filtered = computed(() => {
  return items.value.filter((it) => {
    if (province.value && !String(it.province || '').includes(province.value)) return false
    if (city.value && !String(it.city || '').includes(city.value)) return false
    if (status.value && String(it.status || '') !== status.value) return false
    return true
  })
})

const summary = computed(() => {
  let totalBytes = 0
  let totalDuration = 0
  const rules = new Set<string>()
  const cities = new Set<string>()
  for (const it of filtered.value) {
    totalBytes += Number(it.total_bytes) || 0
    totalDuration += Number(it.duration_ms) || 0
    if (it.rule_id) rules.add(it.rule_id)
    if (it.city) cities.add(`${it.province}-${it.city}`)
  }
  const conn = filtered.value.length
  return {
    conn,
    totalBytes,
    avgDuration: conn > 0 ? Math.round(totalDuration / conn) : 0,
    rules: rules.size,
    cities: cities.size,
  }
})

const topCities = computed(() => {
  const m = new Map<string, number>()
  for (const it of filtered.value) {
    const k = `${it.province}${it.city}`
    m.set(k, (m.get(k) || 0) + (Number(it.total_bytes) || 0))
  }
  return [...m.entries()].sort((a, b) => b[1] - a[1]).slice(0, 10)
})

const topRules = computed(() => {
  const m = new Map<string, number>()
  for (const it of filtered.value) {
    const k = it.rule_id || 'unknown'
    m.set(k, (m.get(k) || 0) + (Number(it.total_bytes) || 0))
  }
  return [...m.entries()].sort((a, b) => b[1] - a[1]).slice(0, 10)
})

async function search() {
  try {
    loading.value = true
    error.value = ''
    const all: SessionItem[] = []
    const pageSize = 500
    const maxPages = 200
    for (let pageNo = 0; pageNo < maxPages; pageNo++) {
      const page = await fetchSessions({
        window: windowText.value,
        rule_id: ruleID.value,
        limit: String(pageSize),
        offset: String(pageNo * pageSize),
      })
      all.push(...page)
      if (page.length < pageSize) break
      if (pageNo === maxPages - 1) {
        error.value = `数据量过大，已截断前 ${maxPages * pageSize} 条，请缩小检索范围`
      }
    }
    items.value = all
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'unknown error'
  } finally {
    loading.value = false
  }
}
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
          <input v-model="ruleID" placeholder="可选" />
        </label>
        <label>
          省
          <input v-model="province" placeholder="可选" />
        </label>
        <label>
          市
          <input v-model="city" placeholder="可选" />
        </label>
        <label>
          状态
          <input v-model="status" placeholder="可选" />
        </label>
        <button class="btn" @click="search">检索</button>
      </div>
    </div>

    <p class="meta">分析型页面：不自动刷新（手动检索）</p>
    <p v-if="loading" class="meta">加载中...</p>
    <p v-if="error" class="error">{{ error }}</p>

    <div class="analytics-grid">
      <article class="analytics-card">
        <h3>总览</h3>
        <p>连接数：{{ summary.conn }}</p>
        <p>总字节：{{ formatBytes(summary.totalBytes) }}</p>
        <p>平均时长：{{ summary.avgDuration }} ms</p>
        <p>活跃规则：{{ summary.rules }}</p>
        <p>活跃城市：{{ summary.cities }}</p>
      </article>

      <article class="analytics-card">
        <h3>Top 城市（按字节）</h3>
        <ol>
          <li v-for="item in topCities" :key="item[0]">{{ item[0] }} - {{ formatBytes(item[1]) }}</li>
        </ol>
      </article>

      <article class="analytics-card">
        <h3>Top 规则（按字节）</h3>
        <ol>
          <li v-for="item in topRules" :key="item[0]">{{ item[0] }} - {{ formatBytes(item[1]) }}</li>
        </ol>
      </article>
    </div>
  </section>
</template>
