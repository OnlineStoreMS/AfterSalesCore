import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '../layouts/AdminLayout.vue'
import { getToken, redirectToPortal, ensureSession, clearToken } from '../utils/auth'

const APP_TITLE = 'AfterSalesCore - 售后管理'

const router = createRouter({
  history: createWebHistory(),
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
        { path: 'unboxing', name: 'UnboxingList', component: () => import('../views/unboxing/UnboxingList.vue'), meta: { title: '开箱记录' } },
        { path: 'unboxing/create', name: 'UnboxingCreate', component: () => import('../views/unboxing/UnboxingCreate.vue'), meta: { title: '录制开箱' } },
        { path: 'unboxing/:id', name: 'UnboxingDetail', component: () => import('../views/unboxing/UnboxingDetail.vue'), meta: { title: '开箱详情' } },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true
  if (!getToken()) {
    redirectToPortal()
    return false
  }
  const ok = await ensureSession()
  if (!ok) {
    clearToken()
    redirectToPortal()
    return false
  }
  return true
})

router.afterEach((to) => {
  const page = to.meta.title as string | undefined
  document.title = page ? `${page} - ${APP_TITLE}` : APP_TITLE
})

export default router
