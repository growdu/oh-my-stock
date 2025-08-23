<template>
  <el-card>
    <el-form :model="newRule" inline style="margin-bottom:12px">
      <el-form-item label="规则名">
        <el-input v-model="newRule.rule_name" placeholder="规则名称"/>
      </el-form-item>
      <el-form-item label="表达式">
        <el-input v-model="newRule.rule_expressionStr" placeholder='{"change_percent":{"gt":5}}'/>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" size="small" @click="addOrUpdateRule">保存规则</el-button>
      </el-form-item>
    </el-form>

    <el-table :data="rules" stripe style="width:100%">
      <el-table-column prop="rule_name" label="规则名"/>
      <el-table-column prop="rule_expressionStr" label="表达式" width="760"/>
      <el-table-column label="操作" width="240">
      <template #default="{ row }">
          <el-button size="small" type="primary" @click="editRuleItem(row)">编辑</el-button>
          <el-button size="small" type="success" @click="applyRule(row)">应用</el-button>
          <el-button size="small" type="danger" @click="deleteRuleItem(row.id)">删除</el-button>
      </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getRules as fetchRules, addRule, updateRule, deleteRule } from '@/utils/api/rules'
import { ElMessage } from 'element-plus'

const rules = ref([])
const newRule = ref({ id:null, rule_name:'', rule_expressionStr:'' })

const getRules = async ()=>{
  try{
    const res = await fetchRules()
    rules.value = Array.isArray(res)?res.map(r=>({
      ...r,
      rule_expressionStr: r.rule_expression ? JSON.stringify(r.rule_expression) : '{}'
    })):[]
  }catch(err){ ElMessage.error('获取规则失败') }
}

const addOrUpdateRule = async ()=>{
  if(!newRule.value.rule_name || !newRule.value.rule_expressionStr) return
  try{
    const expr = JSON.parse(newRule.value.rule_expressionStr)
    if(newRule.value.id){
      await updateRule(newRule.value.id,newRule.value.rule_name,expr)
    }else{
      await addRule(newRule.value.rule_name,expr)
    }
    newRule.value = {id:null, rule_name:'', rule_expressionStr:''}
    await getRules()
  }catch(err){ console.error(err) }
}

const applyRule = (rule)=>{
  window.dispatchEvent(new CustomEvent('select-rule', { detail: rule }))
  }

const editRuleItem = (rule)=>{ newRule.value={id:rule.id,rule_name:rule.rule_name,rule_expressionStr:rule.rule_expressionStr} }
const deleteRuleItem = async (id)=>{ await deleteRule(id); await getRules() }

onMounted(async ()=>{ await getRules() })
</script>

<style scoped>
h2 { margin-bottom:12px; }
</style>
