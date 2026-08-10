<template>
  <div class="p-4">
    <!-- 我的自定义规则区 -->
    <el-card class="mb-4">
      <h2 class="title">我的选股规则</h2>

      <el-alert type="info" :closable="false" class="mb-3" v-if="lastRunMsg">
        {{ lastRunMsg }}
      </el-alert>

      <el-form :inline="true" :model="newRule" class="rule-form">
        <el-form-item label="规则名">
          <el-input v-model="newRule.rule_name" placeholder="如：连续3天上涨且主力净流入" />
        </el-form-item>
        <el-form-item label="规则表达式 (JSON)">
          <el-input
            v-model="newRule.rule_expressionStr"
            type="textarea"
            :rows="3"
            style="min-width: 480px"
            placeholder='{"change_percent": {"gt": 5}, "industry": {"in": ["银行"]}}'
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="addOrUpdateRule">保存规则</el-button>
          <el-button @click="newRule = blankRule">清空</el-button>
          <el-button type="success" plain @click="$router.push('/presets')">
            使用系统默认策略 →
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 自定义规则列表 -->
    <el-card class="mb-4">
      <el-table :data="rules" stripe style="width: 100%">
        <el-table-column prop="rule_name"        label="规则名" />
        <el-table-column                     label="表达式摘要" width="380">
          <template #default="{ row }">
            <code class="expr">{{ shortExpr(row.rule_expression) }}</code>
          </template>
        </el-table-column>
        <el-table-column                     label="最近命中数" width="120">
          <template #default="{ row }">
            <el-tag v-if="row._hits != null" :type="row._hits ? 'success' : 'info'">
              {{ row._hits ?? '-' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="320" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="editRuleItem(row)">编辑</el-button>
            <el-button size="small" type="primary" :loading="row._running" @click="runRuleNow(row)">执行</el-button>
            <el-button size="small" type="danger"  @click="deleteRuleItem(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 最近一次执行结果：卡片式 -->
    <el-card v-if="lastRunResults.length" class="mt-4">
      <div class="result-header">
        <h3 class="title">最近执行结果：{{ lastRunRuleName }}</h3>
        <div class="result-meta">
          <el-tag type="success">共 {{ lastRunResults.length }} 只命中</el-tag>
          <span class="date-tip">{{ lastRunDate }}</span>
        </div>
      </div>
      <el-row :gutter="12" class="result-grid">
        <el-col
          v-for="s in lastRunResults"
          :key="s.symbol"
          :xs="24" :sm="12" :md="8" :lg="6" :xl="4"
        >
          <div class="stock-card" @click="openChart(s)">
            <div class="card-row1">
              <span class="sym">{{ s.symbol }}</span>
              <span class="name">{{ s.name }}</span>
              <el-tag size="small" effect="plain" class="board">{{ s.industry || '—' }}</el-tag>
            </div>
            <div class="card-row2">
              <span class="price">{{ fmt(s.current_price) }}</span>
              <span :class="['pct', (s.change_percent ?? 0) >= 0 ? 'up' : 'down']">
                {{ (s.change_percent ?? 0) >= 0 ? '+' : '' }}{{ fmt(s.change_percent) }}%
              </span>
            </div>
            <div class="card-row3">
              <div class="kv"><span>净流入</span><b>{{ fmtMoney(s.net_inflow) }}</b></div>
              <div class="kv"><span>换手</span><b>{{ fmt(s.turnover_rate) }}%</b></div>
              <div class="kv"><span>PE</span><b>{{ fmt(s.pe_ttm) }}</b></div>
            </div>
          </div>
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import {
  getRules, addRule, updateRule, deleteRule, runRule,
} from '@/utils/api/rules'
import { ElMessage } from 'element-plus'

const blankRule = () => ({ id: null, rule_name: '', rule_expressionStr: '' })

const rules      = ref([])
const newRule    = ref(blankRule())
const lastRunResults   = ref([])
const lastRunRuleName  = ref('')
const lastRunMsg       = ref('')
const lastRunDate      = ref('')

const fmt = (v) => (v == null || Number.isNaN(Number(v))) ? '-' : Number(v).toFixed(2)
const fmtMoney = (v) => v == null ? '-' : (Number(v) / 1e4).toFixed(2) + ' 万'
const shortExpr = (e) => {
  if (!e) return ''
  const s = JSON.stringify(e)
  return s.length > 80 ? s.slice(0, 80) + '…' : s
}

const openChart = (s) => {
  // 复用全局事件，让 StockChart / 详情页接收
  window.dispatchEvent(new CustomEvent('show-stock', { detail: s }))
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

const addOrUpdateRule = async () => {
  if (!newRule.value.rule_name || !newRule.value.rule_expressionStr) {
    ElMessage.warning('请填写规则名和表达式')
    return
  }
  let expr
  try { expr = JSON.parse(newRule.value.rule_expressionStr) }
  catch (e) { return ElMessage.error('表达式不是合法 JSON: ' + e.message) }

  if (newRule.value.id) {
    await updateRule(newRule.value.id, newRule.value.rule_name, expr)
  } else {
    await addRule(newRule.value.rule_name, expr)
  }
  newRule.value = blankRule()
  await getRules_()
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

const editRuleItem   = (row) => { newRule.value = { id: row.id, rule_name: row.rule_name, rule_expressionStr: row.rule_expressionStr } }
const deleteRuleItem = async (id) => { await deleteRule(id); await getRules_() }

onMounted(getRules_)
</script>

<style scoped>
.title { margin-top: 0; margin-bottom: 12px; }
.rule-form { align-items: flex-start; }
.expr { font-size: 12px; }
.mt-4 { margin-top: 16px; }
.mb-3 { margin-bottom: 12px; }
.mb-4 { margin-bottom: 16px; }
.p-4 { padding: 16px; }

/* 卡片结果区 */
.result-header {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 12px;
}
.result-meta { display: flex; align-items: center; gap: 12px; }
.date-tip   { color: #888; font-size: 12px; }

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
.card-row1 .sym  { font-weight: 700; color: #303133; }
.card-row1 .name { color: #303133; font-weight: 600; max-width: 110px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.card-row1 .board{ margin-left: auto; }

.card-row2 {
  display: flex; align-items: baseline; justify-content: space-between;
  margin: 8px 0 6px;
}
.card-row2 .price { font-size: 22px; font-weight: 700; color: #303133; }
.card-row2 .pct    { font-size: 14px; font-weight: 700; }
.up   { color: #c0392b; }
.down { color: #27ae60; }

.card-row3 {
  display: flex; justify-content: space-between;
  border-top: 1px dashed #ebeef5; padding-top: 6px;
  font-size: 12px; color: #606266;
}
.card-row3 .kv { display: flex; flex-direction: column; align-items: flex-start; }
.card-row3 .kv b { color: #303133; font-weight: 600; margin-top: 2px; }
</style>
