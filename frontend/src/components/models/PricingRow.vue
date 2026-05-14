<template>
  <tr v-if="value != null" class="border-b border-gray-100 last:border-0 dark:border-dark-700/60">
    <td class="py-1.5 text-gray-600 dark:text-dark-300">{{ label }}</td>
    <td class="py-1.5 text-right font-mono text-gray-900 dark:text-white">
      <span>${{ formatted }}</span>
      <span class="ml-1 text-[10px] text-gray-400 dark:text-dark-500">/ {{ unit }}</span>
    </td>
  </tr>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    label: string
    /** 后端已转为 USD per million tokens 或 USD per call。 */
    value: number | null
    /** 有效倍率：base × user。 */
    mult: number
    /** 显示单位（MTok / call）。 */
    unit?: string
  }>(),
  { unit: 'MTok' },
)

const formatted = computed(() => {
  if (props.value == null) return '-'
  const v = props.value * (props.mult || 1)
  // 大于 1：保留 2 位；小于 1：保留 4 位；非常小：保留 6 位。
  if (v >= 1) return v.toFixed(2)
  if (v >= 0.01) return v.toFixed(4)
  return v.toFixed(6)
})
</script>
