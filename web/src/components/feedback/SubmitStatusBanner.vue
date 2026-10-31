<script setup>
import { computed } from 'vue'

const props = defineProps({ feedback: { type: Object, required: true } })
const visible = computed(() => props.feedback.status === 'success' || props.feedback.status === 'error')
const isDiagnosis = computed(() => ['save', 'amend', 'commit', 'abate'].includes(props.feedback.action) || Boolean(props.feedback.diagnoseId))
const isOrder = computed(() => Boolean(props.feedback.orderId || props.feedback.orderNo))
const isCharge = computed(() => ['pay', 'refund'].includes(props.feedback.action) || Boolean(props.feedback.chargeId || props.feedback.squareNo))
</script>

<template>
  <p
    v-if="visible"
    class="banner"
    :class="feedback.status === 'success' ? 'banner-ok' : ''"
    :role="feedback.status === 'error' ? 'alert' : 'status'"
    data-testid="register-submit-feedback"
    :data-diagnosis-testid="isDiagnosis && !isOrder && !isCharge ? 'diagnosis-submit-feedback' : undefined"
    :data-order-testid="isOrder && !isCharge ? 'order-submit-feedback' : undefined"
    :data-charge-testid="isCharge ? 'charge-submit-feedback' : undefined"
  >
    {{ feedback.message }}
  </p>
</template>
