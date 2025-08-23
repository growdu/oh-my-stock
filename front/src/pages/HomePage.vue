<template>
  <el-card>
    <!-- 顶部搜索区 -->
    <div class="table-header">
      <!-- ✅ 自动补全 + 回车搜索 -->
      <el-autocomplete
        v-model="searchQuery"
        :fetch-suggestions="querySearch"
        placeholder="输入股票代码或名称"
        size="small"
        clearable
        @select="handleSearchSelect"
        @keyup.enter.native="handleSearchEnter"
        style="width:240px"
      />
    </div>

    <!-- 表格 -->
    <el-table
      :data="stocks"
      stripe
      :row-class-name="rowClassName"
      style="width:100%"
    >
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
            {{ isFavorite(row.symbol) ? '★' : '☆' }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- ✅ 分页区域：左右布局 -->
    <div class="pagination">
      <div class="pagination-left">
        <el-pagination
          :current-page="page"
          :page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="sizes"
          @size-change="handlePageSizeChange"
        />
      </div>
      <div class="pagination-right">
        <el-pagination
          :current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="prev, pager, next, ->, total"
          @current-change="handlePageChange"
        />
      </div>
    </div>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getStocks, searchStocks } from '@/utils/api/stocks'
import { getFavorites, addFavorite, removeFavorite } from '@/utils/api/favorites'

const stocks = ref([])
const favorites = ref([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const searchQuery = ref('')

// ✅ 模糊搜索接口
const querySearch = async (query, cb) => {
  if (!query) return cb([])
  try {
    const res = await searchStocks(query)
    cb(res.data.map(item => ({
      value: `${item.symbol} ${item.name}`,
      symbol: item.symbol
    })))
  } catch (err) {
    console.error(err)
    cb([])
  }
}

// 选中候选项时
const handleSearchSelect = (item) => {
  searchQuery.value = item.symbol
  page.value = 1
  loadStocks()
}

// ✅ 输入回车直接搜索
const handleSearchEnter = () => {
  page.value = 1
  loadStocks()
}

const loadStocks = async () => {
  try {
    const res = await getStocks(page.value, pageSize.value, searchQuery.value)
    stocks.value = Array.isArray(res.data)
      ? res.data.sort((a, b) => b.change_percent - a.change_percent)
      : []
    total.value = res.total || 0
  } catch (err) {
    console.error(err)
  }
}

const handlePageChange = (newPage) => {
  page.value = newPage
  loadStocks()
}

const handlePageSizeChange = (newSize) => {
  pageSize.value = newSize
  page.value = 1
  loadStocks()
}

const refreshFavorites = async () => {
  const res = await getFavorites()
  favorites.value = Array.isArray(res.data) ? res.data : []
}

const toggleFavorite = async (symbol) => {
  if (favorites.value.some(f => f.symbol === symbol)) {
    await removeFavorite(symbol)
  } else {
    await addFavorite(symbol)
  }
  await refreshFavorites()
}

const isFavorite = (symbol) => favorites.value.some(f => f.symbol === symbol)

const rowClassName = ({ row }) => row.change_percent >= 5 ? 'highlight-row' : ''

onMounted(async () => {
  await refreshFavorites()
  await loadStocks()
})
</script>

<style scoped>
.highlight-row {
  background-color:#fff7e6;
}
.table-header {
  display:flex;
  justify-content:flex-start;
  margin-bottom:10px;
  align-items:center;
}
.pagination {
  display:flex;
  justify-content:space-between;
  align-items:center;
  margin-top:12px;
}
</style>
