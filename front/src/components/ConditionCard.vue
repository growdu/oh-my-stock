<!--
  ConditionCard.vue — 单条条件的卡片
  根据 cond.type 渲染对应的表单（数字 / 下拉 / 多选 / 文本）
-->
<template>
  <div class="cc-card">
    <div class="cc-head">
      <el-tag size="small" :type="tagType" effect="plain">{{ opMeta?.label || cond.type }}</el-tag>
      <span class="cc-desc">{{ opMeta?.hint }}</span>
      <el-button text type="danger" size="small" class="cc-del" @click="$emit('remove')">×</el-button>
    </div>

    <div v-if="opMeta?.params?.length" class="cc-body">
      <div
        v-for="p in opMeta.params"
        :key="p.key"
        class="cc-field"
      >
        <label class="cc-label">{{ p.label }}</label>

        <el-input-number
          v-if="p.type === 'number'"
          :model-value="cond[p.key]"
          @update:model-value="update(p.key, $event)"
          :step="p.step || 1"
          :min="p.min"
          :max="p.max"
          size="small"
          :placeholder="p.placeholder"
          style="width: 120px"
        />
        <span v-if="p.suffix" class="cc-suffix">{{ p.suffix }}</span>

        <el-select
          v-else-if="p.type === 'select'"
          :model-value="cond[p.key]"
          @update:model-value="update(p.key, $event)"
          size="small"
          style="width: 140px"
        >
          <el-option v-for="o in p.options" :key="o.value" :label="o.label" :value="o.value" />
        </el-select>

        <div v-else-if="p.type === 'multi'" class="cc-multi">
          <el-checkbox-group
            :model-value="cond[p.key] || []"
            @update:model-value="update(p.key, $event)"
            size="small"
          >
            <el-checkbox v-for="o in p.options" :key="o.value" :label="o.value" border size="small">
              {{ o.label }}
            </el-checkbox>
          </el-checkbox-group>
        </div>

        <el-input
          v-else-if="p.type === 'text'"
          :model-value="textValue(p.key)"
          @update:model-value="updateText(p.key, $event)"
          size="small"
          :placeholder="p.placeholder"
          style="width: 220px"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { getOpMeta } from './ruleOperators.js'

const props = defineProps({
  cond: { type: Object, required: true },
})
const emit = defineEmits(['remove', 'update'])

const opMeta = computed(() => getOpMeta(props.cond.type))

const TAG_COLOR = {
  change_percent_range: 'danger', turnover_rate_range: 'warning', volume_ratio: 'success',
  yang_streak: 'success', yin_streak: 'danger', ma_alignment: 'success',
  ma_compare: 'success', close_vs_ma: '', kdj_cross: 'warning',
  macd_cross: 'warning', board_in: '', industry_in: 'info',
  breakout_high: 'success', is_st: 'danger', industry_not_in: 'info',
  list_age_days_lt: '', market_cap_yi: '',
}
const tagType = computed(() => TAG_COLOR[props.cond.type] || '')

function update(key, val) {
  const next = { ...props.cond, [key]: val }
  if (val === undefined || val === null || val === '') delete next[key]
  emit('update', next)
}

function textValue(key) {
  const v = props.cond[key]
  if (Array.isArray(v)) return v.join(',')
  return v ?? ''
}
function updateText(key, text) {
  const arr = String(text || '').split(',').map(s => s.trim()).filter(Boolean)
  emit('update', { ...props.cond, [key]: arr })
}
</script>

<style scoped>
.cc-card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  padding: 8px 10px;
  display: flex; flex-direction: column; gap: 6px;
}
.cc-card:hover { border-color: #93c5fd; }
.cc-head {
  display: flex; align-items: center; gap: 8px;
}
.cc-desc { color: #9ca3af; font-size: 11px; flex: 1; }
.cc-del { font-size: 16px; padding: 0 6px; line-height: 1; }

.cc-body {
  display: flex; flex-wrap: wrap; gap: 6px 12px; padding-top: 2px;
}
.cc-field {
  display: flex; align-items: center; gap: 4px;
  background: #f9fafb; border-radius: 4px; padding: 3px 6px;
}
.cc-label { font-size: 12px; color: #6b7280; min-width: 32px; }
.cc-suffix { font-size: 11px; color: #9ca3af; }
.cc-multi { display: flex; }
.cc-multi :deep(.el-checkbox.is-bordered) { margin-right: 4px; }
.cc-multi :deep(.el-checkbox__label) { padding-left: 4px; font-size: 12px; }
</style>
