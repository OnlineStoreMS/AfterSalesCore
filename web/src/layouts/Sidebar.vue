<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { HomeFilled, VideoCamera, Box, Search, Monitor, Shop, Bell, RefreshLeft, Tickets } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const collapsed = defineModel<boolean>('collapsed', { default: false })

const activeMenu = computed(() => {
  if (route.path.startsWith('/shops')) return '/shops'
  if (route.path.startsWith('/returns/intercept')) return '/returns/intercept'
  if (route.path.startsWith('/returns/shipped-success')) return '/returns/shipped-success'
  if (route.path.startsWith('/returns')) return '/returns'
  if (route.path.startsWith('/service-orders')) return '/service-orders'
  if (route.path.startsWith('/notifications')) return '/notifications'
  return route.path
})

const menuItems = [
  { path: '/notifications', title: '通知管理', icon: Bell },
  { path: '/unboxing/create', title: '录制开箱', icon: VideoCamera },
  { path: '/packing/create', title: '录制打包', icon: Box },
  { path: '/unboxing', title: '开箱记录', icon: Search },
  { path: '/packing', title: '打包记录', icon: Search },
  { path: '/edge-devices', title: '录制端管理', icon: Monitor },
]

const logoText = computed(() => (collapsed.value ? 'AS' : '售后中心'))

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
      :default-openeds="['returns']"
      :collapse="collapsed"
      background-color="#001529"
      text-color="#ffffffa6"
      active-text-color="#fff"
    >
      <el-menu-item index="/dashboard" @click="navigate('/dashboard')">
        <el-icon><HomeFilled /></el-icon>
        <span>工作台</span>
      </el-menu-item>
      <el-menu-item index="/shops" @click="navigate('/shops')">
        <el-icon><Shop /></el-icon>
        <span>店铺管理</span>
      </el-menu-item>
      <el-sub-menu index="returns">
        <template #title>
          <el-icon><RefreshLeft /></el-icon>
          <span>退回管理</span>
        </template>
        <el-menu-item index="/returns" @click="navigate('/returns')">退回件</el-menu-item>
        <el-menu-item index="/returns/shipped-success" @click="navigate('/returns/shipped-success')">
          已发货退款成功
        </el-menu-item>
        <el-menu-item index="/returns/intercept" @click="navigate('/returns/intercept')">
          需商家拦截快递
        </el-menu-item>
      </el-sub-menu>
      <el-menu-item index="/service-orders" @click="navigate('/service-orders')">
        <el-icon><Tickets /></el-icon>
        <span>服务工单</span>
      </el-menu-item>
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
.sidebar :deep(.el-sub-menu .el-menu-item) {
  min-width: 0;
}
</style>
