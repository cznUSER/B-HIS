<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { session } from '../auth/session.js'
import { decorateModules } from '../workbench/catalog.js'
import { outpatientTarget } from '../workbench/context.js'

const route = useRoute()
const steps = computed(() => decorateModules(session.user).filter((item) => item.key !== 'patient' && item.status === 'available'))
</script>

<template>
  <nav class="workflow-steps" data-testid="outpatient-step-nav" aria-label="门诊流程">
    <span class="workflow-caption">门诊流程</span>
    <div class="workflow-track">
      <component
        :is="step.allowed && route.params.module !== step.key ? 'RouterLink' : 'span'"
        v-for="(step, index) in steps"
        :key="step.key"
        :to="step.allowed && route.params.module !== step.key ? outpatientTarget(step.key, route.query) : undefined"
        class="workflow-step"
        :class="{ 'is-current': route.params.module === step.key, 'is-disabled': !step.allowed }"
        :aria-current="route.params.module === step.key ? 'step' : undefined"
        :aria-disabled="!step.allowed || undefined"
        :title="step.reason || step.blurb"
      >
        <span class="workflow-number" aria-hidden="true">{{ String(index + 1).padStart(2, '0') }}</span>
        <span>{{ step.name }}</span>
        <span v-if="route.params.module === step.key" class="workflow-current">当前</span>
      </component>
    </div>
    <RouterLink class="workflow-overview" :to="{ name: 'workbench' }">流程总览 <span aria-hidden="true">↗</span></RouterLink>
  </nav>
</template>

<style scoped>
.workflow-steps { display: flex; align-items: center; gap: 20px; margin-bottom: 24px; padding: 10px 16px; border: 1px solid #e0e9e2; border-radius: 14px; background: rgba(255, 255, 255, .78); }
.workflow-caption { flex: none; padding-right: 18px; border-right: 1px solid #e0e9e2; color: #7d9185; font-size: 11px; }
.workflow-track { display: flex; flex: 1; align-items: center; gap: 12px; overflow: auto; scrollbar-width: none; }
.workflow-track::-webkit-scrollbar { display: none; }
.workflow-step { display: flex; flex: 1; align-items: center; justify-content: center; gap: 8px; min-height: 36px; padding: 0 10px; border-radius: 9px; color: #748a7c; font-size: 12px; text-decoration: none; white-space: nowrap; transition: background .2s, color .2s; }
.workflow-number { font-size: 10px; color: #95a99a; font-variant-numeric: tabular-nums; }
.workflow-step:hover:not(.is-disabled) { background: #eef6f0; color: #137b68; }
.workflow-step.is-current { background: #e3f2e9; color: #167763; font-weight: 600; }
.workflow-step.is-current .workflow-number { color: #167763; }
.workflow-step.is-disabled { color: #a5b1a9; }
.workflow-current { font-size: 9px; font-weight: 400; opacity: .7; }
.workflow-overview { flex: none; display: flex; gap: 8px; color: #718b7b; font-size: 11px; text-decoration: none; }
.workflow-overview:hover { color: #137b68; }
@media (max-width: 1200px) { .workflow-caption { display: none; } .workflow-steps { gap: 10px; padding: 8px; } .workflow-track { gap: 6px; } .workflow-current { display: none; } }
@media (max-width: 760px) { .workflow-overview { display: none; } .workflow-step { flex: none; padding: 0 12px; } .workflow-steps { margin-bottom: 18px; } }
</style>
