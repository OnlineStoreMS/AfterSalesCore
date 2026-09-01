import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '../layouts/AdminLayout.vue'
import {redirectToPortal, ensureSession, clearToken} from '../utils/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/auth/callback',
      name: 'AuthCallback',
      component: () => import('../views/AuthCallback.vue'),
      meta: { public: true },
    },
    {
      path: '/auth/logout',
      name: 'AuthLogout',
      component: () => import('../views/AuthLogout.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: AdminLayout,
      redirect: '/dashboard',
      children: [
        { path: 'dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue'), meta: { title: '工作台' } },
        {
          path: 'unboxing',
          name: 'UnboxingList',
          component: () => import('../views/record/RecordList.vue'),
          meta: { title: '开箱记录', recordType: 'unboxing' },
        },
        {
          path: 'unboxing/create',
          name: 'UnboxingCreate',
          component: () => import('../views/record/RecordCreate.vue'),
          meta: { title: '录制开箱', recordType: 'unboxing' },
        },
        {
          path: 'unboxing/:id',
          name: 'UnboxingDetail',
          component: () => import('../views/record/RecordDetail.vue'),
          meta: { title: '开箱详情', recordType: 'unboxing' },
        },
        {
          path: 'packing',
          name: 'PackingList',
          component: () => import('../views/record/RecordList.vue'),
          meta: { title: '打包记录', recordType: 'packing' },
        },
        {
          path: 'packing/create',
          name: 'PackingCreate',
          component: () => import('../views/record/RecordCreate.vue'),
          meta: { title: '录制打包', recordType: 'packing' },
        },
        {
          path: 'packing/:id',
          name: 'PackingDetail',
          component: () => import('../views/record/RecordDetail.vue'),
          meta: { title: '打包详情', recordType: 'packing' },
        },
        {
          path: 'shops',
          name: 'Shops',
          component: () => import('../views/shop/ShopList.vue'),
          meta: { title: '店铺管理' },
        },
        {
          path: 'returns',
          name: 'Returns',
          component: () => import('../views/shop/ReturnList.vue'),
          meta: { title: '退回管理' },
        },
        {
          path: 'returns/shipped-success',
          name: 'ShippedRefunds',
          component: () => import('../views/shop/ShippedRefundList.vue'),
          meta: { title: '已发货退款成功' },
        },
        {
          path: 'returns/intercept',
          name: 'InterceptOrders',
          component: () => import('../views/shop/InterceptList.vue'),
          meta: { title: '需商家拦截快递' },
        },
        {
          path: 'service-orders',
          name: 'ServiceOrders',
          component: () => import('../views/shop/ServiceOrderList.vue'),
          meta: { title: '服务工单' },
        },
        {
          path: 'shops/:id',
          name: 'ShopWorkbench',
          component: () => import('../views/shop/ShopWorkbench.vue'),
          meta: { title: '店铺售后工作台' },
        },
        {
          path: 'notifications',
          name: 'Notifications',
          component: () => import('../views/notification/NotificationSettings.vue'),
          meta: { title: '通知管理' },
        },
        {
          path: 'edge-devices',
          name: 'EdgeDevices',
          component: () => import('../views/edge/EdgeDeviceList.vue'),
          meta: { title: '录制端管理' },
        },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true
  const ok = await ensureSession()
  if (!ok) {
    clearToken()
    redirectToPortal()
    return false
  }
  return true
})

export default router
