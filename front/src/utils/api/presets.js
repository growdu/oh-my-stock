import request from '../request'

// 列出所有内置默认策略
export const listPresets = () => request.get('/presets')

// 执行某个默认策略，返回分页命中
export const runPreset = (id, page = 1, pageSize = 8) =>
  request.get(`/presets/${id}/run`, { params: { page, page_size: pageSize } })
