<script setup>
// 2026-10-13 P1-H05-MASTER-UI 基础数据壳：二级页签。仅系统账号可看数据。
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useMock } from '../api/config.js'
import { session } from '../auth/session.js'
import { canMaintainMaster } from '../master/access.js'

const route = useRoute()
const allowed = computed(() => canMaintainMaster(session.user))
const mockMode = useMock()

const tabs = [
  { name: 'master', label: '总览' },
  { name: 'master-departments', label: '科室' },
  { name: 'master-doctors', label: '医生' },
  { name: 'master-catalog', label: '收费/药品目录' },
  { name: 'master-inventory', label: '药品库存' },
  { name: 'master-dicts', label: '基础字典' },
  { name: 'master-users', label: '用户账号' },
]

function tabActive(name) {
  return route.name === name
}
</script>

<template>
  <section class="patient-page master-page" aria-labelledby="master-title">
    <p class="crumb">
      <RouterLink :to="{ name: 'home' }">首页</RouterLink>
      <span aria-hidden="true"> / </span>
      <span>系统设置</span>
      <span aria-hidden="true"> / </span>
      <span>基础数据</span>
    </p>

    <header class="panel-head">
      <h1 id="master-title">基础数据</h1>
    </header>
    <p class="field-hint master-lead">演示账号 system / 1。这里维护科室、医生、目录、库存、号别和账号。完整的用户与权限仍是二期。</p>
    <p v-if="allowed && !mockMode" class="field-hint" data-testid="master-live">当前连接基础数据接口。登录仍用 system / 1。</p>

    <nav class="master-tabs" aria-label="基础数据分页" data-testid="master-tabs">
      <RouterLink
        v-for="tab in tabs"
        :key="tab.name"
        class="master-tab"
        :class="{ 'is-active': tabActive(tab.name) }"
        :to="{ name: tab.name }"
        :aria-current="tabActive(tab.name) ? 'page' : undefined"
      >
        {{ tab.label }}
      </RouterLink>
    </nav>

    <div v-if="!allowed" class="empty-panel" data-testid="master-denied">
      <h2 class="empty-lead">需要管理员账号</h2>
      <p class="muted">当前账号不能维护基础数据。请使用系统账号进入。</p>
      <RouterLink class="button button-ghost empty-back" :to="{ name: 'home' }">返回首页</RouterLink>
    </div>

    <router-view v-else />
  </section>
</template>
