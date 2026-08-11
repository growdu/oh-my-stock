<template>
  <div class="results-page">
    <!-- 8 个系统规则 grid -->
    <div class="carousel-section">
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
        <h3 class="stocks-title">
          <span class="stock-red-bar"></span>
          「{{ selected.name }}」命中结果
        </h3>
        <span class="stocks-meta">共 {{ total }} 只<span v-if="tradeDate"> · {{ tradeDate }}</span></span>
      </div>

      <el-skeleton v-if="loading" :rows="4" animated />

      <el-empty v-else-if="!rows.length" description="当前没有命中股票" />

      <div v-else class="cards-stage stocks-grid">
        <div
          v-for="s in rows"
          :key="s.symbol"
          class="stock-card"
          @mousemove="onTiltMove"
          @mouseleave="onTiltLeave"
        >
          <div class="card-row1 layer-1">
            <span class="sym">{{ s.symbol }}</span>
            <span class="name" :title="s.name">{{ s.name }}</span>
            <el-tag size="small" effect="plain" class="market-tag">
              {{ boardLabel(s) }}
            </el-tag>
          </div>
          <div class="card-row2 layer-2">
            <span class="price">{{ fmt(s.close) }}</span>
            <span :class="['pct', (s.change_percent ?? 0) >= 0 ? 'up' : 'down']">
              {{ (s.change_percent ?? 0) >= 0 ? '+' : '' }}{{ fmt(s.change_percent) }}%
            </span>
          </div>
          <div class="card-row3 layer-3">
            <div class="kv"><span>开</span><b>{{ fmt(s.open) }}</b></div>
            <div class="kv"><span>高</span><b>{{ fmt(s.high) }}</b></div>
            <div class="kv"><span>低</span><b>{{ fmt(s.low) }}</b></div>
          </div>
          <div class="card-row4 layer-4">
            <div class="kv"><span>换手</span><b>{{ fmt(s.turnover_rate) }}%</b></div>
            <div class="kv"><span>净流入(万)</span><b>{{ fmtW(s.net_amount) }}</b></div>
            <div class="kv"><span>PE</span><b>{{ fmt(s.pe_ttm) }}</b></div>
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
import { useCard3D } from '@/composables/useCard3D'
import { usePresetCache } from '@/composables/usePresetCache'

const { onTiltMove, onTiltLeave } = useCard3D({ max: 9 })
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

const syncSelected = async () => {
  const p = presets.value[selectedIndex.value]
  if (!p) return
  selectedId.value = p.id
  selected.value  = p
  page.value      = 1

  if (cache[p.id]) {
    const c = cache[p.id]
    rows.value      = c.rows
    total.value     = c.total
    tradeDate.value = c.tradeDate
    loading.value   = false
  } else {
    await fetchRows(1)
  }
}

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

const onPage = (p) => { page.value = p; fetchRows(p) }

onMounted(async () => {
  try {
    const res = await listPresets()
    presets.value = res.data?.data || []
    if (presets.value.length) {
      selectedIndex.value = 0
      await syncSelected()
      preloadAll(presets.value)
    }
  } catch (e) {
    ElMessage.error('获取系统规则失败')
  }
})
</script>

<style scoped>
/* =========================================================
   Results.vue 配色（蓝色管理主题，区分公开页）
   公开页（Display）：红色主题
   管理页（Results）：蓝色主题
   ========================================================= */
.results-page {
  padding: 16px;
  height: 100vh;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  overflow: hidden;
  background: #f0f6ff;
}

.carousel-section {
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
  border: 1.5px solid #d6e8ff;
  box-shadow: 0 3px 10px rgba(64,158,255,.06);
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
  border-color: #99c8ff;
  box-shadow: 0 8px 22px rgba(64,158,255,.22);
  transform: translateY(-4px);
}
/* 选中卡片：蓝色高亮 + 更大放大 */
.carousel-card.active {
  transform: scale(1.15) translateY(-14px);
  border-color: #409eff;
  background: linear-gradient(180deg, #ffffff 0%, #f5faff 100%);
  box-shadow:
    0 0 0 3px rgba(64,158,255,.30),
    0 22px 56px rgba(64,158,255,.40);
  z-index: 10;
}

.pcard-head {
  display: flex; align-items: flex-start;
  margin-bottom: 8px;
  min-height: 20px;
}
.board-tag {
  background: #ecf5ff !important;
  border-color: #b3d8ff !important;
  color: #1d6fcf !important;
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
  border-top: 1.5px dashed #d6e8ff;
  padding-top: 6px;
  font-size: 11px;
  margin-top: 6px;
}
.pcard-id   { color: #999; font-family: monospace; }
.pcard-hint { color: #409eff; font-weight: 700; font-size: 14px; }

.carousel-arrow {
  position: absolute;
  top: 50%; transform: translateY(-50%);
  width: 36px; height: 36px;
  border-radius: 50%;
  border: 1.5px solid #b3d8ff;
  background: rgba(255,255,255,.95);
  color: #1d6fcf;
  font-size: 20px; line-height: 1;
  cursor: pointer;
  z-index: 20;
  transition: all .25s ease;
  box-shadow: 0 2px 8px rgba(64,158,255,.12);
}
.carousel-arrow:hover {
  border-color: #409eff;
  color: #409eff;
  background: #ecf5ff;
  box-shadow: 0 4px 14px rgba(64,158,255,.30);
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
  background: #d6e8ff;
  cursor: pointer;
  transition: all .3s cubic-bezier(.4,0,.2,1);
}
.dot:hover { background: #99c8ff; }
.dot.cached { background: #99c8ff; }
.dot.active {
  background: #409eff;
  width: 26px;
  border-radius: 4px;
  box-shadow: 0 0 8px rgba(64,158,255,.5);
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
  display: flex; align-items: center; gap: 12px;
  margin-bottom: 14px;
  padding: 0 4px;
}
.stocks-title {
  margin: 0; font-size: 19px; font-weight: 800; color: #000;
  display: flex; align-items: center; gap: 10px;
}
.stock-red-bar {
  display: inline-block;
  width: 4px; height: 20px;
  background: #409eff;
  border-radius: 2px;
  box-shadow: 0 0 8px rgba(64,158,255,.4);
}
.stocks-meta {
  font-size: 13px; color: #1d6fcf;
  background: #ecf5ff;
  padding: 5px 12px;
  border-radius: 12px;
  border: 1px solid #b3d8ff;
  font-weight: 600;
}

.cards-stage { perspective: 1200px; }
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
  border: 1.5px solid #d6e8ff;
  border-radius: 12px;
  padding: 14px;
  background: #ffffff;
  transform-style: preserve-3d;
  will-change: transform, box-shadow;
}
.stock-card::before {
  content: ''; position: absolute; inset: 0;
  border-radius: 12px; pointer-events: none;
  background: radial-gradient(120% 80% at 0% 0%, rgba(64,158,255,.06), transparent 60%);
}
.stock-card:hover {
  border-color: #99c8ff;
  box-shadow: 0 6px 18px rgba(64,158,255,.15);
}

.card-row1 {
  display: flex; align-items: center; gap: 8px;
  transform: translateZ(6px);
}
.card-row1 .sym {
  font-size: 15px;
  font-weight: 800;
  color: #000;
  letter-spacing: -0.3px;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
}
.card-row1 .name {
  color: #000;
  font-weight: 700;
  font-size: 14px;
  max-width: 130px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.card-row1 .market-tag {
  margin-left: auto;
  background: #ecf5ff !important;
  border-color: #b3d8ff !important;
  color: #1d6fcf !important;
  font-weight: 600 !important;
}

.card-row2 {
  display: flex; align-items: baseline; justify-content: space-between;
  margin: 10px 0 10px;
  transform: translateZ(20px);
}
.card-row2 .price {
  font-size: 30px;
  font-weight: 900;
  color: #000;
  letter-spacing: -0.8px;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
}
.card-row2 .pct {
  font-size: 17px;
  font-weight: 800;
}
.up   { color: #e63946; }                /* A股红涨 */
.down { color: #16a34a; }                /* A股绿跌 */

.card-row3, .card-row4 {
  display: flex; justify-content: space-between;
  border-top: 1.5px dashed #d6e8ff;
  padding-top: 8px;
  margin-top: 8px;
  transform: translateZ(4px);
}
.card-row3 { margin-top: 10px; }
.card-row4 { transform: translateZ(10px); }
.card-row3 .kv, .card-row4 .kv {
  display: flex; flex-direction: column; align-items: flex-start;
  gap: 4px;
}
.card-row3 .kv span, .card-row4 .kv span {
  font-size: 11px;
  color: #888;
  font-weight: 500;
}
.card-row3 .kv b, .card-row4 .kv b {
  color: #000;
  font-weight: 800;
  font-size: 14px;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
}

.pager {
  flex: 0 0 auto;
  margin-top: 14px;
  text-align: right;
}
</style>
