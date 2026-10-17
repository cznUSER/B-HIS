<script setup>
// 2026-10-14 P2-1 退药/退费独立工作区：复用既有发药、收费组件和后端规则。
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { session } from '../auth/session.js'
import BusinessPageHeader from '../components/BusinessPageHeader.vue'
import ChargeView from './ChargeView.vue'
import DispenseView from './DispenseView.vue'
import { hasEffectivePermission } from '../workbench/catalog.js'

const route = useRoute()
const router = useRouter()

const canReturn = computed(() => hasEffectivePermission(session.user, ['dispense:create', 'dispense:return']))
const canRefund = computed(() => hasEffectivePermission(session.user, ['charge:refund']))
const activeTab = computed(() => {
  if (route.query.view === 'records' && canRefund.value) return 'refund'
  if (route.query.view === 'issued' && canReturn.value) return 'return'
  return canReturn.value ? 'return' : 'refund'
})
const chargeId = computed(() => route.query.charge_id || route.query.chargeId || '')

function go(tab) {
  if ((tab === 'return' && !canReturn.value) || (tab === 'refund' && !canRefund.value)) return
  void router.push({
    name: 'outpatient-module',
    params: { module: 'refund' },
    query: tab === 'refund'
      ? { view: 'records', ...(chargeId.value ? { charge_id: chargeId.value } : {}) }
      : { view: 'issued', ...(chargeId.value ? { charge_id: chargeId.value } : {}) },
  })
}
</script>

<template>
  <div class="refund-workspace">
    <BusinessPageHeader
      title-id="refund-workspace-title"
      title="退药/退费工作区"
      eyebrow="OUTPATIENT · RETURNS"
      icon="refund"
      description="已发药先退药，再按原收费单现金退费；含优惠收费单暂不支持部分退费。"
    />
    <nav class="refund-tabs" aria-label="退药/退费工作区">
      <button v-if="canReturn" type="button" :class="{ 'is-active': activeTab === 'return' }" :aria-pressed="activeTab === 'return'" @click="go('return')">
        退药
      </button>
      <button v-if="canRefund" type="button" :class="{ 'is-active': activeTab === 'refund' }" :aria-pressed="activeTab === 'refund'" @click="go('refund')">
        退费
      </button>
    </nav>

    <DispenseView v-if="activeTab === 'return' && canReturn" return-only />
    <ChargeView v-else-if="canRefund" refund-only />
    <p v-else class="result-empty" role="status">当前账号没有退药或退费权限</p>
  </div>
</template>

<style scoped>
.refund-workspace {
  display: grid;
  gap: 18px;
}

.refund-tabs {
  display: flex;
  gap: 8px;
  border-bottom: 1px solid #d9e4e1;
}

.refund-tabs button {
  min-height: 36px;
  padding: 0 16px;
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: #607372;
  cursor: pointer;
  font: inherit;
  font-weight: 650;
}

.refund-tabs button.is-active {
  border-bottom-color: #1a6f68;
  color: #165a55;
}
</style>
