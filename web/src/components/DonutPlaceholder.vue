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
    <header class="pie-heading"><h3>{{ title }}</h3><span>演示占位</span></header>
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
            stroke-width="3.7"
            :stroke="slice.color"
            stroke-opacity=".42"
            :stroke-dasharray="arcs[index].dash"
            :stroke-dashoffset="arcs[index].offset"
          />
        </g>
        <text x="21" y="18.9" text-anchor="middle" dominant-baseline="middle" font-size="7" font-weight="500" fill="#3d675b">
          0
        </text>
        <text x="21" y="25.9" text-anchor="middle" dominant-baseline="middle" font-size="2.5" fill="#899c95">
          暂无数据
        </text>
      </svg>
      <ul class="pie-legend">
        <li v-for="slice in slices" :key="`${slice.key}-legend`">
          <i :style="{ background: slice.color }" aria-hidden="true"></i>
          <span>{{ slice.label }}</span>
          <span class="pie-num">0</span>
        </li>
      </ul>
    </div>
    <p class="pie-footnote">等分图环仅作占位，各项均为 0。</p>
  </article>
</template>

<style scoped>
.pie-card {
  align-items: stretch;
  gap: 0;
  min-width: 0;
  padding: 22px 24px 15px;
  border: 1px solid #dde6e5;
  border-radius: 18px;
  background: #fff;
  box-shadow: 0 3px 12px #1b394704;
}
.pie-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.pie-heading h3 { color: var(--ink, #20394a); font-size: 14px; font-weight: 650; }
.pie-heading > span { padding: 3px 8px; border: 1px solid #e4ece7; border-radius: 6px; color: #8b9c94; background: #f7faf8; font-size: 9px; }
.pie-body { display: grid; grid-template-columns: minmax(110px, 170px) minmax(0, 1fr); align-items: center; justify-content: normal; gap: 25px; padding: 15px 0 10px; }
.donut { display: block; width: 100%; max-width: 170px; height: auto; }
.pie-legend { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 17px 22px; }
.pie-legend li { display: grid; grid-template-columns: 7px minmax(0, 1fr) auto; gap: 7px; align-items: center; margin: 0; color: #6d8388; font-size: 11px; }
.pie-legend i { width: 7px; height: 7px; border-radius: 50%; opacity: .7; }
.pie-num { color: #3c5d61; font-size: 13px; font-weight: 500; font-variant-numeric: tabular-nums; }
.pie-footnote { margin-top: 6px; padding-top: 12px; border-top: 1px solid #edf1ed; color: #99a5a1; font-size: 10px; }
@media (max-width: 1280px) {
  .pie-body { grid-template-columns: minmax(95px, 140px) minmax(0, 1fr); gap: 16px; }
  .pie-legend { grid-template-columns: minmax(0, 1fr); gap: 8px; }
}
@media (max-width: 720px) {
  .pie-card { padding: 20px 18px 14px; }
  .pie-body { grid-template-columns: minmax(100px, 140px) minmax(0, 1fr); }
  .pie-legend { gap: 10px; }
}
</style>
