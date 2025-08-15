<template>
  <div style="padding: 20px">
    <!-- 搜索栏 -->
    <el-row :gutter="10" style="margin-bottom: 20px;">
      <el-col :span="8">
        <el-input
          v-model="symbol"
          placeholder="输入股票代码（例如 600519）"
          clearable
        />
      </el-col>
      <el-col :span="4">
        <el-button type="primary" @click="fetchData">查询</el-button>
      </el-col>
    </el-row>

    <!-- K线图 -->
    <div ref="chartRef" style="width: 100%; height: 500px;"></div>
  </div>
</template>

<script setup>
import * as echarts from 'echarts'
import axios from 'axios'
import { ref, onMounted } from 'vue'

const symbol = ref('600519') // 默认股票代码
const chartRef = ref(null)
let chartInstance = null

// 计算均线
function calculateMA(dayCount, data) {
  const result = []
  for (let i = 0; i < data.length; i++) {
    if (i < dayCount) {
      result.push('-')
      continue
    }
    let sum = 0
    for (let j = 0; j < dayCount; j++) {
      sum += parseFloat(data[i - j][1]) // 收盘价
    }
    result.push((sum / dayCount).toFixed(2))
  }
  return result
}

const fetchData = async () => {
  if (!symbol.value) return
  const res = await axios.get(`/api/v1/stock-daily-data/${symbol.value}`)
  const sorted = res.data.sort((a, b) => new Date(a.trade_date) - new Date(b.trade_date))

  const dates = sorted.map(d => d.trade_date)
  const kData = sorted.map(d => [d.open, d.close, d.low, d.high])

  const option = {
    title: { text: `股票 ${symbol.value} K线图`, left: 'center' },
    tooltip: { trigger: 'axis' },
    legend: { data: ['日K', 'MA5', 'MA10', 'MA20'] },
    xAxis: { type: 'category', data: dates, boundaryGap: false },
    yAxis: { scale: true },
    series: [
      {
        name: '日K',
        type: 'candlestick',
        data: kData
      },
      {
        name: 'MA5',
        type: 'line',
        data: calculateMA(5, kData),
        smooth: true
      },
      {
        name: 'MA10',
        type: 'line',
        data: calculateMA(10, kData),
        smooth: true
      },
      {
        name: 'MA20',
        type: 'line',
        data: calculateMA(20, kData),
        smooth: true
      }
    ]
  }

  if (!chartInstance) {
    chartInstance = echarts.init(chartRef.value)
  }
  chartInstance.setOption(option)
}

onMounted(() => {
  fetchData()
})
</script>
