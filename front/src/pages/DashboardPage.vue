<template>
  <div class="dashboard">
    <el-row :gutter="16">
      <!-- 涨幅榜 -->
      <el-col :span="12">
        <el-card shadow="never">
          <template #header><span class="card-title">📈 涨幅榜 TOP20</span></template>
          <el-table :data="gainers" stripe size="small" max-height="600">
            <el-table-column type="index" width="50" />
            <el-table-column prop="symbol" label="代码" width="90" />
            <el-table-column prop="name" label="名称" width="100" />
            <el-table-column prop="close" label="现价" width="80" />
            <el-table-column prop="change_percent" label="涨跌幅" width="100" sortable>
              <template #default="{ row }">
                <span :style="{ color: row.change_percent >= 0 ? 'red' : 'green', fontWeight: 'bold' }">
                  {{ row.change_percent.toFixed(2) }}%
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="volume" label="成交量" width="100">
              <template #default="{ row }">{{ (row.volume / 10000).toFixed(0) }}万</template>
            </el-table-column>
            <el-table-column prop="turnover_rate" label="换手率" width="80">
              <template #default="{ row }">{{ row.turnover_rate.toFixed(2) }}%</template>
            </el-table-column>
            <el-table-column label="操作" width="70">
              <template #default="{ row }">
                <el-button size="small" link @click="goDetail(row.symbol)">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <!-- 跌幅榜 -->
      <el-col :span="12">
        <el-card shadow="never">
          <template #header><span class="card-title">📉 跌幅榜 TOP20</span></template>
          <el-table :data="losers" stripe size="small" max-height="600">
            <el-table-column type="index" width="50" />
            <el-table-column prop="symbol" label="代码" width="90" />
            <el-table-column prop="name" label="名称" width="100" />
            <el-table-column prop="close" label="现价" width="80" />
            <el-table-column prop="change_percent" label="涨跌幅" width="100" sortable>
              <template #default="{ row }">
                <span :style="{ color: row.change_percent >= 0 ? 'red' : 'green', fontWeight: 'bold' }">
                  {{ row.change_percent.toFixed(2) }}%
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="volume" label="成交量" width="100">
              <template #default="{ row }">{{ (row.volume / 10000).toFixed(0) }}万</template>
            </el-table-column>
            <el-table-column prop="turnover_rate" label="换手率" width="80">
              <template #default="{ row }">{{ row.turnover_rate.toFixed(2) }}%</template>
            </el-table-column>
            <el-table-column label="操作" width="70">
              <template #default="{ row }">
                <el-button size="small" link @click="goDetail(row.symbol)">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <!-- 第二行：成交量榜 / 资金净额榜 -->
    <el-row :gutter="16" style="margin-top:16px">
      <el-col :span="12">
        <el-card shadow="never">
          <template #header><span class="card-title">📦 成交量榜 TOP20</span></template>
          <el-table :data="volumeRank" stripe size="small" max-height="600">
            <el-table-column type="index" width="50" />
            <el-table-column prop="symbol" label="代码" width="90" />
            <el-table-column prop="name" label="名称" width="100" />
            <el-table-column prop="close" label="现价" width="80" />
            <el-table-column prop="change_percent" label="涨跌幅" width="100">
              <template #default="{ row }">
                <span :style="{ color: row.change_percent >= 0 ? 'red' : 'green' }">
                  {{ row.change_percent.toFixed(2) }}%
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="volume" label="成交量" width="100">
              <template #default="{ row }">{{ (row.volume / 10000).toFixed(0) }}万</template>
            </el-table-column>
            <el-table-column label="操作" width="70">
              <template #default="{ row }">
                <el-button size="small" link @click="goDetail(row.symbol)">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card shadow="never">
          <template #header><span class="card-title">💰 资金净额榜 TOP20</span></template>
          <el-table :data="netAmountRank" stripe size="small" max-height="600">
            <el-table-column type="index" width="50" />
            <el-table-column prop="symbol" label="代码" width="90" />
            <el-table-column prop="name" label="名称" width="100" />
            <el-table-column prop="close" label="现价" width="80" />
            <el-table-column prop="change_percent" label="涨跌幅" width="100">
              <template #default="{ row }">
                <span :style="{ color: row.change_percent >= 0 ? 'red' : 'green' }">
                  {{ row.change_percent.toFixed(2) }}%
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="net_amount" label="资金净额" width="110">
              <template #default="{ row }">
                <span :style="{ color: row.net_amount >= 0 ? 'red' : 'green', fontWeight: 'bold' }">
                  {{ row.net_amount.toFixed(0) }}万
                </span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="70">
              <template #default="{ row }">
                <el-button size="small" link @click="goDetail(row.symbol)">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted } from "vue"
import { useRouter } from "vue-router"
import { getStockRanking } from "@/utils/api/screen.js"

const router = useRouter()
const gainers = ref([])
const losers = ref([])
const volumeRank = ref([])
const netAmountRank = ref([])

function goDetail(symbol) {
  router.push({ path: "/stocks", query: { symbol } })
}

onMounted(async () => {
  try {
    const [gRes, lRes, vRes, nRes] = await Promise.all([
      getStockRanking("change_percent", 20, "desc"),
      getStockRanking("change_percent", 20, "asc"),
      getStockRanking("volume", 20),
      getStockRanking("net_amount", 20)
    ])
    gainers.value = gRes.data.data || []
    losers.value = lRes.data.data || []
    volumeRank.value = vRes.data.data || []
    netAmountRank.value = nRes.data.data || []
  } catch (e) { console.error(e) }
})
</script>

<style scoped>
.dashboard { padding: 16px; }
.card-title { font-size: 16px; font-weight: bold; }
</style>