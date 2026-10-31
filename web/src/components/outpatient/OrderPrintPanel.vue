<script setup>
import { computed } from 'vue'
import { printStatusLabel } from '../../workbench/orderPrintStatus.js'

const props = defineProps({
  previewModel: { type: Object, default: null },
  printStatus: { type: String, default: 'IDLE' },
  locked: { type: Boolean, default: false },
  canPreview: { type: Boolean, default: false },
  honestyNote: { type: String, default: '' },
  sessionNote: { type: String, default: '' },
  lastPrintedAt: { type: String, default: '' },
  errorMessage: { type: String, default: '' },
})

const emit = defineEmits(['preview', 'print', 'confirm', 'retry', 'reprint', 'close'])

const statusLabel = computed(() => printStatusLabel(props.printStatus))
const showPanel = computed(() => props.canPreview || props.printStatus !== 'IDLE')
const lines = computed(() => props.previewModel?.lines || [])
</script>

<template>
  <section v-if="showPanel" class="order-print-panel" data-testid="order-print-panel" aria-label="处方输出状态">
    <div class="order-print-head">
      <div>
        <span class="clinical-kicker">处方输出</span>
        <h3>本机预览与输出状态</h3>
      </div>
      <p class="order-print-status" data-testid="order-print-status" role="status">
        状态：{{ statusLabel }}
      </p>
    </div>

    <p class="order-print-honesty" data-testid="order-print-honesty" role="note">{{ honestyNote }}</p>
    <p class="order-print-session" data-testid="order-print-session" role="note">{{ sessionNote }}</p>

    <div v-if="previewModel" class="order-print-preview" data-testid="order-print-preview">
      <dl class="order-print-meta">
        <div><dt>患者</dt><dd>{{ previewModel.patientName || '—' }}（{{ previewModel.patientNo || previewModel.patientId || '无号' }}）</dd></div>
        <div><dt>就诊号</dt><dd>{{ previewModel.registerNo || previewModel.registerId || '—' }}</dd></div>
        <div><dt>处方流水</dt><dd>{{ previewModel.orderNo || previewModel.orderId || '—' }}</dd></div>
        <div><dt>诊断</dt><dd>{{ previewModel.diagnoseName || '—' }}</dd></div>
        <div><dt>业务状态</dt><dd>{{ previewModel.orderStatus || '—' }}</dd></div>
      </dl>
      <ul v-if="lines.length" class="order-print-lines">
        <li v-for="(line, index) in lines" :key="line.itemNo + '-' + index">
          {{ index + 1 }}. {{ line.itemName || line.itemNo }} × {{ line.amount || '—' }}{{ line.units || '' }}
          <span v-if="line.usageText || line.frequencyText">（{{ [line.usageText, line.frequencyText].filter(Boolean).join(' / ') }}）</span>
        </li>
      </ul>
      <p v-else class="order-print-empty">暂无明细可预览</p>
    </div>

    <p v-if="errorMessage" class="order-print-error" role="alert" data-testid="order-print-error">{{ errorMessage }}</p>
    <p v-if="lastPrintedAt" class="order-print-time" data-testid="order-print-time">本机确认时间：{{ lastPrintedAt }}</p>

    <div class="order-print-actions">
      <button
        class="button button-ghost button-compact"
        type="button"
        data-testid="order-print-preview-btn"
        :disabled="locked || !canPreview"
        @click="emit('preview')"
      >
        打开预览
      </button>
      <button
        v-if="printStatus === 'PREVIEW' || printStatus === 'FAILED'"
        class="button button-compact"
        type="button"
        data-testid="order-print-output-btn"
        :disabled="locked"
        @click="emit('print')"
      >
        {{ locked ? '正在输出…' : '本机打印占位' }}
      </button>
      <button
        v-if="printStatus === 'PRINTING'"
        class="button button-compact"
        type="button"
        data-testid="order-print-confirm-btn"
        @click="emit('confirm')"
      >
        确认已输出
      </button>
      <button
        v-if="printStatus === 'FAILED'"
        class="button button-ghost button-compact"
        type="button"
        data-testid="order-print-retry-btn"
        :disabled="locked"
        @click="emit('retry')"
      >
        失败重试
      </button>
      <button
        v-if="printStatus === 'PRINTED'"
        class="button button-ghost button-compact"
        type="button"
        data-testid="order-print-reprint-btn"
        :disabled="locked"
        @click="emit('reprint')"
      >
        重打
      </button>
      <button
        v-if="printStatus !== 'IDLE'"
        class="button button-ghost button-compact"
        type="button"
        data-testid="order-print-close-btn"
        :disabled="locked"
        @click="emit('close')"
      >
        收起
      </button>
    </div>
  </section>
</template>

<style scoped>
.order-print-panel {
  margin-top: 0.75rem;
  padding: 0.85rem 1rem;
  border: 1px solid var(--border, #d7dee8);
  border-radius: 10px;
  background: var(--surface-muted, #f7f9fc);
}
.order-print-head {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  align-items: flex-start;
}
.order-print-head h3 {
  margin: 0.15rem 0 0;
  font-size: 1rem;
}
.order-print-status {
  margin: 0;
  font-weight: 600;
}
.order-print-honesty,
.order-print-session,
.order-print-time,
.order-print-empty {
  margin: 0.45rem 0 0;
  color: var(--text-muted, #5b677a);
  font-size: 0.85rem;
}
.order-print-error {
  margin: 0.45rem 0 0;
  color: #a11;
  font-size: 0.9rem;
}
.order-print-preview {
  margin-top: 0.65rem;
  padding: 0.65rem 0.75rem;
  border-radius: 8px;
  background: #fff;
  border: 1px dashed var(--border, #d7dee8);
}
.order-print-meta {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
  gap: 0.35rem 0.85rem;
  margin: 0;
}
.order-print-meta dt {
  font-size: 0.75rem;
  color: var(--text-muted, #5b677a);
}
.order-print-meta dd {
  margin: 0;
  font-weight: 600;
}
.order-print-lines {
  margin: 0.55rem 0 0;
  padding-left: 1.1rem;
}
.order-print-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
  margin-top: 0.7rem;
}
</style>