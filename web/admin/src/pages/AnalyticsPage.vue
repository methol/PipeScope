<script setup lang="ts">
import { ref } from 'vue'
import { fetchAnalytics, type AnalyticsBucket, type AnalyticsResult } from '../api/client'
import { formatBytes } from '../utils/format'

const windowText = ref('1d')
const ruleID = ref('')
const province = ref('')
const city = ref('')
const status = ref('')
const loading = ref(false)
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

function formatBucket(item: AnalyticsBucket): string {
  return `${item.name} - ${formatBytes(item.total_bytes)}`
}

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
      top_n: '10',
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
