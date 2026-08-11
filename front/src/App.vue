<template>
  <div id="app">
    <Header v-if="showHeader" />
    <router-view />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import Header from '@/components/Header.vue'

const route = useRoute()
// 公开页 / 用自己的 bar，Header 仅在 /admin/* 显示
const showHeader = computed(() => route.path.startsWith('/admin'))
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  height: 100%;
}
</style>
