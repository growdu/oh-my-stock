<template>
    <div class="stock-charts p-4">
      <!-- 搜索框（仅在独立模式显示） -->
      <el-card v-if="standalone" shadow="never" class="mb-4">
        <div class="flex flex-wrap items-center gap-3">
          <el-autocomplete
            v-model.trim="querySymbol"
            :fetch-suggestions="querySearch"
            placeholder="输入股票代码或名称"
            style="max-width: 240px"
            clearable
            @select="handleSelect"
            @keyup.enter.native="fetchStock"
          />
          <el-button type="primary" :loading="loading" @click="fetchStock">查询</el-button>
          <el-select v-model="range" placeholder="区间" style="width: 140px">
            <el-option :value="60" label="近60日" />
            <el-option :value="120" label="近120日" />
            <el-option :value="240" label="近240日" />
            <el-option :value="0" label="全部" />
          </el-select>
          <el-switch v-model="showMoneyFlow" active-text="显示资金流" />
        </div>
      </el-card>
  
      <!-- 多股票标签页 -->
      <el-tabs v-if="multiMode" v-model="activeStock" type="card" @tab-remove="removeStock">
        <el-tab-pane
          v-for="stock in stocksData"
          :key="stock.meta.symbol"
          :label="`${stock.meta.name}(${stock.meta.symbol})`"
          :name="stock.meta.symbol"
          closable
        >
          <stock-chart :data="stock" :range="range" :show-money-flow="showMoneyFlow" />
        </el-tab-pane>
      </el-tabs>
  
      <!-- 单股票图表 -->
      <stock-chart
        v-else
        :data="currentStock"
        :range="range"
        :show-money-flow="showMoneyFlow"
      />
  
      <el-empty v-if="!loading && (!stocksData || stocksData.length === 0)" description="暂无数据" class="mt-8" />
    </div>
  </template>
  
  <script setup>
  import { ref, computed, watch, onMounted } from 'vue'
  import axios from 'axios'
  import debounce from 'lodash/debounce'
  import StockChart from './StockChart.vue'
  
  const props = defineProps({
    // 初始股票代码/名称
    symbol: {
      type: [String, Array],
      default: ''
    },
    // 是否显示搜索框（独立模式）
    standalone: {
      type: Boolean,
      default: true
    },
    // 默认显示区间
    defaultRange: {
      type: Number,
      default: 120
    },
    // 是否显示资金流
    defaultShowMoneyFlow: {
      type: Boolean,
      default: true
    }
  })
  
  const API_BASE = import.meta.env.VITE_API_BASE || 'http://192.168.3.99:3003'
  
  const querySymbol = ref('')
  const loading = ref(false)
  const range = ref(props.defaultRange)
  const showMoneyFlow = ref(props.defaultShowMoneyFlow)
  const stocksData = ref([])
  const activeStock = ref('')
  
  // 计算属性
  const multiMode = computed(() => Array.isArray(props.symbol))
  const currentStock = computed(() => stocksData.value[0] || null)
  
  // 初始化
  onMounted(() => {
    if (props.symbol) {
      if (Array.isArray(props.symbol)) {
        // 多股票模式
        props.symbol.forEach(s => addStock(s))
      } else {
        // 单股票模式
        querySymbol.value = props.symbol
        fetchStock()
      }
    }
  })
  
  // 添加股票
  async function addStock(symbol) {
    if (!symbol) return
    if (stocksData.value.some(s => s.meta.symbol === symbol)) return
    
    loading.value = true
    try {
      const url = `${API_BASE}/api/v1/stocks/history?symbol=${encodeURIComponent(symbol)}`
      const { data } = await axios.get(url)
      
      const stockData = {
        meta: { 
          name: data.name, 
          symbol: data.symbol, 
          market: data.market, 
          industry: data.industry, 
          listing_date: data.listing_date 
        },
        rows: Array.isArray(data.daily_data) ? data.daily_data : []
      }
      
      stocksData.value.push(stockData)
      activeStock.value = symbol
    } catch (e) {
      console.error(e)
      ElMessage.error(`获取股票 ${symbol} 数据失败`)
    } finally {
      loading.value = false
    }
  }
  
  // 移除股票
  function removeStock(symbol) {
    const index = stocksData.value.findIndex(s => s.meta.symbol === symbol)
    if (index !== -1) {
      stocksData.value.splice(index, 1)
    }
  }
  
  // 查询股票
  async function fetchStock() {
    if (!querySymbol.value) return
    await addStock(querySymbol.value)
    querySymbol.value = ''
  }
  
  // 自动补全
  const doSearch = async (queryString, cb) => {
    if (!queryString) {
      cb([])
      return
    }
    try {
      const { data } = await axios.get(
        `${API_BASE}/api/v1/stocks/search?q=${encodeURIComponent(queryString)}`
      )
      cb(
        data.map(item => ({
          value: `${item.symbol} ${item.name}`,
          symbol: item.symbol,
          name: item.name
        }))
      )
    } catch (e) {
      console.error(e)
      cb([])
    }
  }
  
  const querySearch = debounce(doSearch, 300)
  
  const handleSelect = (item) => {
    querySymbol.value = item.symbol
    fetchStock()
  }
  </script>
  
  <style scoped>
  .stock-charts {
    .mb-4 { margin-bottom: 1rem; }
    .mt-8 { margin-top: 2rem; }
  }
  </style>