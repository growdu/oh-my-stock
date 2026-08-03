import request from '../request'

export const getFavorites = () => request.get('/user/favorites')

export const addFavorite = (symbol) =>
  request.post('/user/favorites', { symbol })

// 后端支持 DELETE /user/favorites/symbol/:symbol
export const removeFavorite = (symbol) =>
  request.delete(`/user/favorites/symbol/${encodeURIComponent(symbol)}`)
