<template>
  <div class="rules-page">
    <!-- 顶部：用户自定义规则卡片网格 -->
    <el-card class="mb-4">
      <div class="header">
        <div>
          <h2 class="title">规则管理</h2>
          <span class="sub">点击「执行」查看该规则的命中股票；卡片右上角删除</span>
        </div>
        <el-button type="primary" @click="openCreate">+ 新增规则</el-button>
      </div>

      <el-row v-if="rules.length" :gutter="12">
        <el-col
          v-for="r in rules"
          :key="r.id"
          :xs="24" :sm="12" :md="8" :lg="6" :xl="6"
        >
          <div class="rule-card">
            <div class="rule-card-head">
              <div class="rule-name">{{ r.rule_name }}</div>
              <el-tag v-if="r._hits != null" :type="r._hits ? 'success' : 'info'" size="small">
                命中 {{ r._hits ?? '-' }}
              </el-tag>
            </div>
            <div class="rule-expr" :title="exprFull(r)">{{ exprFull(r) }}</div>
            <div class="rule-foot">
              <el-button size="small" type="primary" :loading="r._running" @click="runRuleNow(r)">执行</el-button>
              <el-button size="small" type="danger" plain @click="deleteRuleItem(r.id)">删除</el-button>
            </div>
          </div>
        </el-col>
      </el-row>

      <el-empty v-else description="还没有自定义规则，点击右上角「新增规则」开始" />
    </el-card>

    <!-- 命中股票卡片网格（执行后展示） -->
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

      <el-row :gutter="12" class="result-grid">
        <el-col
          v-for="s in lastRunResults"
          :key="s.symbol"
          :xs="24" :sm="12" :md="8" :lg="6" :xl="6"
        >
          <div class="stock-card">
            <div class="card-row1">
              <span class="sym">{{ s.symbol }}</span>
              <span class="name">{{ s.name }}</span>
              <el-tag size="small" effect="plain" class="market-tag">
                {{ boardLabel(s) }}
              </el-tag>
            </div>
            <div class="card-row2">
              <span class="price">{{ fmt(s.current_price ?? s.close) }}</span>
              <span :class="['pct', (s.change_percent ?? 0) >= 0 ? 'up' : 'down']">
                {{ (s.change_percent ?? 0) >= 0 ? '+' : '' }}{{ fmt(s.change_percent) }}%
              </span>
            </div>
            <div class="card-row3">
              <div class="kv"><span>换手</span><b>{{ fmt(s.turnover_rate) }}%</b></div>
              <div class="kv"><span>净流入(万)</span><b>{{ fmtW(s.net_inflow ?? s.net_amount) }}</b></div>
              <div class="kv"><span>PE</span><b>{{ fmt(s.pe_ttm) }}</b></div>
            </div>
          </div>
        </el-col>
      </el-row>
    </el-card>

    <!-- 新增规则对话框 -->
    <el-dialog v-model="createOpen" title="新增规则" width="640">
      <el-form :model="newRule" label-width="100px">
        <el-form-item label="规则名">
          <el-input v-model="newRule.rule_name" placeholder="如：连续3天上涨且主力净流入" />
        </el-form-item>
        <el-form-item label="规则表达式">
          <el-input
            v-model="newRule.rule_expressionStr"
            type="textarea"
            :rows="6"
            placeholder='示例: {"all":[{"type":"field_gt","name":"change_percent","value":5},{"type":"board_in","boards":["创业板"]}],"exclude":[{"type":"is_st"}]}'
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createOpen = false">取消</el-button>
        <el-button type="primary" @click="submitCreate">保存</el-button>
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

const rules      = ref([])
const newRule    = ref({ rule_name: '', rule_expressionStr: '' })
const createOpen = ref(false)
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
  newRule.value = { rule_name: '', rule_expressionStr: '' }
  createOpen.value = true
}

const submitCreate = async () => {
  if (!newRule.value.rule_name || !newRule.value.rule_expressionStr) {
    ElMessage.warning('请填写规则名和表达式')
    return
  }
  let expr
  try { expr = JSON.parse(newRule.value.rule_expressionStr) }
  catch (e) { return ElMessage.error('表达式不是合法 JSON: ' + e.message) }

  await addRule(newRule.value.rule_name, expr)
  createOpen.value = false
  await getRules_()
  ElMessage.success('已保存')
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

/* 规则卡 */
.rule-card {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 12px 14px;
  margin-bottom: 12px;
  background: #fff;
  height: 130px;
  display: flex; flex-direction: column; justify-content: space-between;
  transition: all .15s ease;
}
.rule-card:hover {
  border-color: #c0c4cc;
  box-shadow: 0 2px 8px rgba(0,0,0,.06);
  transform: translateY(-2px);
}
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
}
.rule-foot { display: flex; gap: 6px; justify-content: flex-end; }

/* 命中卡（与 Results.vue 保持一致） */
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

.card-row3 {
  display: flex; justify-content: space-between;
  border-top: 1px dashed #ebeef5; padding-top: 6px;
  font-size: 12px; color: #606266;
}
.card-row3 .kv {
  display: flex; flex-direction: column; align-items: flex-start;
}
.card-row3 .kv b { color: #303133; font-weight: 600; margin-top: 2px; }
</style>
