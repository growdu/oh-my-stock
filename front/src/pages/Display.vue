<template>
  <div class="page">
    <!-- 8 个系统规则 grid（直接放在最外层 div） -->
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
              <span class="pcard-name">{{ p.name }}</span>
            </div>
            <div class="pcard-tag">
              <el-tag size="small" effect="plain">{{ boardSummary(p) }}</el-tag>
            </div>
            <div class="pcard-desc">{{ p.description }}</div>
            <div class="pcard-foot">
              <span class="pcard-id">{{ p.id }}</span>
              <span class="pcard-hint">▸</span>
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
          :class="{ active: i === selectedIndex }"
          :title="p.name"
          @click="selectByIndex(i)"
        />
      </div>
    </div>

    <!-- 命中股票（直接放在最外层 div，无 el-card） -->
    <div v-if="selected" class="stocks-section">
      <div class="stocks-header">
        <h3 class="stocks-title">「{{ selected.name }}」命中结果</h3>
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
import { listPresets, runPreset } from '@/utils/api/presets'
import { ElMessage } from 'element-plus'
import { useCard3D } from '@/composables/useCard3D'

const { onTiltMove, onTiltLeave } = useCard3D({ max: 9 })

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

const syncSelected = () => {
  const p = presets.value[selectedIndex.value]
  if (!p) return
  selectedId.value = p.id
  selected.value  = p
  page.value      = 1
  fetchRows()
}

const fetchRows = async () => {
  if (!selectedId.value) return
  loading.value = true
  try {
    const res = await runPreset(selectedId.value, page.value, pageSize.value)
    rows.value      = res.data?.data || []
    total.value     = res.data?.total || 0
    tradeDate.value = rows.value[0]?.trade_date || ''
  } catch (e) {
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const onPage = (p) => { page.value = p; fetchRows() }

onMounted(async () => {
  try {
    const res = await listPresets()
    presets.value = res.data?.data || []
    if (presets.value.length) {
      selectedIndex.value = 0
      syncSelected()
    }
  } catch (e) {
    ElMessage.error('获取系统规则失败')
  }
})
</script>

<style scoped>
.page {
  padding: 16px;
  height: 100vh;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  overflow: hidden;
}

/* ============ 8 卡 grid 全可见 + 选中高亮放大 ============ */
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
  height: 170px;
  padding: 12px 12px 10px;
  border-radius: 12px;
  background: linear-gradient(180deg, #ffffff 0%, #fafbfc 100%);
  border: 1px solid #ebeef5;
  box-shadow: 0 2px 6px rgba(0,0,0,.04);
  cursor: pointer;
  position: relative;
  transition:
    transform .35s cubic-bezier(.4,0,.2,1),
    border-color .3s ease,
    box-shadow .3s ease,
    background .3s ease;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.carousel-card:hover {
  border-color: #b3d8ff;
  box-shadow: 0 6px 18px rgba(64,158,255,.18);
  transform: translateY(-2px);
}
.carousel-card.active {
  transform: scale(1.08) translateY(-8px);
  border-color: #409eff;
  background: linear-gradient(180deg, #ffffff 0%, #eaf4ff 100%);
  box-shadow:
    0 0 0 2px rgba(64,158,255,.3),
    0 16px 40px rgba(64,158,255,.35);
  z-index: 10;
}

.pcard-head { display: flex; align-items: center; gap: 6px; margin-bottom: 6px; }
.pcard-name {
  font-size: 14px; font-weight: 700; color: #303133;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  letter-spacing: -0.2px;
}
.pcard-tag { margin-bottom: 6px; }
.pcard-desc {
  font-size: 11.5px; color: #606266; line-height: 1.55;
  margin: 0; flex: 1;
  display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical;
  overflow: hidden;
}
.pcard-foot {
  display: flex; justify-content: space-between; align-items: center;
  border-top: 1px dashed #ebeef5; padding-top: 4px;
  font-size: 10px;
  margin-top: 6px;
}
.pcard-id   { color: #c0c4cc; font-family: monospace; }
.pcard-hint { color: #409eff; opacity: 0; transition: opacity .25s; font-size: 14px; }
.carousel-card.active .pcard-hint { opacity: 1; }

.carousel-arrow {
  position: absolute;
  top: 50%; transform: translateY(-50%);
  width: 32px; height: 32px;
  border-radius: 50%;
  border: 1px solid #dcdfe6;
  background: rgba(255,255,255,.95);
  color: #606266;
  font-size: 18px; line-height: 1;
  cursor: pointer;
  z-index: 20;
  transition: all .25s ease;
  box-shadow: 0 2px 6px rgba(0,0,0,.08);
}
.carousel-arrow:hover {
  border-color: #409eff;
  color: #409eff;
  box-shadow: 0 4px 12px rgba(64,158,255,.30);
  transform: translateY(-50%) scale(1.1);
}
.carousel-arrow.left  { left: 0px; }
.carousel-arrow.right { right: 0px; }

.carousel-dots {
  display: flex; justify-content: center; gap: 8px;
  margin-top: 12px;
}
.dot {
  width: 8px; height: 8px; border-radius: 50%;
  background: #dcdfe6;
  cursor: pointer;
  transition: all .3s cubic-bezier(.4,0,.2,1);
}
.dot:hover { background: #b3d8ff; }
.dot.active {
  background: #409eff;
  width: 24px;
  border-radius: 4px;
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
  display: flex; align-items: baseline; gap: 12px;
  margin-bottom: 12px;
  padding: 0 4px;
}
.stocks-title { margin: 0; font-size: 16px; font-weight: 700; color: #303133; }
.stocks-meta  { font-size: 12px; color: #909399; }

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
  border: 1px solid #ebeef5;
  border-radius: 10px;
  padding: 14px;
  background: linear-gradient(180deg, #ffffff 0%, #fafbfc 100%);
  transform-style: preserve-3d;
  will-change: transform, box-shadow;
}
.stock-card::before {
  content: ''; position: absolute; inset: 0;
  border-radius: 10px; pointer-events: none;
  background: radial-gradient(120% 80% at 0% 0%, rgba(64,158,255,.05), transparent 60%);
}

.card-row1 {
  display: flex; align-items: center; gap: 6px;
  font-size: 12px; color: #606266;
  transform: translateZ(6px);
}
.card-row1 .sym { font-weight: 700; color: #303133; }
.card-row1 .name {
  color: #303133; font-weight: 600; max-width: 100px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.card-row1 .market-tag { margin-left: auto; }

.card-row2 {
  display: flex; align-items: baseline; justify-content: space-between;
  margin: 8px 0 8px;
  transform: translateZ(20px);
}
.card-row2 .price { font-size: 24px; font-weight: 800; color: #303133; letter-spacing: -0.5px; }
.card-row2 .pct   { font-size: 14px; font-weight: 700; }
.up   { color: #c0392b; }
.down { color: #27ae60; }

.card-row3, .card-row4 {
  display: flex; justify-content: space-between;
  border-top: 1px dashed #ebeef5; padding-top: 6px;
  font-size: 12px; color: #606266;
  transform: translateZ(4px);
}
.card-row3 { margin-top: 6px; }
.card-row4 { transform: translateZ(10px); border-top-color: #d9ecff; }
.card-row3 .kv, .card-row4 .kv {
  display: flex; flex-direction: column; align-items: flex-start;
}
.card-row3 .kv b, .card-row4 .kv b {
  color: #303133; font-weight: 600; margin-top: 2px;
}

.pager {
  flex: 0 0 auto;
  margin-top: 14px;
  text-align: right;
}
</style>
