import { createRouter, createWebHistory } from 'vue-router'
import LoginPage from '@/pages/LoginPage.vue'
import Results   from '@/pages/Results.vue'
import Rules     from '@/components/Rules.vue'

const routes = [
  { path: '/login', name: 'Login', component: LoginPage },
  { path: '/',      redirect: '/home' },
  { path: '/home',  name: 'Results', component: Results, meta: { requiresAuth: true } },
  { path: '/rules', name: 'Rules',   component: Rules,   meta: { requiresAuth: true } },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) {
    next({ path: '/login', query: { redirect: to.fullPath } })
  } else if (to.path === '/login' && token) {
    next('/home')
  } else {
    next()
  }
})

export default router
