import request from '../request'

// 精选 Top N：传入勾选的预设 id 列表，返回按综合分排好序的 Top N 股票
export const finalPick = (presetIds, topN = 2) =>
  request.post('/screen/final-pick', {
    preset_ids: presetIds,
    top_n: topN,
  })


// 历史精选：返回最近 N 天（默认 7 天）的 Top N 落库记录
export const finalPickHistory = (days = 7) =>
  request.get('/screen/final-pick/history', { params: { days } })


// 最新精选（读缓存）：返回 final_picks 表最新交易日的 Top N
// 不做计算，由后端 daily_refresh 预算好落库
export const finalPickLatest = () =>
  request.get('/screen/final-pick/latest')
