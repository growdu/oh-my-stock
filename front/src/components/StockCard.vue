<template>
  <div
    class="stock-card"
    :class="stockClass"
    @click="$emit('click', stock)"
    :title="clickable ? '点击查看 K 线' : ''"
  >
    <!-- 头部：代码 + 名称 + 板型 + 排名 -->
    <div class="card-header">
      <span class="sym">{{ stock.symbol }}</span>
      <span class="name" :title="stock.name">{{ stock.name }}</span>
      <el-tag size="small" effect="plain" class="market-tag" :class="boardTagClass">{{ boardLabel }}</el-tag>
      <span v-if="rank" class="rank-no">#{{ rank }}</span>
      <slot name="badge" />
    </div>

    <!-- 行业 -->
    <div v-if="stock.industry" class="card-industry">
      <span class="ind-label">行业</span>{{ stock.industry }}
    </div>

    <!-- 价格 -->
    <div class="price-block" :class="stockClass">
      <span class="price">{{ fmt(stock.close) }}</span>
      <span class="pct">{{ pctArrow }} {{ (stock.change_percent ?? 0) >= 0 ? '+' : '' }}{{ fmt(stock.change_percent) }}%</span>
    </div>

    <!-- 2x2 核心数据 -->
    <div class="core-grid">
      <div class="core-cell">
        <span class="core-label">净利润</span>
        <b class="core-value" :class="yoyClass(stock.net_profit_yoy)">
          {{ fmtYi(stock.net_profit) }}
          <small v-if="stock.net_profit_yoy != null" class="core-sub">同比 {{ fmtPctSigned(stock.net_profit_yoy) }}</small>
        </b>
      </div>
      <div class="core-cell">
        <span class="core-label">营收增长</span>
        <b class="core-value" :class="yoyClass(stock.revenue_yoy)">{{ fmtPctSigned(stock.revenue_yoy) }}</b>
      </div>
      <div class="core-cell">
        <span class="core-label">成交量</span>
        <b class="core-value">{{ fmtVol(stock.volume) }}</b>
      </div>
      <div class="core-cell">
        <span class="core-label">MA5</span>
        <b class="core-value" :class="maPosClass(stock.ma5)">
          {{ fmt(stock.ma5) }}
          <em v-if="maCross" class="ts-flag gold">金叉</em>
          <em v-else-if="maDeathCross" class="ts-flag death">死叉</em>
        </b>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  stock: { type: Object, required: true },
  rank: { type: [Number, String], default: null },
  clickable: { type: Boolean, default: false },
})
defineEmits(['click'])

// === 格式化 ===
const fmt = (v) => (v == null || Number.isNaN(Number(v))) ? '-' : Number(v).toFixed(2)
const fmtYi = (v) => {
  if (v == null || Number.isNaN(Number(v))) return '-'
  const n = Number(v)
  if (Math.abs(n) >= 1e8) return (n / 1e8).toFixed(2) + '亿'
  if (Math.abs(n) >= 1e4) return (n / 1e4).toFixed(2) + '万'
  return n.toFixed(0)
}
const fmtVol = (v) => {
  if (v == null) return '-'
  const n = Number(v)
  if (Math.abs(n) >= 1e8) return (n / 1e8).toFixed(2) + '亿'
  if (Math.abs(n) >= 1e4) return (n / 1e4).toFixed(2) + '万'
  return n.toFixed(0)
}
const fmtPctSigned = (v) => {
  if (v == null) return '-'
  const n = Number(v)
  return (n >= 0 ? '+' : '') + n.toFixed(1) + '%'
}

const yoyClass = (v) => {
  if (v == null) return ''
  return Number(v) >= 0 ? 'up' : 'down'
}

const stockClass = computed(() => {
  const cp = props.stock.change_percent ?? 0
  if (cp > 0) return 'up'
  if (cp < 0) return 'down'
  return 'flat'
})

const pctArrow = computed(() => {
  const cp = props.stock.change_percent ?? 0
  if (cp > 0) return '▲'
  if (cp < 0) return '▼'
  return '—'
})

// === 技术指标 ===
const hasValue = (v) => v != null && Math.abs(v) > 1e-9
const maPosClass = (v) => {
  if (!hasValue(v) || props.stock.close == null) return ''
  return Number(props.stock.close) >= Number(v) ? 'up' : 'down'
}
const maCross = computed(() => {
  const f = props.stock.ma5, fp = props.stock.ma5_prev
  const sl = props.stock.ma10
  if (!hasValue(f) || !hasValue(fp) || !hasValue(sl)) return false
  return fp <= sl && f > sl
})
const maDeathCross = computed(() => {
  const f = props.stock.ma5, fp = props.stock.ma5_prev
  const sl = props.stock.ma10
  if (!hasValue(f) || !hasValue(fp) || !hasValue(sl)) return false
  return fp >= sl && f < sl
})

// === 板型标签 ===
const boardTagClass = computed(() => {
  const m = props.stock.market || ''
  const sym = String(props.stock.symbol || '')
  if (m === '科创板' || /^688/.test(sym)) return 'tag-STAR'
  if (m === '创业板' || /^30[01]/.test(sym)) return 'tag-CHINEXT'
  if (m === '主板' || m === '深主板' || m === '沈主板') return 'tag-MAIN'
  return 'tag-OTHER'
})

const boardLabel = computed(() => {
  const sym = String(props.stock.symbol || '')
  if (sym.startsWith('688')) return '科创板'
  if (sym.startsWith('300') || sym.startsWith('301')) return '创业板'
  if (sym.startsWith('60') || sym.startsWith('00') || sym.startsWith('20')) return '主板'
  if (sym.startsWith('8')  || sym.startsWith('43') || sym.startsWith('92')) return '北交所'
  return props.stock.market || ''
})
</script>

<style scoped>
/* === 卡片容器 === */
.stock-card {
  position: relative;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  padding: 14px 16px 12px 16px;
  background: #ffffff;
  overflow: hidden;
  transition: transform .25s ease, box-shadow .25s ease, border-color .25s ease;
  box-shadow: 0 1px 3px rgba(0,0,0,.04);
  display: flex; flex-direction: column;
  cursor: pointer;
}
.stock-card::after {
  content: '';
  position: absolute;
  top: 0; right: 0; bottom: 0;
  width: 4px;
}
.stock-card.up::after { background: #e63946; }
.stock-card.down::after { background: #16a34a; }
.stock-card.flat::after { background: #d1d5db; }

.stock-card.up   { background: #fff8f8; }
.stock-card.down { background: #f6fbf6; }
.stock-card:hover {
  transform: translateY(-3px);
  border-color: #c7d2fe;
  box-shadow: 0 10px 28px rgba(0,0,0,.10);
}

/* === 头部 === */
.card-header {
  display: flex; align-items: center; gap: 10px;
  margin-bottom: 10px;
  padding-bottom: 10px;
  border-bottom: 1px dashed #f3f4f6;
}
.card-header .sym {
  font-size: 20px; font-weight: 800; color: #111;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  letter-spacing: -0.5px;
}
.card-header .name {
  font-size: 17px; color: #222; font-weight: 700;
  max-width: 140px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.card-header .market-tag {
  margin-left: auto;
  font-weight: 700 !important;
  font-size: 13px !important;
  padding: 0 10px !important;
  height: 24px !important;
  line-height: 22px !important;
}
.card-header .market-tag.tag-STAR   { background: #fdf2f8 !important; border-color: #f9a8d4 !important; color: #be185d !important; }
.card-header .market-tag.tag-CHINEXT{ background: #fff7ed !important; border-color: #fdba74 !important; color: #c2410c !important; }
.card-header .market-tag.tag-MAIN   { background: #f0f9ff !important; border-color: #bae6fd !important; color: #0369a1 !important; }
.card-header .market-tag.tag-OTHER  { background: #f3f4f6 !important; border-color: #d1d5db !important; color: #4b5563 !important; }
.card-header .rank-no {
  font-size: 14px; color: #999;
  font-family: 'SF Mono', Menlo, monospace;
  font-weight: 700;
}

/* === 行业 === */
.card-industry {
  font-size: 14px;
  color: #4b5563;
  font-weight: 600;
  margin: 0 0 12px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.card-industry .ind-label { color: #9ca3af; margin-right: 6px; font-weight: 500; }

/* === 价格 === */
.price-block {
  display: flex; align-items: baseline; gap: 10px;
  padding: 4px 0 6px;
  border-bottom: 1px dashed #f3f4f6;
  margin-bottom: 8px;
}
.price-block.up   .price { color: #e63946; }
.price-block.down .price { color: #16a34a; }
.price-block .price {
  font-size: 26px; font-weight: 900;
  letter-spacing: -0.5px;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  line-height: 1.1;
}
.price-block .pct {
  font-size: 14px; font-weight: 800;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
}
.price-block.up   .pct { color: #e63946; }
.price-block.down .pct { color: #16a34a; }
.price-block.flat .pct { color: #6b7280; }

/* === 2x2 核心数据 === */
.core-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px 10px;
  flex: 1;
  align-content: center;
}
.core-cell {
  display: flex; flex-direction: column; align-items: flex-start; justify-content: center;
  gap: 1px;
  background: #fafbfc;
  border-radius: 6px;
  padding: 6px 8px;
  min-width: 0;
}
.core-cell .core-label {
  color: #9ca3af;
  font-weight: 500;
  font-size: 11px;
  letter-spacing: 0.2px;
}
.core-cell .core-value {
  color: #111;
  font-weight: 800;
  font-size: 15px;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
  line-height: 1.25;
}
.core-cell .core-value.up   { color: #e63946; }
.core-cell .core-value.down { color: #16a34a; }
.core-cell .core-sub {
  display: block;
  font-size: 10px;
  font-weight: 500;
  color: #9ca3af;
  margin-top: 1px;
  font-family: inherit;
  letter-spacing: 0;
}

/* 金叉/死叉标签 */
.ts-flag {
  display: inline-block;
  font-style: normal;
  font-size: 10px;
  font-weight: 700;
  padding: 1px 4px;
  border-radius: 3px;
  margin-left: 4px;
  vertical-align: 1px;
}
.ts-flag.gold { background: #fef3c7; color: #d97706; }
.ts-flag.death { background: #fee2e2; color: #dc2626; }
</style>
