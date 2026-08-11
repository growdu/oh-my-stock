import { createRouter, createWebHistory } from 'vue-router'

import LoginPage  from '@/pages/LoginPage.vue'
import Display    from '@/pages/Display.vue'
import Results    from '@/pages/Results.vue'
import Rules      from '@/components/Rules.vue'

const routes = [
  // 公开展示页（无需登录）
  { path: '/',             name: 'Display', component: Display },

  // 管理后台（需登录）
  { path: '/admin',        redirect: '/admin/results' },
  { path: '/admin/login',  name: 'AdminLogin', component: LoginPage },
  { path: '/admin/results', name: 'AdminResults', component: Results, meta: { requiresAuth: true } },
  { path: '/admin/rules',   name: 'AdminRules',   component: Rules,   meta: { requiresAuth: true } },

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
  // 已登录访问管理后台登录页 → 直奔 results
  else if (to.path === '/admin/login' && token) {
    next('/admin/results')
  }
  else {
    next()
  }
})

export default router
