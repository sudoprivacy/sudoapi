<template>
  <div v-if="options.length" class="flex flex-wrap items-center gap-2">
    <span class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ label }}</span>
    <button
      v-for="opt in options"
      :key="opt.value"
      type="button"
      @click="$emit('toggle', opt.value)"
      :class="[
        'rounded-full border px-2.5 py-0.5 text-xs font-medium transition',
        selected.includes(opt.value)
          ? 'border-primary-500 bg-primary-500/10 text-primary-700 dark:text-primary-300'
          : 'border-gray-200 bg-white text-gray-600 hover:border-gray-300 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-300',
      ]"
    >
      {{ opt.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label: string
  options: Array<{ value: string; label: string }>
  selected: string[]
}>()
defineEmits<{ (e: 'toggle', value: string): void }>()
</script>
