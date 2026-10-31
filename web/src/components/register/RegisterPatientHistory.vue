<script setup>
import { registerSortName, statusLabel } from '../../register/master.js'
import { paymentSummary, slotTime } from '../../register/slotCardModel.js'
defineProps({ patient: Object, rows: { type: Array, default: () => [] }, loading: Boolean, error: String, departmentLabel: Function, doctorLabel: Function, charges: Object })
defineEmits(['retry'])
</script>
<template>
  <section class="register-history-panel register-empty-history" data-testid="register-history-panel" aria-labelledby="register-history-title">
    <header><h2 id="register-history-title">该患者历史挂号 / 预约记录</h2><span>近 90 天 · 含退号及后续预约</span><button v-if="patient" type="button" class="text-btn" :disabled="loading" @click="$emit('retry')">刷新记录</button></header>
    <div v-if="loading" class="register-card-empty" role="status">正在读取挂号列表…</div>
    <div v-else-if="error" class="register-card-empty" role="alert">{{ error }}<button type="button" class="text-btn" @click="$emit('retry')">重试挂号列表</button></div>
    <div v-else class="register-history-scroll" tabindex="0" role="region" aria-label="挂号历史宽表，可横向滚动">
      <table class="register-history-table" data-testid="register-history-table"><thead><tr><th>就诊日期</th><th>科室</th><th>医生</th><th>号别</th><th>时段</th><th>费用</th><th>支付</th><th>状态</th><th>流水号</th><th>操作</th></tr></thead><tbody>
        <tr v-for="row in rows" :key="row.registerId" :data-testid="row.visitType === 'APPOINTMENT' ? 'appointment-list' : 'register-list'" :data-register-id="row.registerId"><td>{{ row.visitDate }}<small>{{ row.visitType === 'APPOINTMENT' ? '预约号' : '现场号' }}</small></td><td>{{ departmentLabel(row.departmentId) }}</td><td>{{ doctorLabel(row.doctorId) }}</td><td>{{ registerSortName(row.registerSort) }}</td><td>{{ row.appointmentStart ? `${slotTime(row.appointmentStart)}–${slotTime(row.appointmentEnd)}` : '现场号' }}</td><td>￥{{ Number(row.registerFee || 0).toFixed(2) }}</td><td>{{ paymentSummary(row, charges?.[row.registerId]) }}</td><td><span class="register-history-status" :data-status="row.status">{{ statusLabel(row.status) }}</span></td><td>{{ row.registerNo || row.registerId }}</td><td class="register-history-actions"><slot name="actions" :row="row" /></td></tr>
        <tr v-if="!rows.length"><td colspan="10" class="register-card-empty">{{ patient ? '近 90 天暂无挂号 / 预约记录。' : '选择患者后读取历史挂号记录；诊断信息在诊断环节维护。' }}</td></tr>
      </tbody></table>
    </div>
  </section>
</template>
