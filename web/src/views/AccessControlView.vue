<script setup>
// 2026-10-14 P1-H05-RBAC 用户、角色、权限统一管理工作区。
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { session } from '../auth/session.js'
import { accessDeniedTitle, canOpenAccessRoute, visibleAccessTabs } from '../master/accessControlPolicy.js'
import MasterSymbol from './master/MasterSymbol.vue'
import './access/access-control.css'
import '../views/master/master-design.css'

const route = useRoute()
const tabs = computed(() => visibleAccessTabs(session.user))
const currentAllowed = computed(() => canOpenAccessRoute(session.user, route.name))
const deniedTitle = computed(() => accessDeniedTitle(route.name))
</script>

<template>
  <section class="patient-page master-page access-page" aria-labelledby="access-title">
    <p class="crumb">
      <RouterLink :to="{ name: 'home' }">首页</RouterLink><span aria-hidden="true"> / </span>
      <span>系统设置</span><span aria-hidden="true"> / </span><span>用户与权限</span>
    </p>
    <header class="master-page-header access-page-header">
      <div>
        <p class="master-eyebrow">ACCOUNT &amp; AUTHORIZATION / 用户与权限</p>
        <h1 id="access-title">用户与权限</h1>
        <p class="master-page-description">按华易通的用户、角色、模块范围和权限例外维护工作账号。</p>
      </div>
      <div class="master-access-tag"><span></span>{{ currentAllowed ? '权限已验证' : '权限不足' }}</div>
    </header>
    <p class="access-scope-note">
      <MasterSymbol name="users" />
      <span>角色权限采用并集，用户级明确拒绝优先。科室范围用于限制数据可见性；临床二次认证仍保持关闭，工卡模式仅作预留。</span>
    </p>
    <nav class="master-tabs access-tabs" aria-label="用户与权限分页">
      <RouterLink v-for="tab in tabs" :key="tab.name" class="master-tab" :class="{ 'is-active': route.name === tab.name }" :to="{ name: tab.name }" :aria-current="route.name === tab.name ? 'page' : undefined">
        <MasterSymbol :name="tab.icon" /><span>{{ tab.label }}</span>
      </RouterLink>
    </nav>
    <div v-if="!currentAllowed" class="empty-panel" data-testid="access-denied">
      <h2 class="empty-lead">{{ deniedTitle }}</h2>
      <p class="muted">当前账号无权查看这个管理页面。</p>
      <RouterLink class="button button-ghost empty-back" :to="{ name: 'home' }">返回首页</RouterLink>
    </div>
    <router-view v-if="currentAllowed" v-slot="{ Component, route: viewRoute }">
      <Transition name="master-page-change" mode="out-in"><component :is="Component" :key="viewRoute.name" /></Transition>
    </router-view>
  </section>
</template>
