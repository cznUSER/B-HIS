<script setup>
import { computed } from 'vue'

const props = defineProps({ feedback: { type: Object, required: true } })
const visible = computed(() => props.feedback.status === 'success' || props.feedback.status === 'error')
</script>

<template>
  <p v-if="visible" class="banner" :class="feedback.status === 'success' ? 'banner-ok' : ''" :role="feedback.status === 'error' ? 'alert' : 'status'" data-testid="register-submit-feedback">
    {{ feedback.message }}
  </p>
</template>

