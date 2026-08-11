<template>
  <div class="page">
    <!-- 8 个系统规则 grid -->
    <div class="rules-section">
      <div class="carousel">
        <button class="carousel-arrow left" @click="prev" aria-label="上一条">‹</button>
        <div class="carousel-track">
          <div
            v-for="(p, i) in presets"
            :key="p.id"
            class="carousel-card"
            :class="{ active: i === selectedIndex }"
            @click="selectByIndex(i)"
          >
            <div class="pcard-head">
              <el-tag size="small" effect="plain" class="board-tag">
                {{ boardSummary(p) }}
              </el-tag>
            </div>
            <div class="pcard-name">{{ p.name }}</div>
            <div class="pcard-desc">{{ p.description }}</div>
            <div class="pcard-foot">
              <span class="pcard-id">{{ p.id }}</span>
              <span class="pcard-hint">{{ cache[p.id] ? '✓' : '…' }}</span>
            </div>
          </div>
        </div>
        <button class="carousel-arrow right" @click="next" aria-label="下一条">›</button>
      </div>
      <div class="carousel-dots">
        <span
          v-for="(p, i) in presets"
          :key="p.id"
          class="dot"
          :class="{ active: i === selectedIndex, cached: cache[p.id] }"
          :title="p.name"
          @click="selectByIndex(i)"
        />
      </div>
    </div>

    <!-- 命中股票 -->
    <div v-if="selected" class="stocks-section">
      <div class="stocks-header">
        <div class="stocks-title">
          <span class="stock-red-bar"></span>
          「{{ selected.name }}」命中结果
          <span class="stocks-meta">共 {{ total }} 只<span v-if="tradeDate"> · {{ tradeDate }}</span></span>
        </div>
        <div class="stocks-actions">
          <el-button size="small" plain @click="exportCSV">
            <span style="margin-right: 4px">⬇</span>导出 CSV
          </el-button>
        </div>
      </div>

      <el-skeleton v-if="loading" :rows="4" animated />

      <el-empty v-else-if="!rows.length" description="当前没有命中股票" />

      <div v-else class="stocks-grid">
        <div
          v-for="s in rows"
          :key="s.symbol"
          class="stock-card"
          :class="stockClass(s)"
        >
          <!-- 头部：代码 + 名称 + 板型 + 涨跌标签 -->
          <div class="card-header">
            <span class="sym">{{ s.symbol }}</span>
            <span class="name" :title="s.name">{{ s.name }}</span>
            <el-tag size="small" effect="plain" class="market-tag">{{ boardLabel(s) }}</el-tag>
            <span class="rank-no">#{{ rows.indexOf(s) + 1 }}</span>
          </div>

          <!-- 主体：左侧价格区 + 右侧指标网格 -->
          <div class="card-body">
            <div class="price-section" :class="stockClass(s)">
              <div class="price">{{ fmt(s.close) }}</div>
              <div class="pct-row" :class="stockClass(s)">
                <span class="arrow">{{ pctArrow(s) }}</span>
                <span class="pct">
                  {{ (s.change_percent ?? 0) >= 0 ? '+' : '' }}{{ fmt(s.change_percent) }}%
                </span>
              </div>
              <div class="amt" :class="stockClass(s)">{{ fmtChangeAmt(s) }}</div>
            </div>

            <div class="metrics-grid">
              <div class="metric"><span>开盘</span><b>{{ fmt(s.open) }}</b></div>
              <div class="metric"><span>最高</span><b>{{ fmt(s.high) }}</b></div>
              <div class="metric"><span>最低</span><b>{{ fmt(s.low) }}</b></div>
              <div class="metric"><span>振幅</span><b>{{ fmt(amplitude(s)) }}%</b></div>
              <div class="metric"><span>换手</span><b>{{ fmt(s.turnover_rate) }}%</b></div>
              <div class="metric"><span>量比</span><b class="placeholder">—</b></div>
              <div class="metric"><span>PE</span><b>{{ fmt(s.pe_ttm) }}</b></div>
              <div class="metric"><span>PB</span><b>{{ fmt(s.pb) }}</b></div>
              <div class="metric"><span>净流入(万)</span><b :class="stockClass(s)">{{ fmtW(s.net_amount) }}</b></div>
              <div class="metric"><span>成交量</span><b>{{ fmtVol(s.volume) }}</b></div>
            </div>
          </div>
        </div>
      </div>

      <div class="pager" v-if="rows.length">
        <el-pagination
          :current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="prev, pager, next, ->, total"
          @current-change="onPage"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { listPresets } from '@/utils/api/presets'
import { ElMessage } from 'element-plus'

import { usePresetCache } from '@/composables/usePresetCache'


const { cache, fetchPage, preloadAll, invalidate } = usePresetCache(8)

const presets       = ref([])
const selected      = ref(null)
const selectedId    = ref('')
const selectedIndex = ref(0)
const rows          = ref([])
const total         = ref(0)
const page          = ref(1)
const pageSize      = ref(8)
const loading       = ref(false)
const tradeDate     = ref('')

const fmt  = (v) => (v == null || Number.isNaN(Number(v))) ? '-' : Number(v).toFixed(2)
const fmtW = (v) => v == null ? '-' : (Number(v) / 1e4).toFixed(2)
const stockClass = (s) => {
  const cp = s.change_percent ?? 0
  if (cp > 0) return 'up'
  if (cp < 0) return 'down'
  return 'flat'
}
const pctArrow = (s) => {
  const cp = s.change_percent ?? 0
  if (cp > 0) return '▲'
  if (cp < 0) return '▼'
  return '─'
}
const fmtChangeAmt = (s) => {
  if (s.close == null || s.change_percent == null) return '-'
  const prev = s.close / (1 + s.change_percent / 100)
  const amt = s.close - prev
  return (amt >= 0 ? '+' : '') + amt.toFixed(2)
}
const amplitude = (s) => {
  if (!s.high || !s.low || !s.close || s.change_percent == null) return null
  const pc = s.close / (1 + s.change_percent / 100)
  if (!pc) return null
  return ((s.high - s.low) / pc) * 100
}

const fmtVol = (v) => {
  if (v == null) return '-'
  if (v >= 1e8) return (v/1e8).toFixed(2) + '亿'
  if (v >= 1e4) return (v/1e4).toFixed(0) + '万'
  return String(v)
}


const boardLabel = (s) => {
  const sym = String(s.symbol || '')
  if (sym.startsWith('688')) return '科创板'
  if (sym.startsWith('300') || sym.startsWith('301')) return '创业板'
  if (sym.startsWith('60') || sym.startsWith('00') || sym.startsWith('20')) return '主板'
  if (sym.startsWith('8')  || sym.startsWith('43') || sym.startsWith('92')) return '北交所'
  if (sym.startsWith('9')) return 'B股'
  return s.market || '—'
}

const boardSummary = (p) => {
  const all = p?.expression?.all || []
  for (const c of all) {
    if (c?.type === 'board_in' && Array.isArray(c.boards)) return c.boards.join(' / ')
  }
  return '全部'
}

const selectByIndex = (i) => {
  if (i === selectedIndex.value) return
  selectedIndex.value = i
  syncSelected()
}

const prev = () => {
  const total = presets.value.length
  if (!total) return
  selectedIndex.value = (selectedIndex.value - 1 + total) % total
  syncSelected()
}

const next = () => {
  const total = presets.value.length
  if (!total) return
  selectedIndex.value = (selectedIndex.value + 1) % total
  syncSelected()
}

// 切换规则：优先读缓存，无缓存才 fetch
const syncSelected = async () => {
  const p = presets.value[selectedIndex.value]
  if (!p) return
  selectedId.value = p.id
  selected.value  = p
  page.value      = 1

  if (cache[p.id]) {
    // 命中缓存：瞬时切换，无 loading 闪烁
    const c = cache[p.id]
    rows.value      = c.rows
    total.value     = c.total
    tradeDate.value = c.tradeDate
    loading.value   = false
  } else {
    // 未缓存：fetch 首页
    await fetchRows(1)
  }
}

// 真正发起请求（首次 / 翻页）
const fetchRows = async (pageNum) => {
  if (!selectedId.value) return
  loading.value = true
  try {
    const result = await fetchPage(selectedId.value, pageNum)
    rows.value      = result.rows
    total.value     = result.total
    tradeDate.value = result.tradeDate
  } catch (e) {
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}


const exportCSV = () => {
  if (!rows.value.length) return
  const headers = ['代码', '名称', '板型', '现价', '涨跌幅%', '涨跌额', '振幅%', '换手%', 'PE', 'PB', '净流入(万)']
  const csv = [
    headers.join(','),
    ...rows.value.map(s => [
      s.symbol,
      '"' + (s.name || '').replace(/"/g, '""') + '"',
      boardLabel(s),
      s.close,
      s.change_percent?.toFixed(2),
      fmtChangeAmt(s),
      amplitude(s)?.toFixed(2),
      s.turnover_rate?.toFixed(2),
      s.pe_ttm?.toFixed(2),
      s.pb?.toFixed(2),
      s.net_amount ? (s.net_amount / 1e4).toFixed(2) : '',
    ].join(','))
  ].join('\n')
  const blob = new Blob(['\ufeff' + csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${selectedId.value}_${tradeDate.value || 'today'}.csv`
  a.click()
  URL.revokeObjectURL(url)
  ElMessage.success(`已导出 ${rows.value.length} 条数据`)
}

const onPage = (p) => { page.value = p; fetchRows(p) }

onMounted(async () => {
  try {
    const res = await listPresets()
    presets.value = res.data?.data || []
    if (presets.value.length) {
      selectedIndex.value = 0
      await syncSelected()
      // 后台预加载所有规则的首页（已缓存的自动跳过）
      preloadAll(presets.value)
    }
  } catch (e) {
    ElMessage.error('获取系统规则失败')
  }
})
</script>

<style scoped>
/* =========================================================
   配色方案（红色主题，呼应 A 股"红涨绿跌"惯例）
   - 主色：涨 #e63946 / 跌 #16a34a
   - 选中卡片：红色高亮
   - 整体减少灰色使用
   ========================================================= */
:root { /* fall back if not in global */ }

.page {
  padding: 16px;
  height: 100vh;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  overflow: hidden;
  background: #fff5f5;   /* 整页极浅红背景，呼应主题 */
}

/* ============ 8 卡 grid 全可见 + 选中红色高亮放大 ============ */
.rules-section {
  flex: 0 0 auto;
  padding: 16px 0 8px;
  margin-bottom: 8px;
}

.carousel {
  position: relative;
  padding: 0 36px;
}
.carousel-track {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 12px;
}
@media (max-width: 1280px) { .carousel-track { grid-template-columns: repeat(4, 1fr); } }
@media (max-width: 700px)  { .carousel-track { grid-template-columns: repeat(2, 1fr); } }

.carousel-card {
  min-width: 0;
  height: 220px;
  padding: 14px 14px 12px;
  border-radius: 14px;
  background: #ffffff;
  border: 1.5px solid #ffe0e0;        /* 浅红边框替代灰色 */
  box-shadow: 0 3px 10px rgba(230,57,70,.06);
  cursor: pointer;
  position: relative;
  transition:
    transform .4s cubic-bezier(.4,0,.2,1),
    border-color .3s ease,
    box-shadow .35s ease,
    background .3s ease;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.carousel-card:hover {
  border-color: #ffb4b4;
  box-shadow: 0 8px 22px rgba(230,57,70,.18);
  transform: translateY(-4px);
}
/* 选中卡片：红色高亮 + 更大放大 */
.carousel-card.active {
  transform: scale(1.15) translateY(-14px);
  border-color: #e63946;
  background: linear-gradient(180deg, #ffffff 0%, #fff5f5 100%);
  box-shadow:
    0 0 0 3px rgba(230,57,70,.25),
    0 22px 56px rgba(230,57,70,.35);
  z-index: 10;
}

.pcard-head {
  display: flex; align-items: flex-start;
  margin-bottom: 8px;
  min-height: 20px;
}
.board-tag {
  background: #fff0f0 !important;
  border-color: #ffd0d0 !important;
  color: #c1292e !important;
  font-weight: 600 !important;
  font-size: 10px !important;
  height: 18px !important;
  line-height: 16px !important;
  padding: 0 7px !important;
}

.pcard-name {
  font-size: 16px;
  font-weight: 800;
  color: #000;
  letter-spacing: -0.4px;
  line-height: 1.4;
  margin-bottom: 8px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
}

.pcard-desc {
  font-size: 12.5px; color: #444; line-height: 1.6;
  margin: 0; flex: 1;
  display: -webkit-box; -webkit-line-clamp: 4; -webkit-box-orient: vertical;
  overflow: hidden;
}
.pcard-foot {
  display: flex; justify-content: space-between; align-items: center;
  border-top: 1.5px dashed #ffd0d0;   /* 浅红虚线替代灰色 */
  padding-top: 6px;
  font-size: 11px;
  margin-top: 6px;
}
.pcard-id   { color: #999; font-family: monospace; }
.pcard-hint { color: #16a34a; font-weight: 700; font-size: 14px; }
.carousel-card.active .pcard-hint { color: #e63946; }

.carousel-arrow {
  position: absolute;
  top: 50%; transform: translateY(-50%);
  width: 36px; height: 36px;
  border-radius: 50%;
  border: 1.5px solid #ffd0d0;
  background: rgba(255,255,255,.95);
  color: #c1292e;
  font-size: 20px; line-height: 1;
  cursor: pointer;
  z-index: 20;
  transition: all .25s ease;
  box-shadow: 0 2px 8px rgba(230,57,70,.12);
}
.carousel-arrow:hover {
  border-color: #e63946;
  color: #e63946;
  background: #fff5f5;
  box-shadow: 0 4px 14px rgba(230,57,70,.30);
  transform: translateY(-50%) scale(1.1);
}
.carousel-arrow.left  { left: 0px; }
.carousel-arrow.right { right: 0px; }

.carousel-dots {
  display: flex; justify-content: center; gap: 8px;
  margin-top: 14px;
}
.dot {
  width: 8px; height: 8px; border-radius: 50%;
  background: #ffd0d0;
  cursor: pointer;
  transition: all .3s cubic-bezier(.4,0,.2,1);
}
.dot:hover { background: #ffb4b4; }
.dot.cached { background: #ffb4b4; }    /* 已缓存的圆点更深一点 */
.dot.active {
  background: #e63946;
  width: 26px;
  border-radius: 4px;
  box-shadow: 0 0 8px rgba(230,57,70,.5);
}

/* ============ 命中股票 ============ */
.stocks-section {
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.stocks-header {
  flex: 0 0 auto;
  display: flex; align-items: center; justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
  padding: 4px 4px;
  background: #fff;
  border-radius: 10px;
  border: 1px solid #ffe0e0;
}
.stocks-title {
  margin: 0; font-size: 19px; font-weight: 800; color: #000;
  display: flex; align-items: center; gap: 10px;
}
.stock-red-bar {
  display: inline-block;
  width: 4px; height: 20px;
  background: #e63946;
  border-radius: 2px;
  box-shadow: 0 0 8px rgba(230,57,70,.4);
}
.stocks-meta {
  font-size: 13px; color: #c1292e;
  background: #fff0f0;
  padding: 5px 12px;
  border-radius: 12px;
  border: 1px solid #ffd0d0;
  font-weight: 600;
}


.stocks-grid {
  flex: 1 1 auto;
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  grid-template-rows: repeat(2, 1fr);
  gap: 12px;
  min-height: 0;
}
@media (max-width: 1280px) { .stocks-grid { grid-template-columns: repeat(3, 1fr); } }
@media (max-width: 900px)  { .stocks-grid { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 600px)  { .stocks-grid { grid-template-columns: 1fr; } }

.stock-card {
  position: relative;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  padding: 10px 14px 10px 14px;
  background: #ffffff;
  overflow: hidden;
  transition: border-color .2s ease, box-shadow .2s ease;
}
/* 右侧涨跌色条 */
.stock-card::after {
  content: '';
  position: absolute;
  top: 0; right: 0; bottom: 0;
  width: 3px;
}
.stock-card.up::after { background: #e63946; }
.stock-card.down::after { background: #16a34a; }
.stock-card.flat::after { background: #d1d5db; }

.stock-card.up   { background: #fff8f8; }
.stock-card.down { background: #f6fbf8; }
.stock-card:hover {
  border-color: #d1d5db;
  box-shadow: 0 4px 16px rgba(0,0,0,.06);
}

/* 头部 */
.card-header {
  display: flex; align-items: center; gap: 8px;
  margin-bottom: 10px;
  padding-bottom: 8px;
  border-bottom: 1px dashed #f3f4f6;
}
.card-header .sym {
  font-size: 15px; font-weight: 800; color: #1a1a1a;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  letter-spacing: -0.3px;
}
.card-header .name {
  font-size: 13px; color: #555; font-weight: 600;
  max-width: 110px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.card-header .market-tag {
  margin-left: auto;
  background: #f0f9ff !important;
  border-color: #bae6fd !important;
  color: #0369a1 !important;
  font-weight: 600 !important;
}
.card-header .rank-no {
  font-size: 11px; color: #999;
  font-family: 'SF Mono', Menlo, monospace;
  font-weight: 600;
}

/* 主体：左侧价格区 + 右侧指标网格 */
.card-body {
  display: flex;
  gap: 12px;
  align-items: stretch;
}

.price-section {
  flex: 0 0 auto;
  min-width: 115px;
  padding-right: 12px;
  border-right: 1px dashed #e5e7eb;
  display: flex; flex-direction: column; justify-content: center;
}
.price-section .price {
  font-size: 28px;
  font-weight: 900;
  letter-spacing: -0.8px;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  line-height: 1.05;
}
.price-section.up   .price { color: #e63946; }
.price-section.down .price { color: #16a34a; }

.pct-row {
  display: flex; align-items: center; gap: 3px;
  margin-top: 5px;
  font-weight: 700;
}
.pct-row.up   { color: #e63946; }
.pct-row.down { color: #16a34a; }
.pct-row .arrow { font-size: 12px; }
.pct-row .pct {
  font-size: 14px;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
}

.amt {
  font-size: 12px; font-weight: 600;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  margin-top: 3px;
}
.amt.up   { color: #e63946; }
.amt.down { color: #16a34a; }

.metrics-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 5px 14px;
  align-content: center;
}
.metric {
  display: flex; align-items: center; justify-content: space-between;
  gap: 6px;
  font-size: 11px;
  line-height: 1.5;
}
.metric span {
  color: #888;
  font-weight: 500;
}
.metric b {
  color: #1a1a1a;
  font-weight: 700;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
}
.metric b.up   { color: #e63946; }
.metric b.down { color: #16a34a; }
.metric b.placeholder { color: #cbd5e1; font-weight: 400; }







.pager {
  flex: 0 0 auto;
  margin-top: 14px;
  text-align: right;
}
</style>
