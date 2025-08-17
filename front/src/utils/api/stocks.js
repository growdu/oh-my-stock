import request from '../request'

export const getStocks = async (page = 1, pageSize = 20) => {
  try {
    const res = await request.get('/stocks/list', {
      params: { page, page_size: pageSize }
    })
    return res.data
  } catch (err) {
    console.error(err)
    return { data: [], total: 0 }
  }
}

// 获取股票列表（分页）
export function fetchStockList(params) {
    return request.get('/api/v1/stocks', { params })
  }
  
  // 模糊搜索（用于 autocomplete）
  export function searchStocks(keyword) {
    return request.get('/api/v1/stocks/search', { params: { q: keyword } })
  }
  
  // 获取单个股票详情
  export function fetchStockDetail(symbol) {
    return request.get(`/api/v1/stocks/${symbol}`)
  }