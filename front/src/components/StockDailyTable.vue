<template>
  <div class="stock-container">
    <h1>{{ symbol }} 股票日线数据</h1>

    <div ref="chartRef" style="width: 100%; height: 500px;"></div>

    <table border="1" cellspacing="0" cellpadding="5">
      <thead>
        <tr>
          <th>日期</th>
          <th>开盘</th>
          <th>最高</th>
          <th>最低</th>
          <th>收盘</th>
          <th>成交量</th>
          <th>MA5</th>
          <th>MA10</th>
          <th>MACD</th>
          <th>BOLL(中轨)</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in tableData" :key="row.trade_date">
          <td>{{ row.trade_date }}</td>
          <td>{{ row.open }}</td>
          <td>{{ row.high }}</td>
          <td>{{ row.low }}</td>
          <td>{{ row.close }}</td>
          <td>{{ row.volume }}</td>
          <td>{{ row.ma5 }}</td>
          <td>{{ row.ma10 }}</td>
          <td>{{ row.macd }}</td>
          <td>{{ row.boll_mid }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { ref, onMounted } from "vue";
import * as echarts from "echarts";
import axios from "axios";

const chartRef = ref(null);
const symbol = "600519"; // 示例股票
const tableData = ref([]);

const calcMA = (dayCount, data) => {
  let result = [];
  for (let i = 0; i < data.length; i++) {
    if (i < dayCount - 1) {
      result.push(null);
      continue;
    }
    let sum = 0;
    for (let j = 0; j < dayCount; j++) {
      sum += +data[i - j][1]; // 收盘价
    }
    result.push((sum / dayCount).toFixed(2));
  }
  return result;
};

const calcMACD = (data, short = 12, long = 26, signal = 9) => {
  let emaShort = [];
  let emaLong = [];
  let diff = [];
  let dea = [];
  let macd = [];

  let kShort = 2 / (short + 1);
  let kLong = 2 / (long + 1);
  let kSignal = 2 / (signal + 1);

  for (let i = 0; i < data.length; i++) {
    let close = +data[i][1];
    emaShort[i] = i === 0 ? close : emaShort[i - 1] * (1 - kShort) + close * kShort;
    emaLong[i] = i === 0 ? close : emaLong[i - 1] * (1 - kLong) + close * kLong;
    diff[i] = emaShort[i] - emaLong[i];
    dea[i] = i === 0 ? diff[i] : dea[i - 1] * (1 - kSignal) + diff[i] * kSignal;
    macd[i] = (diff[i] - dea[i]) * 2;
  }
  return macd.map(v => v.toFixed(2));
};

const calcBOLL = (data, n = 20) => {
  let mid = [];
  for (let i = 0; i < data.length; i++) {
    if (i < n - 1) {
      mid.push(null);
      continue;
    }
    let sum = 0;
    for (let j = 0; j < n; j++) {
      sum += +data[i - j][1]; // 收盘价
    }
    mid.push((sum / n).toFixed(2));
  }
  return mid;
};

onMounted(async () => {
  const res = await axios.get(`http://192.168.3.99:3003/api/v1/stock-daily-data/${symbol}`);
  tableData.value = res.data.map(item => ({
    ...item,
    ma5: null,
    ma10: null,
    macd: null,
    boll_mid: null
  }));

  const kData = res.data.map(d => [
    d.open,
    d.close,
    d.low,
    d.high
  ]);
  const dates = res.data.map(d => d.trade_date);

  const ma5 = calcMA(5, res.data.map(d => [0, d.close]));
  const ma10 = calcMA(10, res.data.map(d => [0, d.close]));
  const macd = calcMACD(res.data.map(d => [0, d.close]));
  const boll = calcBOLL(res.data.map(d => [0, d.close]));

  tableData.value.forEach((row, i) => {
    row.ma5 = ma5[i];
    row.ma10 = ma10[i];
    row.macd = macd[i];
    row.boll_mid = boll[i];
  });

  const chart = echarts.init(chartRef.value);
  chart.setOption({
    title: { text: `${symbol} 日K线` },
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'category', data: dates },
    yAxis: { scale: true },
    series: [
      { type: 'candlestick', data: kData },
      { name: 'MA5', type: 'line', data: ma5 },
      { name: 'MA10', type: 'line', data: ma10 }
    ]
  });
});
</script>
