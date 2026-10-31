<script setup>
defineProps({ department: Object, doctor: Object, slotLabel: String })
</script>
<template>
  <ol class="register-step-pills" data-testid="register-steps" aria-label="挂号选号进度">
    <li :class="{ done: department, current: !department }"><b>1</b><div><span>选科室</span><strong>{{ department?.departmentName || '请选择科室' }}</strong></div></li>
    <li :class="{ done: doctor, current: department && !doctor }"><b>2</b><div><span>选医生</span><strong>{{ doctor?.doctorName || (department ? '请选择医生' : '请先选科室') }}</strong></div></li>
    <li :class="{ done: slotLabel, current: doctor && !slotLabel }"><b>3</b><div><span>选号源</span><strong>{{ slotLabel || (doctor ? '请选择时段' : '请先选医生') }}</strong></div></li>
  </ol>
</template>
