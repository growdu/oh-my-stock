import request from '../request'   // 复用之前的 request.js

/**
 * 获取股票历史数据
 * @param {string} symbol 股票代码
 */
export function fetchDailyData(symbol) {
  return request.get(`/stocks/history`, { params: { symbol } })
}

/**
 * 模糊搜索股票
 * @param {string} query 查询关键字
 */
export function searchStocks(query) {
  return request.get(`/stocks/search`, { params: { q: query } })
}
