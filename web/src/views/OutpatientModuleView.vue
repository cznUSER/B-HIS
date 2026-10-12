<script setup>
// 2026-10-12 P1-H05-PATIENT-UI 建档走可用页，其余门诊模块保持空态。
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { session } from '../auth/session.js'
import { decorateModules, findModule } from '../workbench/catalog.js'
import ModulePlaceholderView from './ModulePlaceholderView.vue'
import PatientView from './PatientView.vue'

const route = useRoute()

const showPatient = computed(() => {
  const found = findModule(String(route.params.module || ''))
  if (!found || found.key !== 'patient') return false
  const decorated = decorateModules(session.user && session.user.roles).find((item) => item.key === 'patient')
  return Boolean(decorated && decorated.allowed)
})
</script>

<template>
  <PatientView v-if="showPatient" />
  <ModulePlaceholderView v-else />
</template>
