// src/router/index.js
import { createRouter, createWebHistory } from 'vue-router'
import LoginPage from '@/pages/LoginPage.vue'
import HomePage from '@/pages/HomePage.vue'
import StockPage from '@/pages/StockPage.vue'
import Rules from '../components/Rules.vue'
import Favorites from '../components/Favorites.vue'
import HotStocks from '../components/HotStocks.vue'
import StockDailyTable from '../components/StockDailyTable.vue'

const routes = [
  { 
    path: '/login', 
    name: 'Login', 
    component: LoginPage 
},
  {
    path: '/',
    component: LoginPage,
    meta: { requiresAuth: true },   // 需要登录
  },
  {
    path: '/home',
    component: HomePage,
    meta: { requiresAuth: true },   // 需要登录
  },
  {
    path: '/stocks',
    name: 'Stocks',
    component: StockPage,
    meta: { requiresAuth: true }
  } ,
  {
    path: '/rules',
    name: 'Rules',
    component: Rules,
    meta: { requiresAuth: true }
  },
  {
    path: '/favorites',
    name: 'Favorites',
    component: Favorites,
    meta: { requiresAuth: true }
  },
  {
    path: '/hot',
    name: 'HotStocks',
    component: HotStocks,
    meta: { requiresAuth: true }
  },
  {
    path: '/daily',
    name: 'StockDaily',
    component: StockDailyTable,
    meta: { requiresAuth: true }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

//简单的登录守卫
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) {
    next({ path: '/login' })
  } else {
    next()
  }
})

export default router
