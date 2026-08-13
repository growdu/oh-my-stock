<!--
  RuleEditor.vue
  自定义规则可视化编辑器
  - 支持 13 个常用算子的表单输入（覆盖 16 个预设中 ~90% 的用法）
  - 实时生成 JSON 表达式，双向绑定
  - 复杂算子可通过"高级 JSON 模式"直接编辑
-->
<template>
  <div class="rule-editor">
    <!-- 左侧：可视化 -->
    <div class="re-visual">
      <el-tabs v-model="activeTab" class="re-tabs">
        <el-tab-pane label="满足全部 (AND)" name="all">
          <div class="re-list">
            <ConditionCard
              v-for="(c, i) in modelValue.all || []"
              :key="i"
              :cond="c"
              @remove="removeAt('all', i)"
              @update="updateAt('all', i, $event)"
            />
            <div class="re-add">
              <el-dropdown @command="(t) => addAt('all', t)">
                <el-button size="small" plain>
                  <span style="margin-right:4px">+</span>添加条件
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item v-for="o in VISIBLE_OPERATORS" :key="o.type" :command="o.type">
                      <span class="re-op-name">{{ o.label }}</span>
                      <span class="re-op-desc">{{ o.hint }}</span>
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
            <el-empty v-if="!modelValue.all?.length" description="尚无 AND 条件" :image-size="50" />
          </div>
        </el-tab-pane>

        <el-tab-pane label="满足任一 (OR)" name="any">
          <div class="re-list">
            <ConditionCard
              v-for="(c, i) in modelValue.any || []"
              :key="i"
              :cond="c"
              @remove="removeAt('any', i)"
              @update="updateAt('any', i, $event)"
            />
            <div class="re-add">
              <el-dropdown @command="(t) => addAt('any', t)">
                <el-button size="small" plain>
                  <span style="margin-right:4px">+</span>添加条件
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item v-for="o in VISIBLE_OPERATORS" :key="o.type" :command="o.type">
                      <span class="re-op-name">{{ o.label }}</span>
                      <span class="re-op-desc">{{ o.hint }}</span>
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
            <el-empty v-if="!modelValue.any?.length" description="尚无 OR 条件" :image-size="50" />
          </div>
        </el-tab-pane>

        <el-tab-pane label="排除 (NOT)" name="exclude">
          <div class="re-list">
            <ConditionCard
              v-for="(c, i) in modelValue.exclude || []"
              :key="i"
              :cond="c"
              @remove="removeAt('exclude', i)"
              @update="updateAt('exclude', i, $event)"
            />
            <div class="re-add">
              <el-dropdown @command="(t) => addAt('exclude', t)">
                <el-button size="small" plain>
                  <span style="margin-right:4px">+</span>添加条件
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item
                      v-for="o in EXCLUDE_OPERATORS"
                      :key="o.type"
                      :command="o.type"
                    >
                      <span class="re-op-name">{{ o.label }}</span>
                      <span class="re-op-desc">{{ o.hint }}</span>
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
            <el-empty v-if="!modelValue.exclude?.length" description="尚无排除条件" :image-size="50" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- 右侧：实时 JSON -->
    <div class="re-json">
      <div class="re-json-head">
        <span class="re-json-title">JSON 预览</span>
        <span class="re-json-hint">右键/折叠切换高级编辑</span>
      </div>
      <el-input
        :model-value="jsonStr"
        type="textarea"
        :rows="14"
        spellcheck="false"
        @update:model-value="onJsonEdit"
      />
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import ConditionCard from './ConditionCard.vue'
import { VISIBLE_OPERATORS, EXCLUDE_OPERATORS, defaultParamsFor } from './ruleOperators.js'

const props = defineProps({
  modelValue: { type: Object, default: () => ({ all: [], exclude: [{ type: 'is_st' }] }) },
})
const emit = defineEmits(['update:modelValue'])

const activeTab = ref('all')

const jsonStr = computed(() => JSON.stringify(props.modelValue, null, 2))

function emit_(newVal) {
  emit_('update:modelValue', JSON.parse(JSON.stringify(newVal)))
}

function addAt(section, type) {
  const next = { ...props.modelValue }
  next[section] = [...(next[section] || []), defaultParamsFor(type)]
  emit(next)
}
function removeAt(section, i) {
  const next = { ...props.modelValue }
  next[section] = next[section].filter((_, idx) => idx !== i)
  emit(next)
}
function updateAt(section, i, newCond) {
  const next = { ...props.modelValue }
  next[section] = next[section].map((c, idx) => idx === i ? newCond : c)
  emit(next)
}
function onJsonEdit(text) {
  try {
    const obj = JSON.parse(text || '{}')
    emit(obj)
  } catch (e) {
    // ignore parse error while typing
  }
}

// 暴露给外部：拿当前 JSON
defineExpose({ json: () => jsonStr.value })
</script>

<style scoped>
.rule-editor {
  display: grid;
  grid-template-columns: minmax(0, 1.55fr) minmax(0, 1fr);
  gap: 12px;
  min-height: 360px;
}
.re-visual { background: #fafbfc; border-radius: 6px; padding: 8px 10px; }
.re-tabs :deep(.el-tabs__header) { margin-bottom: 6px; }
.re-tabs :deep(.el-tabs__nav-wrap::after) { background: #e5e7eb; }
.re-tabs :deep(.el-tabs__item) { font-size: 13px; padding: 0 10px; height: 32px; line-height: 32px; }

.re-list { display: flex; flex-direction: column; gap: 6px; padding: 4px 0; }
.re-add { padding: 4px 0; }

.re-op-name { font-weight: 600; margin-right: 6px; }
.re-op-desc { color: #9ca3af; font-size: 11px; }

.re-json {
  display: flex; flex-direction: column;
  background: #0f172a; border-radius: 6px; padding: 8px 10px;
}
.re-json-head { display: flex; justify-content: space-between; color: #cbd5e1; font-size: 11px; margin-bottom: 4px; }
.re-json-title { font-weight: 700; color: #e2e8f0; }
.re-json-hint { color: #64748b; }
.re-json :deep(.el-textarea__inner) {
  background: transparent; color: #e2e8f0;
  font-family: 'SF Mono', Menlo, Consolas, monospace; font-size: 12px; line-height: 1.5;
  border: none; box-shadow: none;
}
</style>
