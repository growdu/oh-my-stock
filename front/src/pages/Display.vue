<template>
  <div class="page">
    <!-- 单列堆叠交错的规则卡片 -->
    <div class="rules-section">
      <div class="rule-stack" @wheel.prevent="onWheel" @touchstart.passive="onTouchStart" @touchend.passive="onTouchEnd">
        <button class="stack-arrow left" @click="prev" aria-label="上一条">‹</button>
        <button class="stack-arrow right" @click="next" aria-label="下一条">›</button>
        <div
          v-for="(p, i) in presets"
          :key="p.id"
          class="stack-card"
          :class="{ active: i === selectedIndex, cached: cache[p.id] }"
          :data-offset="stackOffset(i)"
          :data-pos="i"
          @click="selectByIndex(i)"
        >
          <div class="pcard-head">
            <el-tag size="small" effect="plain" class="board-tag">
              {{ boardSummary(p) }}
            </el-tag>
            <span class="pcard-index">{{ i + 1 }} / {{ presets.length }}</span>
          </div>
          <div class="pcard-name">{{ p.name }}</div>
          <div class="pcard-desc">{{ p.description }}</div>
          <div class="pcard-foot">
            <span class="pcard-id">{{ p.id }}</span>
            <span class="pcard-hint">{{ cache[p.id] ? '✓ 已加载' : '… 加载中' }}</span>
          </div>
        </div>
      </div>
      <div class="stack-dots">
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
          <span class="stocks-meta">共 {{ filteredRows.length }} 只<span v-if="industryFilter"> / {{ total }}</span><span v-if="tradeDate"> · {{ tradeDate }}</span></span>
        </div>
        <div class="stocks-actions">
          <el-button size="small" plain @click="exportCSV">
            <span style="margin-right: 4px">⬇</span>导出 CSV
          </el-button>
        </div>
      </div>

      <el-skeleton v-if="loading" :rows="4" animated />

      <el-empty v-else-if="!rows.length" description="当前没有命中股票" />

      <div v-else>
        <div v-if="industries.length" class="industry-filter">
          <span class="if-label">行业</span>
          <button
            class="if-chip"
            :class="{ active: !industryFilter }"
            @click="industryFilter = ''"
          >全部</button>
          <button
            v-for="ind in industries"
            :key="ind"
            class="if-chip"
            :class="{ active: industryFilter === ind }"
            @click="industryFilter = ind"
          >{{ ind }}<span class="if-count">{{ industryCount(ind) }}</span></button>
        </div>

      <div class="stocks-grid">
        <div
          v-for="s in filteredRows"
          :key="s.symbol"
          class="stock-card"
          :class="stockClass(s)"
          @click="openKLine(s)"
          title="点击查看 K 线"
        >
          <!-- 头部：代码 + 名称 + 板型 + 涨跌标签 -->
          <div class="card-header">
            <span class="sym">{{ s.symbol }}</span>
            <span class="name" :title="s.name">{{ s.name }}</span>
            <el-tag size="small" effect="plain" class="market-tag" :class="boardTagClass(s)">{{ boardLabel(s) }}</el-tag>
            <span class="rank-no">#{{ rows.indexOf(s) + 1 }}</span>
          </div>

          <div v-if="s.industry" class="card-industry">
            <span class="ind-label">行业</span>{{ s.industry }}
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
              <div class="metric"><span>PE</span><b>{{ fmt(s.pe_ttm) }}</b></div>
              <div class="metric"><span>PB</span><b>{{ fmt(s.pb) }}</b></div>
              <div class="metric"><span>净流入(万)</span><b :class="stockClass(s)">{{ fmtW(s.net_amount) }}</b></div>
              <div class="metric"><span>成交量</span><b>{{ fmtVol(s.volume) }}</b></div>
              <div class="metric"><span>MACD柱</span><b :class="macdBarClass(s)">{{ fmt(macdBar(s)) }}</b></div>
            </div>
          </div>

          <!-- 技术指标快照：MA / KDJ / RSI / BOLL 关键值，金叉死叉高亮 -->
          <div class="tech-strip">
              <span class="ts-chip" :class="maPosClass(s, 'ma5')" :title="'MA5 昨值 ' + fmt(s.ma5_prev)">
                MA5 {{ fmt(s.ma5) }}
                <em v-if="maCross(s, 'ma5', 'ma10')" class="ts-flag gold">金叉</em>
                <em v-else-if="maDeathCross(s, 'ma5', 'ma10')" class="ts-flag death">死叉</em>
              </span>
              <span class="ts-chip" :class="maPosClass(s, 'ma10')">MA10 {{ fmt(s.ma10) }}</span>
              <span class="ts-chip" :class="maPosClass(s, 'ma20')">MA20 {{ fmt(s.ma20) }}</span>
              <span class="ts-chip" :class="maPosClass(s, 'ma60')">MA60 {{ fmt(s.ma60) }}</span>
              <span class="ts-chip" :class="kdjClass(s)" :title="'K ' + fmt(s.k) + ' / D ' + fmt(s.d)">
                KDJ {{ fmt(s.k) }}/{{ fmt(s.d) }}
                <em v-if="kdjCross(s)" class="ts-flag gold">金叉</em>
              </span>
              <span class="ts-chip" :class="rsiClass(s)">RSI6 {{ fmt(s.rsi6) }}</span>
              <span class="ts-chip" :class="bollClass(s)" :title="'上轨 ' + fmt(s.boll_upper) + ' / 中轨 ' + fmt(s.boll_mid) + ' / 下轨 ' + fmt(s.boll_lower)">
                BOLL {{ fmt(bollPctB(s) * 100) }}%
              </span>
            </div>
          </div>
        </div>
      </div>

      <KLineDialog v-model="klineOpen" :stock="klineStock" />

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
import { ref, onMounted, computed } from 'vue'
import { listPresets } from '@/utils/api/presets'
import { ElMessage } from 'element-plus'

import { usePresetCache } from '@/composables/usePresetCache'
import KLineDialog from '@/components/KLineDialog.vue'


const { cache, fetchPage, preloadAll, invalidate } = usePresetCache(8)

// K线弹窗状态
const klineOpen  = ref(false)
const klineStock = ref(null)
function openKLine(s) {
  klineStock.value = s
  klineOpen.value  = true
}

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

// === 技术指标工具函数 ===
// 后端对缺失数据 COALESCE 成 0，所以这里要再过滤一下，避免 "无数据" 被显示成 "持平"
const hasValue = (v) => v != null && Math.abs(v) > 1e-9
const macdBar = (s) => {
  if (!hasValue(s.dif) || !hasValue(s.dea)) return null
  return (s.dif - s.dea) * 2
}
const macdBarClass = (s) => {
  const b = macdBar(s)
  if (b == null) return ''
  if (Math.abs(b) < 0.001) return ''
  return b >= 0 ? 'up' : 'down'
}
const maPosClass = (s, key) => {
  const ma = s[key]
  if (!hasValue(ma) || s.close == null) return ''
  return s.close >= ma ? 'up' : 'down'
}
const maCross = (s, fastKey, slowKey) => {
  const f = s[fastKey], fp = s[fastKey + '_prev']
  const sl = s[slowKey], slp = s[slowKey + '_prev']
  if (!hasValue(f) || !hasValue(fp) || !hasValue(sl) || !hasValue(slp)) return false
  return fp <= slp && f > sl
}
const maDeathCross = (s, fastKey, slowKey) => {
  const f = s[fastKey], fp = s[fastKey + '_prev']
  const sl = s[slowKey], slp = s[slowKey + '_prev']
  if (!hasValue(f) || !hasValue(fp) || !hasValue(sl) || !hasValue(slp)) return false
  return fp >= slp && f < sl
}
const kdjClass = (s) => {
  if (!hasValue(s.k) || !hasValue(s.d)) return ''
  if (s.k > 80 && s.d > 80) return 'overbought'
  if (s.k < 20 && s.d < 20) return 'oversold'
  return s.k >= s.d ? 'up' : 'down'
}
const kdjCross = (s) => {
  if (!hasValue(s.k) || !hasValue(s.d)) return false
  return s.k > s.d && (s.k - s.d) >= 0.5
}
const rsiClass = (s) => {
  const r = s.rsi6
  if (!hasValue(r)) return ''
  if (r >= 80) return 'overbought'
  if (r <= 30) return 'oversold'
  return ''
}
const bollPctB = (s) => {
  if (!hasValue(s.boll_upper) || !hasValue(s.boll_lower) || s.close == null) return null
  const range = s.boll_upper - s.boll_lower
  if (range <= 0) return null
  return Math.max(0, Math.min(1, (s.close - s.boll_lower) / range))
}
const bollClass = (s) => {
  const p = bollPctB(s)
  if (p == null) return ''
  if (p >= 1) return 'overbought'
  if (p <= 0) return 'oversold'
  return p >= 0.5 ? 'up' : 'down'
}

// === 行业筛选（前端按当前 rows 过滤） ===
const industryFilter = ref('')
const industries = computed(() => {
  const set = new Set()
  for (const s of rows.value) {
    if (s.industry) set.add(s.industry)
  }
  return Array.from(set).sort()
})
const industryCount = (ind) => rows.value.filter(s => s.industry === ind).length
const filteredRows = computed(() => {
  if (!industryFilter.value) return rows.value
  return rows.value.filter(s => s.industry === industryFilter.value)
})


const boardTagClass = (s) => {
  const m = s.market || ''
  if (m === '科创板' || /^688/.test(s.symbol || '')) return 'tag-STAR'
  if (m === '创业板' || /^30[01]/.test(s.symbol || '')) return 'tag-CHINEXT'
  if (m === '主板' || m === '深主板' || m === '沈主板') return 'tag-MAIN'
  return 'tag-OTHER'
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

// 卡片相对 active 索引的偏移：负数=左，正数=右
// 用环形距离：选 i 时显示 (i - selected) mod N，平铺到 -N/2..N/2
const stackOffset = (i) => {
  const n = presets.value.length
  if (!n) return 0
  let d = i - selectedIndex.value
  if (d > n/2) d -= n
  if (d < -n/2) d += n
  return d
}

// 鼠标滚轮切换（debounce）
let _wheelLock = false
const onWheel = (e) => {
  if (_wheelLock) return
  if (Math.abs(e.deltaY) < 10 && Math.abs(e.deltaX) < 10) return
  _wheelLock = true
  setTimeout(() => { _wheelLock = false }, 600)
  if (e.deltaY > 0 || e.deltaX > 0) next()
  else prev()
}

// 触摸滑动切换
let _touchX = 0
const onTouchStart = (e) => { _touchX = e.touches[0].clientX }
const onTouchEnd = (e) => {
  const dx = (e.changedTouches[0].clientX - _touchX)
  if (Math.abs(dx) < 40) return
  if (dx < 0) next()
  else prev()
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
  const headers = ['代码', '名称', '板型', '行业', '现价', '涨跌幅%', '涨跌额', '振幅%', '换手%', 'PE', 'PB', '净流入(万)', 'MA5', 'MA10', 'MA20', 'MACD', 'KDJ-K', 'KDJ-D', 'RSI6', 'BOLL%']
  const csv = [
    headers.join(','),
    ...filteredRows.value.map(s => [
      s.symbol,
      '"' + (s.name || '').replace(/"/g, '""') + '"',
      boardLabel(s),
      s.industry || '',
      s.close,
      s.change_percent?.toFixed(2),
      fmtChangeAmt(s),
      amplitude(s)?.toFixed(2),
      s.turnover_rate?.toFixed(2),
      s.pe_ttm?.toFixed(2),
      s.pb?.toFixed(2),
      s.net_amount ? (s.net_amount / 1e4).toFixed(2) : '',
      s.ma5?.toFixed(3),
      s.ma10?.toFixed(3),
      s.ma20?.toFixed(3),
      macdBar(s)?.toFixed(3),
      s.k?.toFixed(2),
      s.d?.toFixed(2),
      s.rsi6?.toFixed(2),
      bollPctB(s) != null ? (bollPctB(s) * 100).toFixed(1) + '%' : '',
    ].join(','))
  ].join('\n')
  const blob = new Blob(['\ufeff' + csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${selectedId.value}${industryFilter ? '_' + industryFilter : ''}_${tradeDate.value || 'today'}.csv`
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

/* ============ 单列堆叠交错卡片（球面 3D 循环）============ */
.rule-stack {
  position: relative;
  height: 220px;            /* 保持原卡片高度 */
  width: 100%;
  perspective: 1200px;
  perspective-origin: 50% 50%;
  overflow: hidden;
  touch-action: pan-y;
  padding: 0 0;
}

/* 左右切换箭头 — 绝对定位在卡片左右两边 */
.stack-arrow {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 44px; height: 44px;
  border-radius: 50%;
  border: 1.5px solid #ffd0d0;
  background: rgba(255,255,255,.96);
  color: #c1292e;
  font-size: 26px;
  line-height: 1;
  cursor: pointer;
  z-index: 50;
  transition: all .2s ease;
  box-shadow: 0 4px 14px rgba(230,57,70,.18);
  display: flex; align-items: center; justify-content: center;
  font-weight: 700;
  padding: 0;
}
.stack-arrow:hover {
  border-color: #e63946;
  color: #e63946;
  background: #fff5f5;
  transform: translateY(-50%) scale(1.10);
  box-shadow: 0 6px 20px rgba(230,57,70,.32);
}
.stack-arrow:active { transform: translateY(-50%) scale(.95); }
.stack-arrow.left  { left: 60px; }
.stack-arrow.right { right: 60px; }
@media (max-width: 760px) {
  .stack-arrow.left  { left: 12px; }
  .stack-arrow.right { right: 12px; }
  .stack-arrow { width: 36px; height: 36px; font-size: 22px; }
}

/* 下方 dots 指示器 */
.stack-dots {
  display: flex;
  justify-content: center;
  gap: 7px;
  margin-top: 12px;
}
.dot {
  width: 7px; height: 7px; border-radius: 50%;
  background: #ffd0d0;
  cursor: pointer;
  transition: all .25s cubic-bezier(.4,0,.2,1);
}
.dot:hover { background: #ffb4b4; }
.dot.cached { background: #ffb4b4; }
.dot.active {
  background: #e63946;
  width: 24px;
  border-radius: 4px;
  box-shadow: 0 0 6px rgba(230,57,70,.5);
}



.stack-card {
  position: absolute;
  top: 50%;
  left: 50%;
  width: 380px;             /* 卡片宽度更大 */
  min-height: 220px;
  padding: 14px 14px 12px;
  border-radius: 14px;
  background: #ffffff;
  border: 1.5px solid #ffe0e0;
  box-shadow: 0 3px 10px rgba(230,57,70,.06);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition:
    transform .6s cubic-bezier(.22,.9,.28,1),
    opacity .45s ease,
    box-shadow .45s ease,
    border-color .35s ease,
    filter .45s ease;
  will-change: transform, opacity;
  transform-origin: center center;
  user-select: none;
}

/* 当前卡：最前，不旋转 */
.stack-card[data-offset="0"] {
  transform: translate(-50%, -50%) translateX(0) translateZ(0) scale(1) rotateY(0);
  z-index: 30;
  opacity: 1;
  border-color: #e63946;
  background: linear-gradient(180deg, #ffffff 0%, #fff5f5 100%);
  box-shadow:
    0 0 0 3px rgba(230,57,70,.22),
    0 24px 50px rgba(230,57,70,.32);
}
/* 紧邻：左 60% 宽、右 60% 宽 + rotateY 25° 球面感 */
.stack-card[data-offset="-1"] {
  transform: translate(-50%, -50%) translateX(-78%) translateZ(-60px) scale(.86) rotateY(32deg);
  z-index: 20;
  opacity: .65;
  filter: brightness(.97);
  border-color: #ffd0d0;
}
.stack-card[data-offset="1"] {
  transform: translate(-50%, -50%) translateX(78%) translateZ(-60px) scale(.86) rotateY(-32deg);
  z-index: 20;
  opacity: .65;
  filter: brightness(.97);
  border-color: #ffd0d0;
}
/* 第二圈：再往两边，重叠更多 */
.stack-card[data-offset="-2"] {
  transform: translate(-50%, -50%) translateX(-130%) translateZ(-130px) scale(.7) rotateY(42deg);
  z-index: 10;
  opacity: .35;
  filter: brightness(.94);
  border-color: #ffe0e0;
}
.stack-card[data-offset="2"] {
  transform: translate(-50%, -50%) translateX(130%) translateZ(-130px) scale(.7) rotateY(-42deg);
  z-index: 10;
  opacity: .35;
  filter: brightness(.94);
  border-color: #ffe0e0;
}
/* 第三圈：几乎只露一条边 */
.stack-card[data-offset="-3"] {
  transform: translate(-50%, -50%) translateX(-160%) translateZ(-200px) scale(.55) rotateY(48deg);
  z-index: 5;
  opacity: .15;
}
.stack-card[data-offset="3"] {
  transform: translate(-50%, -50%) translateX(160%) translateZ(-200px) scale(.55) rotateY(-48deg);
  z-index: 5;
  opacity: .15;
}
/* 远端：完全隐藏（环回，准备滑入）*/
.stack-card[data-offset^="-4"],
.stack-card[data-offset^="-5"],
.stack-card[data-offset^="-6"],
.stack-card[data-offset^="-7"],
.stack-card[data-offset^="-8"],
.stack-card[data-offset^="4"],
.stack-card[data-offset^="5"],
.stack-card[data-offset^="6"],
.stack-card[data-offset^="7"],
.stack-card[data-offset^="8"] {
  transform: translate(-50%, -50%) translateX(0) translateZ(-300px) scale(.4) rotateY(0);
  z-index: 1;
  opacity: 0;
  pointer-events: none;
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
  border: 1px solid #ffd0d0;
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
  gap: 16px;
  min-height: 0;
}
@media (max-width: 1280px) { .stocks-grid { grid-template-columns: repeat(3, 1fr); } }
@media (max-width: 900px)  { .stocks-grid { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 600px)  { .stocks-grid { grid-template-columns: 1fr; } }

.stock-card {
  position: relative;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  padding: 14px 16px 12px 16px;
  background: #ffffff;
  overflow: hidden;
  transition: transform .25s ease, box-shadow .25s ease, border-color .25s ease;
  box-shadow: 0 1px 3px rgba(0,0,0,.04);
  display: flex; flex-direction: column;
}
.stock-card::after {
  content: '';
  position: absolute;
  top: 0; right: 0; bottom: 0;
  width: 4px;
}
.stock-card.up::after { background: #e63946; }
.stock-card.down::after { background: #16a34a; }
.stock-card.flat::after { background: #d1d5db; }

.stock-card.up   { background: #fff8f8; }
.stock-card.down { background: #f6fbf6; }
.stock-card:hover {
  transform: translateY(-3px);
  border-color: #c7d2fe;
  box-shadow: 0 10px 28px rgba(0,0,0,.10);
}

.card-header {
  display: flex; align-items: center; gap: 10px;
  margin-bottom: 10px;
  padding-bottom: 10px;
  border-bottom: 1px dashed #f3f4f6;
}
.card-header .sym {
  font-size: 20px; font-weight: 800; color: #111;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  letter-spacing: -0.5px;
}
.card-header .name {
  font-size: 17px; color: #222; font-weight: 700;
  max-width: 140px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.card-header .market-tag {
  margin-left: auto;
  font-weight: 700 !important;
  font-size: 13px !important;
  padding: 0 10px !important;
  height: 24px !important;
  line-height: 22px !important;
}
.card-header .market-tag.tag-STAR   { background: #fdf2f8 !important; border-color: #f9a8d4 !important; color: #be185d !important; }
.card-header .market-tag.tag-CHINEXT{ background: #fff7ed !important; border-color: #fdba74 !important; color: #c2410c !important; }
.card-header .market-tag.tag-MAIN   { background: #f0f9ff !important; border-color: #bae6fd !important; color: #0369a1 !important; }
.card-header .market-tag.tag-OTHER  { background: #f3f4f6 !important; border-color: #d1d5db !important; color: #4b5563 !important; }
.card-header .rank-no {
  font-size: 14px; color: #999;
  font-family: 'SF Mono', Menlo, monospace;
  font-weight: 700;
}

.card-industry {
  font-size: 14px;
  color: #4b5563;
  font-weight: 600;
  margin: 0 0 12px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.card-industry .ind-label { color: #9ca3af; margin-right: 6px; font-weight: 500; }

.card-body {
  flex: 1;
  display: flex;
  gap: 12px;
  align-items: stretch;
}

.price-section {
  flex: 0 0 auto;
  width: 110px;
  padding-right: 10px;
  border-right: 1px dashed #e5e7eb;
  display: flex; flex-direction: column; justify-content: center;
  gap: 2px;
}
.price-section .price {
  font-size: 30px;
  font-weight: 900;
  letter-spacing: -0.8px;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  line-height: 1.15;
}
.price-section.up   .price { color: #e63946; }
.price-section.down .price { color: #16a34a; }

.pct-row {
  display: flex; align-items: center; gap: 4px;
  margin-top: 6px;
  font-weight: 800;
}
.pct-row.up   { color: #e63946; }
.pct-row.down { color: #16a34a; }
.pct-row .arrow { font-size: 14px; }
.pct-row .pct {
  font-size: 17px;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
}

.amt {
  font-size: 13px; font-weight: 700;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  margin-top: 4px;
}
.amt.up   { color: #e63946; }
.amt.down { color: #16a34a; }

.metrics-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px 12px;
  align-content: center;
  padding-left: 6px;
}
.metric {
  display: flex; flex-direction: column; align-items: flex-start; justify-content: center;
  gap: 2px;
  font-size: 12px;
  line-height: 1.3;
  min-width: 0;
  padding: 4px 6px;
  background: #fafbfc;
  border-radius: 6px;
}
.metric span {
  color: #9ca3af;
  font-weight: 500;
  font-size: 11px;
}
.metric b {
  color: #111;
  font-weight: 800;
  font-size: 14px;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}
.metric b.up   { color: #e63946; }
.metric b.down { color: #16a34a; }
.metric b.placeholder { color: #cbd5e1; font-weight: 400; }

/* ========= 移动端适配 (≤760px) ========= */
@media (max-width: 760px) {
  /* 整页 */
  .page { padding: 10px 8px; }

  /* 规则堆叠区 */
  .rule-stack {
    height: 180px;
    perspective: 800px;
  }
  .stack-card {
    width: 88vw;
    max-width: 360px;
    min-height: 180px;
    padding: 12px 12px 10px;
  }
  .stack-arrow.left  { left: 6px;  }
  .stack-arrow.right { right: 6px; }
  .stack-arrow { width: 36px; height: 36px; font-size: 22px; }

  .stack-card[data-offset="-1"] { transform: translate(-50%, -50%) translateX(-72%) translateZ(-50px) scale(.84) rotateY(32deg); }
  .stack-card[data-offset="1"]  { transform: translate(-50%, -50%) translateX(72%)  translateZ(-50px) scale(.84) rotateY(-32deg); }
  .stack-card[data-offset="-2"] { transform: translate(-50%, -50%) translateX(-118%) translateZ(-110px) scale(.68) rotateY(42deg); }
  .stack-card[data-offset="2"]  { transform: translate(-50%, -50%) translateX(118%)  translateZ(-110px) scale(.68) rotateY(-42deg); }
  .stack-card[data-offset="-3"] { transform: translate(-50%, -50%) translateX(-148%) translateZ(-180px) scale(.50) rotateY(48deg); }
  .stack-card[data-offset="3"]  { transform: translate(-50%, -50%) translateX(148%)  translateZ(-180px) scale(.50) rotateY(-48deg); }

  .pcard-name  { font-size: 15px; }
  .pcard-desc  { font-size: 12px; }
  .board-tag   { font-size: 9px  !important; height: 16px !important; line-height: 14px !important; }

  /* 顶部标题条 */
  .stocks-header {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }
  .stocks-title { font-size: 14px; }
  .stocks-meta  { font-size: 11px; }

  /* 行业筛选 chip 横向滚动 */
  .industry-filter {
    overflow-x: auto;
    flex-wrap: nowrap;
    -webkit-overflow-scrolling: touch;
  }
  .industry-filter .if-chip { flex: 0 0 auto; }

  /* 股票卡：单列全宽 + 紧凑 */
  .stocks-grid {
    grid-template-columns: 1fr !important;
    gap: 12px;
  }
  .stock-card {
    padding: 12px 12px 10px;
  }
  .card-header { gap: 8px; margin-bottom: 8px; padding-bottom: 8px; }
  .card-header .sym { font-size: 18px; }
  .card-header .name { font-size: 15px; max-width: 110px; }
  .card-header .rank-no { font-size: 12px; }

  /* 主体：价格在上、参数网格在下，单列排版 */
  .card-body {
    flex-direction: column;
    gap: 10px;
  }
  .price-section {
    width: 100%;
    min-width: 0;
    flex-direction: row;
    align-items: baseline;
    gap: 12px;
    padding: 0 0 8px;
    border-right: none;
    border-bottom: 1px dashed #e5e7eb;
  }
  .price-section .price { font-size: 26px; }
  .pct-row { margin-top: 0; }
  .pct-row .pct  { font-size: 16px; }
  .pct-row .arrow { font-size: 13px; }
  .amt { font-size: 12px; margin-top: 0; }

  /* 5 列 → 2 列 */
  .metrics-grid {
    padding-left: 0;
    grid-template-columns: repeat(2, 1fr) !important;
    gap: 6px;
  }
  .metric { padding: 4px 6px; }
  .metric span { font-size: 10.5px; }
  .metric b    { font-size: 13px; }

  /* 技术指标条：紧凑可读 */
  .tech-strip { gap: 4px 6px; margin-top: 10px; padding-top: 6px; }
  .ts-chip { font-size: 11px; padding: 2px 6px; }

  /* 分页 */
  .pager { justify-content: center; padding: 6px 0; }
}

@media (max-width: 380px) {
  .stack-card { width: 92vw; }
  .metric b { font-size: 12.5px; }
  .price-section .price { font-size: 22px; }
}

/* 移动端：让整页可滚动看更多股票 */
@media (max-width: 760px) {
  .page {
    height: auto;
    min-height: 100vh;
    min-height: 100dvh;
    overflow-y: auto;
    overflow-x: hidden;
    -webkit-overflow-scrolling: touch;
    overscroll-behavior-y: contain;
  }
  .rules-section { flex: 0 0 auto; }
  .stocks-section { flex: 0 1 auto; }
}

/* ===== 技术指标条 ===== */
.tech-strip {
  margin-top: 12px;
  padding: 8px 0 0;
  border-top: 1px dashed #f3f4f6;
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-start;
  gap: 6px 8px;
}
.ts-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 9px;
  font-size: 12px;
  font-weight: 700;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  color: #475569;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  white-space: nowrap;
  line-height: 1.5;
}
.ts-chip.up          { color: #e63946; background: #fff5f5; border-color: #ffd0d0; }
.ts-chip.down        { color: #16a34a; background: #f6fbf6; border-color: #bbf7d0; }
.ts-chip.overbought  { color: #c2410c; background: #fff7ed; border-color: #fdba74; }
.ts-chip.oversold    { color: #0369a1; background: #f0f9ff; border-color: #bae6fd; }
.ts-flag {
  font-style: normal;
  font-size: 10px;
  font-weight: 800;
  padding: 0 4px;
  border-radius: 3px;
  margin-left: 2px;
}
.ts-flag.gold  { color: #fff; background: #e63946; }
.ts-flag.death { color: #fff; background: #16a34a; }

/* ===== 行业 chip 过滤栏 ===== */
.industry-filter {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin: 0 0 10px 0;
  padding: 8px 10px;
  background: #fff;
  border: 1px solid #ffd0d0;
  border-radius: 8px;
}
.if-label {
  font-size: 12px;
  color: #c1292e;
  font-weight: 800;
  background: #fff0f0;
  padding: 3px 8px;
  border-radius: 4px;
  border: 1px solid #ffd0d0;
}
.if-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 600;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 999px;
  color: #475569;
  cursor: pointer;
  transition: all .2s ease;
}
.if-chip:hover { border-color: #ffb4b4; color: #c1292e; }
.if-chip.active {
  background: #e63946;
  color: #fff;
  border-color: #e63946;
  box-shadow: 0 2px 6px rgba(230,57,70,.3);
}
.if-count {
  font-size: 10px;
  background: rgba(255,255,255,.3);
  padding: 0 5px;
  border-radius: 8px;
  font-weight: 700;
}
.if-chip:not(.active) .if-count {
  background: #fff;
  color: #94a3b8;
}







.pager {
  flex: 0 0 auto;
  margin-top: 14px;
  text-align: right;
}
</style>
