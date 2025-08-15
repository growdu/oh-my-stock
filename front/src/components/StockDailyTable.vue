<template>
  <div class="p-4">
    <!-- Search + Actions -->
    <el-card shadow="never" class="mb-4">
      <div class="flex flex-wrap items-center gap-3">
        <el-input
          v-model.trim="querySymbol"
          placeholder="输入股票代码，例如 300687"
          style="max-width: 220px"
          @keyup.enter.native="fetchStock"
          clearable
        />
        <el-button type="primary" :loading="loading" @click="fetchStock">查询</el-button>
        <el-select v-model="range" placeholder="区间" style="width: 140px">
          <el-option :value="60" label="近60日" />
          <el-option :value="120" label="近120日" />
          <el-option :value="240" label="近240日" />
          <el-option :value="0" label="全部" />
        </el-select>
        <el-switch v-model="showMoneyFlow" active-text="显示资金流" />
      </div>
    </el-card>

    <!-- Basic Info -->
    <el-card v-if="meta" shadow="never" class="mb-4">
      <div class="flex flex-wrap items-center gap-4 text-sm">
        <div><span class="opacity-70">名称：</span>{{ meta.name }}</div>
        <div><span class="opacity-70">代码：</span>{{ meta.symbol }}</div>
        <div><span class="opacity-70">市场：</span>{{ meta.market }}</div>
        <div><span class="opacity-70">行业：</span>{{ meta.industry }}</div>
        <div><span class="opacity-70">上市：</span>{{ formatDate(meta.listing_date) }}</div>
      </div>
    </el-card>

    <!-- Charts -->
    <el-card shadow="never">
      <div class="grid gap-6">
        <!-- K-line + Volume -->
        <div ref="klineRef" style="width: 100%; height: 460px" />
        <!-- Money Flow -->
        <div v-if="showMoneyFlow" ref="moneyRef" style="width: 100%; height: 320px" />
      </div>
    </el-card>

    <!-- Empty State -->
    <el-empty v-if="!loading && (!rows || rows.length === 0)" description="暂无数据" class="mt-8" />
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import * as echarts from 'echarts'
import axios from 'axios'

// ===== Config =====
// 修改为你的后端地址（可用 .env.* 配置 VITE_API_BASE）
const API_BASE = import.meta.env.VITE_API_BASE || 'http://192.168.3.99:3003'

// ===== State =====
const querySymbol = ref('300687')
const loading = ref(false)
const range = ref(120) // 默认近 120 日
const showMoneyFlow = ref(true)

const meta = ref(null) // { name, symbol, market, industry, listing_date }
const rows = ref([])   // 原始 daily_data

// ECharts refs
const klineRef = ref(null)
const moneyRef = ref(null)
let kChart = null
let mChart = null

// ===== Helpers =====
function formatDate(iso) {
  if (!iso) return '-'
  const d = new Date(iso)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${dd}`
}

function toFixed(n, k = 2) {
  if (n === null || n === undefined || Number.isNaN(Number(n))) return null
  return Number(n).toFixed(k)
}

function ma(arr, day) {
  const out = new Array(arr.length).fill(null)
  let sum = 0
  for (let i = 0; i < arr.length; i++) {
    sum += arr[i]
    if (i >= day) sum -= arr[i - day]
    if (i >= day - 1) out[i] = Number((sum / day).toFixed(3))
  }
  return out
}

// 处理数据：排序、裁剪、计算 MA
const processed = computed(() => {
  if (!rows.value || rows.value.length === 0) return null
  // 按日期升序
  const sorted = [...rows.value].sort((a, b) => new Date(a.trade_date) - new Date(b.trade_date))
  const clipped = range.value && range.value > 0 ? sorted.slice(-range.value) : sorted

  const dates = clipped.map(r => formatDate(r.trade_date))
  const opens = clipped.map(r => Number(r.open))
  const closes = clipped.map(r => Number(r.close))
  const lows = clipped.map(r => Number(r.low))
  const highs = clipped.map(r => Number(r.high))
  const vols = clipped.map(r => Number(r.volume || 0))
  const changePct = clipped.map(r => r.change_percent)

  const candlesticks = clipped.map(r => [Number(r.open), Number(r.close), Number(r.low), Number(r.high)])

  const ma5 = ma(closes, 5)
  const ma10 = ma(closes, 10)
  const ma20 = ma(closes, 20)
  const ma60 = ma(closes, 60)

  const inAmt = clipped.map(r => (r.in_amount == null ? null : Number(r.in_amount)))
  const outAmt = clipped.map(r => (r.out_amount == null ? null : Number(r.out_amount)))
  const netAmt = clipped.map((_, i) => (inAmt[i] == null || outAmt[i] == null) ? null : Number((inAmt[i] - outAmt[i]).toFixed(0)))

  return { dates, opens, closes, lows, highs, vols, candlesticks, ma5, ma10, ma20, ma60, changePct, inAmt, outAmt, netAmt }
})

// ===== Charts =====
function renderKline() {
  if (!klineRef.value || !processed.value) return
  if (!kChart) kChart = echarts.init(klineRef.value)
  const p = processed.value

  const option = {
    animation: false,
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross' },
      valueFormatter: (v) => (v == null ? '-' : String(v))
    },
    axisPointer: { link: [{ xAxisIndex: 'all' }] },
    grid: [
      { left: 40, right: 10, top: 20, height: 260 },
      { left: 40, right: 10, top: 300, height: 100 }
    ],
    xAxis: [
      { type: 'category', data: p.dates, boundaryGap: false, axisLine: { onZero: false }, splitLine: { show: false }, min: 'dataMin', max: 'dataMax' },
      { type: 'category', data: p.dates, gridIndex: 1, boundaryGap: false, axisLine: { onZero: false }, splitLine: { show: false }, axisLabel: { show: false }, min: 'dataMin', max: 'dataMax' }
    ],
    yAxis: [
      { scale: true, splitArea: { show: true } },
      { scale: true, gridIndex: 1, splitNumber: 2 }
    ],
    dataZoom: [
      { type: 'inside', xAxisIndex: [0, 1], start: 20, end: 100 },
      { type: 'slider', xAxisIndex: [0, 1], top: 420 }
    ],
    series: [
      {
        name: '日K',
        type: 'candlestick',
        xAxisIndex: 0, yAxisIndex: 0,
        data: p.candlesticks,
        itemStyle: { color: '#ef5350', color0: '#26a69a', borderColor: '#ef5350', borderColor0: '#26a69a' }
      },
      { name: 'MA5', type: 'line', data: p.ma5, showSymbol: false, smooth: true },
      { name: 'MA10', type: 'line', data: p.ma10, showSymbol: false, smooth: true },
      { name: 'MA20', type: 'line', data: p.ma20, showSymbol: false, smooth: true },
      { name: 'MA60', type: 'line', data: p.ma60, showSymbol: false, smooth: true },
      {
        name: '成交量', type: 'bar', xAxisIndex: 1, yAxisIndex: 1, data: p.vols,
        itemStyle: {
          color: (params) => {
            const i = params.dataIndex
            return p.closes[i] >= p.opens[i] ? '#ef5350' : '#26a69a'
          }
        }
      }
    ]
  }
  kChart.setOption(option)
}

function renderMoneyFlow() {
  if (!moneyRef.value || !processed.value) return
  if (!mChart) mChart = echarts.init(moneyRef.value)
  const p = processed.value

  const option = {
    animation: false,
    tooltip: { trigger: 'axis', valueFormatter: v => (v == null ? '-' : String(v)) },
    legend: { data: ['主力流入', '主力流出', '净额'] },
    grid: { left: 40, right: 10, top: 30, bottom: 20 },
    xAxis: { type: 'category', data: p.dates, boundaryGap: true },
    yAxis: { type: 'value', name: '金额(元)' },
    dataZoom: [ { type: 'inside', start: 20, end: 100 }, { type: 'slider' } ],
    series: [
      { name: '主力流入', type: 'bar', stack: 'flow', data: p.inAmt },
      { name: '主力流出', type: 'bar', stack: 'flow', data: p.outAmt },
      { name: '净额', type: 'line', showSymbol: false, data: p.netAmt, smooth: true }
    ]
  }
  mChart.setOption(option)
}

function resizeAll() {
  kChart && kChart.resize()
  mChart && mChart.resize()
}

watch(processed, async () => {
  await nextTick()
  renderKline()
  if (showMoneyFlow.value) renderMoneyFlow()
})

watch(showMoneyFlow, async () => {
  await nextTick()
  if (showMoneyFlow.value) renderMoneyFlow()
  resizeAll()
})

onMounted(async () => {
  await fetchStock()
  window.addEventListener('resize', resizeAll)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', resizeAll)
  kChart && kChart.dispose()
  mChart && mChart.dispose()
})

// ===== API =====
async function fetchStock() {
  if (!querySymbol.value) return
  loading.value = true
  try {
    const url = `${API_BASE}/api/v1/stocks/history?symbol=${encodeURIComponent(querySymbol.value)}`
    const { data } = await axios.get(url)
    meta.value = {
      name: data.name,
      symbol: data.symbol,
      market: data.market,
      industry: data.industry,
      listing_date: data.listing_date
    }
    rows.value = Array.isArray(data.daily_data) ? data.daily_data : []
  } catch (e) {
    console.error(e)
    meta.value = null
    rows.value = []
    ElMessage.error('获取数据失败，请检查后端接口或网络')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.mb-4 { margin-bottom: 1rem; }
.grid { display: grid; }
.flex { display: flex; }
.flex-wrap { flex-wrap: wrap; }
.items-center { align-items: center; }
.gap-3 { gap: .75rem; }
.gap-4 { gap: 1rem; }
.gap-6 { gap: 1.5rem; }
.p-4 { padding: 1rem; }
.opacity-70 { opacity: .7; }
.text-sm { font-size: .9rem; }
.mt-8 { margin-top: 2rem; }
</style>
