<template>
  <el-card>
    <div class="header">
      <h2>热点股票</h2>
      <div>
        <el-select v-model="market" placeholder="全部市场" clearable @change="loadData" style="width:140px">
          <el-option label="全部" value="" />
          <el-option label="主板" value="主板" />
          <el-option label="创业板" value="创业板" />
          <el-option label="科创板" value="科创板" />
        </el-select>
        <el-input v-model="keyword" placeholder="代码或名称" clearable style="width: 200px; margin-left: 8px" @keyup.enter.native="loadData" />
        <el-button type="primary" @click="loadData" style="margin-left: 8px">刷新</el-button>
      </div>
    </div>

    <h3 v-if="tab === 'spot'" class="sub">全市场涨幅榜（默认涨幅 ≥ 5%）</h3>
    <h3 v-else class="sub">规则选股候选（{{ ruleName || '全部' }}）</h3>

    <el-tabs v-model="tab" @tab-change="onTabChange">
      <el-tab-pane label="实时热门" name="spot" />
      <el-tab-pane label="规则候选" name="target" />
    </el-tabs>

    <el-table :data="rows" stripe :row-class-name="rowClass" @row-click="showDetail">
      <el-table-column prop="symbol"  label="代码"   width="100" />
      <el-table-column prop="name"    label="名称"   width="160" />
      <el-table-column prop="industry"      label="行业"  width="120" />
      <el-table-column prop="market"        label="市场"  width="100" />
      <el-table-column prop="current_price" label="现价"  width="120">
        <template #default="{ row }">{{ fmt(row.current_price ?? row.close) }}</template>
      </el-table-column>
      <el-table-column prop="change_percent" label="涨幅%" width="120">
        <template #default="{ row }">
          <span :class="row.change_percent >= 0 ? 'up' : 'down'">
            {{ fmt(row.change_percent) }}%
          </span>
        </template>
      </el-table-column>
      <el-table-column prop="turnover_rate" label="换手率%" width="100">
        <template #default="{ row }">{{ fmt(row.turnover_rate) }}</template>
      </el-table-column>
      <el-table-column prop="net_inflow" label="主力净流入" width="140">
        <template #default="{ row }">{{ fmtMoney(row.net_inflow ?? row.net_amount) }}</template>
      </el-table-column>
      <el-table-column v-if="tab === 'target'" prop="rule_name" label="命中规则" width="180" />
      <el-table-column label="收藏" width="80">
        <template #default="{ row }">
          <el-button size="small" link @click.stop="toggleFavorite(row.symbol)">
            {{ isFavorite(row.symbol) ? '★' : '☆' }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager" v-if="tab === 'spot'">
      <el-pagination
        :current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next, ->, total"
        @current-change="p => { page = p; loadData() }"
      />
    </div>

    <el-dialog v-model="showDetailDialog" width="500" :title="selected?.name || '详情'">
      <div v-if="selected">
        <p><b>代码:</b> {{ selected.symbol }}</p>
        <p><b>行业:</b> {{ selected.industry }}</p>
        <p><b>现价:</b> {{ fmt(selected.current_price ?? selected.close) }}</p>
        <p><b>涨幅:</b>
          <span :class="selected.change_percent >= 0 ? 'up' : 'down'">
            {{ fmt(selected.change_percent) }}%
          </span>
        </p>
        <p v-if="selected.rule_name"><b>命中规则:</b> {{ selected.rule_name }}</p>
      </div>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { fetchHotStocks } from '@/utils/api/stocks'
import request from '@/utils/request'
import { getFavorites, addFavorite, removeFavorite } from '@/utils/api/favorites'

const tab = ref('spot')
const rows = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const market = ref('')
const keyword = ref('')
const showDetailDialog = ref(false)
const selected = ref(null)

const favorites = ref([])
const refreshFavorites = async () => {
  const res = await getFavorites()
  favorites.value = Array.isArray(res.data?.data) ? res.data.data : []
}
const isFavorite = (s) => favorites.value.some(f => f.symbol === s)
const toggleFavorite = async (s) => {
  isFavorite(s) ? await removeFavorite(s) : await addFavorite(s)
  await refreshFavorites()
}

const fmt = (v) => (v == null || Number.isNaN(Number(v))) ? '-' : Number(v).toFixed(2)
const fmtMoney = (v) => v == null ? '-' : (Number(v) / 1e4).toFixed(2) + ' 万'

const loadData = async () => {
  try {
    if (tab.value === 'spot') {
      const res = await fetchHotStocks({ page: page.value, page_size: pageSize.value })
      rows.value = (res.data?.data || []).filter(s =>
        (!market.value || s.market === market.value) &&
        (!keyword.value || s.symbol.includes(keyword.value) || s.name?.includes(keyword.value))
      )
      total.value = res.data?.total || rows.value.length
    } else {
      const res = await request.get('/target-stocks', { params: { rule_name: ruleName.value } })
      rows.value = res.data?.data || []
    }
  } catch (e) { console.error(e) }
}

const onTabChange = (name) => {
  page.value = 1
  loadData()
}

const ruleName = ref('')

// 监听来自 Rules 页"执行后刷新"
onMounted(async () => {
  await refreshFavorites()
  await loadData()
  window.addEventListener('select-rule', (e) => {
    ruleName.value = e.detail?.rule_name || ''
    tab.value = 'target'
    loadData()
  })
})

const showDetail = (row) => { selected.value = row; showDetailDialog.value = true }
const rowClass   = ({ row }) => (row.change_percent >= 5 ? 'highlight-row' : '')
</script>

<style scoped>
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.sub { margin: 0 0 8px; font-size: 14px; color: #888; }
.pager { margin-top: 12px; text-align: right; }
.up { color: #d33; font-weight: 600; }
.down { color: #1a9; font-weight: 600; }
.highlight-row { background: #fff7e6; }
h2 { margin: 0; }
</style>
