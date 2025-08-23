<template>
  <div class="p-4">
    <el-card>
      <div class="flex items-center gap-2 mb-4">
        <el-input
          v-model="inputCode"
          placeholder="输入股票代码，例如 sh600000"
          style="max-width: 220px"
          @keyup.enter="addStock"
        />
        <el-button type="primary" @click="addStock">添加</el-button>
        <el-button @click="fetchStocks">刷新</el-button>
      </div>

      <el-table :data="stocks" stripe border style="width: 100%">
        <el-table-column prop="code" label="代码" width="120" />
        <el-table-column prop="name" label="名称" width="120" />
        <el-table-column prop="current" label="现价" />
        <el-table-column prop="open" label="今开" />
        <el-table-column prop="yesterdayClose" label="昨收" />
        <el-table-column prop="high" label="最高" />
        <el-table-column prop="low" label="最低" />
        <el-table-column prop="volume" label="成交量" />
        <el-table-column prop="amount" label="成交额" />
        <el-table-column prop="time" label="时间" width="140" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { sinaStockProvider } from "@/services/SinaStockProvider";

const inputCode = ref("");
const stockCodes = ref<string[]>(["sh600000", "sz000001", "sz300750"]);
const stocks = ref<any[]>([]);

const fetchStocks = async () => {
  if (stockCodes.value.length === 0) return;
  stocks.value = await sinaStockProvider.fetch(stockCodes.value);
};

const addStock = () => {
  if (inputCode.value && !stockCodes.value.includes(inputCode.value)) {
    stockCodes.value.push(inputCode.value);
    fetchStocks();
  }
  inputCode.value = "";
};

// 启动时拉取一次
fetchStocks();
</script>
