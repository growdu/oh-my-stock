<template>
  <div class="presets-page">
    <!-- 顶部 8 个默认策略卡片 -->
    <el-card class="mb-4">
      <div class="header">
        <h2 class="title">系统默认策略</h2>
        <span class="sub">点击下方任一预设卡片，查看当日命中股票</span>
      </div>

      <el-row :gutter="12">
        <el-col
          v-for="p in presets"
          :key="p.id"
          :xs="24" :sm="12" :md="8" :lg="6" :xl="6"
        >
          <div
            class="preset-card"
            :class="{ active: selectedId === p.id }"
            @click="selectPreset(p)"
          >
            <div class="pcard-name">{{ p.name }}</div>
            <div class="pcard-desc">{{ p.description }}</div>
            <div class="pcard-foot">
              <el-tag size="small" effect="plain">{{ boardSummary(p) }}</el-tag>
              <span class="pcard-id">{{ p.id }}</span>
            </div>
          </div>
        </el-col>
      </el-row>
    </el-card>

    <!-- 命中股票卡片网格 -->
    <el-card v-if="selected">
      <div class="result-header">
        <div>
          <h3 class="title-inline">「{{ selected.name }}」命中结果</h3>
          <span class="meta-tip">{{ selected.description }}</span>
        </div>
        <div class="result-meta">
          <el-tag type="success">共 {{ total }} 只</el-tag>
          <span class="date-tip">{{ tradeDate }}</span>
          <el-button size="small" @click="reload">刷新</el-button>
        </div>
      </div>

      <el-skeleton v-if="loading" :rows="4" animated />

      <el-empty v-else-if="!rows.length" description="当前没有命中股票" />

      <el-row v-else :gutter="12" class="result-grid">
        <el-col
          v-for="s in rows"
          :key="s.symbol"
          :xs="24" :sm="12" :md="8" :lg="6" :xl="6"
        >
          <div class="stock-card" @click="openChart(s)">
            <div class="card-row1">
              <span class="sym">{{ s.symbol }}</span>
              <span class="name" :title="s.name">{{ s.name }}</span>
              <el-tag size="small" effect="plain" class="market-tag">
                {{ boardLabel(s) }}
              </el-tag>
            </div>
            <div class="card-row2">
              <span class="price">{{ fmt(s.close) }}</span>
              <span :class="['pct', (s.change_percent ?? 0) >= 0 ? 'up' : 'down']">
                {{ (s.change_percent ?? 0) >= 0 ? '+' : '' }}{{ fmt(s.change_percent) }}%
              </span>
            </div>
            <div class="card-row3">
              <div class="kv"><span>开</span><b>{{ fmt(s.open) }}</b></div>
              <div class="kv"><span>高</span><b>{{ fmt(s.high) }}</b></div>
              <div class="kv"><span>低</span><b>{{ fmt(s.low) }}</b></div>
            </div>
            <div class="card-row4">
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

const presets    = ref([])
const selected   = ref(null)
const selectedId = ref('')
const rows       = ref([])
const total      = ref(0)
const page       = ref(1)
const pageSize   = ref(12)
const loading    = ref(false)
const tradeDate  = ref('')

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

const openChart = (s) => {
  window.dispatchEvent(new CustomEvent('show-stock', { detail: s }))
}

const loadPresets = async () => {
  try {
    const res = await listPresets()
    presets.value = res.data?.data || []
    if (presets.value.length) {
      selectPreset(presets.value[0])
    }
  } catch (e) {
    ElMessage.error('获取默认策略失败')
  }
}

const selectPreset = async (p) => {
  selectedId.value = p.id
  selected.value  = p
  page.value      = 1
  await fetchRows()
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

const onPage  = (p) => { page.value = p; fetchRows() }
const reload  = ()    => fetchRows()

onMounted(loadPresets)
</script>

<style scoped>
.presets-page { padding: 16px; }
.mb-4 { margin-bottom: 16px; }
.header {
  display: flex; align-items: baseline; gap: 12px; margin-bottom: 12px;
}
.title { margin: 0; }
.sub  { color: #909399; font-size: 12px; }

/* 预设卡 */
.preset-card {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 12px 14px;
  margin-bottom: 12px;
  cursor: pointer;
  background: #fff;
  transition: all .15s ease;
  height: 110px;
  display: flex; flex-direction: column; justify-content: space-between;
}
.preset-card:hover {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(64,158,255,.12);
  transform: translateY(-2px);
}
.preset-card.active {
  border-color: #409eff;
  background: linear-gradient(135deg, #ecf5ff 0%, #ffffff 100%);
  box-shadow: 0 2px 12px rgba(64,158,255,.18);
}
.pcard-name { font-size: 15px; font-weight: 700; color: #303133; }
.pcard-desc { font-size: 12px; color: #606266; line-height: 1.5; margin: 6px 0; }
.pcard-foot { display: flex; justify-content: space-between; align-items: center; }
.pcard-id   { color: #c0c4cc; font-size: 11px; font-family: monospace; }

/* 命中卡 */
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
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 12px;
  background: #fff;
  cursor: pointer;
  transition: transform .12s ease, box-shadow .12s ease, border-color .12s ease;
}
.stock-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 14px rgba(0,0,0,.08);
  border-color: #c0c4cc;
}
.card-row1 {
  display: flex; align-items: center; gap: 6px;
  font-size: 12px; color: #606266;
}
.card-row1 .sym { font-weight: 700; color: #303133; }
.card-row1 .name {
  color: #303133; font-weight: 600; max-width: 100px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.card-row1 .market-tag { margin-left: auto; }

.card-row2 {
  display: flex; align-items: baseline; justify-content: space-between;
  margin: 8px 0 6px;
}
.card-row2 .price { font-size: 22px; font-weight: 700; color: #303133; }
.card-row2 .pct   { font-size: 14px; font-weight: 700; }
.up   { color: #c0392b; }
.down { color: #27ae60; }

.card-row3, .card-row4 {
  display: flex; justify-content: space-between;
  border-top: 1px dashed #ebeef5; padding-top: 6px;
  font-size: 12px; color: #606266;
}
.card-row3 { margin-top: 6px; }
.card-row3 .kv, .card-row4 .kv {
  display: flex; flex-direction: column; align-items: flex-start;
}
.card-row3 .kv b, .card-row4 .kv b {
  color: #303133; font-weight: 600; margin-top: 2px;
}

.pager { margin-top: 14px; text-align: right; }
</style>
