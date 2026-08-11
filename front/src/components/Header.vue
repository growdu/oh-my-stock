<template>
  <el-header height="60px" class="app-header">
    <!-- Logo -->
    <div class="logo">
      <img src="@/assets/logo.png" alt="MyStock" />
      <span>MyStock</span>
    </div>

    <!-- 横向菜单：仅 2 项 -->
    <el-menu
      mode="horizontal"
      :default-active="activePage"
      class="menu desktop-menu"
    >
      <el-menu-item index="/home">
        <router-link to="/home">选股结果</router-link>
      </el-menu-item>
      <el-menu-item index="/rules">
        <router-link to="/rules">规则管理</router-link>
      </el-menu-item>
    </el-menu>

    <!-- 用户区 -->
    <div class="user-actions">
      <span v-if="username">{{ username }}，欢迎</span>
      <el-button class="login-btn" @click="handleLoginLogout">
        {{ token ? '退出登录' : '登录' }}
      </el-button>
    </div>
  </el-header>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()

const token = ref('')
const username = ref('')
const activePage = ref('/home')

onMounted(() => {
  token.value = localStorage.getItem('token') || ''
  username.value = localStorage.getItem('username') || ''
  activePage.value = route.path
})

watch(() => route.path, (newPath) => {
  activePage.value = newPath
})

const handleLoginLogout = () => {
  if (token.value) {
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    token.value = ''
    username.value = ''
    router.push('/login')
  } else {
    router.push('/login')
  }
}
</script>

<style scoped>
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  height: 60px;
  font-family: 'Segoe UI', Roboto, sans-serif;
}

.logo {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 20px;
  font-weight: bold;
}
.logo img {
  height: 36px;
}

.menu {
  flex: 1;
  margin-left: 30px;
}
.menu :deep(.el-menu-item) > a {
  font-weight: 500;
  font-size: 16px;
  padding: 0 12px;
}
.menu :deep(.el-menu-item.is-active > a) {
  border-radius: 4px;
}

.user-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}
.login-btn {
  border-radius: 4px;
  padding: 4px 12px;
  font-size: 14px;
}
</style>
