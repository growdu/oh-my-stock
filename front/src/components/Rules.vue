<template>
  <Header></Header>
    <el-card class="module-card">
      <h3 class="module-title mb-3">个人规则配置</h3>
      <el-form :model="rules" label-width="120px">
        <el-form-item label="关注涨幅">
          <el-input-number v-model="rules.priceChange" :min="0" :max="50"/>
        </el-form-item>
        <el-form-item label="关注跌幅">
          <el-input-number v-model="rules.priceDrop" :min="0" :max="50"/>
        </el-form-item>
        <el-form-item label="通知方式">
          <el-select v-model="rules.notify" placeholder="请选择">
            <el-option label="邮件" value="email"/>
            <el-option label="消息" value="message"/>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="saveRules">保存配置</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </template>
  
  <script setup>
  import { ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import Header from './Header.vue'
  
  const rules = ref({ priceChange: 5, priceDrop: 5, notify: 'email' })
  const saveRules = () => {
    console.log('保存规则:', rules.value)
    ElMessage.success('个人规则已保存')
  }
  </script>
  
  <style scoped>
  .module-card { padding: 20px; border-radius: 12px; background-color: #fff; box-shadow: 0 4px 12px rgba(0,0,0,0.05); margin-bottom: 20px; }
  .module-title { font-size: 18px; font-weight: bold; }
  </style>
  