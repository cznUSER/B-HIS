<script setup>
import MasterSymbol from './MasterSymbol.vue'
defineProps({
  title: { type: String, required: true },
  titleId: { type: String, required: true },
  description: { type: String, default: '' },
  icon: { type: String, default: 'overview' },
  count: { type: Number, default: null },
  loading: Boolean,
  countLabel: { type: String, default: '当前列表' },
  badge: { type: String, default: '' },
})
</script>

<template>
  <header class="master-panel-header">
    <span class="master-panel-symbol"><MasterSymbol :name="icon" /></span>
    <div class="master-panel-copy">
      <div class="master-panel-title"><h2 :id="titleId">{{ title }}</h2><span v-if="badge" class="master-soft-badge">{{ badge }}</span></div>
      <p v-if="description">{{ description }}</p>
    </div>
    <span v-if="count !== null" class="master-count"><span>{{ countLabel }}</span><strong>{{ loading ? '—' : count }}</strong><span>条</span></span>
    <div v-if="$slots.actions" class="master-header-actions"><slot name="actions" /></div>
  </header>
</template>
