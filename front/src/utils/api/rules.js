import request from '../request'

export const getRules = () => request.get('/user/rules')

export const addRule = (rule_name, rule_expression) =>
  request.post('/user/rules', { rule_name, rule_expression })

export const updateRule = (id, rule_name, rule_expression) =>
  request.put(`/user/rules/${id}`, { rule_name, rule_expression })

export const deleteRule = (id) => request.delete(`/user/rules/${id}`)

export const runRule = (id) => request.post(`/user/rules/${id}/run`, {})

export const previewRule = (rule_name, rule_expression) =>
  request.post('/user/rules/preview', { rule_name, rule_expression })
