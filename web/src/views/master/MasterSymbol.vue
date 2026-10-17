<script setup>
defineProps({ name: { type: String, default: 'overview' } })
</script>

<template>
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.55" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <template v-if="name === 'departments'"><path d="M5 20V6l7-3 7 3v14M3 20h18M9 20v-5h6v5M9 7v4M15 7v4M7 9h4M13 9h4" /></template>
    <template v-else-if="name === 'doctors'"><circle cx="12" cy="7" r="3"/><path d="M5 21v-2a7 7 0 0 1 14 0v2M8 14v4a2 2 0 0 0 4 0v-5M17 17v2"/><circle cx="17" cy="20" r="1.4"/></template>
    <template v-else-if="name === 'catalog'"><rect x="5" y="4" width="14" height="17" rx="2"/><path d="M9 4V2h6v2M9 9h6M9 13h6M9 17h3"/></template>
    <template v-else-if="name === 'inventory'"><path d="m3 7 9-4 9 4-9 4-9-4ZM3 7v10l9 4 9-4V7M12 11v10M7.5 5l9 4M16.5 9v4"/></template>
    <template v-else-if="name === 'dicts'"><path d="M4 4h6a3 3 0 0 1 3 3v14a4 4 0 0 0-4-3H4V4ZM13 7a3 3 0 0 1 3-3h4v14h-4a3 3 0 0 0-3 3M7 8h3M7 12h3M16 8h1M16 12h1"/></template>
    <template v-else-if="name === 'users'"><circle cx="9" cy="7" r="3"/><path d="M3 20v-2a6 6 0 0 1 12 0v2M16 4a3 3 0 0 1 0 6M18 14a5 5 0 0 1 3 4v2"/></template>
    <template v-else><rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/></template>
  </svg>
</template>
