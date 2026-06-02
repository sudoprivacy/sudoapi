<!-- sudoapi: Model Square model catalog. -->

<template>
  <span v-if="value == null" class="text-gray-400 dark:text-dark-500">-</span>
  <span v-else class="inline-flex flex-wrap items-baseline justify-end gap-x-1.5 gap-y-0.5">
    <span class="whitespace-nowrap">${{ formattedCurrent }}</span>
    <span
      v-if="showOriginal"
      class="whitespace-nowrap text-[10px] text-gray-400 line-through dark:text-dark-500"
    >
      ${{ formattedOriginal }}
    </span>
    <span class="whitespace-nowrap text-[10px] text-gray-400 dark:text-dark-500">
      / {{ unit }}
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    /** 后端已转为 USD per million tokens 或 USD per call。 */
    value: number | null
    /** 有效倍率：用户专属倍率覆盖分组默认倍率后得到的最终值。 */
    mult: number
    /** 显示单位（MTok / call）。 */
    unit?: string
  }>(),
  { unit: 'MTok' },
)

const effectiveMult = computed(() => (Number.isFinite(props.mult) ? props.mult : 1))

const currentValue = computed(() => {
  if (props.value == null) return null
  return props.value * effectiveMult.value
})

const formattedCurrent = computed(() => {
  if (currentValue.value == null) return '-'
  return formatMoney(currentValue.value)
})

const formattedOriginal = computed(() => {
  if (props.value == null) return '-'
  return formatMoney(props.value)
})

const showOriginal = computed(() => {
  if (props.value == null) return false
  return effectiveMult.value < 1 && Math.abs(currentValue.value! - props.value) > 1e-12
})

function formatMoney(v: number): string {
  if (v >= 1) return v.toFixed(2)
  if (v >= 0.01) return v.toFixed(4)
  return v.toFixed(6)
}
</script>
