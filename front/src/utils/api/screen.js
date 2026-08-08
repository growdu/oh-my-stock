import request from "../request"

// Stock screening with multiple filters
export function screenStocks(params) {
  return request.get("/stocks/screen", { params })
}

// Get industry list for dropdown
export function getIndustries() {
  return request.get("/stocks/industries")
}

// Get market type list for dropdown
export function getMarkets() {
  return request.get("/stocks/markets")
}

// Get stock ranking by dimension.
// order 默认 desc；传 "asc" 取最末位（用于跌幅榜）。
export function getStockRanking(rankBy = "change_percent", limit = 20, order = "desc") {
  return request.get("/stocks/ranking", { params: { rank_by: rankBy, limit, order } })
}