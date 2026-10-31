<script setup>
defineProps({ departments: { type: Array, default: () => [] }, selected: String, disabled: Boolean, today: Boolean })
defineEmits(['select'])
</script>
<template>
  <div class="register-dept-grid" data-testid="register-department-selected">
    <button v-for="dept in departments" :key="dept.departmentId" type="button" class="register-dept-card" :class="{ selected: selected === dept.departmentId }" :data-testid="`dept-${dept.departmentId}`" :aria-pressed="selected === dept.departmentId" :disabled="disabled" @click="$emit('select', dept.departmentId)">
      <strong>{{ dept.departmentName }}</strong><span>{{ today ? '今日' : '当日' }}余号 <b>{{ dept.remainTotal === null ? '—' : dept.remainTotal }}</b></span><small v-if="dept.remainTotal === null">含现场号 · 未配置时段</small>
    </button>
    <p v-if="!departments.length" class="register-card-empty">暂无启用科室。</p>
  </div>
</template>
