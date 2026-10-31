<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  DEFAULT_FIND_CATEGORY,
  EMPTY_MATCH_MESSAGE,
  FIND_CATEGORIES,
  filterByKeyword,
  findSelectedLabel,
} from '../../workbench/orderItemFind.js'

const props = defineProps({
  modelValue: { type: String, default: '' },
  items: { type: Array, default: () => [] },
  index: { type: Number, default: 0 },
  disabled: { type: Boolean, default: false },
  /** 已落库快照（真接口下选中项可能不在当前 ACTIVE 列表） */
  sourceItem: { type: Object, default: null },
})

const emit = defineEmits(['update:modelValue'])

const open = ref(false)
const category = ref(DEFAULT_FIND_CATEGORY)
const keyword = ref('')
const rootEl = ref(null)
const searchEl = ref(null)

const allItems = computed(() => {
  const list = Array.isArray(props.items) ? [...props.items] : []
  if (
    props.sourceItem?.itemNo &&
    props.sourceItem.itemNo === props.modelValue &&
    !list.some((item) => item.itemNo === props.sourceItem.itemNo)
  ) {
    list.unshift({
      itemNo: props.sourceItem.itemNo,
      itemName: props.sourceItem.itemName,
      itemType: props.sourceItem.itemType || 'DRUG',
      spec: props.sourceItem.spec || '',
      units: props.sourceItem.units || '',
      price: props.sourceItem.price,
    })
  }
  return list
})

const filtered = computed(() =>
  filterByKeyword(allItems.value, { category: category.value, keyword: keyword.value }),
)

const triggerLabel = computed(() => {
  if (!props.modelValue) return '请选择项目'
  return findSelectedLabel(allItems.value, props.modelValue) || props.modelValue
})

const testId = computed(() => `line-item-${props.index}`)

watch(
  () => props.disabled,
  (locked) => {
    if (locked) closePanel()
  },
)

function openPanel() {
  if (props.disabled) return
  open.value = true
  nextTick(() => searchEl.value?.focus?.())
}

function closePanel() {
  open.value = false
  keyword.value = ''
}

function togglePanel() {
  if (props.disabled) return
  if (open.value) closePanel()
  else openPanel()
}

function setCategory(id) {
  if (props.disabled) return
  category.value = id === 'TREAT' ? 'TREAT' : 'DRUG'
}

function selectItem(item) {
  if (props.disabled || !item?.itemNo) return
  emit('update:modelValue', item.itemNo)
  closePanel()
}

function clearSelection() {
  if (props.disabled) return
  emit('update:modelValue', '')
  closePanel()
}

function onDocPointer(event) {
  if (!open.value) return
  const root = rootEl.value
  if (root && event.target && !root.contains(event.target)) closePanel()
}

function onDocKey(event) {
  if (event.key === 'Escape' && open.value) closePanel()
}

onMounted(() => {
  document.addEventListener('pointerdown', onDocPointer, true)
  document.addEventListener('keydown', onDocKey, true)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocPointer, true)
  document.removeEventListener('keydown', onDocKey, true)
})
</script>

<template>
  <div ref="rootEl" class="order-item-finder" :data-testid="testId" data-order-item-finder="1">
    <button
      class="order-item-finder__trigger"
      type="button"
      :disabled="disabled"
      :aria-label="`第 ${index + 1} 行项目`"
      :aria-expanded="open ? 'true' : 'false'"
      aria-haspopup="listbox"
      data-testid-role="item-trigger"
      @click="togglePanel"
    >
      <span class="order-item-finder__trigger-text">{{ triggerLabel }}</span>
      <span class="order-item-finder__chev" aria-hidden="true">▾</span>
    </button>

    <div
      v-if="open"
      class="order-item-finder__panel"
      role="listbox"
      :aria-label="`第 ${index + 1} 行找项`"
      data-testid-role="item-panel"
    >
      <div class="order-item-finder__tabs" role="tablist" aria-label="项目分类">
        <button
          v-for="tab in FIND_CATEGORIES"
          :key="tab.id"
          class="order-item-finder__tab"
          type="button"
          role="tab"
          :aria-selected="category === tab.id ? 'true' : 'false'"
          :data-testid="`line-item-cat-${index}-${tab.id}`"
          :class="{ 'is-active': category === tab.id }"
          @click="setCategory(tab.id)"
        >
          {{ tab.label }}
        </button>
      </div>

      <label class="order-item-finder__search">
        <span class="sr-only">关键词</span>
        <input
          ref="searchEl"
          v-model="keyword"
          type="search"
          autocomplete="off"
          placeholder="编码 / 名称 / 规格"
          :data-testid="`line-item-q-${index}`"
          :aria-label="`第 ${index + 1} 行项目关键词`"
        />
      </label>

      <ul class="order-item-finder__list" data-testid-role="item-list">
        <li v-if="!filtered.length" class="order-item-finder__empty" role="status" :data-testid="`line-item-empty-${index}`">
          {{ EMPTY_MATCH_MESSAGE }}
        </li>
        <li v-for="item in filtered" :key="item.itemNo">
          <button
            class="order-item-finder__option"
            type="button"
            role="option"
            :aria-selected="modelValue === item.itemNo ? 'true' : 'false'"
            :data-testid="`line-item-opt-${index}-${item.itemNo}`"
            @click="selectItem(item)"
          >
            <span class="order-item-finder__name">{{ item.itemName }}</span>
            <span class="order-item-finder__meta">{{ item.itemNo }}<template v-if="item.spec"> · {{ item.spec }}</template></span>
          </button>
        </li>
      </ul>

      <div class="order-item-finder__foot">
        <button class="text-btn" type="button" :data-testid="`line-item-clear-${index}`" @click="clearSelection">清空选择</button>
        <button class="text-btn" type="button" @click="closePanel">关闭</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.order-item-finder {
  position: relative;
  min-width: 10rem;
  z-index: 2;
}

.order-item-finder__trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.35rem;
  width: 100%;
  min-height: 2rem;
  padding: 0.25rem 0.45rem;
  border: 1px solid var(--border-color, #c9d2df);
  border-radius: 0.35rem;
  background: #fff;
  text-align: left;
  cursor: pointer;
}

.order-item-finder__trigger:disabled {
  opacity: 0.65;
  cursor: not-allowed;
  background: #f5f7fa;
}

.order-item-finder__trigger-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.order-item-finder__chev {
  flex: none;
  opacity: 0.55;
}

.order-item-finder__panel {
  position: absolute;
  left: 0;
  top: calc(100% + 0.2rem);
  z-index: 40;
  width: min(22rem, 70vw);
  max-height: 18rem;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  padding: 0.5rem;
  border: 1px solid var(--border-color, #c9d2df);
  border-radius: 0.45rem;
  background: #fff;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.12);
}

.order-item-finder__tabs {
  display: flex;
  gap: 0.25rem;
}

.order-item-finder__tab {
  flex: 1;
  min-height: 1.85rem;
  border: 1px solid transparent;
  border-radius: 0.3rem;
  background: #f1f5f9;
  cursor: pointer;
}

.order-item-finder__tab.is-active {
  border-color: #3b82f6;
  background: #eff6ff;
  color: #1d4ed8;
  font-weight: 600;
}

.order-item-finder__search input {
  width: 100%;
  min-height: 1.9rem;
  padding: 0.25rem 0.45rem;
  border: 1px solid var(--border-color, #c9d2df);
  border-radius: 0.3rem;
}

.order-item-finder__list {
  list-style: none;
  margin: 0;
  padding: 0;
  overflow: auto;
  max-height: 10.5rem;
}

.order-item-finder__option {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.1rem;
  width: 100%;
  padding: 0.35rem 0.4rem;
  border: 0;
  border-radius: 0.3rem;
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.order-item-finder__option:hover,
.order-item-finder__option[aria-selected='true'] {
  background: #eff6ff;
}

.order-item-finder__name {
  font-weight: 600;
}

.order-item-finder__meta {
  font-size: 0.78rem;
  color: #64748b;
}

.order-item-finder__empty {
  padding: 0.65rem 0.4rem;
  color: #64748b;
  text-align: center;
}

.order-item-finder__foot {
  display: flex;
  justify-content: space-between;
  gap: 0.5rem;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
</style>