<template>
  <div class="picker-page">
    <!-- 顶栏：返回 + 标题 -->
    <div class="topbar">
      <button class="back" @click="$router.push('/admin/results')" aria-label="返回">
        ‹ 返回
      </button>
      <h1>🎯 进阶精选 Top {{ topN }}</h1>
      <div class="topbar-right">
        <el-tag type="info" effect="plain">{{ tradeDate || '加载中…' }}</el-tag>
      </div>
    </div>

    <!-- 加载预设 + 候选 -->
    <div v-if="loading" class="loading">加载预设列表…</div>

    <template v-else>
      <!-- 候选预设 -->
      <el-card class="config-card" shadow="never">
        <template #header>
          <div class="config-head">
            <span class="config-title">参与预设（勾选即参与二次精选）</span>
            <div class="config-actions">
              <el-button size="small" @click="selectAll">全选</el-button>
              <el-button size="small" @click="selectAggressive">激进预设</el-button>
              <el-button size="small" type="warning" plain @click="resetToDefault">重置默认</el-button>
              <el-input-number v-model="topN" :min="1" :max="5" size="small" style="width:110px" />
            </div>
          </div>
        </template>
        <div class="preset-grid">
          <div
            v-for="p in presets"
            :key="p.id"
            class="preset-pill"
            :class="{ active: selected.has(p.id), excluded: defaultExcluded.has(p.id) }"
            @click="toggle(p.id)"
          >
            <el-checkbox :model-value="selected.has(p.id)" @click.stop="toggle(p.id)" />
            <div class="pill-body">
              <div class="pill-name">{{ p.name }}</div>
              <div class="pill-meta">
                <span class="pill-id">{{ p.id }}</span>
                <span class="pill-count">{{ counts[p.id] ?? '…' }}</span>
              </div>
            </div>
          </div>
        </div>
        <div class="legend">
          <el-tag size="small" type="warning" effect="plain">橙色</el-tag> = 默认排除（反向 / 高风险 / 几乎无命中）；点击 <strong>激进预设</strong> 按钮一键切换。
        </div>
      </el-card>

      <!-- 操作 -->
      <div class="action-row">
        <el-button
          type="primary"
          size="large"
          :loading="running"
          :disabled="selected.size === 0"
          @click="runFinalPick"
        >
          🎯 开始精选 Top {{ topN }}
        </el-button>
        <span v-if="lastResult" class="action-meta">
          候选池 {{ lastResult.candidates }} 只 · 评分通过 {{ lastResult.scored }} 只 · 命中 {{ lastResult.picks.length }} 只
        </span>
      </div>

      <!-- 结果 -->
      <div v-if="lastResult && lastResult.picks.length > 0" class="picks">
        <div
          v-for="(p, i) in lastResult.picks"
          :key="p.symbol"
          class="pick-card"
          :class="{ gold: i === 0, silver: i === 1 }"
        >
          <div class="pick-medal">{{ i === 0 ? '🥇' : i === 1 ? '🥈' : `#${i + 1}` }}</div>

          <div class="pick-head">
            <div class="pick-name">
              {{ p.name }}
              <span class="pick-symbol">{{ p.symbol }}</span>
            </div>
            <div class="pick-tags">
              <el-tag size="small" :type="marketTagType(p.market)" effect="dark">{{ p.market }}</el-tag>
              <el-tag v-if="p.industry" size="small" effect="plain">{{ p.industry }}</el-tag>
            </div>
          </div>

          <div class="pick-score">
            <span class="score-num">{{ p.score }}</span>
            <span class="score-label">综合分</span>
          </div>

          <!-- 评分明细 -->
          <div class="bd">
            <div class="bd-title">评分明细</div>
            <div class="bd-row">
              <span class="bd-name">板型</span>
              <el-progress :percentage="bdPct(p.breakdown.board, 10)" :stroke-width="8" :show-text="false" :color="barColor(p.breakdown.board, 10)" />
              <span class="bd-val">{{ p.breakdown.board }} / 10</span>
            </div>
            <div class="bd-row">
              <span class="bd-name">技术</span>
              <el-progress :percentage="bdPct(p.breakdown.technical, 30)" :stroke-width="8" :show-text="false" :color="barColor(p.breakdown.technical, 30)" />
              <span class="bd-val">{{ p.breakdown.technical }} / 30</span>
            </div>
            <div class="bd-row">
              <span class="bd-name">动量</span>
              <el-progress :percentage="bdPct(p.breakdown.momentum, 20)" :stroke-width="8" :show-text="false" :color="barColor(p.breakdown.momentum, 20)" />
              <span class="bd-val">{{ p.breakdown.momentum }} / 20</span>
            </div>
            <div class="bd-row">
              <span class="bd-name">量价</span>
              <el-progress :percentage="bdPct(p.breakdown.volume_price, 18)" :stroke-width="8" :show-text="false" :color="barColor(p.breakdown.volume_price, 18)" />
              <span class="bd-val">{{ p.breakdown.volume_price }} / 18</span>
            </div>
            <div class="bd-row">
              <span class="bd-name">成长</span>
              <el-progress :percentage="bdPct(p.breakdown.growth, 12)" :stroke-width="8" :show-text="false" :color="barColor(p.breakdown.growth, 12)" />
              <span class="bd-val">{{ p.breakdown.growth }} / 12</span>
            </div>
            <div class="bd-row">
              <span class="bd-name">资金</span>
              <el-progress :percentage="bdPct(p.breakdown.fund, 10)" :stroke-width="8" :show-text="false" :color="barColor(p.breakdown.fund, 10)" />
              <span class="bd-val">{{ p.breakdown.fund }} / 10</span>
            </div>
            <div v-if="p.breakdown.penalty < 0" class="bd-row penalty">
              <span class="bd-name">减分</span>
              <span class="bd-val">{{ p.breakdown.penalty }}</span>
            </div>
          </div>

          <!-- 关键指标 -->
          <div class="metrics">
            <div class="metric">
              <span class="m-label">收盘</span>
              <span class="m-val">{{ p.close?.toFixed(2) }}</span>
            </div>
            <div class="metric" :class="{ up: p.change_percent > 0, down: p.change_percent < 0 }">
              <span class="m-label">涨幅</span>
              <span class="m-val">{{ p.change_percent?.toFixed(2) }}%</span>
            </div>
            <div class="metric">
              <span class="m-label">换手</span>
              <span class="m-val">{{ p.turnover_rate?.toFixed(2) }}%</span>
            </div>
            <div class="metric">
              <span class="m-label">PE</span>
              <span class="m-val">{{ p.pe_ttm ? p.pe_ttm.toFixed(1) : '—' }}</span>
            </div>
            <div class="metric">
              <span class="m-label">资金净流</span>
              <span class="m-val" :class="{ pos: p.net_amount > 0, neg: p.net_amount < 0 }">
                {{ formatAmount(p.net_amount) }}
              </span>
            </div>
          </div>

          <!-- 命中预设 -->
          <div class="matched">
            <span class="matched-label">命中预设：</span>
            <el-tag
              v-for="mp in p.matched_presets"
              :key="mp"
              size="small"
              effect="plain"
              type="success"
              class="matched-tag"
            >{{ presetName(mp) }}</el-tag>
          </div>
        </div>
      </div>

      <el-empty
        v-else-if="lastResult && lastResult.picks.length === 0"
        description="候选池为空。试着勾选更多预设。"
      />
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { listPresets } from '@/utils/api/presets'
import { runPreset } from '@/utils/api/presets'
import { finalPick } from '@/utils/api/screen'

// === 状态 ===
const loading = ref(true)
const running = ref(false)
const presets = ref([])
const counts = ref({})           // 每条预设的命中数（用于卡片显示）
const selected = ref(new Set())  // 勾选中的预设 id
const topN = ref(2)
const tradeDate = ref('')
const lastResult = ref(null)

// === 默认排除 ===
const defaultExcluded = new Set([
  'ma-death-cross',         // 死叉卖出信号（反向）
  'oversold-bounce',        // 超卖反弹（博弈性质）
  'bottom-reversal',        // 近期 0 命中
])
const aggressiveSet = new Set([
  'ma-trend','volume-price','ma-golden-cross','limit-up-strong',
  'high-position-breakout','tech-bounce','boll-bounce','breakout-5d',
])

// === 加载预设 + 各预设命中数（默认 page=1, page_size=1 拿 total） ===
async function loadPresets() {
  loading.value = true
  try {
    const { data } = await listPresets()
    presets.value = data || []
    tradeDate.value = new Date().toISOString().slice(0, 10)
    // 拉每条预设的命中数
    const tasks = presets.value.map(async (p) => {
      try {
        const r = await runPreset(p.id, 1, 1)
        counts.value[p.id] = r.data?.total ?? 0
      } catch {
        counts.value[p.id] = 0
      }
    })
    await Promise.all(tasks)
    resetToDefault()
  } catch (e) {
    ElMessage.error('加载预设失败：' + (e?.message || e))
  } finally {
    loading.value = false
  }
}

function resetToDefault() {
  selected.value = new Set(presets.value.map((p) => p.id).filter((id) => !defaultExcluded.has(id)))
}

function selectAll() {
  selected.value = new Set(presets.value.map((p) => p.id))
}

function selectAggressive() {
  // 仅选「短线 / 激进 / 双创优先」常用的预设
  selected.value = new Set(aggressiveSet)
}

function toggle(id) {
  const s = new Set(selected.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  selected.value = s
}

function presetName(id) {
  return presets.value.find((p) => p.id === id)?.name ?? id
}

function marketTagType(m) {
  if (m === '创业板' || m === '科创板') return 'danger'
  if (m === '主板') return 'success'
  return 'info'
}

function bdPct(v, max) {
  if (max <= 0) return 0
  return Math.max(0, Math.min(100, Math.round((v / max) * 100)))
}

function barColor(v, max) {
  const pct = bdPct(v, max)
  if (pct >= 75) return '#67c23a'
  if (pct >= 40) return '#409eff'
  return '#909399'
}

function formatAmount(n) {
  if (!n) return '0'
  if (n >= 1e8) return (n / 1e8).toFixed(2) + '亿'
  if (n >= 1e4) return (n / 1e4).toFixed(2) + '万'
  return n.toFixed(0)
}

// === 跑精选 ===
async function runFinalPick() {
  if (selected.value.size === 0) {
    ElMessage.warning('请至少勾选一条预设')
    return
  }
  running.value = true
  lastResult.value = null
  try {
    const { data } = await finalPick([...selected.value], topN.value)
    lastResult.value = data
    tradeDate.value = data.trade_date
    if (data.picks.length === 0) {
      ElMessage.warning(`候选池 ${data.candidates} 只但无股票通过最低分筛选`)
    } else {
      ElMessage.success(`精选出 ${data.picks.length} 只（候选 ${data.candidates} 只）`)
    }
  } catch (e) {
    ElMessage.error('精选失败：' + (e?.message || e))
  } finally {
    running.value = false
  }
}

onMounted(loadPresets)
</script>

<style scoped>
.picker-page {
  max-width: 1280px;
  margin: 0 auto;
  padding: 16px;
}

.topbar {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}
.topbar h1 {
  flex: 1;
  font-size: 22px;
  margin: 0;
  color: #303133;
}
.back {
  background: transparent;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  padding: 6px 12px;
  cursor: pointer;
  font-size: 14px;
}
.back:hover { background: #f5f7fa; }

.loading {
  text-align: center;
  padding: 40px;
  color: #909399;
}

.config-card { margin-bottom: 16px; }
.config-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.config-title { font-weight: 600; }
.config-actions { display: flex; gap: 8px; align-items: center; }

.preset-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 8px;
}
.preset-pill {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
  background: #fff;
}
.preset-pill:hover {
  border-color: #409eff;
  background: #ecf5ff;
}
.preset-pill.active {
  border-color: #409eff;
  background: #ecf5ff;
  box-shadow: 0 0 0 2px rgba(64,158,255,0.15);
}
.preset-pill.excluded {
  background: #fdf6ec;
  border-color: #faecd8;
}
.preset-pill.excluded.active {
  background: #fef0e6;
  border-color: #e6a23c;
}
.pill-body { flex: 1; min-width: 0; }
.pill-name {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.pill-meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
}
.pill-count { font-variant-numeric: tabular-nums; }
.legend {
  margin-top: 12px;
  font-size: 12px;
  color: #909399;
}

.action-row {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}
.action-meta { color: #909399; font-size: 14px; }

.picks {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 16px;
}

.pick-card {
  position: relative;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.04);
}
.pick-card.gold {
  background: linear-gradient(135deg, #fff8e1 0%, #fff 60%);
  border-color: #f0c14b;
}
.pick-card.silver {
  background: linear-gradient(135deg, #f5f5f5 0%, #fff 60%);
  border-color: #c0c4cc;
}

.pick-medal {
  position: absolute;
  top: -12px;
  left: 16px;
  font-size: 28px;
  filter: drop-shadow(0 2px 4px rgba(0,0,0,0.2));
}

.pick-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  margin-top: 4px;
}
.pick-name {
  font-size: 18px;
  font-weight: 700;
  color: #303133;
}
.pick-symbol {
  font-size: 12px;
  color: #909399;
  margin-left: 6px;
  font-weight: normal;
}
.pick-tags { display: flex; gap: 6px; flex-wrap: wrap; }

.pick-score {
  text-align: center;
  margin: 16px 0;
}
.score-num {
  font-size: 48px;
  font-weight: 800;
  color: #f56c6c;
  letter-spacing: -2px;
  font-variant-numeric: tabular-nums;
}
.score-label {
  display: block;
  font-size: 12px;
  color: #909399;
  margin-top: -4px;
}

.bd {
  background: #fafbfc;
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 12px;
}
.bd-title {
  font-size: 12px;
  color: #909399;
  margin-bottom: 8px;
}
.bd-row {
  display: grid;
  grid-template-columns: 50px 1fr 60px;
  align-items: center;
  gap: 8px;
  margin: 4px 0;
  font-size: 12px;
}
.bd-name { color: #606266; }
.bd-val { text-align: right; color: #303133; font-variant-numeric: tabular-nums; }
.bd-row.penalty .bd-val { color: #f56c6c; font-weight: 600; }

.metrics {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-bottom: 12px;
}
.metric {
  background: #f5f7fa;
  border-radius: 6px;
  padding: 6px 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.m-label { font-size: 11px; color: #909399; }
.m-val { font-size: 14px; color: #303133; font-weight: 600; font-variant-numeric: tabular-nums; }
.m-val.pos { color: #f56c6c; }
.m-val.neg { color: #67c23a; }
.metric.up .m-val { color: #f56c6c; }
.metric.down .m-val { color: #67c23a; }

.matched {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  font-size: 12px;
}
.matched-label { color: #909399; }
.matched-tag { font-size: 11px !important; }
</style>
