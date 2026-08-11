<template>
  <div class="results-page">
    <!-- 顶部：8 个系统规则卡片网格（3D 翻牌） -->
    <el-card class="mb-4">
      <div class="header">
        <div>
          <h2 class="title">选股结果</h2>
          <span class="sub">选择下方任一系统规则，查看当日命中股票（卡片悬停可翻转看规则条件）</span>
        </div>
        <el-button size="small" @click="fetchRows" :loading="loading">刷新</el-button>
      </div>

      <el-row :gutter="12" class="cards-stage">
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
            <!-- 正面 -->
            <div class="card-face card-front">
              <div class="pcard-name">{{ p.name }}</div>
              <div class="pcard-desc">{{ p.description }}</div>
              <div class="pcard-foot">
                <el-tag size="small" effect="plain">{{ boardSummary(p) }}</el-tag>
                <span class="pcard-id">{{ p.id }}</span>
              </div>
              <div class="flip-hint">悬停翻转 ↻</div>
            </div>
            <!-- 背面：规则条件 -->
            <div class="card-face card-back">
              <div class="back-title">📜 规则条件</div>
              <ul class="cond-list">
                <li v-for="(c, i) in (p.expression?.all || []).slice(0, 6)" :key="i">
                  <span class="ctype">{{ c.type }}</span>
                  <span class="cname" v-if="c.name">{{ c.name }} {{ c.value ?? '' }}</span>
                  <span class="cname" v-if="c.boards">{{ c.boards.join('/') }}</span>
                  <span class="cname" v-if="c.days">{{ c.days }}日</span>
                </li>
              </ul>
              <div v-if="(p.expression?.exclude || []).length" class="back-exclude">
                排除：{{ (p.expression.exclude || []).map(e => e.type).join('、') }}
              </div>
            </div>
          </div>
        </el-col>
      </el-row>
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

const presets    = ref([])
const selected   = ref(null)
const selectedId = ref('')
const rows       = ref([])
const total      = ref(0)
const page       = ref(1)
const pageSize   = ref(8)
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

const loadPresets = async () => {
  try {
    const res = await listPresets()
    presets.value = res.data?.data || []
    if (presets.value.length) {
      selectPreset(presets.value[0])
    }
  } catch (e) {
    ElMessage.error('获取系统规则失败')
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

/* ============ 3D 场景：父级建立透视 ============ */
.cards-stage { perspective: 1200px; }

/* ============ 规则卡（3D 翻牌） ============ */
.preset-card {
  position: relative;
  height: 150px;
  margin-bottom: 14px;
  cursor: pointer;
  transform-style: preserve-3d;
  transition: transform .45s cubic-bezier(.2,.7,.3,1);
}
.preset-card:hover { transform: translateY(-4px); }

.preset-card .card-face {
  position: absolute;
  inset: 0;
  border-radius: 10px;
  padding: 12px 14px;
  border: 1px solid #ebeef5;
  background: #fff;
  box-shadow: 0 2px 8px rgba(0,0,0,.04);
  -webkit-backface-visibility: hidden;
  backface-visibility: hidden;
  display: flex;
  flex-direction: column;
  transition: transform .65s cubic-bezier(.4,.0,.2,1), box-shadow .35s;
}
.preset-card .card-front { transform: rotateY(0deg); }
.preset-card .card-back {
  transform: rotateY(180deg);
  background: linear-gradient(135deg, #ecf5ff 0%, #ffffff 75%);
  border-color: #409eff;
  box-shadow: 0 6px 18px rgba(64,158,255,.18);
}
.preset-card:hover .card-front { transform: rotateY(-180deg); }
.preset-card:hover .card-back  { transform: rotateY(0deg); }

.preset-card.active {
  border-color: #409eff;
}
.preset-card.active .card-front {
  background: linear-gradient(135deg, #d6eaff 0%, #ffffff 70%);
  border-color: #409eff;
  box-shadow: 0 6px 18px rgba(64,158,255,.22);
}

.pcard-name { font-size: 15px; font-weight: 700; color: #303133; }
.pcard-desc {
  font-size: 12px; color: #606266; line-height: 1.5;
  margin: 6px 0; flex: 1;
  display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical;
  overflow: hidden;
}
.pcard-foot { display: flex; justify-content: space-between; align-items: center; }
.pcard-id   { color: #c0c4cc; font-size: 11px; font-family: monospace; }
.flip-hint  {
  position: absolute; bottom: 4px; right: 8px;
  font-size: 10px; color: #c0c4cc;
  font-family: monospace;
}

.back-title {
  font-size: 13px; font-weight: 700; color: #303133;
  margin-bottom: 6px;
}
.cond-list {
  list-style: none; padding: 0; margin: 0;
  display: flex; flex-direction: column; gap: 4px;
  flex: 1; overflow: hidden;
}
.cond-list li {
  display: flex; gap: 6px; align-items: center;
  font-size: 11px; color: #606266;
  line-height: 1.3;
}
.cond-list .ctype {
  background: #409eff; color: #fff;
  padding: 1px 6px; border-radius: 4px;
  font-family: monospace; font-size: 10px;
  white-space: nowrap;
}
.cond-list .cname {
  font-family: monospace; color: #303133;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.back-exclude {
  font-size: 10px; color: #909399;
  border-top: 1px dashed #d9ecff; padding-top: 4px; margin-top: 4px;
}

/* ============ 命中股票卡（3D 倾斜 + 分层景深） ============ */
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
  /* transform 由 JS 注入（鼠标跟随） */
}
.stock-card::before {
  /* 卡片"光照"高光，左上到右下的内阴影 */
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
  transform: translateZ(20px);  /* 价格这一行最靠前 */
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
