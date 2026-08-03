import request from '../request'

// 分页获取股票列表（来自 stock_history_mv）
export const getStocks = (page = 1, pageSize = 20, keyword = '') =>
  request.get('/stocks/list', { params: { page, page_size: pageSize } })

// 模糊搜索（用于 autocomplete）
export const searchStocks = (keyword) =>
  request.get('/stocks/search', { params: { q: keyword } })

// 单个股票详情
export const fetchStockDetail = (symbol) =>
  request.get(`/stocks/symbol/${symbol}`)

// 综合历史（K 线 + 指标 + 资金流）
export const fetchStockHistory = (symbol, days = 120) =>
  request.get('/stocks/history', { params: { symbol, days } })

// 热门
export const fetchHotStocks = (params = {}) =>
  request.get('/stocks/hot', { params })
