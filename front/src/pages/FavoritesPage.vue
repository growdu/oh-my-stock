<template>
  <Header></Header>
    <el-card>
      <h2>我的收藏</h2>
      <el-table :data="favorites" stripe style="width:100%">
        <el-table-column prop="symbol" label="股票代码" width="120"/>
        <el-table-column prop="name" label="股票名称" width="150"/>
        <el-table-column prop="change_percent" label="涨跌幅" width="100">
          <template #default="{ row }">
            <span :style="{ color: row.change_percent>=0 ? 'red':'green' }">
              {{ row.change_percent.toFixed(2) }}%
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button type="danger" size="small" @click="removeFavoriteItem(row.symbol)">取消收藏</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </template>
  
  <script setup>
  import { ref, onMounted } from 'vue'
  import { getFavorites, removeFavorite } from '@/utils/api/favorites'
import Header from '../components/Header.vue'
  
  const favorites = ref([])
  
  const refreshFavorites = async ()=>{
    const res = await getFavorites()
    favorites.value = Array.isArray(res.data)?res.data:[]
  }
  
  const removeFavoriteItem = async (symbol)=>{
    await removeFavorite(symbol)
    await refreshFavorites()
  }
  
  onMounted(async ()=>{
    await refreshFavorites()
  })
  </script>
  
  <style scoped>
  h2 { margin-bottom: 12px; }
  </style>
  