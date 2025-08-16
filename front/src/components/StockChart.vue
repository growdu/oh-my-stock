<!-- src/components/StockChart.vue -->
<template>
    <div>
      <!-- 股票信息 -->
      <el-card v-if="meta" shadow="never" class="mb-4">
        <div class="flex flex-wrap items-center gap-4 text-sm">
          <div><span class="opacity-70">名称：</span>{{ meta.name }}</div>
          <div><span class="opacity-70">代码：</span>{{ meta.symbol }}</div>
          <div><span class="opacity-70">市场：</span>{{ meta.market }}</div>
          <div><span class="opacity-70">行业：</span>{{ meta.industry }}</div>
          <div><span class="opacity-70">上市：</span>{{ formatDate(meta.listing_date) }}</div>
        </div>
      </el-card>
  
      <!-- K线图 & 资金流 -->
      <el-card shadow="never">
        <div class="grid gap-6">
          <div ref="klineRef" style="width: 100%; height: 480px" />
          <div v-if="showMoneyFlow" ref="moneyRef" style="width: 100%; height: 320px" />
        </div>
      </el-card>
    </div>
  </template>
  
  <script setup>
  import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
  import * as echarts from 'echarts'
  import axios from 'axios'
  
  const props = defineProps({
    symbol: { type: String, required: true }
  })
  
  const API_BASE = import.meta.env.VITE_API_BASE || 'http://192.168.3.99:3003'
  
  const meta = ref(null)
  const rows = ref([])
  const showMoneyFlow = ref(true)
  const range = ref(120)
  const loading = ref(false)
  
  const klineRef = ref(null)
  const moneyRef = ref(null)
  let kChart = null
  let mChart = null
  
  // 你的工具函数（formatDate, ma, processed 等）照搬即可...
  
  async function fetchStock() {
    if (!props.symbol) return
    loading.value = true
    try {
      const url = `${API_BASE}/api/v1/stocks/history?symbol=${encodeURIComponent(props.symbol)}`
      const { data } = await axios.get(url)
      meta.value = data
      rows.value = Array.isArray(data.daily_data) ? data.daily_data : []
    } catch (e) {
      console.error(e)
    } finally {
      loading.value = false
    }
  }
  
  watch(rows, async () => {
    await nextTick()
    renderKline()
    if (showMoneyFlow.value) renderMoneyFlow()
  })
  
  onMounted(fetchStock)
  onBeforeUnmount(() => {
    kChart && kChart.dispose()
    mChart && mChart.dispose()
  })
  </script>
  