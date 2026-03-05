<script setup lang="ts">
import { ref } from 'vue'
import MapPage from './pages/MapPage.vue'
import RulesPage from './pages/RulesPage.vue'
import SessionsPage from './pages/SessionsPage.vue'

type Tab = 'map' | 'rules' | 'sessions'
const tab = ref<Tab>('map')
</script>

<template>
  <div class="app-shell">
    <header class="topbar">
      <h1>PipeScope Admin</h1>
      <p>TCP 转发链路观测台</p>
    </header>

    <nav class="tabs">
      <button :class="{ active: tab === 'map' }" @click="tab = 'map'">地图</button>
      <button :class="{ active: tab === 'rules' }" @click="tab = 'rules'">规则</button>
      <button :class="{ active: tab === 'sessions' }" @click="tab = 'sessions'">明细</button>
    </nav>

    <main class="content">
      <MapPage v-if="tab === 'map'" />
      <RulesPage v-else-if="tab === 'rules'" />
      <SessionsPage v-else />
    </main>
  </div>
</template>
