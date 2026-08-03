import request from '../request'

export const listTargetStocks = (ruleName) =>
  request.get('/target-stocks', { params: ruleName ? { rule_name: ruleName } : {} })
