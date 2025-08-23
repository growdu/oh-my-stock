<template>
  <div class="p-4">
    <!-- Search + Actions -->
    <el-card shadow="never" class="mb-4">
      <div class="flex flex-wrap items-center gap-3">
        <!-- ✅ 改为自动补全 -->
        <el-autocomplete
          v-model.trim="querySymbol"
          :fetch-suggestions="querySearch"
          placeholder="输入股票代码或名称"
          style="max-width: 240px"
          clearable
          @select="handleSelect"
          @keyup.enter.native="fetchStock"
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
        <div ref="klineRef" style="width: 100%; height: 480px" />
        <div v-if="showMoneyFlow" ref="moneyRef" style="width: 100%; height: 320px" />
      </div>
    </el-card>

    <el-empty v-if="!loading && (!rows || rows.length === 0)" description="暂无数据" class="mt-8" />
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import * as echarts from 'echarts'
import axios from 'axios'
import debounce from 'lodash/debounce'   // ✅ 引入lodash.debounce
import { ElMessage } from 'element-plus'
import { fetchDailyData, searchStocks } from '@/utils/api/daily.js'


const querySymbol = ref('000001')
const loading = ref(false)
const range = ref(120)
const showMoneyFlow = ref(true)

const meta = ref(null)
const rows = ref([])

const klineRef = ref(null)
const moneyRef = ref(null)
let kChart = null
let mChart = null

/* ===========================
   工具函数
=========================== */
function formatDate(iso) {
  if (!iso) return '-'
  const d = new Date(iso)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function toFixed(n, k = 2) {
  if (n == null || Number.isNaN(Number(n))) return null
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

const processed = computed(() => {
  if (!rows.value || rows.value.length === 0) return null
  const sorted = [...rows.value].sort((a, b) => new Date(a.trade_date) - new Date(b.trade_date))
  const clipped = range.value && range.value > 0 ? sorted.slice(-range.value) : sorted

  const dates = clipped.map(r => formatDate(r.trade_date))
  const opens = clipped.map(r => Number(r.open))
  const closes = clipped.map(r => Number(r.close))
  const lows = clipped.map(r => Number(r.low))
  const highs = clipped.map(r => Number(r.high))
  const vols = clipped.map(r => Number(r.volume || 0))
  const candlesticks = clipped.map(r => [Number(r.open), Number(r.close), Number(r.low), Number(r.high)])

  const ma3 = ma(closes, 3)
  const ma5 = ma(closes, 5)
  const ma7 = ma(closes, 7)
  const ma10 = ma(closes, 10)

  // ✅ 改成 "万"
  const inAmt = clipped.map(r => (r.in_amount == null ? null : Math.round(Number(r.in_amount) / 1e4)))
  const outAmt = clipped.map(r => (r.out_amount == null ? null : Math.round(Number(r.out_amount) / 1e4)))
  const netAmt = clipped.map((_, i) => (inAmt[i] == null || outAmt[i] == null) ? null : inAmt[i] - outAmt[i])

  return { dates, opens, closes, lows, highs, vols, candlesticks, ma3, ma5, ma7, ma10, inAmt, outAmt, netAmt }
})


function renderKline() {
  if (!klineRef.value || !processed.value) return
  if (!kChart) kChart = echarts.init(klineRef.value)
  const p = processed.value

  const option = {
    animation: false,
    tooltip: { trigger: 'axis', axisPointer: { type: 'cross' } },
    axisPointer: { link: [{ xAxisIndex: 'all' }] },
    grid: [
      { left: 60, right: 10, top: 20, height: 260 },   // ← grid.left 加宽，保证坐标轴文字有空间
      { left: 60, right: 10, top: 330, height: 100 }
    ],
    xAxis: [
      {
        type: 'category',
        data: p.dates,
        boundaryGap: true,
        min: -0.5,                                // ← 往左留半个柱子
        max: p.dates.length - 0.5                 // ← 往右留半个柱子
      },
      {
        type: 'category',
        data: p.dates,
        gridIndex: 1,
        boundaryGap: true,
        axisLabel: { show: false },
        min: -0.5,
        max: p.dates.length - 0.5
      }
    ],
    yAxis: [
      { scale: true, splitArea: { show: true } },
      { scale: true, gridIndex: 1, splitNumber: 2 }
    ],
    dataZoom: [
      { type: 'inside', xAxisIndex: [0, 1], start: 20, end: 100 },
      { type: 'slider', xAxisIndex: [0, 1], top: 440 }
    ],
    series: [
      { name: '日K', type: 'candlestick', data: p.candlesticks },
      { name: 'MA5', type: 'line', data: p.ma3, showSymbol: false, smooth: true },
      { name: 'MA10', type: 'line', data: p.ma5, showSymbol: false, smooth: true },
      { name: 'MA20', type: 'line', data: p.ma7, showSymbol: false, smooth: true },
      { name: 'MA60', type: 'line', data: p.ma10, showSymbol: false, smooth: true },
      { 
        name: '成交量', 
        type: 'bar', 
        xAxisIndex: 1, 
        yAxisIndex: 1, 
        data: p.vols, 
        barCategoryGap: '30%'   // ← 让柱子和边缘、坐标轴有间隔
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
    tooltip: { 
      trigger: 'axis', 
      valueFormatter: v => (v == null ? '-' : v + '万')   // ✅ 单位改为“万”
    },
    legend: { data: ['主力流入', '主力流出', '净额'] },
    grid: { left: 60, right: 10, top: 30, bottom: 20 },
    xAxis: { 
      type: 'category', 
      data: p.dates, 
      boundaryGap: true, 
      min: -0.5, 
      max: p.dates.length - 0.5 
    },
    yAxis: { type: 'value', name: '金额(万)' },   // ✅ 单位改为万
    dataZoom: [ { type: 'inside', start: 20, end: 100 }, { type: 'slider' } ],
    series: [
      { name: '主力流入', type: 'bar', stack: 'flow', data: p.inAmt, barCategoryGap: '30%' },
      { name: '主力流出', type: 'bar', stack: 'flow', data: p.outAmt, barCategoryGap: '30%' },
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

/* ===========================
   API 请求
=========================== */

// ===========================
// API 请求
// ===========================
async function fetchStock() {
  if (!querySymbol.value) return
  loading.value = true
  try {
    const { data } = await fetchDailyData(querySymbol.value)
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

// ===========================
// 自动补全 + 防抖
// ===========================
const doSearch = async (queryString, cb) => {
  if (!queryString) {
    cb([])
    return
  }
  try {
    const { data } = await searchStocks(queryString)
    cb(
      data.map(item => ({
        value: `${item.symbol} ${item.name}`,
        symbol: item.symbol,
        name: item.name
      }))
    )
  } catch (e) {
    console.error(e)
    cb([])
  }
}

const querySearch = debounce(doSearch, 300)
const handleSelect = (item) => {
  querySymbol.value = item.symbol
  fetchStock()
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
