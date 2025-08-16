// src/router/index.js
import { createRouter, createWebHistory } from 'vue-router'
import LoginPage from '@/pages/LoginPage.vue'
import HomePage from '@/pages/HomePage.vue'
import StockPage from '@/pages/StockPage.vue'

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
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

//简单的登录守卫
router.beforeEach((to, from, next) => {
  const isLoggedIn = localStorage.getItem('token')
  if (to.meta.requiresAuth && !isLoggedIn) {
    next('/login')
  } else {
    next()
  }
})

export default router
