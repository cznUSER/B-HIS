<script setup>
// 2026-10-12 P1-H05-HOME-DASH 等分环，图例固定为 0。
defineProps({
  title: { type: String, required: true },
  slices: { type: Array, required: true },
  arcs: { type: Array, required: true },
})
</script>

<template>
  <article class="pie-card">
    <h3>{{ title }}</h3>
    <div class="pie-body">
      <svg class="donut" viewBox="0 0 42 42" role="img" :aria-label="`${title}，演示占位，各项为 0`">
        <g transform="rotate(-90 21 21)">
          <circle
            v-for="(slice, index) in slices"
            :key="slice.key"
            cx="21"
            cy="21"
            r="15.915"
            fill="none"
            stroke-width="5.2"
            :stroke="slice.color"
            :stroke-dasharray="arcs[index].dash"
            :stroke-dashoffset="arcs[index].offset"
          />
        </g>
        <circle cx="21" cy="21" r="11.3" fill="#fff" />
        <text x="21" y="21" text-anchor="middle" dominant-baseline="middle" font-size="3.2" fill="#8aa0ae">
          占位
        </text>
      </svg>
      <ul class="pie-legend">
        <li v-for="slice in slices" :key="`${slice.key}-legend`">
          <i :style="{ background: slice.color }" aria-hidden="true"></i>
          <span>{{ slice.label }}</span>
          <span class="pie-num">(0)</span>
        </li>
      </ul>
    </div>
  </article>
</template>
