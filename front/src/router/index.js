import { createRouter, createWebHistory } from 'vue-router'

import LoginPage  from '@/pages/LoginPage.vue'
import Display    from '@/pages/Display.vue'
import Rules      from '@/components/Rules.vue'
import Picker     from '@/pages/Picker.vue'

const routes = [
  // 首页：进阶精选 Top 3（公开）
  { path: '/',             name: 'Picker',     component: Picker },

  // 全部预设：Display 堆叠卡片（公开）
  { path: '/display',      name: 'Display',    component: Display },

  // 管理后台（仅管规则）
  { path: '/admin/login',  name: 'AdminLogin', component: LoginPage },
  { path: '/admin/rules',  name: 'AdminRules', component: Rules, meta: { requiresAuth: true } },

  // 兜底
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  // 未登录访问管理后台 → 跳管理后台登录页
  if (to.meta.requiresAuth && !token) {
    next({ path: '/admin/login', query: { redirect: to.fullPath } })
  }
  // 已登录访问管理后台登录页 → 直奔 rules
  else if (to.path === '/admin/login' && token) {
    next('/admin/rules')
  }
  else {
    next()
  }
})

export default router
