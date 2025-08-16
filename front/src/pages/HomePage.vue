<template>
  <el-container style="height: 100vh">
    <!-- 左侧侧边栏 -->
    <el-aside :width="collapsed ? '64px' : '220px'" class="sidebar">
      <div class="logo" v-if="!collapsed">📈 股票管理</div>
      <div class="logo-collapsed" v-else>📈</div>
      <el-menu
        :default-active="activeMenu"
        class="el-menu-vertical-demo"
        background-color="#1f2d3d"
        text-color="#bfcbd9"
        active-text-color="#409EFF"
        @select="handleMenuSelect"
        :collapse="collapsed"
      >
        <el-menu-item index="allStocks">全部股票</el-menu-item>
        <el-menu-item index="hotStocks">热点股票</el-menu-item>
        <el-menu-item index="favorites">收藏</el-menu-item>
        <el-menu-item index="rules">规则配置</el-menu-item>
      </el-menu>
    </el-aside>

    <!-- 右侧主内容 -->
    <el-container>
      <el-header class="header">
        <el-button
          icon="el-icon-menu"
          @click="toggleCollapsed"
          class="toggle-btn"
          circle
        ></el-button>
        <span class="header-title">{{ currentTitle }}</span>
      </el-header>
      <el-main class="main-content">
        <component :is="currentComponent" />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import AllStocks from '@/components/AllStocks.vue'
import HotStocks from '@/components/HotStocks.vue'
import Favorites from '@/components/Favorites.vue'
import Rules from '@/components/Rules.vue'

const activeMenu = ref('allStocks')
const collapsed = ref(false)
const componentsMap = { allStocks: AllStocks, hotStocks: HotStocks, favorites: Favorites, rules: Rules }
const titleMap = { allStocks: '全部股票', hotStocks: '热点股票', favorites: '收藏股票', rules: '个人规则配置' }
const currentComponent = computed(() => componentsMap[activeMenu.value])
const currentTitle = computed(() => titleMap[activeMenu.value])

function handleMenuSelect(key) { activeMenu.value = key }
function toggleCollapsed() { collapsed.value = !collapsed.value }

const handleResize = () => { collapsed.value = window.innerWidth < 768 }
onMounted(() => { handleResize(); window.addEventListener('resize', handleResize) })
onBeforeUnmount(() => { window.removeEventListener('resize', handleResize) })
</script>

<style scoped>
.sidebar { background-color: #1f2d3d; color: #fff; display:flex; flex-direction: column; transition: width 0.3s; }
.logo { height: 60px; font-size: 18px; color: #409EFF; font-weight:bold; display:flex; align-items:center; justify-content:center; border-bottom:1px solid #2e3a4b; margin-bottom:10px; }
.logo-collapsed { height:60px; display:flex; align-items:center; justify-content:center; font-size:20px; border-bottom:1px solid #2e3a4b; margin-bottom:10px; }
.header { padding: 10px 20px; font-size: 20px; font-weight: bold; background-color:#f5f7fa; border-bottom:1px solid #ebeef5; display:flex; align-items:center; }
.toggle-btn { margin-right:15px; }
.header-title { font-weight:bold; }
.main-content { padding:20px; background-color:#f0f2f5; overflow-y:auto; }
</style>
