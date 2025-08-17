<template>
  <Header></Header>
  <el-card>
    <div class="table-header">
      <el-input v-model="searchQuery" placeholder="搜索股票代码或名称" size="small" clearable @keyup.enter.native="loadStocks" style="width:200px"/>
      <el-select v-model="pageSize" placeholder="每页显示" size="small" @change="loadStocks">
        <el-option v-for="n in [10,20,50,100]" :key="n" :label="n+'条'" :value="n"/>
      </el-select>
    </div>

    <el-table :data="stocks" stripe :row-class-name="rowClassName" style="width:100%">
      <el-table-column prop="symbol" label="代码" width="100"/>
      <el-table-column prop="name" label="名称" width="150"/>
      <el-table-column prop="open" label="开盘" width="100"/>
      <el-table-column prop="close" label="收盘" width="100"/>
      <el-table-column prop="change_percent" label="涨跌幅" width="100">
        <template #default="{ row }">
          <span :style="{ color: row.change_percent >= 0 ? 'red' : 'green' }">
            {{ row.change_percent.toFixed(2) }}%
          </span>
        </template>
      </el-table-column>
      <el-table-column prop="trade_date" label="日期" width="120">
        <template #default="{ row }">{{ row.trade_date.slice(0,10) }}</template>
      </el-table-column>
      <el-table-column label="收藏" width="100">
        <template #default="{ row }">
          <el-button size="small" type="text" @click="toggleFavorite(row.symbol)">
            {{ isFavorite(row.symbol)?'★':'☆' }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination">
      <el-pagination
        :current-page="page"
        :page-size="pageSize"
        :total="total"
        @current-change="handlePageChange"
        layout="prev, pager, next, ->, total"
      />
    </div>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getStocks } from '@/utils/api/stocks'
import { getFavorites, addFavorite, removeFavorite } from '@/utils/api/favorites'
import Header from '../components/Header.vue'

const stocks = ref([])
const favorites = ref([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const searchQuery = ref('')

const loadStocks = async ()=>{
  try{
    const res = await getStocks(page.value,pageSize.value,searchQuery.value)
    stocks.value = Array.isArray(res.data)?res.data.sort((a,b)=>b.change_percent-a.change_percent):[]
    total.value = res.total||0
  }catch(err){ console.error(err) }
}

const handlePageChange = (newPage)=>{ page.value=newPage; loadStocks() }

const refreshFavorites = async ()=>{
  const res = await getFavorites()
  favorites.value = Array.isArray(res.data)?res.data:[]
}

const toggleFavorite = async (symbol)=>{
  if(favorites.value.some(f=>f.symbol===symbol)) await removeFavorite(symbol)
  else await addFavorite(symbol)
  await refreshFavorites()
}

const isFavorite = (symbol)=> favorites.value.some(f=>f.symbol===symbol)

const rowClassName = ({ row })=> row.change_percent>=5?'highlight-row':''

onMounted(async ()=>{
  await refreshFavorites()
  await loadStocks()
})
</script>

<style scoped>
.highlight-row { background-color:#fff7e6; }
.table-header { display:flex; justify-content:space-between; margin-bottom:10px; align-items:center; }
.pagination { margin-top:12px; text-align:right; }
</style>
