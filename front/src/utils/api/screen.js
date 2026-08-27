import request from '../request'

// 精选 Top N：传入勾选的预设 id 列表，返回按综合分排好序的 Top N 股票
export const finalPick = (presetIds, topN = 2) =>
  request.post('/screen/final-pick', {
    preset_ids: presetIds,
    top_n: topN,
  })
