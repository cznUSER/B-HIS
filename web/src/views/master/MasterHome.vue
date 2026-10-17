<script setup>
// 2026-10-13 P1-H05-MASTER-UI 基础数据总览。
import { onMounted, ref } from 'vue'
import { ApiError } from '../../api/errors.js'
import { session } from '../../auth/session.js'
import { fetchMasterSummary } from '../../master/api.js'
import MasterSymbol from './MasterSymbol.vue'

const summary = ref(null)
const errorMessage = ref('')

const cards = [
  { name: 'master-departments', label: '科室', key: 'departments', empty: '暂无科室', description: '组织科室信息与启停状态', unit: '个科室', category: '诊疗组织' },
  { name: 'master-doctors', label: '医生', key: 'doctors', empty: '暂无医生；请先维护科室', description: '维护医生信息与科室归属', unit: '位医生', category: '诊疗组织' },
  { name: 'master-catalog', label: '收费/药品目录', key: 'catalog', empty: '暂无收费/药品项目', description: '管理收费项目、药品与定价', unit: '个项目', category: '业务基础' },
  { name: 'master-inventory', label: '药品库存', key: 'inventory', empty: '暂无药品库存（无 DRUG 或未建 SKU）', description: '查看库存数量与调整记录', unit: '条库存', category: '业务基础' },
  { name: 'master-dicts', label: '基础字典', key: 'dicts', empty: '无可维护码表（仅系统枚举）', description: '查看系统枚举与维护号别', unit: '条码表记录', category: '系统管理' },
  { name: 'master-users', label: '用户账号', key: 'users', empty: '暂无用户', description: '维护人员账号与预置角色', unit: '个账号', category: '系统管理' },
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
  <div class="master-overview">
    <div class="master-overview-head"><div><h2>管理总览</h2><p>从组织、业务基础到账号，集中维护六类信息。</p></div><span class="master-soft-badge">当前数据概况</span></div>
    <p v-if="errorMessage" class="banner" role="alert">{{ errorMessage }}</p>
    <ul v-else-if="summary" class="master-cards" data-testid="master-home">
    <li v-for="(card, index) in cards" :key="card.key" :style="{ '--master-card-delay': `${index * 40}ms` }">
      <RouterLink class="master-card" :to="{ name: card.name }">
        <div class="master-card-top"><span class="master-card-symbol" :class="`master-tone-${card.key}`"><MasterSymbol :name="card.key" /></span><span class="master-card-category">{{ card.category }}</span></div>
        <h3>{{ card.label }}</h3>
        <p class="master-card-description">{{ card.description }}</p>
        <div class="master-card-value"><strong>{{ summary[card.key] }}</strong><span>{{ card.unit }}</span></div>
        <div class="master-card-bottom"><span class="field-hint">{{ summary[card.key] ? '进入维护' : card.empty }}</span><svg viewBox="0 0 20 20" aria-hidden="true"><path d="M4 10h12M11 5l5 5-5 5"/></svg></div>
      </RouterLink>
    </li>
    </ul>
    <p v-else class="shell-status">正在读取…</p>
    <div class="master-overview-guide"><span class="master-guide-icon"><MasterSymbol name="departments" /></span><div><h3>建议维护顺序</h3><p>先建立科室与医生，再维护收费目录及药品库存，最后配置号别与工作账号。</p></div></div>
  </div>
</template>
