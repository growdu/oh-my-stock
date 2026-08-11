// usePresetCache —— 规则命中结果前端缓存
// 用法：
//   const { cache, fetchPage, preloadAll, invalidate } = usePresetCache(8)
//   - 切换规则时 syncSelected() 直接读 cache，秒切换
//   - 首次进入页面调用 preloadAll(presets) 后台预加载 8 个规则
//   - 翻页时只缓存首页（首页是切换场景的高频数据）
//   - invalidate() 清除所有缓存
import { reactive } from 'vue'
import { runPreset } from '@/utils/api/presets'

export function usePresetCache(pageSize = 8) {
  const cache = reactive({})
  // { [ruleId]: { rows, total, tradeDate } }

  const fetchPage = async (ruleId, page) => {
    // 首页 + 已缓存 → 走 cache
    if (page === 1 && cache[ruleId]) {
      return cache[ruleId]
    }
    const res = await runPreset(ruleId, page, pageSize)
    const rows = res.data?.data || []
    const result = {
      rows,
      total: res.data?.total || 0,
      tradeDate: rows[0]?.trade_date || '',
    }
    if (page === 1) cache[ruleId] = result  // 只缓存首页
    return result
  }

  const preloadAll = async (presets) => {
    // 并发请求所有规则的首页（已缓存的跳过）
    const tasks = (presets || [])
      .filter((p) => p && p.id && !cache[p.id])
      .map((p) =>
        fetchPage(p.id, 1).catch(() => null)
      )
    if (tasks.length) await Promise.all(tasks)
  }

  const invalidate = (ruleId) => {
    if (ruleId) {
      delete cache[ruleId]
    } else {
      Object.keys(cache).forEach((k) => delete cache[k])
    }
  }

  return { cache, fetchPage, preloadAll, invalidate }
}
