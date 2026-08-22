<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { HomeFilled, VideoCamera, Box, Search, Monitor, Shop, Bell, Warning } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const collapsed = defineModel<boolean>('collapsed', { default: false })

const activeMenu = computed(() => {
  if (route.path.startsWith('/shops')) return '/shops'
  if (route.path.startsWith('/notifications')) return '/notifications'
  if (route.path.startsWith('/plugin-debug-logs')) return '/plugin-debug-logs'
  return route.path
})

const menuItems = [
  { path: '/dashboard', title: '工作台', icon: HomeFilled },
  { path: '/shops', title: '店铺管理', icon: Shop },
  { path: '/notifications', title: '通知管理', icon: Bell },
  { path: '/plugin-debug-logs', title: '插件诊断日志', icon: Warning },
  { path: '/unboxing/create', title: '录制开箱', icon: VideoCamera },
  { path: '/packing/create', title: '录制打包', icon: Box },
  { path: '/unboxing', title: '开箱记录', icon: Search },
  { path: '/packing', title: '打包记录', icon: Search },
  { path: '/edge-devices', title: '录制端管理', icon: Monitor },
]

const logoText = computed(() => (collapsed.value ? 'AS' : '售后管理'))

function navigate(path: string) {
  router.push(path)
}

watch(() => route.path, () => {})
</script>

<template>
  <aside class="sidebar" :class="{ collapsed }">
    <div class="logo">{{ logoText }}</div>
    <el-menu
      :default-active="activeMenu"
      :collapse="collapsed"
      background-color="#001529"
      text-color="#ffffffa6"
      active-text-color="#fff"
    >
      <el-menu-item
        v-for="item in menuItems"
        :key="item.path"
        :index="item.path"
        @click="navigate(item.path)"
      >
        <el-icon><component :is="item.icon" /></el-icon>
        <span>{{ item.title }}</span>
      </el-menu-item>
    </el-menu>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 220px;
  background: #001529;
  transition: width 0.2s;
  flex-shrink: 0;
}
.sidebar.collapsed {
  width: 64px;
}
.logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 600;
  font-size: 16px;
  border-bottom: 1px solid #ffffff14;
}
.sidebar :deep(.el-menu) {
  border-right: none;
}
</style>
