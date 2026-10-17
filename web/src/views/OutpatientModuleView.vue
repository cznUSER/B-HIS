<script setup>
// 2026-10-12 P1-H05-PATIENT-UI 建档走可用页，其余门诊模块保持空态。
// 2026-10-12 P1-H05-REGISTER-UI 挂号走可用页。无权限仍回空态。
// 2026-10-12 P1-H05-ORDER-UI 开单走可用页。
// 2026-10-12 P1-H05-CHARGE-UI 收费走可用页。
// 2026-10-14 P2-1 退药/退费挂载独立工作区，内部复用既有发药和收费流程。
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { session } from '../auth/session.js'
import { decorateModules, findModule } from '../workbench/catalog.js'
import ChargeView from './ChargeView.vue'
import DispenseView from './DispenseView.vue'
import ModulePlaceholderView from './ModulePlaceholderView.vue'
import OrderView from './OrderView.vue'
import PatientView from './PatientView.vue'
import RegisterView from './RegisterView.vue'
import RefundView from './RefundView.vue'
import DiagnosisView from './DiagnosisView.vue'
import OutpatientWorkbenchFrame from '../components/OutpatientWorkbenchFrame.vue'
import { outpatientContext } from '../workbench/outpatientContext.js'

const route = useRoute()

function allowed(key) {
  const found = findModule(String(route.params.module || ''))
  if (!found || found.key !== key) return false
  const decorated = decorateModules(session.user).find((item) => item.key === key)
  return Boolean(decorated && decorated.allowed)
}

const showPatient = computed(() => allowed('patient'))
const showRegister = computed(() => allowed('register'))
const showDiagnosis = computed(() => allowed('diagnosis'))
const showOrder = computed(() => allowed('order'))
const showCharge = computed(() => allowed('charge'))
const showDispense = computed(() => allowed('dispense'))
const showRefund = computed(() => allowed('refund'))
const moduleKey = computed(() => String(route.params.module || ''))
const moduleInfo = computed(() => findModule(moduleKey.value))
const moduleTitle = computed(() => moduleInfo.value?.name || '门诊业务')
const moduleDescription = computed(() => moduleInfo.value?.blurb || '门诊业务工作区')
const context = computed(() => {
  const query = route.query || {}
  const read = (key) => {
    const value = query[key]
    return Array.isArray(value) ? value[0] : value
  }
  return {
    patientId: read('patientId') || outpatientContext.patient?.patientId || '',
    patientName: read('patientName') || outpatientContext.patient?.name || outpatientContext.patient?.patientName || '',
    registerId: read('registerId') || outpatientContext.register?.registerId || '',
    orderId: read('orderId') || outpatientContext.orders.at(-1)?.orderId || '',
    chargeId: read('chargeId') || read('charge_id') || outpatientContext.charges.at(-1)?.chargeId || '',
  }
})
const frameStatus = computed(() => {
  if (!moduleInfo.value || !allowed(moduleKey.value)) return '当前账号无此模块权限'
  if (moduleKey.value !== 'patient' && !Object.values(context.value).some(Boolean) && !outpatientContext.register && !outpatientContext.orders.length && !outpatientContext.charges.length) return '等待患者或就诊上下文'
  return '工作区已就绪'
})
const frameTone = computed(() => frameStatus.value.includes('无') ? 'blocked' : frameStatus.value.includes('等待') ? 'waiting' : 'ready')
</script>

<template>
  <OutpatientWorkbenchFrame :module-key="moduleKey" :title="moduleTitle" :description="moduleDescription" :context="context" :status="frameStatus" :status-tone="frameTone">
    <PatientView v-if="showPatient" />
    <RegisterView v-else-if="showRegister" />
    <DiagnosisView v-else-if="showDiagnosis" />
    <OrderView v-else-if="showOrder" />
    <ChargeView v-else-if="showCharge" />
    <DispenseView v-else-if="showDispense" />
    <RefundView v-else-if="showRefund" />
    <ModulePlaceholderView v-else />
  </OutpatientWorkbenchFrame>
</template>
