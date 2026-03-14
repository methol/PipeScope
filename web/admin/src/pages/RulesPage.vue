<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { fetchRules, type RulePoint } from '../api/client'
import { formatBytes } from '../utils/format'

const windowText = ref('15m')
const items = ref<RulePoint[]>([])
const error = ref('')
let timer: number | null = null

async function load() {
  try {
    error.value = ''
    items.value = await fetchRules({ window: windowText.value })
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
      <h2>规则统计</h2>
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
    </div>

    <p v-if="error" class="error">{{ error }}</p>

    <table class="table">
      <thead>
        <tr>
          <th>Rule ID</th>
          <th>连接数</th>
          <th>总字节</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in items" :key="item.rule_id">
          <td>{{ item.rule_id }}</td>
          <td>{{ item.conn_count }}</td>
          <td>{{ formatBytes(item.total_bytes) }}</td>
        </tr>
      </tbody>
    </table>
  </section>
</template>
