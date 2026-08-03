import { createRouter, createWebHistory } from 'vue-router'
import LoginPage    from '@/pages/LoginPage.vue'
import HomePage     from '@/pages/HomePage.vue'
import Favorites    from '@/components/Favorites.vue'
import Rules        from '@/components/Rules.vue'
import HotStocks    from '@/components/HotStocks.vue'
import StockDailyTable from '@/components/StockDailyTable.vue'

const routes = [
  { path: '/login', name: 'Login', component: LoginPage },
  { path: '/',      redirect: '/home' },
  { path: '/home',     name: 'Home',     component: HomePage,     meta: { requiresAuth: true } },
  { path: '/hot',      name: 'HotStocks',component: HotStocks,    meta: { requiresAuth: true } },
  { path: '/daily',    name: 'Daily',    component: StockDailyTable, meta: { requiresAuth: true } },
  { path: '/favorites',name: 'Favorites',component: Favorites,    meta: { requiresAuth: true } },
  { path: '/rules',    name: 'Rules',    component: Rules,        meta: { requiresAuth: true } },
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
