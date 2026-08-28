<template>
  <el-dialog
    v-model="visible"
    :title="title"
    :width="dialogWidth"
    :fullscreen="isMobile"
    top="4vh"
    :close-on-click-modal="true"
    destroy-on-close
    class="kline-dialog"
    :class="{ mobile: isMobile }"
  >
    <div class="kline-toolbar">
      <div class="kline-info">
        <span class="sym">{{ stock?.symbol }}</span>
        <span class="name">{{ stock?.name }}</span>
        <el-tag size="small" :class="boardTagClass(stock)" class="board-tag">{{ boardLabel(stock) }}</el-tag>
      </div>
      <div class="kline-price">
        <span class="price" :class="stockClass(stock)">{{ stock && fmt(stock.close) }}</span>
        <span class="pct" :class="stockClass(stock)">
          {{ stock && ((stock.change_percent ?? 0) >= 0 ? '+' : '') + fmt(stock.change_percent) }}%
        </span>
      </div>
      <div class="kline-range">
        <el-radio-group v-model="days" size="small" @change="loadKLine">
          <el-radio-button :value="30">30日</el-radio-button>
          <el-radio-button :value="60">60日</el-radio-button>
          <el-radio-button :value="90">90日</el-radio-button>
          <el-radio-button :value="180">半年</el-radio-button>
        </el-radio-group>
      </div>
    </div>

    <div v-loading="loading" class="kline-chart" ref="chartEl"></div>

    <div class="kline-legend">
      <span><i class="dot up"></i> 阳线</span>
      <span><i class="dot down"></i> 阴线</span>
      <span class="ma-line ma5">MA5</span>
      <span class="ma-line ma10">MA10</span>
      <span class="ma-line ma20">MA20</span>
      <span class="avg-line">五日均价</span>
      <span class="bar-up">成交量</span>
      <span class="vol-avg">5日均量</span>
      <span class="macd-line dif">DIF</span>
      <span class="macd-line dea">DEA</span>
      <span class="macd-bar">MACD柱</span>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, watch, watchEffect, nextTick, onBeforeUnmount, onMounted, computed } from 'vue'
import * as echarts from 'echarts'
import { ElMessage } from 'element-plus'

const props = defineProps({
  modelValue: Boolean,
  stock: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue'])

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const days = ref(90)
const loading = ref(false)
const chartEl = ref(null)
let chart = null
let loadingToken = 0   // 单飞：每次新 load 自增，旧的 await 直接放弃

// 响应式宽度与字号
const windowWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1280)
const isMobile = computed(() => windowWidth.value <= 760)
const dialogWidth = computed(() => isMobile.value ? '100vw' : '900px')

function onResize() {
  windowWidth.value = window.innerWidth
  chart && chart.resize()
}
onMounted(() => window.addEventListener('resize', onResize))
onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  if (chart) { chart.dispose(); chart = null }
})

const title = computed(() => props.stock ? `${props.stock.symbol} ${props.stock.name} · K 线` : 'K 线')

function fmt(v, digits = 2) {
  if (v === null || v === undefined || Number.isNaN(v)) return '-'
  return Number(v).toFixed(digits)
}
function boardLabel(s) {
  if (!s) return ''
  if (s.market === '创业板') return '创业板'
  if (s.market === '科创板') return '科创板'
  if (s.market === '北交所') return '北交所'
  return '主板'
}
function boardTagClass(s) {
  if (!s) return ''
  if (s.market === '科创板') return 'tag-STAR'
  if (s.market === '创业板') return 'tag-CHINEXT'
  if (s.market === '北交所') return 'tag-BSE'
  return 'tag-MAIN'
}
function stockClass(s) {
  if (!s) return ''
  const c = s.change_percent
  if (c === undefined || c === null) return 'flat'
  if (c > 0) return 'up'
  if (c < 0) return 'down'
  return 'flat'
}

async function loadKLine() {
  if (!props.stock) return
  const myToken = ++loadingToken
  loading.value = true
  try {
    const url = `/api/v1/stock-daily-data/${props.stock.symbol}/kline?days=${days.value}`
    const r = await fetch(url)
    if (myToken !== loadingToken) return
    if (!r.ok) {
      ElMessage.error(`K 线接口 ${r.status}`)
      loading.value = false
      return
    }
    const data = await r.json()
    if (myToken !== loadingToken) return
    render(data.candles || [])
  } catch (e) {
    if (myToken !== loadingToken) return
    ElMessage.error(`加载 K 线失败：${e.message || e}`)
  } finally {
    if (myToken === loadingToken) loading.value = false
  }
}

function render(candles) {
  if (!chartEl.value) return
  if (!chart) {
    chart = echarts.init(chartEl.value, null, { renderer: 'canvas' })
  }
  if (!candles.length) {
    chart.clear()
    loading.value = false
    return
  }
  const dates = candles.map(c => c.date)
  const ohlc = candles.map(c => [c.open, c.close, c.low, c.high])
  const vols = candles.map((c, i) => [i, c.volume, c.close >= c.open ? 1 : -1])
  const ma5 = candles.map(c => c.ma5 || null)
  const ma10 = candles.map(c => c.ma10 || null)
  const ma20 = candles.map(c => c.ma20 || null)
  const avg5 = candles.map(c => c.avg5 || null)
  // 5 日均量（成交量子图叠加参考线）
  const volAvg5 = candles.map((c, i) => {
    if (i < 4) return null
    let s = 0
    for (let k = i - 4; k <= i; k++) s += (candles[k]?.volume || 0)
    return s / 5
  })
  // MACD
  const difArr = candles.map(c => c.dif || null)
  const deaArr = candles.map(c => c.dea || null)
  const macdArr = candles.map((c, i) => {
    const v = c.macd || 0
    // 用前一根 DEA 判正负柱
    const base = i > 0 ? (candles[i-1].dea || 0) : 0
    return [i, v, v >= 0 ? 1 : -1, base]
  })

  // 网格比例：主图 60%，成交量 18%，MACD 22%
  const h = isMobile.value ? '50%' : '58%'
  const vh = isMobile.value ? '18%' : '17%'
  const mh = isMobile.value ? '22%' : '20%'
  const top1 = '2%'
  const top2 = isMobile.value ? '56%' : '62%'
  const top3 = isMobile.value ? '76%' : '81%'

  chart.setOption({
    animation: false,
    backgroundColor: '#ffffff',
    legend: { show: false },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross' },
      backgroundColor: 'rgba(255,255,255,.96)',
      borderColor: '#e5e7eb',
      borderWidth: 1,
      textStyle: { color: '#111', fontSize: isMobile.value ? 11 : 12 },
      formatter: (params) => {
        const candle = params.find(p => p.seriesType === 'candlestick')
        if (!candle) return ''
        const idx = candle.dataIndex
        const c = candles[idx]
        const up = c.close >= c.open
        const color = up ? '#e63946' : '#16a34a'
        const pad = (s) => `<div style="display:flex;justify-content:space-between;gap:14px"><span>${s}</span></div>`
        const lines = [
          `<div style="font-weight:700;margin-bottom:4px">${c.date}</div>`,
          `<div>开 <b style="color:${color}">${fmt(c.open)}</b></div>`,
          `<div>收 <b style="color:${color}">${fmt(c.close)}</b></div>`,
          `<div>高 <b style="color:${color}">${fmt(c.high)}</b></div>`,
          `<div>低 <b style="color:${color}">${fmt(c.low)}</b></div>`,
          `<div>量 <b>${fmt(c.volume / 10000, 0)}万</b> 换手 <b>${fmt(c.turnover_rate, 2)}%</b></div>`,
          `<div>5日均量 <b>${fmt((volAvg5[idx] || 0) / 10000, 0)}万</b></div>`,
          `<div style="border-top:1px dashed #e5e7eb;margin:4px 0;padding-top:4px;color:#6b7280">MA5 ${fmt(c.ma5)} MA10 ${fmt(c.ma10)} MA20 ${fmt(c.ma20)}</div>`,
          `<div style="color:#6b7280">五日均价 ${fmt(c.avg5)}</div>`,
          `<div style="border-top:1px dashed #e5e7eb;margin:4px 0;padding-top:4px">DIF <b style="color:#3b82f6">${fmt(c.dif)}</b>  DEA <b style="color:#f59e0b">${fmt(c.dea)}</b>  MACD <b style="color:${(c.macd||0)>=0?'#e63946':'#16a34a'}">${fmt(c.macd)}</b></div>`,
          `<div style="color:#6b7280">K ${fmt(c.k)} D ${fmt(c.d)} J ${fmt(c.j)} RSI6 ${fmt(c.rsi6)}</div>`,
        ]
        return lines.join('')
      },
    },
    axisPointer: { link: [{ xAxisIndex: 'all' }] },
    grid: [
      { left: 56, right: 16, top: top1, height: h,
        label: { show: false } },
      { left: 56, right: 16, top: top2, height: vh },
      { left: 56, right: 16, top: top3, height: mh },
    ],
    graphic: [
      { type: 'text', right: 20, top: top2, style: { text: '成交量', fill: '#9ca3af', fontSize: 10 } },
      { type: 'text', right: 20, top: top3, style: { text: 'MACD',  fill: '#9ca3af', fontSize: 10 } },
    ],
    xAxis: [
      {
        type: 'category', data: dates,
        boundaryGap: false,
        axisLine:  { lineStyle: { color: '#e5e7eb' } },
        axisLabel: { color: '#6b7280', fontSize: 11 },
        splitLine: { show: false },
      },
      {
        type: 'category', gridIndex: 1, data: dates,
        boundaryGap: false,
        axisLine:  { lineStyle: { color: '#e5e7eb' } },
        axisTick:  { show: false },
        axisLabel: { show: false },
        splitLine: { show: false },
      },
      {
        type: 'category', gridIndex: 2, data: dates,
        boundaryGap: false,
        axisLine:  { lineStyle: { color: '#e5e7eb' } },
        axisTick:  { show: false },
        axisLabel: { show: true, color: '#6b7280', fontSize: 10 },
        splitLine: { show: false },
      },
    ],
    yAxis: [
      {
        scale: true,
        axisLine:  { show: false },
        axisLabel: { color: '#6b7280', fontSize: 11 },
        splitLine: { lineStyle: { color: '#f3f4f6' } },
        axisTick:  { show: false },
      },
      {
        gridIndex: 1,
        axisLine:  { show: false },
        axisLabel: { color: '#6b7280', fontSize: 10, formatter: v => v >= 10000 ? (v / 10000).toFixed(0) + '万' : v },
        splitLine: { show: false },
        axisTick:  { show: false },
      },
      {
        gridIndex: 2,
        axisLine:  { show: false },
        axisLabel: { color: '#6b7280', fontSize: 10 },
        splitLine: { show: false },
        axisTick:  { show: false },
      },
    ],
    dataZoom: [
      { type: 'inside', xAxisIndex: [0, 1, 2], start: 60, end: 100 },
      { type: 'slider', xAxisIndex: [0, 1, 2], height: 16, bottom: 4, start: 60, end: 100, borderColor: '#e5e7eb', fillerColor: 'rgba(230,57,70,.10)', handleStyle: { color: '#e63946' } },
    ],
    series: [
      {
        name: 'K线',
        type: 'candlestick',
        data: ohlc,
        itemStyle: {
          color: '#e63946',        // 阳线
          color0: '#16a34a',       // 阴线
          borderColor: '#e63946',
          borderColor0: '#16a34a',
        },
      },
      {
        name: 'MA5',  type: 'line', data: ma5,  smooth: true, symbol: 'none',
        lineStyle: { width: 1.2, color: '#f59e0b' },
      },
      {
        name: 'MA10', type: 'line', data: ma10, smooth: true, symbol: 'none',
        lineStyle: { width: 1.2, color: '#3b82f6' },
      },
      {
        name: 'MA20', type: 'line', data: ma20, smooth: true, symbol: 'none',
        lineStyle: { width: 1.2, color: '#a855f7' },
      },
      {
        name: '5日均价', type: 'line', data: avg5, smooth: true, symbol: 'none',
        lineStyle: { width: 1.4, color: '#dc2626', type: 'dashed' },
      },
      // 成交量子图
      {
        name: '成交量', type: 'bar', xAxisIndex: 1, yAxisIndex: 1, data: vols,
        itemStyle: { color: (p) => p.data[2] === 1 ? '#e63946' : '#16a34a' },
        barWidth: '60%',
      },
      {
        name: '5日均量', type: 'line', xAxisIndex: 1, yAxisIndex: 1, data: volAvg5,
        smooth: true, symbol: 'none',
        lineStyle: { width: 1.2, color: '#0891b2', type: 'dashed' },
      },
      // MACD子图：DIF、DEA 曲线 + MACD 柱
      {
        name: 'DIF', type: 'line', xAxisIndex: 2, yAxisIndex: 2, data: difArr,
        smooth: true, symbol: 'none',
        lineStyle: { width: 1.2, color: '#3b82f6' },
      },
      {
        name: 'DEA', type: 'line', xAxisIndex: 2, yAxisIndex: 2, data: deaArr,
        smooth: true, symbol: 'none',
        lineStyle: { width: 1.2, color: '#f59e0b' },
      },
      {
        name: 'MACD', type: 'bar', xAxisIndex: 2, yAxisIndex: 2, data: macdArr,
        itemStyle: { color: (p) => p.data[2] >= 0 ? '#e63946' : '#16a34a' },
        barWidth: '50%',
      },
    ],
  }, true)
}

// 合并：modelValue=true OR stock?.symbol 变化时触发 load；close 时清场
let prevSymbol = null
watchEffect(async (onCleanup) => {
  const isOpen = props.modelValue
  const sym = props.stock?.symbol
  if (!isOpen) {
    loadingToken++
    if (chart) { chart.dispose(); chart = null }  // destroy-on-close 时旧 chart 实例已无 DOM，必须 dispose
    prevSymbol = null
    return
  }
  if (!sym || sym === prevSymbol) return  // 同一只股票不重复加载
  prevSymbol = sym

  onCleanup(() => { loadingToken++ })
  await nextTick()
  if (!chartEl.value) {
    await new Promise(r => setTimeout(r, 50))
  }
  if (!chart && chartEl.value) {
    chart = echarts.init(chartEl.value, null, { renderer: 'canvas' })
  }
  loadKLine()
})
watch(() => isMobile.value, () => { if (visible.value) loadKLine() })
</script>

<style scoped>
.kline-toolbar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 4px 10px;
  flex-wrap: wrap;
  gap: 8px 14px;
}
.kline-info {
  display: flex; align-items: center; gap: 10px;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
}
.kline-info .sym { font-size: 18px; font-weight: 800; color: #111; }
.kline-info .name { font-size: 14px; font-weight: 700; color: #222; }
.kline-info .board-tag { font-weight: 700; }

.kline-price {
  display: flex; align-items: baseline; gap: 8px;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
}
.kline-price .price { font-size: 22px; font-weight: 900; }
.kline-price .pct   { font-size: 14px; font-weight: 800; }
.kline-info .up, .kline-price .up   { color: #e63946; }
.kline-info .down, .kline-price .down { color: #16a34a; }
.kline-info .flat, .kline-price .flat { color: #6b7280; }

.kline-range { display: flex; }

.kline-chart {
  width: 100%; height: 540px;
  background: #fff;
  border: 1px solid #f3f4f6; border-radius: 8px;
}

.kline-legend {
  display: flex; align-items: center; flex-wrap: wrap;
  gap: 8px 18px;
  padding: 10px 4px 0;
  font-size: 12px; color: #6b7280;
}
.kline-legend .dot {
  display: inline-block; width: 10px; height: 10px; border-radius: 2px;
  margin-right: 4px; vertical-align: -1px;
}
.kline-legend .dot.up   { background: #e63946; }
.kline-legend .dot.down { background: #16a34a; }
.kline-legend .ma-line::before, .kline-legend .avg-line::before {
  content: ''; display: inline-block; width: 14px; height: 2px;
  margin-right: 5px; vertical-align: 3px;
}
.kline-legend .ma5::before  { background: #f59e0b; }
.kline-legend .ma10::before { background: #3b82f6; }
.kline-legend .ma20::before { background: #a855f7; }
.kline-legend .avg-line::before { background: #dc2626; }
.kline-legend .bar-up::before {
  content: ''; display: inline-block; width: 10px; height: 8px;
  margin-right: 5px; vertical-align: 0;
  background: linear-gradient(to top, #e63946 50%, #16a34a 50%);
  border-radius: 1px;
}
.kline-legend .vol-avg::before {
  content: ''; display: inline-block; width: 14px; height: 0;
  border-top: 2px dashed #0891b2;
  margin-right: 5px; vertical-align: 3px;
}
.kline-legend .macd-line::before {
  content: ''; display: inline-block; width: 14px; height: 2px;
  margin-right: 5px; vertical-align: 3px;
}
.kline-legend .macd-line.dif::before { background: #3b82f6; }
.kline-legend .macd-line.dea::before { background: #f59e0b; }
.kline-legend .macd-bar::before {
  content: ''; display: inline-block; width: 10px; height: 8px;
  margin-right: 5px; vertical-align: 0;
  background: linear-gradient(to top, #e63946 50%, #16a34a 50%);
  border-radius: 1px;
}

/* ========= 移动端适配 ========= */
@media (max-width: 760px) {
  .kline-toolbar {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }
  .kline-info { flex-wrap: wrap; }
  .kline-info .sym { font-size: 16px; }
  .kline-info .name { font-size: 13px; }
  .kline-price { justify-content: flex-start; }
  .kline-price .price { font-size: 20px; }
  .kline-range { justify-content: center; }
  .kline-range :deep(.el-radio-button__inner) {
    padding: 4px 10px;
    font-size: 12px;
  }
  .kline-chart {
    height: 70vh;
    min-height: 460px;
  }
  .kline-legend {
    gap: 6px 12px;
    font-size: 11px;
    justify-content: center;
  }
}

@media (max-width: 380px) {
  .kline-chart {
    height: 65vh;
    min-height: 400px;
  }
  .kline-price .price { font-size: 18px; }
}
</style>
