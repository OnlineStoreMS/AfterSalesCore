<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { HomeFilled, VideoCamera, Box, Search, Monitor, Shop, Bell, RefreshLeft, Tickets } from '@element-plus/icons-vue'
import { fetchNavCounts, type NavCounts } from '../api/shop'
import { useSessionStore } from '../stores/session'

const route = useRoute()
const router = useRouter()
const sessionStore = useSessionStore()
const collapsed = defineModel<boolean>('collapsed', { default: false })
const counts = ref<NavCounts>({
  pendingServiceOrders: 0,
  interceptOrders: 0,
  ticketTotal: 0,
  buyerReturnPickup: 0,
  reviewShippedRefund: 0,
})
let pollTimer = 0

const activeMenu = computed(() => {
  if (route.path.startsWith('/shops/await-pickup')) return '/shops/await-pickup'
  if (route.path.startsWith('/shops/shipped-refund')) return '/shops/shipped-refund'
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

function badgeText(n: number) {
  if (!n) return ''
  return n > 99 ? '99+' : String(n)
}

async function loadCounts() {
  try {
    counts.value = await fetchNavCounts()
  } catch {
    /* keep last */
  }
}

function navigate(path: string) {
  router.push(path)
}

onMounted(() => {
  loadCounts()
  pollTimer = window.setInterval(loadCounts, 30000)
})
onUnmounted(() => {
  if (pollTimer) window.clearInterval(pollTimer)
})
watch(() => sessionStore.session?.tenant.id, () => {
  loadCounts()
})
watch(() => route.path, () => {
  loadCounts()
})
</script>

<template>
  <aside class="sidebar" :class="{ collapsed }">
    <div class="logo">{{ logoText }}</div>
    <el-menu
      :default-active="activeMenu"
      :default-openeds="['shops', 'returns']"
      :collapse="collapsed"
      background-color="#001529"
      text-color="#ffffffa6"
      active-text-color="#fff"
    >
      <el-menu-item index="/dashboard" @click="navigate('/dashboard')">
        <el-icon><HomeFilled /></el-icon>
        <span>工作台</span>
      </el-menu-item>
      <el-sub-menu index="shops">
        <template #title>
          <el-icon><Shop /></el-icon>
          <span>店铺管理</span>
          <span
            v-if="counts.ticketTotal"
            class="nav-badge"
            :title="`全部店铺售后单 ${counts.ticketTotal}`"
          >{{ badgeText(counts.ticketTotal) }}</span>
        </template>
        <el-menu-item index="/shops" @click="navigate('/shops')">店铺列表</el-menu-item>
        <el-menu-item index="/shops/await-pickup" @click="navigate('/shops/await-pickup')">
          待取件
          <span
            v-if="counts.buyerReturnPickup"
            class="nav-badge"
            :title="`待取件 ${counts.buyerReturnPickup}`"
          >{{ badgeText(counts.buyerReturnPickup) }}</span>
        </el-menu-item>
        <el-menu-item index="/shops/shipped-refund" @click="navigate('/shops/shipped-refund')">
          已发货退款
          <span
            v-if="counts.reviewShippedRefund"
            class="nav-badge"
            :title="`已发货退款 ${counts.reviewShippedRefund}`"
          >{{ badgeText(counts.reviewShippedRefund) }}</span>
        </el-menu-item>
      </el-sub-menu>
      <el-sub-menu index="returns">
        <template #title>
          <el-icon><RefreshLeft /></el-icon>
          <span>退回管理</span>
          <span
            v-if="counts.interceptOrders"
            class="nav-badge"
            :title="`需商家拦截 ${counts.interceptOrders}`"
          >{{ badgeText(counts.interceptOrders) }}</span>
        </template>
        <el-menu-item index="/returns" @click="navigate('/returns')">退回件</el-menu-item>
        <el-menu-item index="/returns/shipped-success" @click="navigate('/returns/shipped-success')">
          已发货退款成功
        </el-menu-item>
        <el-menu-item index="/returns/intercept" @click="navigate('/returns/intercept')">
          需商家拦截快递
          <span
            v-if="counts.interceptOrders"
            class="nav-badge"
            :title="`需商家拦截 ${counts.interceptOrders}`"
          >{{ badgeText(counts.interceptOrders) }}</span>
        </el-menu-item>
      </el-sub-menu>
      <el-menu-item index="/service-orders" @click="navigate('/service-orders')">
        <el-icon><Tickets /></el-icon>
        <span>服务工单</span>
        <span
          v-if="counts.pendingServiceOrders"
          class="nav-badge"
          :title="`待处理工单 ${counts.pendingServiceOrders}`"
        >{{ badgeText(counts.pendingServiceOrders) }}</span>
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
.sidebar :deep(.el-menu-item),
.sidebar :deep(.el-sub-menu__title) {
  position: relative;
}
.nav-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  margin-left: 8px;
  border-radius: 9px;
  background: #f56c6c;
  color: #fff;
  font-size: 11px;
  line-height: 1;
  font-weight: 600;
}
.sidebar.collapsed .nav-badge {
  position: absolute;
  top: 6px;
  right: 6px;
  margin-left: 0;
  min-width: 16px;
  height: 16px;
  font-size: 10px;
}
</style>
