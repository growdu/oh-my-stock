import request from '../request'
import { ElMessage } from 'element-plus'

const getUserId = () => localStorage.getItem('user_id') || ''

export const getFavorites = async () => {
  try {
    const res = await request.get('/user/favorites', { params: { user_id: getUserId() } })
    return res.data
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '获取收藏失败')
    return []
  }
}

export const addFavorite = async (symbol) => {
  try {
    await request.post('/user/favorites', { symbol, user_id: getUserId() })
    ElMessage.success('收藏成功')
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '收藏失败')
  }
}

export const removeFavorite = async (symbol) => {
  try {
    await request.delete(`/user/favorites/${symbol}`, { data: { user_id: getUserId() } })
    ElMessage.success('取消收藏成功')
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '取消收藏失败')
  }
}
