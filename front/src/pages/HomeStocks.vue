<template>
  <div>
    <DataTable
      v-bind="tableProps"
      :columns="columns"
    />
  </div>
</template>

<script setup>
import DataTable from '@/components/DataTable.vue'
import { usePaginatedData } from '@/composables/usePaginatedData'
import { fetchStockList } from '@/utils/api/stocks'

// ✅ 使用通用 Hook 绑定股票数据
const {
  data,
  total,
  loading,
  query,
  page,
  pageSize,
} = usePaginatedData(fetchStockList)

// ✅ 定义表格列
const columns = [
  { prop: 'symbol', label: '股票代码', width: 120 },
  { prop: 'name', label: '股票名称', width: 160 },
  { prop: 'price', label: '当前价格', width: 120 },
  { prop: 'change_percent', label: '涨跌幅(%)', width: 120 },
]

// ✅ 传给 DataTable 的属性
const tableProps = {
  data,
  total,
  loading,
  query,
  page,
  pageSize,
}
</script>
