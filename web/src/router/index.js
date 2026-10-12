// 2026-10-12 P1-H05 登录界面
// 2026-10-12 P1-H05-HOME 登录后进入门诊工作台
// 2026-10-12 P1-H05-HOME-DASH 默认页改为首页看板，六步仍走 /app/outpatient/*
// 2026-10-12 P1-H05-PATIENT-UI 建档页替换空态，其余模块仍走空态
// 2026-10-12 P1-H05-REGISTER-UI 挂号页替换空态
import { createRouter, createWebHistory } from 'vue-router'
import { isAuthenticated } from '../auth/session.js'
import AppShell from '../components/AppShell.vue'
import LoginView from '../views/LoginView.vue'
import DashboardView from '../views/DashboardView.vue'
import OutpatientModuleView from '../views/OutpatientModuleView.vue'
import WorkbenchView from '../views/WorkbenchView.vue'
import { findModule } from '../workbench/catalog.js'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView, meta: { guest: true, title: '登录' } },
    {
      path: '/app',
      component: AppShell,
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'home',
          component: DashboardView,
          meta: { title: '工作台' },
        },
        {
          path: 'outpatient',
          name: 'workbench',
          component: WorkbenchView,
          meta: { title: '门诊业务' },
        },
        {
          path: 'outpatient/:module',
          name: 'outpatient-module',
          component: OutpatientModuleView,
          beforeEnter: (to) => {
            if (!findModule(String(to.params.module || ''))) return { name: 'home' }
            return true
          },
        },
      ],
    },
    { path: '/', redirect: { name: 'home' } },
    { path: '/:pathMatch(.*)*', redirect: { name: 'home' } },
  ],
})

router.beforeEach((to) => {
  const needsAuth = to.matched.some((record) => record.meta.requiresAuth)
  if (needsAuth && !isAuthenticated.value) return { name: 'login' }
  if (to.meta.guest && isAuthenticated.value) return { name: 'home' }
  return true
})

router.afterEach((to) => {
  let title = typeof to.meta.title === 'string' ? to.meta.title : 'B-HIS'
  if (to.name === 'outpatient-module') {
    const mod = findModule(String(to.params.module || ''))
    if (mod) title = mod.name
  }
  document.title = `${title} · B-HIS`
})

export default router
