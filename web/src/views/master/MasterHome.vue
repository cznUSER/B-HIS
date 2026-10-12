<script setup>
// 2026-10-13 P1-H05-MASTER-UI 基础数据总览。
import { onMounted, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchMasterSummary } from '../../master/api.js'

const summary = ref(null)
const errorMessage = ref('')

const cards = [
  { name: 'master-departments', label: '科室', key: 'departments', empty: '暂无科室' },
  { name: 'master-doctors', label: '医生', key: 'doctors', empty: '暂无医生；请先维护科室' },
  { name: 'master-catalog', label: '收费/药品目录', key: 'catalog', empty: '暂无收费/药品项目' },
  { name: 'master-inventory', label: '药品库存', key: 'inventory', empty: '暂无药品库存（无 DRUG 或未建 SKU）' },
  { name: 'master-dicts', label: '基础字典', key: 'dicts', empty: '无可维护码表（仅系统枚举）' },
  { name: 'master-users', label: '用户账号', key: 'users', empty: '暂无用户' },
]

onMounted(async () => {
  try {
    summary.value = await fetchMasterSummary(session.accessToken)
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法读取基础数据'
  }
})
</script>

<template>
  <p v-if="errorMessage" class="banner" role="alert">{{ errorMessage }}</p>
  <ul v-else-if="summary" class="master-cards" data-testid="master-home">
    <li v-for="card in cards" :key="card.key">
      <RouterLink class="master-card" :to="{ name: card.name }">
        <span>{{ card.label }}</span>
        <strong>{{ summary[card.key] }}</strong>
        <span class="field-hint">{{ summary[card.key] ? '进入维护' : card.empty }}</span>
      </RouterLink>
    </li>
  </ul>
  <p v-else class="shell-status">正在读取…</p>
</template>
