// 2026-10-12 P1-H05 登录界面
// 2026-10-12 P1-H05-HOME 登录后进入门诊工作台
// 2026-10-12 P1-H05-HOME-DASH 默认页改为首页看板，门诊工作区仍走 /app/outpatient/*
// 2026-10-12 P1-H05-PATIENT-UI 建档页替换空态，其余模块仍走空态
// 2026-10-12 P1-H05-REGISTER-UI 挂号页替换空态
// 2026-10-12 P1-H05-CHARGE-UI 收费页对接 /charges；Mock 仅在 VITE_USE_MOCK=true
// 2026-10-13 P1-H05-MASTER-UI 系统设置 → 基础数据（Mock）
import { createRouter, createWebHistory } from 'vue-router'
import { isAuthenticated, session, setSessionExpiredHandler } from '../auth/session.js'
import { rememberReturnTo } from './returnTo.js'
import AppShell from '../components/AppShell.vue'
import LoginView from '../views/LoginView.vue'
import DashboardView from '../views/DashboardView.vue'
import OutpatientModuleView from '../views/OutpatientModuleView.vue'
import WorkbenchView from '../views/WorkbenchView.vue'
import PatientView from '../views/PatientView.vue'
import PatientSearchView from '../views/PatientSearchView.vue'
import { canEnter, findModule } from '../workbench/catalog.js'
import MasterView from '../views/MasterView.vue'
import MasterHome from '../views/master/MasterHome.vue'
import DepartmentsPanel from '../views/master/DepartmentsPanel.vue'
import DoctorsPanel from '../views/master/DoctorsPanel.vue'
import CatalogPanel from '../views/master/CatalogPanel.vue'
import InventoryPanel from '../views/master/InventoryPanel.vue'
import DictsPanel from '../views/master/DictsPanel.vue'
import AccessControlView from '../views/AccessControlView.vue'
import UsersAccessPanel from '../views/access/UsersAccessPanel.vue'
import RolesPanel from '../views/access/RolesPanel.vue'
import PermissionsPanel from '../views/access/PermissionsPanel.vue'
import UserOperationsLogPanel from '../views/access/UserOperationsLogPanel.vue'
import PasswordChangeView from '../views/PasswordChangeView.vue'

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior(to, from, savedPosition) {
    return savedPosition || { left: 0, top: 0 }
  },
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
          path: 'archive/patients/create',
          name: 'patient-create',
          component: PatientView,
          meta: { title: '患者建档', patientAccess: true },
        },
        {
          path: 'archive/patients/search',
          name: 'patient-search',
          component: PatientSearchView,
          meta: { title: '患者查询', patientAccess: true },
        },
        {
          path: 'outpatient/patient',
          name: 'legacy-outpatient-patient',
          redirect: (to) => ({ name: 'patient-create', query: to.query, hash: to.hash }),
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
        {
          path: 'change-password',
          name: 'password-change',
          component: PasswordChangeView,
          meta: { title: '修改密码' },
        },
        {
          path: 'settings/users',
          component: AccessControlView,
          meta: { title: '用户与权限' },
          children: [
            { path: '', name: 'access-users', component: UsersAccessPanel, meta: { title: '用户账号' } },
            { path: 'roles', name: 'access-roles', component: RolesPanel, meta: { title: '角色与范围' } },
            { path: 'permissions', name: 'access-permissions', component: PermissionsPanel, meta: { title: '权限目录' } },
            { path: 'logs', name: 'access-user-logs', component: UserOperationsLogPanel, meta: { title: '用户操作日志' } },
          ],
        },
        {
          path: 'settings/master',
          component: MasterView,
          meta: { title: '基础数据' },
          children: [
            { path: '', name: 'master', component: MasterHome, meta: { title: '基础数据' } },
            { path: 'departments', name: 'master-departments', component: DepartmentsPanel, meta: { title: '科室' } },
            { path: 'doctors', name: 'master-doctors', component: DoctorsPanel, meta: { title: '医生' } },
            { path: 'catalog', name: 'master-catalog', component: CatalogPanel, meta: { title: '收费/药品目录' } },
            { path: 'inventory', name: 'master-inventory', component: InventoryPanel, meta: { title: '药品库存' } },
            { path: 'dicts', name: 'master-dicts', component: DictsPanel, meta: { title: '基础字典' } },
            // Keep the legacy基础数据 URL as a compatibility entry point for the full access workspace.
            { path: 'users', name: 'master-users', redirect: { name: 'access-users' }, meta: { title: '用户账号' } },
          ],
        },
      ],
    },
    { path: '/', redirect: { name: 'home' } },
    { path: '/:pathMatch(.*)*', redirect: { name: 'home' } },
  ],
})

router.beforeEach((to) => {
  const needsAuth = to.matched.some((record) => record.meta.requiresAuth)
  if (needsAuth && !isAuthenticated.value) {
    const redirect = rememberReturnTo(to.fullPath)
    return { name: 'login', query: redirect ? { redirect } : {} }
  }
  if (isAuthenticated.value && session.user?.mustChangePassword && to.name !== 'password-change') {
    return { name: 'password-change' }
  }
  if (to.name === 'password-change' && !session.user?.mustChangePassword) return { name: 'home' }
  if (to.matched.some((record) => record.meta.patientAccess) && !canEnter(findModule('patient'), session.user)) {
    return { name: 'home' }
  }
  // beforeEnter is not rerun for module param changes on the same route.
  if (to.name === 'outpatient-module' && !findModule(String(to.params.module || ''))) return { name: 'home' }
  if (to.meta.guest && isAuthenticated.value) return { name: 'home' }
  return true
})

setSessionExpiredHandler(() => {
  const to = router.currentRoute.value
  if (!to.matched.some((record) => record.meta.requiresAuth)) return
  const redirect = rememberReturnTo(to.fullPath)
  void router.replace({ name: 'login', query: { ...(redirect ? { redirect } : {}), expired: '1' } })
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
