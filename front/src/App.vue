<template>
  <div id="app">
    <!-- 这里可以放公共头部/导航栏 -->
    <!-- <HeaderComponent /> -->

    <!-- 路由渲染区域 -->
    <router-view></router-view>

    <!-- 这里可以放公共底部 -->
    <!-- <FooterComponent /> -->
  </div>
</template>

<script setup>
import LoginPage from '@/pages/LoginPage.vue'
</script>
<style>
/* 全局样式 */
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  height: 100%;
}
</style>