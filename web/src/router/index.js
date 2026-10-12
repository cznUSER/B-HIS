// 2026-10-12 P1-H05 登录界面
import { createRouter, createWebHistory } from 'vue-router'
import { isAuthenticated } from '../auth/session.js'
import LoginView from '../views/LoginView.vue'
import HomeView from '../views/HomeView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView, meta: { guest: true } },
    { path: '/', name: 'home', component: HomeView, meta: { requiresAuth: true } },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

router.beforeEach((to) => {
  if (to.meta.requiresAuth && !isAuthenticated.value) {
    return { name: 'login' }
  }
  if (to.meta.guest && isAuthenticated.value) {
    return { name: 'home' }
  }
  return true
})

export default router
