import request from '../request'

export const fetchDailyData = (symbol) =>
  request.get('/stocks/history', { params: { symbol, days: 240 } })

export const searchStocks = (q) =>
  request.get('/stocks/search', { params: { q } })
