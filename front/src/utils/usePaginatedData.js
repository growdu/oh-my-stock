import { ref, watch } from 'vue'

export function usePaginatedData(apiFn) {
  const data = ref([])
  const total = ref(0)
  const loading = ref(false)
  const query = ref('')
  const page = ref(1)
  const pageSize = ref(20)

  const fetchData = async () => {
    loading.value = true
    try {
      const res = await apiFn({
        page: page.value,
        pageSize: pageSize.value,
        query: query.value
      })
      data.value = res.data || []
      total.value = res.total || 0
    } catch (err) {
      console.error('加载数据失败', err)
    } finally {
      loading.value = false
    }
  }

  // 自动监听参数变化重新请求
  watch([query, page, pageSize], fetchData, { immediate: true })

  return {
    data,
    total,
    loading,
    query,
    page,
    pageSize,
    fetchData,
  }
}
