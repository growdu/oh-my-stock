<template>
  <div class="home-page">
    <!-- 顶栏 -->
    <div class="topbar">
      <h1>每日精选</h1>
      <div class="topbar-right">
        <span v-if="updatedAt" class="update-tip">更新于 {{ updatedAt }}</span>
        <el-tag type="info" effect="plain">{{ tradeDate || '暂无数据' }}</el-tag>
        <button class="topbar-btn" @click="router.push('/display')">更多预设 ›</button>
      </div>
    </div>

    <!-- 加载态 -->
    <div v-if="loading" class="loading">加载中…</div>

    <!-- 空态：今日还没预算 -->
    <div v-else-if="picks.length === 0" class="empty-state">
      <el-empty :description="errorMsg || '今日精选尚未生成，等待每日数据更新后自动计算'" />
    </div>

    <!-- 精选卡片 -->
    <div v-else class="stocks-grid">
      <StockCard
        v-for="(p, i) in picks"
        :key="p.symbol"
        :stock="p"
        :rank="i + 1"
        clickable
        @click="openKLine"
      />
    </div>

    <KLineDialog v-model="klineOpen" :stock="klineStock" />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { finalPickLatest } from '@/utils/api/screen'
import StockCard from '@/components/StockCard.vue'
import KLineDialog from '@/components/KLineDialog.vue'

const router = useRouter()
const loading   = ref(true)
const errorMsg  = ref('')
const picks     = ref([])
const tradeDate = ref('')
const updatedAt = ref('')

// K 线弹窗
const klineOpen  = ref(false)
const klineStock = ref(null)
function openKLine(stock) {
  klineStock.value = { symbol: stock.symbol, name: stock.name }
  klineOpen.value  = true
}

// 适配器：PickStock → StockCard 期望字段
function pickToCard(p) {
  return {
    symbol: p.symbol,
    name: p.name,
    industry: p.industry,
    market: p.market,
    close: p.close,
    change_percent: p.change_percent,
    volume: p.volume ?? 0,
    turnover_rate: p.turnover_rate,
    pe_ttm: p.pe_ttm,
    pb: p.pb,
    net_profit: p.net_profit ?? null,
    net_profit_yoy: p.net_profit_yoy ?? null,
    revenue_yoy: p.revenue_yoy ?? null,
    ma5: p.ma5,
    ma10: p.ma10,
    ma5_prev: p.ma5_prev,
  }
}

// 读取后端预算好的缓存（不计算）
async function loadLatest() {
  loading.value = true
  errorMsg.value = ''
  try {
    const { data } = await finalPickLatest()
    picks.value = (data.picks || []).map(pickToCard)
    tradeDate.value = data.trade_date || ''
    updatedAt.value = data.updated_at || ''
    if (data.picks.length === 0 && !data.cached) {
      errorMsg.value = '今日精选尚未生成'
    }
  } catch (e) {
    errorMsg.value = '加载失败：' + (e?.message || e)
    ElMessage.error(errorMsg.value)
  } finally {
    loading.value = false
  }
}

onMounted(loadLatest)
</script>

<style scoped>
.home-page {
  max-width: 1280px;
  margin: 0 auto;
  padding: 16px;
}

.topbar {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
}
.topbar h1 {
  flex: 1;
  font-size: 22px;
  margin: 0;
  color: #303133;
}
.topbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}
.update-tip {
  font-size: 12px;
  color: #909399;
  font-variant-numeric: tabular-nums;
}
.topbar-btn {
  background: transparent;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  padding: 6px 12px;
  cursor: pointer;
  font-size: 13px;
  color: #606266;
  transition: all 0.15s;
}
.topbar-btn:hover { background: #f5f7fa; border-color: #409eff; color: #409eff; }

.loading {
  text-align: center;
  padding: 80px 0;
  color: #909399;
  font-size: 14px;
}
.empty-state {
  margin-top: 40px;
}

.stocks-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}
@media (max-width: 900px) { .stocks-grid { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 600px) { .stocks-grid { grid-template-columns: 1fr; } }
</style>
