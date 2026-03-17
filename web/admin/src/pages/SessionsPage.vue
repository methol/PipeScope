<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { fetchSessions, fetchSessionsOptions, type SessionItem, type SessionsOptions } from '../api/client'
import { formatBytes } from '../utils/format'

const ruleID = ref('')
const limit = ref('100')
const items = ref<SessionItem[]>([])
const options = ref<SessionsOptions>({ rules: [] })
const error = ref('')
const loading = ref(false)
const optionsLoading = ref(false)
let timer: number | null = null

async function loadOptions() {
  optionsLoading.value = true
  try {
    options.value = await fetchSessionsOptions({ window: '15m' })
  } finally {
    optionsLoading.value = false
  }
}

async function load() {
  try {
    loading.value = true
    error.value = ''
    items.value = await fetchSessions({
      window: '5m',
      rule_id: ruleID.value,
      limit: limit.value,
      offset: '0',
    })
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'unknown error'
  } finally {
    loading.value = false
  }
}

watch([ruleID, limit], () => {
  void load()
})

onMounted(async () => {
  await loadOptions()
  await load()
  timer = window.setInterval(() => {
    void load()
  }, 5000)
})

onUnmounted(() => {
  if (timer !== null) window.clearInterval(timer)
})
</script>

<template>
  <section class="panel">
    <div class="panel-header">
      <h2>实时会话</h2>
      <div class="filters">
        <label>
          Rule
          <select v-model="ruleID" :disabled="optionsLoading">
            <option value="">全部</option>
            <option v-for="item in options.rules" :key="item" :value="item">{{ item }}</option>
          </select>
        </label>
        <label>
          Limit
          <select v-model="limit">
            <option value="100">100</option>
            <option value="1000">1000</option>
            <option value="10000">10000</option>
          </select>
        </label>
      </div>
    </div>

    <p class="meta">固定窗口：5m · 每 5 秒自动刷新</p>
    <p v-if="optionsLoading" class="meta">选项加载中...</p>
    <p v-if="loading" class="meta">加载中...</p>
    <p v-if="error" class="error">{{ error }}</p>

    <table class="table">
      <thead>
        <tr>
          <th>时间</th>
          <th>Rule</th>
          <th>源</th>
          <th>目标</th>
          <th>状态</th>
          <th>总字节</th>
          <th>地域</th>
          <th>拦截原因</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in items" :key="item.id">
          <td>{{ new Date(item.start_ts).toLocaleTimeString() }}</td>
          <td>{{ item.rule_id }}</td>
          <td>{{ item.src_addr }}</td>
          <td>{{ item.dst_addr }}</td>
          <td>{{ item.status }}</td>
          <td>{{ formatBytes(item.total_bytes) }}</td>
          <td>{{ item.province }}{{ item.city }}</td>
          <td>{{ item.blocked_reason || '-' }}</td>
        </tr>
      </tbody>
    </table>
  </section>
</template>
