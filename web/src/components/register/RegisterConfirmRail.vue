<script setup>
defineProps({ patient: Object, department: Object, doctor: Object, slotLabel: String, fee: Number, payment: String, canSave: Boolean, saving: Boolean, mockMode: Boolean, reason: String })
defineEmits(['update:payment', 'update:fee'])
const payments = [{ id: 'CASH', name: '现金' }, { id: 'WECHAT', name: '微信' }, { id: 'ALIPAY', name: '支付宝' }, { id: 'INSURANCE', name: '医保个账' }]
</script>
<template>
  <aside class="register-confirm-rail" data-testid="register-fee-panel" aria-labelledby="register-confirm-title">
    <header><h2 id="register-confirm-title">挂号确认</h2><p>汇总科室 · 医生 · 时段 · 费用</p></header>
    <div class="register-confirm-body">
      <dl class="register-confirm-summary"><div><dt>患者</dt><dd>{{ patient?.name || '未选择' }}</dd></div><div><dt>科室</dt><dd>{{ department?.departmentName || '未选择' }}</dd></div><div><dt>医生</dt><dd>{{ doctor?.doctorName || '未选择' }}</dd></div><div><dt>职称</dt><dd>{{ doctor?.title || '—' }}</dd></div><div><dt>时段</dt><dd>{{ slotLabel || '未选择' }}</dd></div><div><dt>号别</dt><dd data-testid="pending-register-sort">普通号</dd></div></dl>


      
      <p v-if="mockMode" class="register-card-note">演示模式不收取挂号费。</p>
      <slot />
    </div>
    <footer><fieldset class="register-pay-group" data-testid="register-payments"><legend>支付方式偏好</legend><div><button v-for="item in payments" :key="item.id" type="button" :data-testid="`payment-${item.id}`" :aria-pressed="payment === item.id" :class="{ selected: payment === item.id }" :disabled="saving" @click="$emit('update:payment', item.id)">{{ item.name }}</button></div><p>尚未实收；请另行现金收费。</p></fieldset><div class="register-fee-big"><label class="register-fee-edit">应付挂号费<input aria-label="挂号费（元）" :value="fee" type="number" min="0" step="0.01" inputmode="decimal" data-testid="pending-register-fee" :disabled="mockMode || saving" @input="$emit('update:fee', Number($event.target.value))" /></label><strong data-testid="register-total-fee">￥{{ Number(fee || 0).toFixed(2) }}</strong></div><button class="register-confirm-button" type="submit" data-testid="save-register" :disabled="!canSave">{{ saving ? '正在保存…' : '确认挂号' }}</button><p data-testid="register-confirm-reason">{{ reason || '完成科室 → 医生 → 号源后可确认' }}</p></footer>
  </aside>
</template>
