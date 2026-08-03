<template>
  <div class="p-4">
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
        </el-form-item>
      </el-form>
    </el-card>

    <el-card>
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

    <el-card v-if="lastRunResults.length" class="mt-4">
      <h3>最近一次执行结果：{{ lastRunRuleName }}</h3>
      <el-table :data="lastRunResults" stripe max-height="400">
        <el-table-column prop="symbol"        label="代码"   width="100" />
        <el-table-column prop="name"          label="名称"   width="160" />
        <el-table-column prop="industry"      label="行业"   width="120" />
        <el-table-column prop="current_price" label="现价"   width="100" />
        <el-table-column prop="change_percent" label="涨幅%"  width="100" />
        <el-table-column prop="net_inflow"     label="净流入" width="120" />
      </el-table>
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

const shortExpr = (e) => {
  if (!e) return ''
  const s = JSON.stringify(e)
  return s.length > 80 ? s.slice(0, 80) + '…' : s
}

const getRules_ = async () => {
  try {
    const res = await getRules()
    rules.value = (res.data || []).map(r => ({
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
    // 拉目标股表
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
.title { margin-top: 0; }
.rule-form { align-items: flex-start; }
.expr { font-size: 12px; }
.mt-4 { margin-top: 16px; }
.mb-3 { margin-bottom: 12px; }
.mb-4 { margin-bottom: 16px; }
.p-4 { padding: 16px; }
</style>
