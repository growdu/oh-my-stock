<template>
  <div id="app">
    <router-view />
  </div>
</template>

<script setup>
// Header 已移除：极简页面，无顶部导航
import { useRoute } from 'vue-router'
const route = useRoute()  // 占位以保持 router 上下文
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  height: 100%;
}
</style>
