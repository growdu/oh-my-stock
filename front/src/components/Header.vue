<template>
  <el-header height="60px" class="app-header">
    <!-- Logo -->
    <div class="logo">
      <img src="@/assets/logo.png" alt="MyStock" />
      <span>MyStock</span>
      <el-tag v-if="isAdmin" type="warning" size="small" effect="dark" class="mode-tag">管理后台</el-tag>
    </div>

    <!-- 管理后台横向菜单 -->
    <el-menu
      v-if="isAdmin"
      mode="horizontal"
      :default-active="activePage"
      class="menu desktop-menu"
    >
      <el-menu-item index="/admin/results">
        <router-link to="/admin/results">选股结果</router-link>
      </el-menu-item>
      <el-menu-item index="/admin/rules">
        <router-link to="/admin/rules">规则管理</router-link>
      </el-menu-item>
    </el-menu>

    <!-- 公开展示页占位 -->
    <div v-else class="menu-placeholder">
      <span class="placeholder-text">每日公开展示</span>
    </div>

    <!-- 用户区 -->
    <div class="user-actions">
      <template v-if="isAdmin">
        <span v-if="username">{{ username }}，欢迎</span>
        <el-button class="login-btn" @click="handleLogout">退出登录</el-button>
      </template>
      <template v-else>
        <el-button class="login-btn" type="primary" plain @click="$router.push('/admin')">管理入口</el-button>
      </template>
    </div>
  </el-header>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()

const token = ref('')
const username = ref('')
const activePage = ref('/admin/results')

const isAdmin = computed(() => route.path.startsWith('/admin') && route.path !== '/admin/login')

onMounted(() => {
  token.value = localStorage.getItem('token') || ''
  username.value = localStorage.getItem('username') || ''
  activePage.value = route.path
})

watch(() => route.path, (newPath) => {
  activePage.value = newPath
})

const handleLogout = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  token.value = ''
  username.value = ''
  router.push('/admin/login')
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
.mode-tag {
  margin-left: 4px;
  font-weight: 500;
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

.menu-placeholder {
  flex: 1;
  margin-left: 30px;
  display: flex;
  align-items: center;
}
.placeholder-text {
  font-size: 14px;
  color: #909399;
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
