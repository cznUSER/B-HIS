<script setup>
import ModuleIcon from './ModuleIcon.vue'

defineProps({
  title: { type: String, required: true },
  description: { type: String, default: '' },
  icon: { type: String, default: 'patient' },
  eyebrow: { type: String, default: '下一步' },
})
</script>

<template>
  <div class="business-empty-state">
    <div class="business-empty-visual" aria-hidden="true">
      <span class="empty-orbit"></span><span class="empty-orbit empty-orbit-inner"></span>
      <span class="business-empty-symbol"><ModuleIcon :name="icon" /></span>
      <span class="empty-dot empty-dot-one"></span><span class="empty-dot empty-dot-two"></span>
    </div>
    <div class="business-empty-copy">
      <p v-if="eyebrow" class="business-empty-eyebrow">{{ eyebrow }}</p>
      <h2>{{ title }}</h2>
      <p v-if="description" class="business-empty-description">{{ description }}</p>
      <div v-if="$slots.detail" class="business-empty-detail"><slot name="detail" /></div>
      <div v-if="$slots.default" class="business-empty-actions"><slot /></div>
    </div>
  </div>
</template>

<style scoped>
.business-empty-state { display: flex; align-items: center; justify-content: center; gap: 38px; min-height: 256px; padding: 38px; border: 1px solid #e0ebe3; border-radius: 18px; background: radial-gradient(ellipse at 0 100%, #e7f4ec66, transparent 58%), #fff; }
.business-empty-visual { position: relative; display: grid; place-items: center; flex: none; width: 152px; height: 152px; }
.empty-orbit { position: absolute; inset: 2px; border: 1px dashed #c9ded1; border-radius: 50%; }
.empty-orbit-inner { inset: 18px; border-style: solid; border-color: #e1ece4; }
.business-empty-symbol { display: grid; place-items: center; width: 74px; height: 86px; border: 1px solid #c6e2d1; border-radius: 22px; background: linear-gradient(135deg, #e6f4eb, #f9fcfa); color: #4a9c82; box-shadow: 0 12px 20px #24523a0a; transform: rotate(-7deg); }
.business-empty-symbol svg { width: 34px; height: 34px; }
.empty-dot { position: absolute; width: 8px; height: 8px; border-radius: 50%; background: #84bca0; box-shadow: 0 0 0 5px #edf5ef; }
.empty-dot-one { top: 21px; left: 28px; }.empty-dot-two { bottom: 19px; right: 20px; width: 5px; height: 5px; background: #adc9b7; }
.business-empty-copy { max-width: 520px; }
.business-empty-eyebrow { margin-bottom: 8px; color: #88a392; font-size: 10px; letter-spacing: .1em; }
.business-empty-copy h2 { color: #38554a; font-size: 20px; font-weight: 600; }
.business-empty-description { margin-top: 10px; color: #7c9085; font-size: 13px; line-height: 1.8; }
.business-empty-detail { margin-top: 12px; color: #7c9085; font-size: 12px; line-height: 1.8; }
.business-empty-actions { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 22px; }
@media (max-width: 760px) { .business-empty-state { flex-direction: column; align-items: flex-start; gap: 18px; padding: 28px 22px; } .business-empty-visual { width: 106px; height: 106px; } .business-empty-symbol { width: 56px; height: 65px; border-radius: 16px; } .business-empty-symbol svg { width: 27px; height: 27px; } .business-empty-copy h2 { font-size: 18px; } .business-empty-description { font-size: 12px; } }
</style>
