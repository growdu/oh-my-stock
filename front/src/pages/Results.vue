<template>
  <div class="results-page">
    <!-- 顶部：8 个系统规则 3D 轮播 -->
    <el-card class="mb-4">
      <div class="header">
        <div>
          <h2 class="title">选股结果</h2>
          <span class="sub">点击下方卡片切换规则，箭头 / 圆点也可切换；当前居中的规则生效</span>
        </div>
        <el-button size="small" @click="fetchRows" :loading="loading">刷新</el-button>
      </div>

      <div class="carousel">
        <button class="carousel-arrow left" @click="prev" aria-label="上一条">‹</button>

        <div class="carousel-track">
          <div
            v-for="(p, i) in presets"
            :key="p.id"
            class="carousel-item"
            :class="{ active: i === selectedIndex }"
            :style="cardStyle(i)"
            @click="selectByIndex(i)"
          >
            <div class="pcard-head">
              <span class="pcard-name">{{ p.name }}</span>
              <el-tag size="small" effect="plain" class="board-tag">{{ boardSummary(p) }}</el-tag>
            </div>
            <div class="pcard-desc">{{ p.description }}</div>
            <div class="pcard-foot">
              <span class="pcard-id">{{ p.id }}</span>
              <span class="pcard-hint">点击生效 ▸</span>
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
    </el-card>

    <!-- 命中股票卡片网格（3D 鼠标跟随倾斜 + 分层） -->
    <el-card v-if="selected">
      <div class="result-header">
        <div>
          <h3 class="title-inline">「{{ selected.name }}」命中结果</h3>
          <span class="meta-tip">{{ selected.description }}</span>
        </div>
        <div class="result-meta">
          <el-tag type="success">共 {{ total }} 只</el-tag>
          <span class="date-tip">{{ tradeDate }}</span>
        </div>
      </div>

      <el-skeleton v-if="loading" :rows="4" animated />

      <el-empty v-else-if="!rows.length" description="当前没有命中股票，可点击「拉取最新数据」填充缓存后再试" />

      <el-row v-else :gutter="12" class="result-grid cards-stage">
        <el-col
          v-for="s in rows"
          :key="s.symbol"
          :xs="24" :sm="12" :md="8" :lg="6" :xl="6"
        >
          <div
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
        </el-col>
      </el-row>

      <div class="pager" v-if="rows.length">
        <el-pagination
          :current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="prev, pager, next, ->, total"
          @current-change="onPage"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { listPresets, runPreset } from '@/utils/api/presets'
import { ElMessage } from 'element-plus'
import { useCard3D } from '@/composables/useCard3D'

const { onTiltMove, onTiltLeave } = useCard3D({ max: 9 })

const presets      = ref([])
const selected     = ref(null)
const selectedId   = ref('')
const selectedIndex = ref(0)
const rows         = ref([])
const total        = ref(0)
const page         = ref(1)
const pageSize     = ref(8)
const loading      = ref(false)
const tradeDate    = ref('')

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

const wrapDelta = (d, total) => {
  if (d >  total / 2) return d - total
  if (d < -total / 2) return d + total
  return d
}

// Apple Coverflow 风格：35° 倾斜 + 100px 纵深 + 145px 横向扇开 + 0.88^ad 指数缩放
const cardStyle = (i) => {
  const total = presets.value.length
  if (!total) return {}
  const d  = wrapDelta(i - selectedIndex.value, total)
  const ad = Math.abs(d)
  const angle = d * 35
  const tz    = -ad * 100
  const tx    = d * 145
  const scale = Math.pow(0.88, ad)
  const opacity = ad > 3 ? 0 : Math.max(0, 1 - ad * 0.32)
  return {
    transform: `rotateY(${angle}deg) translateZ(${tz}px) translateX(${tx}px) scale(${scale})`,
    opacity,
    zIndex: 100 - ad,
    pointerEvents: ad > 2 ? 'none' : 'auto',
  }
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

const loadPresets = async () => {
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

onMounted(loadPresets)
</script>

<style scoped>
.results-page { padding: 16px; }
.mb-4 { margin-bottom: 16px; }
.header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 12px;
}
.title { margin: 0; }
.sub  { color: #909399; font-size: 12px; margin-left: 8px; }

/* ============ Apple Coverflow 3D 轮播 ============ */
.carousel {
  position: relative;
  perspective: 1200px;
  perspective-origin: 50% 45%;
  height: 320px;          /* 容纳镜面倒影 */
  margin: 4px auto 8px;
}
/* 弧形地板阴影 */
.carousel::before {
  content: '';
  position: absolute;
  bottom: 8px;
  left: 50%;
  transform: translateX(-50%);
  width: 70%;
  height: 36px;
  background: radial-gradient(ellipse at center, rgba(0,0,0,.18) 0%, rgba(0,0,0,0) 70%);
  filter: blur(10px);
  z-index: 1;
  pointer-events: none;
}
.carousel-track {
  position: relative;
  width: 100%;
  height: 100%;
  transform-style: preserve-3d;
}
.carousel-item {
  position: absolute;
  top: 30px;
  left: 50%;
  width: 280px;
  height: 220px;
  margin-left: -140px;
  padding: 16px 18px;
  border-radius: 14px;
  background: linear-gradient(180deg, #ffffff 0%, #f5f7fa 100%);
  border: 1px solid rgba(0,0,0,.05);
  box-shadow:
    0 1px 1px rgba(0,0,0,.06),
    0 6px 12px rgba(0,0,0,.08),
    0 14px 36px rgba(0,0,0,.12);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  /* 镜面倒影（webkit） */
  -webkit-box-reflect: below 14px
    linear-gradient(transparent 50%, rgba(255,255,255,.35) 78%, rgba(0,0,0,.18));
  transition:
    transform .75s cubic-bezier(.22, .8, .35, 1.05),
    opacity .55s ease,
    box-shadow .35s ease,
    border-color .35s ease;
}
.carousel-item:hover {
  box-shadow:
    0 1px 1px rgba(0,0,0,.06),
    0 8px 16px rgba(0,0,0,.10),
    0 18px 44px rgba(64,158,255,.22);
}
.carousel-item.active {
  border-color: rgba(64,158,255,.45);
  background: linear-gradient(180deg, #ffffff 0%, #eaf4ff 100%);
  box-shadow:
    0 1px 1px rgba(0,0,0,.06),
    0 14px 28px rgba(64,158,255,.28),
    0 28px 60px rgba(64,158,255,.35);
}

.pcard-head {
  display: flex; align-items: center; justify-content: space-between;
  gap: 8px;
}
.pcard-name {
  font-size: 17px; font-weight: 700; color: #303133;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  max-width: 180px;
  letter-spacing: -0.2px;
}
.pcard-desc {
  font-size: 12.5px; color: #606266; line-height: 1.55;
  margin: 10px 0; flex: 1;
  display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical;
  overflow: hidden;
}
.pcard-foot {
  display: flex; justify-content: space-between; align-items: center;
  border-top: 1px dashed #ebeef5; padding-top: 6px;
}
.pcard-id   { color: #c0c4cc; font-size: 11px; font-family: monospace; }
.pcard-hint { font-size: 11px; color: #409eff; opacity: 0; transition: opacity .25s; }
.carousel-item.active .pcard-hint { opacity: 1; }

/* 左右箭头（Coverflow 玻璃态） */
.carousel-arrow {
  position: absolute;
  top: 50%; transform: translateY(-50%);
  width: 40px; height: 40px;
  border-radius: 50%;
  border: 1px solid #dcdfe6;
  background: rgba(255,255,255,.92);
  color: #606266;
  font-size: 24px; line-height: 1;
  cursor: pointer;
  z-index: 50;
  transition: all .25s ease;
  backdrop-filter: blur(8px);
  box-shadow: 0 2px 8px rgba(0,0,0,.08);
}
.carousel-arrow:hover {
  border-color: #409eff;
  color: #409eff;
  box-shadow: 0 4px 14px rgba(64,158,255,.30);
  transform: translateY(-50%) scale(1.08);
}
.carousel-arrow.left  { left: 12px; }
.carousel-arrow.right { right: 12px; }

.carousel-dots {
  display: flex; justify-content: center; gap: 8px;
  margin-top: 14px;
  position: relative;
  z-index: 5;
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

/* ============ 命中股票卡 3D 倾斜 ============ */
.cards-stage { perspective: 1200px; }
.result-header {
  display: flex; justify-content: space-between; align-items: flex-start;
  margin-bottom: 12px;
}
.title-inline { margin: 0 0 4px; }
.meta-tip    { color: #909399; font-size: 12px; }
.result-meta { display: flex; align-items: center; gap: 12px; }
.date-tip    { color: #888; font-size: 12px; }

.result-grid { margin-top: 4px; }
.stock-card {
  position: relative;
  border: 1px solid #ebeef5;
  border-radius: 10px;
  padding: 14px;
  margin-bottom: 14px;
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

.pager { margin-top: 14px; text-align: right; }
</style>
