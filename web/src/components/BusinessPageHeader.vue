<script setup>
import ModuleIcon from './ModuleIcon.vue'

defineProps({
  titleId: { type: String, required: true },
  title: { type: String, required: true },
  description: { type: String, default: '' },
  eyebrow: { type: String, default: '门诊业务' },
  icon: { type: String, default: 'order' },
})
</script>

<template>
  <header class="business-page-header">
    <div class="business-heading-main">
      <span class="business-heading-icon" aria-hidden="true"><ModuleIcon :name="icon" /></span>
      <div class="business-heading-copy">
        <p class="business-eyebrow">{{ eyebrow }}</p>
        <h1 :id="titleId">{{ title }}</h1>
        <p v-if="description" class="business-description">{{ description }}</p>
      </div>
    </div>
    <div v-if="$slots.actions" class="business-header-actions"><slot name="actions" /></div>
  </header>
</template>

<style scoped>
.business-page-header { position: relative; display: flex; align-items: center; justify-content: space-between; gap: 24px; min-height: 132px; padding: 25px 28px; overflow: hidden; border: 1px solid #dce9e1; border-radius: 20px; background: linear-gradient(116deg, #fff 10%, #f0f7f2); box-shadow: 0 3px 12px #244b3410; }
.business-page-header::after { content: ''; position: absolute; right: -65px; top: -100px; width: 280px; height: 280px; border: 1px solid #c1deca66; border-radius: 50%; pointer-events: none; }
.business-heading-main { position: relative; z-index: 1; display: flex; align-items: center; gap: 19px; min-width: 0; }
.business-heading-icon { display: grid; place-items: center; flex: none; width: 59px; height: 59px; border: 1px solid #cce6d7; border-radius: 18px; background: linear-gradient(145deg, #e1f1e7, #f1f8f3); color: #268773; box-shadow: inset 0 1px #fff; }
.business-heading-icon svg { width: 28px; height: 28px; }
.business-eyebrow { margin-bottom: 5px; font-size: 10px; font-weight: 600; color: #79a08a; letter-spacing: .09em; }
.business-heading-copy h1 { color: #233e38; font-size: 26px; font-weight: 650; letter-spacing: -.02em; line-height: 1.3; }
.business-description { max-width: 63em; margin-top: 8px; color: #73877e; font-size: 12px; line-height: 1.7; }
.business-header-actions { position: relative; z-index: 1; display: flex; flex-wrap: wrap; justify-content: flex-end; align-items: center; gap: 10px; flex: none; }
@media (max-width: 1050px) { .business-page-header { align-items: flex-start; flex-direction: column; gap: 18px; } .business-header-actions { justify-content: flex-start; } }
@media (max-width: 760px) { .business-page-header { padding: 20px; min-height: 120px; border-radius: 16px; } .business-heading-main { align-items: flex-start; gap: 13px; } .business-heading-icon { width: 45px; height: 45px; border-radius: 13px; } .business-heading-icon svg { width: 23px; height: 23px; } .business-heading-copy h1 { font-size: 22px; } .business-description { font-size: 11px; } }
</style>
