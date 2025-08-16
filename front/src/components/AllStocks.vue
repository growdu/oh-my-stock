<template>
    <el-card class="module-card">
      <div class="flex justify-between items-center mb-3">
        <h3 class="module-title">股票列表</h3>
        <el-input v-model="filter" placeholder="搜索股票" clearable style="max-width: 250px"/>
      </div>
      <el-table :data="filteredStocks" stripe border style="width:100%">
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
  import { ref, computed } from 'vue'
  
  const filter = ref('')
  const stocks = ref([
    { symbol: '300001', name: '股票A', price: 10.5, change: 2.5 },
    { symbol: '300002', name: '股票B', price: 8.3, change: -1.2 },
    { symbol: '300003', name: '股票C', price: 15.7, change: 0.8 },
  ])
  const filteredStocks = computed(() => {
    if (!filter.value) return stocks.value
    return stocks.value.filter(
      s => s.name.includes(filter.value) || s.symbol.includes(filter.value)
    )
  })
  </script>
  
  <style scoped>
  .module-card { padding: 20px; border-radius: 12px; background-color: #fff; box-shadow: 0 4px 12px rgba(0,0,0,0.05); margin-bottom: 20px; }
  .module-title { font-size: 18px; font-weight: bold; }
  .up { color: #f56c6c; }
  .down { color: #67c23a; }
  </style>
  