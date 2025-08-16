<!-- src/pages/StockPage.vue -->
<template>
    <div class="p-4">
      <el-page-header content="股票分析" @back="$router.back()" class="mb-4" />
  
      <div class="grid gap-6">
        <el-card
          v-for="symbol in symbols"
          :key="symbol"
          shadow="never"
        >
          <StockChart :symbol="symbol" />
        </el-card>
      </div>
    </div>
  </template>
  
  <script setup>
  import { useRoute } from 'vue-router'
  import StockChart from '@/components/StockChart.vue'   // 把你现有的逻辑抽出来做成组件
  
  const route = useRoute()
  // 路由参数支持 ?symbols=000001,600519
  const symbols = route.query.symbols ? route.query.symbols.split(',') : []
  </script>
  