import request from '../request'
import { ElMessage } from 'element-plus'

const getUserId = () => localStorage.getItem('user_id') || ''

export const getRules = async () => {
  try {
    const res = await request.get('/user/rules', { params: { user_id: getUserId() } })
    if (res.data && Array.isArray(res.data.data)) {
      return res.data.data.map(r => ({
        ...r,
        rule_expressionStr: r.rule_expression ? JSON.stringify(r.rule_expression) : '{}'
      }))
    }
    return []
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '获取规则失败')
    return []
  }
}

export const addRule = async (rule_name, rule_expression) => {
  try {
    await request.post('/user/rules', { rule_name, rule_expression, user_id: getUserId() })
    ElMessage.success('规则添加成功')
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '添加规则失败')
  }
}

export const updateRule = async (id, rule_name, rule_expression) => {
  try {
    await request.put(`/user/rules/${id}`, { rule_name, rule_expression, user_id: getUserId() })
    ElMessage.success('规则更新成功')
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '更新规则失败')
  }
}

export const deleteRule = async (id) => {
  try {
    await request.delete(`/user/rules/${id}`, { data: { user_id: getUserId() } })
    ElMessage.success('规则删除成功')
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '删除规则失败')
  }
}
