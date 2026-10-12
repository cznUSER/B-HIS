<script setup>
// 2026-10-12 P1-H05-HOME 门诊六步图标
defineProps({
  name: { type: String, required: true },
})
</script>

<template>
  <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
    <g fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">
      <template v-if="name === 'patient'">
        <circle cx="9" cy="8" r="2.7" />
        <path d="M4.2 18.4c.6-2.9 2.6-4.4 4.8-4.4s4.2 1.5 4.8 4.4" />
        <path d="M17 8v5.2M14.4 10.6h5.2" />
      </template>
      <template v-else-if="name === 'register'">
        <rect x="4" y="5" width="16" height="14.5" rx="2" />
        <path d="M4 9.2h16M8 3.6v3M16 3.6v3M8.2 13.2h4.2" />
      </template>
      <template v-else-if="name === 'order'">
        <path d="M7 4.2h6.6L18 8.6V19a1.4 1.4 0 0 1-1.4 1.4H7.4A1.4 1.4 0 0 1 6 19V5.6A1.4 1.4 0 0 1 7.4 4.2H7z" />
        <path d="M13.5 4.5V8.8H18M8.6 12.4h6.6M8.6 15.6h4.4" />
      </template>
      <template v-else-if="name === 'charge'">
        <path d="M7 3.6h10v16.6l-1.8-1.1-1.8 1.1-1.8-1.1L10 19.1 8.2 18 7 19.1V3.6z" />
        <path d="M9.4 8h5.2M9.4 11.4h5.2M9.4 14.8h3.2" />
      </template>
      <template v-else-if="name === 'dispense'">
        <rect x="9" y="3.2" width="6" height="2.8" rx="0.8" />
        <path d="M8.2 6h7.6l.9 2.8v9.2a1.5 1.5 0 0 1-1.5 1.5h-6.4a1.5 1.5 0 0 1-1.5-1.5V8.8L8.2 6z" />
        <path d="M9.2 12.4h5.6" />
      </template>
      <template v-else-if="name === 'refund'">
        <path d="M8.2 8.2H5.4V5.4" />
        <path d="M6 8a6.8 6.8 0 1 1-1.2 6.2" />
        <path d="M12 9.2v3.4l2.3 1.4" />
      </template>
    </g>
  </svg>
</template>
