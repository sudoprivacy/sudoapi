<!-- sudoapi: Model Square model catalog. -->

<template>
  <tr v-if="value != null" class="border-b border-gray-100 last:border-0 dark:border-dark-700/60">
    <td class="py-1.5 text-gray-600 dark:text-dark-300">{{ label }}</td>
    <td class="py-1.5 text-right font-mono text-gray-900 dark:text-white">
      <ModelPriceValue :value="value" :mult="mult" :unit="unit" />
    </td>
  </tr>
</template>

<script setup lang="ts">
import ModelPriceValue from './ModelPriceValue.vue'

withDefaults(
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
</script>
