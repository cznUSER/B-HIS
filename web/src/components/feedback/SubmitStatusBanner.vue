<script setup>
import { computed } from 'vue'

const props = defineProps({ feedback: { type: Object, required: true } })
const visible = computed(() => props.feedback.status === 'success' || props.feedback.status === 'error')
const isDiagnosis = computed(() => ['save', 'amend', 'commit', 'abate'].includes(props.feedback.action) || Boolean(props.feedback.diagnoseId))
</script>

<template>
  <p
    v-if="visible"
    class="banner"
    :class="feedback.status === 'success' ? 'banner-ok' : ''"
    :role="feedback.status === 'error' ? 'alert' : 'status'"
    data-testid="register-submit-feedback"
    :data-diagnosis-testid="isDiagnosis ? 'diagnosis-submit-feedback' : undefined"
  >
    {{ feedback.message }}
  </p>
</template>
