/**
 * ruleOperators.js — 自定义规则可视化编辑器的算子元数据
 *
 * 每个算子定义：
 *  - type:       后端 evaluator 接受的 type 字符串
 *  - label:      下拉菜单里显示的中文名
 *  - hint:       简短说明
 *  - defaults:   添加时的默认参数
 *  - params:     用于渲染表单控件的 schema 数组
 *
 * 表单控件类型：
 *  - number    数字输入
 *  - select    下拉单选
 *  - multi     多选 chip
 *  - text      文本输入
 */
export const VISIBLE_OPERATORS = [
  {
    type: 'change_percent_range', label: '涨跌幅区间', hint: '限定当日涨跌幅 (%)',
    defaults: { min: -3, max: 3 },
    params: [
      { key: 'min', label: '最小', type: 'number', step: 0.1, suffix: '%' },
      { key: 'max', label: '最大', type: 'number', step: 0.1, suffix: '%' },
    ],
  },
  {
    type: 'turnover_rate_range', label: '换手率区间', hint: '限定换手率 (%)',
    defaults: { min: 3, max: 8 },
    params: [
      { key: 'min', label: '最小', type: 'number', step: 0.1, suffix: '%' },
      { key: 'max', label: '最大', type: 'number', step: 0.1, suffix: '%' },
    ],
  },
  {
    type: 'volume_ratio', label: '量比', hint: 'latest.volume / vol_avg5',
    defaults: { min: 1.5 },
    params: [
      { key: 'min', label: '≥', type: 'number', step: 0.1, placeholder: '可选：放量阈值' },
      { key: 'max', label: '≤', type: 'number', step: 0.1, placeholder: '可选：缩量阈值' },
    ],
  },
  {
    type: 'yang_streak', label: '连续阳线', hint: 'N 日连续收阳',
    defaults: { days: 3 },
    params: [{ key: 'days', label: '天数', type: 'number', step: 1, min: 1 }],
  },
  {
    type: 'yin_streak', label: '连续阴线', hint: 'N 日连续收阴',
    defaults: { days: 3 },
    params: [{ key: 'days', label: '天数', type: 'number', step: 1, min: 1 }],
  },
  {
    type: 'ma_alignment', label: '均线多头排列', hint: 'MA5>MA10>MA20>MA60',
    defaults: { order: ['ma5', 'ma10', 'ma20', 'ma60'] },
    params: [
      { key: 'order', label: '均线顺序', type: 'multi',
        options: [
          { value: 'ma5', label: 'MA5' }, { value: 'ma10', label: 'MA10' },
          { value: 'ma20', label: 'MA20' }, { value: 'ma60', label: 'MA60' },
        ] },
    ],
  },
  {
    type: 'ma_compare', label: '均线对比', hint: 'MA fast 与 slow 比较',
    defaults: { fast: 'ma5', slow: 'ma10', op: 'gt' },
    params: [
      { key: 'fast', label: '快线', type: 'select',
        options: [{ value: 'ma5', label: 'MA5' }, { value: 'ma10', label: 'MA10' }, { value: 'ma20', label: 'MA20' }] },
      { key: 'op', label: '关系', type: 'select',
        options: [{ value: 'gt', label: '>' }, { value: 'lt', label: '<' }] },
      { key: 'slow', label: '慢线', type: 'select',
        options: [{ value: 'ma10', label: 'MA10' }, { value: 'ma20', label: 'MA20' }, { value: 'ma60', label: 'MA60' }] },
    ],
  },
  {
    type: 'close_vs_ma', label: '收盘 vs MA', hint: '收盘价相对某 MA',
    defaults: { ma: 'ma5', op: 'gt' },
    params: [
      { key: 'ma', label: 'MA', type: 'select',
        options: [{ value: 'ma5', label: 'MA5' }, { value: 'ma10', label: 'MA10' }, { value: 'ma20', label: 'MA20' }] },
      { key: 'op', label: '关系', type: 'select',
        options: [{ value: 'gt', label: '>' }, { value: 'lt', label: '<' }] },
    ],
  },
  {
    type: 'kdj_cross', label: 'KDJ 交叉', hint: 'K/D 交叉方向与位置',
    defaults: { location: 'any' },
    params: [
      { key: 'direction', label: '方向', type: 'select',
        options: [{ value: 'golden', label: '金叉 (K 上穿 D)' }, { value: 'death', label: '死叉 (K 下穿 D)' }] },
      { key: 'location', label: '位置', type: 'select',
        options: [{ value: 'any', label: '任意' }, { value: 'below_zero', label: '零轴下方' }, { value: 'above_zero', label: '零轴上方' }] },
    ],
  },
  {
    type: 'macd_cross', label: 'MACD 交叉', hint: 'DIF/DEA 交叉方向与位置',
    defaults: { location: 'any' },
    params: [
      { key: 'direction', label: '方向', type: 'select',
        options: [{ value: 'golden', label: '金叉' }, { value: 'death', label: '死叉' }] },
      { key: 'location', label: '位置', type: 'select',
        options: [{ value: 'any', label: '任意' }, { value: 'below_zero', label: '零轴下方' }, { value: 'above_zero', label: '零轴上方' }] },
    ],
  },
  {
    type: 'board_in', label: '限定板型', hint: '主板/创业板/科创板/北交所',
    defaults: { boards: ['主板', '创业板', '科创板'] },
    params: [
      { key: 'boards', label: '板型', type: 'multi',
        options: [
          { value: '主板', label: '主板' },
          { value: '创业板', label: '创业板' },
          { value: '科创板', label: '科创板' },
          { value: '北交所', label: '北交所' },
        ] },
    ],
  },
  {
    type: 'industry_in', label: '限定行业', hint: '指定行业（多个用逗号）',
    defaults: { industries: [] },
    params: [{ key: 'industries', label: '行业', type: 'text', placeholder: '半导体,银行,白酒' }],
  },
  {
    type: 'breakout_high', label: '突破 N 日新高', hint: '收盘创 N 日新高',
    defaults: { lookback: 60 },
    params: [{ key: 'lookback', label: '回溯天数', type: 'number', step: 5, min: 5 }],
  },
]

export const EXCLUDE_OPERATORS = [
  {
    type: 'is_st', label: '排除 ST/*ST', hint: '名称含 ST 的股票',
    defaults: {},
    params: [],
  },
  {
    type: 'is_not_st', label: '只保留 ST', hint: '反向（一般用不到）',
    defaults: {},
    params: [],
  },
  {
    type: 'industry_not_in', label: '排除行业', hint: '指定行业外的股票',
    defaults: { industries: [] },
    params: [{ key: 'industries', label: '行业', type: 'text', placeholder: 'ST,房地产' }],
  },
  {
    type: 'list_age_days_lt', label: '排除次新股', hint: '上市 < N 天的',
    defaults: { days: 60 },
    params: [{ key: 'days', label: '上市天数', type: 'number', step: 30, min: 0 }],
  },
  {
    type: 'market_cap_yi', label: '排除市值区间', hint: '总市值（亿）',
    defaults: { min: 0, max: 20 },
    params: [
      { key: 'min', label: '最小', type: 'number', step: 10, suffix: '亿' },
      { key: 'max', label: '最大', type: 'number', step: 10, suffix: '亿' },
    ],
  },
]

const ALL_OPS = [...VISIBLE_OPERATORS, ...EXCLUDE_OPERATORS]
const BY_TYPE = Object.fromEntries(ALL_OPS.map(o => [o.type, o]))

export function defaultParamsFor(type) {
  const op = BY_TYPE[type]
  if (!op) return { type }
  return { type, ...JSON.parse(JSON.stringify(op.defaults)) }
}

export function getOpMeta(type) {
  return BY_TYPE[type]
}
