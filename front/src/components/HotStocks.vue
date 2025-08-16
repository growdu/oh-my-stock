<template>
    <el-row :gutter="20">
      <el-col :span="6" v-for="stock in stocks" :key="stock.symbol">
        <el-card class="hot-stock-card">
          <div class="hot-stock-name">{{ stock.name }}</div>
          <div class="hot-stock-symbol">{{ stock.symbol }}</div>
          <div :class="stock.change>0?'up':'down'" class="hot-stock-change">{{ stock.change }}%</div>
        </el-card>
      </el-col>
    </el-row>
  </template>
  
  <script setup>
  import { ref } from 'vue'
  
  const stocks = ref([
    { symbol: '300001', name: '股票A', change: 2.5 },
    { symbol: '300003', name: '股票C', change: 0.8 },
  ])
  </script>
  
  <style scoped>
  .hot-stock-card {
    padding: 15px; border-radius: 10px; background: linear-gradient(135deg, #e0f7fa, #b2ebf2); text-align: center;
    box-shadow: 0 4px 8px rgba(0,0,0,0.05); margin-bottom: 20px;
  }
  .hot-stock-name { font-weight: bold; font-size: 16px; }
  .hot-stock-symbol { margin-top: 5px; color: #555; }
  .hot-stock-change { margin-top: 10px; font-size: 16px; font-weight: bold; }
  .up { color: #f56c6c; }
  .down { color: #67c23a; }
  </style>
  