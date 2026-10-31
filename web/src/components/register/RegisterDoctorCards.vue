<script setup>
defineProps({ doctors: { type: Array, default: () => [] }, selected: String, departmentId: String, disabled: Boolean, fee: Number })
defineEmits(['select'])
</script>
<template>
  <div class="register-doctor-list" data-testid="register-doctor-cards">
    <p v-if="!departmentId" class="register-card-empty">请选择科室后查看医生与余号</p>
    <p v-else-if="!doctors.length" class="register-card-empty">该科室当日暂无出诊医生。</p>
    <button v-for="doctor in doctors" :key="doctor.doctorId" class="register-doctor-card" :class="{ selected: selected === doctor.doctorId }" type="button" :data-testid="`doctor-${doctor.doctorId}`" :aria-pressed="selected === doctor.doctorId" :disabled="disabled || (doctor.scheduleConfigured && doctor.remain === 0)" @click="$emit('select', doctor.doctorId)">
      <span class="register-doctor-avatar" aria-hidden="true">{{ doctor.doctorName.slice(0, 1) }}</span>
      <span class="register-doctor-body"><strong>{{ doctor.doctorName }}<small>{{ doctor.title || '职称 —' }}</small></strong><span>擅长：{{ doctor.specialty || '—' }}</span><span class="register-doctor-bottom"><span :class="{ 'register-sold-out': doctor.remain === 0 }">{{ doctor.remain === null ? '现场号 · 未配置时段' : doctor.remain === 0 ? '当日已满 / 无出诊时段' : `余 ${doctor.remain} 号` }}</span><b>{{ selected === doctor.doctorId ? `￥${Number(fee || 0).toFixed(2)}` : '费用待确认' }}</b></span></span>
    </button>
  </div>
</template>
