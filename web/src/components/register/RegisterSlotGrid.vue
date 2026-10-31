<script setup>
import { slotTime } from '../../register/slotCardModel.js'
defineProps({ doctor: Object, selected: String, disabled: Boolean, appointmentMode: Boolean })
defineEmits(['select'])
</script>
<template>
  <div class="register-slot-groups" data-testid="register-slot-grid">
    <p v-if="!doctor" class="register-card-empty">请选择医生后查看上午 / 下午号源</p>
    <div v-else-if="!doctor.scheduleConfigured" class="register-onsite-note" data-testid="register-onsite-note"><b>未配置时段 · 按现场号办理</b><p>{{ appointmentMode ? '该医生未配置预约时段，请切换现场号或选择已排班医生。' : '现场号不预占预约时段，确认后按正常挂号流程办理。' }}</p></div>
    <p v-else-if="!doctor.slots.length" class="register-card-empty">该医生当日无出诊时段，请更换医生或日期。</p>
    <template v-else>
      <section v-for="period in ['AM', 'PM']" :key="period" class="register-slot-period">
        <h4>{{ period === 'AM' ? '上午' : '下午' }}<small>余 {{ doctor.slots.filter(s => s.period === period).reduce((sum, s) => sum + s.remain, 0) }} 号</small></h4>
        <div class="register-slot-grid">
          <button v-for="slot in doctor.slots.filter(s => s.period === period)" :key="slot.start" type="button" class="register-slot-cell" :class="{ selected: selected === slot.start }" :data-testid="`slot-${slotTime(slot.start)}`" :data-slot-start="slot.start" :aria-pressed="selected === slot.start" :disabled="disabled || slot.remain === 0" @click="$emit('select', slot)"><strong>{{ slotTime(slot.start) }}–{{ slotTime(slot.end) }}</strong><span>{{ slot.remain > 0 ? `剩余 ${slot.remain}` : '已约满' }}</span></button>
          <p v-if="!doctor.slots.some(s => s.period === period)" class="register-slot-empty">本时段无号</p>
        </div>
      </section>
    </template>
  </div>
</template>
