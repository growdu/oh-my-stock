<template>
  <el-card>
    <div class="table-header">
      <el-input v-model="searchQuery" placeholder="搜索股票代码或名称" size="small" clearable @keyup.enter.native="loadStocks" style="width:200px"/>
      <el-select v-model="pageSize" placeholder="每页显示" size="small" @change="loadStocks">
        <el-option v-for="n in [10,20,50,100]" :key="n" :label="n+'条'" :value="n"/>
      </el-select>
    </div>

    <el-table :data="stocks" stripe :row-class-name="rowClassName" style="width:100%" @row-click="showStockDetail">
      <el-table-column prop="symbol" label="代码" width="100"/>
      <el-table-column prop="name" label="名称" width="150"/>
      <el-table-column prop="open" label="开盘" width="100"/>
      <el-table-column prop="close" label="收盘" width="100"/>
      <el-table-column prop="change_percent" label="涨跌幅" width="100">
        <template #default="{ row }">
          <span :style="{ color: row.change_percent>=0?'red':'green' }">{{ row.change_percent.toFixed(2) }}%</span>
        </template>
      </el-table-column>
      <el-table-column prop="trade_date" label="日期" width="120">
        <template #default="{ row }">{{ row.trade_date.slice(0,10) }}</template>
      </el-table-column>
      <el-table-column label="收藏" width="100">
        <template #default="{ row }">
          <el-button size="small" type="text" @click.stop="toggleFavorite(row.symbol)">
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

    <!-- 股票详情弹窗 -->
    <el-dialog :visible.sync="showDetailDialog" width="600px" :title="selectedStock?.name || '股票详情'">
      <div v-if="selectedStock">
        <p><strong>代码:</strong> {{ selectedStock.symbol }}</p>
        <p><strong>名称:</strong> {{ selectedStock.name }}</p>
        <p><strong>开盘:</strong> {{ selectedStock.open }}</p>
        <p><strong>收盘:</strong> {{ selectedStock.close }}</p>
        <p><strong>涨跌幅:</strong> <span :style="{ color:selectedStock.change_percent>=0?'red':'green' }">{{ selectedStock.change_percent.toFixed(2) }}%</span></p>
        <p><strong>日期:</strong> {{ selectedStock.trade_date.slice(0,10) }}</p>
      </div>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getStocks } from '@/utils/api/stocks'
import { getFavorites, addFavorite, removeFavorite } from '@/utils/api/favorites'

const stocks = ref([])
const favorites = ref([])
const rules = ref([]) // 所有选股规则
const selectedRule = ref(null) // 当前应用规则

const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const searchQuery = ref('')

const selectedStock = ref(null)
const showDetailDialog = ref(false)

const loadStocks = async ()=>{
  try{
    const res = await getStocks(page.value,pageSize.value,searchQuery.value)
    stocks.value = Array.isArray(res.data)?res.data.sort((a,b)=>b.change_percent-a.change_percent):[]
    total.value = res.total||0
    if(selectedRule.value) applyRule(selectedRule.value)
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

const showStockDetail = (row)=>{ selectedStock.value=row; showDetailDialog.value=true }

// 高亮规则
const applyRule = (rule)=>{
  stocks.value.forEach(stock=>{
    stock.highlight=false
    try{
      const expr = rule.rule_expression||{}
      let match=true
      for(const key in expr){
        const condition=expr[key]
        if('gt' in condition && stock[key]<=condition.gt) match=false
        if('lt' in condition && stock[key]>=condition.lt) match=false
      }
      if(match) stock.highlight=true
    }catch(err){ console.error(err) }
  })
}

const rowClassName = ({ row })=> row.highlight?'highlight-row':''

onMounted(async ()=>{
  await refreshFavorites()
  await loadStocks()
  // 监听选股规则选择
  window.addEventListener('select-rule', (e)=>{
    selectedRule.value = e.detail
    applyRule(selectedRule.value)
  })
})
</script>

<style scoped>
.highlight-row { background-color:#fff7e6; }
.table-header { display:flex; justify-content:space-between; margin-bottom:10px; align-items:center; }
.pagination { margin-top:12px; text-align:right; }
</style>
