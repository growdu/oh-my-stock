<template>
  <div class="rules-page">
    <!-- 顶部：用户自定义规则卡片网格（3D 翻牌） -->
    <el-card class="mb-4">
      <div class="header">
        <div>
          <h2 class="title">规则管理</h2>
          <span class="sub">点击「执行」查看该规则的命中股票；卡片悬停可翻转看完整表达式</span>
        </div>
        <el-button type="primary" @click="openCreate">+ 新增规则</el-button>
      </div>

      <el-row v-if="rules.length" :gutter="12" class="cards-stage">
        <el-col
          v-for="r in rules"
          :key="r.id"
          :xs="24" :sm="12" :md="8" :lg="6" :xl="6"
        >
          <div class="rule-card">
            <!-- 正面 -->
            <div class="card-face card-front">
              <div class="rule-card-head">
                <div class="rule-name">{{ r.rule_name }}</div>
                <el-tag v-if="r._hits != null" :type="r._hits ? 'success' : 'info'" size="small">
                  命中 {{ r._hits ?? '-' }}
                </el-tag>
              </div>
              <div class="rule-expr" :title="exprFull(r)">{{ exprFull(r) }}</div>
              <div class="rule-foot">
                <el-button size="small" type="primary" :loading="r._running" @click.stop="runRuleNow(r)">执行</el-button>
                <el-button size="small" type="danger" plain @click.stop="deleteRuleItem(r.id)">删除</el-button>
              </div>
              <div class="flip-hint">悬停翻转 ↻</div>
            </div>
            <!-- 背面：完整表达式 JSON -->
            <div class="card-face card-back">
              <div class="back-title">📜 规则表达式</div>
              <pre class="rule-expr-full">{{ exprPretty(r) }}</pre>
              <div class="rule-foot">
                <el-button size="small" type="primary" :loading="r._running" @click.stop="runRuleNow(r)">执行</el-button>
                <el-button size="small" type="danger" plain @click.stop="deleteRuleItem(r.id)">删除</el-button>
              </div>
            </div>
          </div>
        </el-col>
      </el-row>

      <el-empty v-else description="还没有自定义规则，点击右上角「新增规则」开始" />
    </el-card>

    <!-- 命中股票卡片网格（3D 鼠标跟随倾斜） -->
    <el-card v-if="lastRunResults.length">
      <div class="result-header">
        <div>
          <h3 class="title-inline">「{{ lastRunRuleName }}」命中结果</h3>
          <span class="meta-tip" v-if="lastRunMsg">{{ lastRunMsg }}</span>
        </div>
        <div class="result-meta">
          <el-tag type="success">共 {{ lastRunResults.length }} 只命中</el-tag>
          <span class="date-tip">{{ lastRunDate }}</span>
        </div>
      </div>

      <el-row :gutter="12" class="result-grid cards-stage">
        <el-col
          v-for="s in lastRunResults"
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
              <span class="name">{{ s.name }}</span>
              <el-tag size="small" effect="plain" class="market-tag">
                {{ boardLabel(s) }}
              </el-tag>
            </div>
            <div class="card-row2 layer-2">
              <span class="price">{{ fmt(s.current_price ?? s.close) }}</span>
              <span :class="['pct', (s.change_percent ?? 0) >= 0 ? 'up' : 'down']">
                {{ (s.change_percent ?? 0) >= 0 ? '+' : '' }}{{ fmt(s.change_percent) }}%
              </span>
            </div>
            <div class="card-row3 layer-3">
              <div class="kv"><span>换手</span><b>{{ fmt(s.turnover_rate) }}%</b></div>
              <div class="kv"><span>净流入(万)</span><b>{{ fmtW(s.net_inflow ?? s.net_amount) }}</b></div>
              <div class="kv"><span>PE</span><b>{{ fmt(s.pe_ttm) }}</b></div>
            </div>
            <div class="card-row4 layer-4 tech-strip">
              <span class="ts-chip" :class="maPosClass(s, 'ma5')">
                MA5 {{ fmt(s.ma5) }}
                <em v-if="maCross(s, 'ma5', 'ma10')" class="ts-flag gold">金叉</em>
                <em v-else-if="maDeathCross(s, 'ma5', 'ma10')" class="ts-flag death">死叉</em>
              </span>
              <span class="ts-chip" :class="maPosClass(s, 'ma10')">MA10 {{ fmt(s.ma10) }}</span>
              <span class="ts-chip" :class="maPosClass(s, 'ma20')">MA20 {{ fmt(s.ma20) }}</span>
              <span class="ts-chip" :class="kdjClass(s)">KDJ {{ fmt(s.k) }}/{{ fmt(s.d) }}</span>
              <span class="ts-chip" :class="rsiClass(s)">RSI6 {{ fmt(s.rsi6) }}</span>
              <span class="ts-chip" :class="bollClass(s)">BOLL {{ fmt(bollPctB(s) * 100) }}%</span>
            </div>
          </div>
        </el-col>
      </el-row>
    </el-card>

    <!-- 新增规则对话框 -->
    <el-dialog v-model="createOpen" title="新增规则" width="1100" top="5vh" :close-on-click-modal="false">
      <el-form :model="newRule" label-width="80px">
        <el-form-item label="规则名">
          <el-input v-model="newRule.rule_name" placeholder="如：连续3天上涨且主力净流入" style="max-width: 360px" />
        </el-form-item>
        <el-form-item label="规则条件">
          <RuleEditor v-model="newRule.expression" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createOpen = false">取消</el-button>
        <el-button type="primary" @click="submitCreate" :loading="submitting">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import {
  getRules, addRule, deleteRule, runRule,
} from '@/utils/api/rules'
import { ElMessage } from 'element-plus'
import { useCard3D } from '@/composables/useCard3D'
import RuleEditor from '@/components/RuleEditor.vue'

const { onTiltMove, onTiltLeave } = useCard3D({ max: 9 })

const rules      = ref([])
const newRule    = ref({ rule_name: '', expression: { all: [], exclude: [{ type: 'is_st' }] } })
const createOpen = ref(false)
const submitting = ref(false)
const lastRunResults   = ref([])
const lastRunRuleName  = ref('')
const lastRunMsg       = ref('')
const lastRunDate      = ref('')

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

const exprFull = (r) => {
  if (!r.rule_expression) return '{}'
  const s = JSON.stringify(r.rule_expression)
  return s.length > 200 ? s.slice(0, 200) + '…' : s
}

const exprPretty = (r) => {
  if (!r.rule_expression) return '{}'
  try { return JSON.stringify(r.rule_expression, null, 2) }
  catch { return String(r.rule_expression) }
}

const getRules_ = async () => {
  try {
    const res = await getRules()
    rules.value = (res.data?.data || []).map(r => ({
      ...r,
      rule_expressionStr: r.rule_expression ? JSON.stringify(r.rule_expression) : '{}',
    }))
  } catch (err) { ElMessage.error('获取规则失败') }
}

const openCreate = () => {
  newRule.value = { rule_name: '', expression: { all: [], exclude: [{ type: 'is_st' }] } }
  createOpen.value = true
}

const submitCreate = async () => {
  if (!newRule.value.rule_name) {
    ElMessage.warning('请填写规则名')
    return
  }
  const expr = newRule.value.expression || {}
  const allCount  = (expr.all  || []).length
  const anyCount  = (expr.any  || []).length
  const exclCount = (expr.exclude || []).length
  if (allCount === 0 && anyCount === 0 && exclCount === 0) {
    return ElMessage.warning('请至少添加一条规则条件')
  }
  submitting.value = true
  try {
    await addRule(newRule.value.rule_name, expr)
    createOpen.value = false
    await getRules_()
    ElMessage.success('已保存')
  } catch (e) {
    ElMessage.error('保存失败: ' + (e.response?.data?.message || e.message || ''))
  } finally {
    submitting.value = false
  }
}

const runRuleNow = async (row) => {
  row._running = true
  try {
    const res = await runRule(row.id)
    row._hits = res.matched
    lastRunRuleName.value = row.rule_name
    lastRunMsg.value = `规则「${row.rule_name}」命中 ${res.matched} 只股票 @ ${res.date}`
    lastRunDate.value  = res.date || ''
    const target = await fetch(`/api/v1/target-stocks?rule_name=${encodeURIComponent(row.rule_name)}`,
      { headers: { Authorization: 'Bearer ' + localStorage.getItem('token') } })
    const targetJson = await target.json()
    lastRunResults.value = targetJson.data || []
    ElMessage.success('执行完成')
  } catch (e) {
    ElMessage.error('执行失败: ' + (e.message || ''))
  } finally {
    row._running = false
  }
}

const deleteRuleItem = async (id) => { await deleteRule(id); await getRules_() }

// === 技术指标辅助函数（与 Display.vue 保持一致） ===
const hasValue = (v) => v != null && Math.abs(v) > 1e-9
const maPosClass = (s, key) => {
  const ma = s[key]
  if (!hasValue(ma) || s.current_price == null) return ''
  return s.current_price >= ma ? 'up' : 'down'
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
const rsiClass = (s) => {
  const r = s.rsi6
  if (!hasValue(r)) return ''
  if (r >= 80) return 'overbought'
  if (r <= 30) return 'oversold'
  return ''
}
const bollPctB = (s) => {
  if (!hasValue(s.boll_upper) || !hasValue(s.boll_lower) || s.current_price == null) return null
  const range = s.boll_upper - s.boll_lower
  if (range <= 0) return null
  return Math.max(0, Math.min(1, (s.current_price - s.boll_lower) / range))
}
const bollClass = (s) => {
  const p = bollPctB(s)
  if (p == null) return ''
  if (p >= 1) return 'overbought'
  if (p <= 0) return 'oversold'
  return p >= 0.5 ? 'up' : 'down'
}

onMounted(getRules_)
</script>

<style scoped>
.rules-page { padding: 16px; }
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
.rule-card {
  position: relative;
  height: 170px;
  margin-bottom: 14px;
  transform-style: preserve-3d;
  transition: transform .45s cubic-bezier(.2,.7,.3,1);
}
.rule-card:hover { transform: translateY(-4px); }

.rule-card .card-face {
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
.rule-card .card-front { transform: rotateY(0deg); }
.rule-card .card-back {
  transform: rotateY(180deg);
  background: linear-gradient(135deg, #fdf6ec 0%, #ffffff 75%);
  border-color: #e6a23c;
  box-shadow: 0 6px 18px rgba(230,162,60,.18);
}
.rule-card:hover .card-front { transform: rotateY(-180deg); }
.rule-card:hover .card-back  { transform: rotateY(0deg); }

.rule-card-head {
  display: flex; align-items: center; justify-content: space-between;
}
.rule-name {
  font-size: 14px; font-weight: 700; color: #303133;
  max-width: 170px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.rule-expr {
  font-size: 11px; color: #909399; font-family: monospace;
  background: #f5f7fa; padding: 6px 8px; border-radius: 4px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  margin-top: 6px;
}
.rule-expr-full {
  font-size: 10px; color: #606266; font-family: monospace;
  background: #fdf6ec; padding: 6px 8px; border-radius: 4px;
  margin: 0;
  flex: 1; overflow: auto;
  white-space: pre-wrap; word-break: break-all;
  line-height: 1.4;
}
.rule-foot { display: flex; gap: 6px; justify-content: flex-end; margin-top: auto; }
.flip-hint {
  position: absolute; bottom: 4px; right: 8px;
  font-size: 10px; color: #c0c4cc;
  font-family: monospace;
}

.back-title {
  font-size: 13px; font-weight: 700; color: #303133;
  margin-bottom: 6px;
}

/* ============ 命中股票卡（3D 倾斜 + 分层） ============ */
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

.card-row3 {
  display: flex; justify-content: space-between;
  border-top: 1px dashed #ebeef5; padding-top: 6px;
  font-size: 12px; color: #606266;
  transform: translateZ(8px);
  margin-top: 6px;
}
.card-row3 .kv {
  display: flex; flex-direction: column; align-items: flex-start;
}
.card-row3 .kv b { color: #303133; font-weight: 600; margin-top: 2px; }

/* ============ 技术状态条 ============ */
.card-row4 {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 5px;
  margin-top: 8px;
  padding-top: 6px;
  border-top: 1px dashed #ebeef5;
  transform: translateZ(4px);
}
.tech-strip { /* 共用 class，做空样式占位，避免 scoped 冲突 */ }
.ts-chip {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 1px 5px;
  font-size: 10.5px;
  font-weight: 600;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  color: #475569;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 4px;
  white-space: nowrap;
  line-height: 1.5;
}
.ts-chip.up          { color: #c0392b; background: #fff5f5; border-color: #ffd0d0; }
.ts-chip.down        { color: #27ae60; background: #f6fbf6; border-color: #bbf7d0; }
.ts-chip.overbought  { color: #c2410c; background: #fff7ed; border-color: #fdba74; }
.ts-chip.oversold    { color: #0369a1; background: #f0f9ff; border-color: #bae6fd; }
.ts-flag {
  font-style: normal;
  font-size: 9px;
  font-weight: 800;
  padding: 0 3px;
  border-radius: 2px;
  margin-left: 2px;
}
.ts-flag.gold  { color: #fff; background: #c0392b; }
.ts-flag.death { color: #fff; background: #27ae60; }
</style>
