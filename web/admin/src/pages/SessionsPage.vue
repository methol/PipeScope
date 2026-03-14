<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { fetchSessions, type SessionItem } from '../api/client'
import { formatBytes } from '../utils/format'

const windowText = ref('15m')
const ruleID = ref('')
const items = ref<SessionItem[]>([])
const error = ref('')
let timer: number | null = null

async function load() {
  try {
    error.value = ''
    items.value = await fetchSessions({
      window: windowText.value,
      rule_id: ruleID.value,
      limit: '100',
      offset: '0',
    })
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'unknown error'
  }
}

onMounted(async () => {
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
      <h2>会话明细</h2>
      <div class="filters">
        <label>
          窗口
          <select v-model="windowText" @change="load">
            <option value="5m">5m</option>
            <option value="15m">15m</option>
            <option value="1h">1h</option>
            <option value="1d">1d</option>
            <option value="1w">1w</option>
            <option value="1mo">1mo</option>
          </select>
        </label>
        <label>
          Rule
          <input v-model="ruleID" placeholder="可选" @change="load" />
        </label>
      </div>
    </div>

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
        </tr>
      </tbody>
    </table>
  </section>
</template>
