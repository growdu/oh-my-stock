<template>
    <el-card class="module-card">
      <h3 class="module-title mb-3">收藏股票</h3>
      <el-table :data="favorites" stripe border style="width:100%">
        <el-table-column prop="symbol" label="代码" width="100"/>
        <el-table-column prop="name" label="名称" width="150"/>
        <el-table-column prop="price" label="当前价" width="120"/>
        <el-table-column prop="change" label="涨跌幅" width="120">
          <template #default="scope">
            <span :class="scope.row.change>0?'up':'down'">{{ scope.row.change }}%</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </template>
  
  <script setup>
  import { ref } from 'vue'
  
  const favorites = ref([
    { symbol: '300001', name: '股票A', price: 10.5, change: 2.5 },
  ])
  </script>
  
  <style scoped>
  .module-card { padding: 20px; border-radius: 12px; background-color: #fff; box-shadow: 0 4px 12px rgba(0,0,0,0.05); margin-bottom: 20px; }
  .module-title { font-size: 18px; font-weight: bold; }
  .up { color: #f56c6c; }
  .down { color: #67c23a; }
  </style>
  