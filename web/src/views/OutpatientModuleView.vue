<script setup>
// 2026-10-12 P1-H05-PATIENT-UI 建档走可用页，其余门诊模块保持空态。
// 2026-10-12 P1-H05-REGISTER-UI 挂号走可用页。无权限仍回空态。
// 2026-10-12 P1-H05-ORDER-UI 开单走可用页。
// 2026-10-12 P1-H05-CHARGE-UI 收费走可用页。
// 2026-10-12 P1-H05-DISPENSE-UI 发药走 Mock 页。退药/退费入口仍是空态。
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

const route = useRoute()

function allowed(key) {
  const found = findModule(String(route.params.module || ''))
  if (!found || found.key !== key) return false
  const decorated = decorateModules(session.user && session.user.roles).find((item) => item.key === key)
  return Boolean(decorated && decorated.allowed)
}

const showPatient = computed(() => allowed('patient'))
const showRegister = computed(() => allowed('register'))
const showOrder = computed(() => allowed('order'))
const showCharge = computed(() => allowed('charge'))
const showDispense = computed(() => allowed('dispense'))
</script>

<template>
  <PatientView v-if="showPatient" />
  <RegisterView v-else-if="showRegister" />
  <OrderView v-else-if="showOrder" />
  <ChargeView v-else-if="showCharge" />
  <DispenseView v-else-if="showDispense" />
  <ModulePlaceholderView v-else />
</template>
