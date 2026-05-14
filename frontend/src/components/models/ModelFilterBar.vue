<template>
  <div class="space-y-4">
    <!-- Search -->
    <div class="relative w-full max-w-md">
      <Icon
        name="search"
        size="md"
        class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
      />
      <input
        type="text"
        :value="modelValue.search"
        @input="onSearchInput"
        :placeholder="t('modelSquare.searchPlaceholder')"
        class="input pl-10"
      />
    </div>

    <!-- Filter chips: category / capability / priceTier -->
    <div class="space-y-3">
      <FilterChipGroup
        :label="t('modelSquare.filter.category')"
        :options="categoryOptions"
        :selected="modelValue.categories"
        @toggle="(v: string) => toggle('categories', v)"
      />
      <FilterChipGroup
        :label="t('modelSquare.filter.capability')"
        :options="capabilityOptions"
        :selected="modelValue.capabilities"
        @toggle="(v: string) => toggle('capabilities', v)"
      />
      <FilterChipGroup
        :label="t('modelSquare.filter.priceRange')"
        :options="priceRangeOptions"
        :selected="modelValue.priceRanges"
        @toggle="(v: string) => toggle('priceRanges', v)"
      />

      <button
        v-if="hasActiveFilter"
        type="button"
        @click="clearAll"
        class="text-xs font-medium text-primary-600 underline-offset-2 hover:underline dark:text-primary-400"
      >
        {{ t('modelSquare.clearFilters') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import FilterChipGroup from './FilterChipGroup.vue'

/**
 * 模型广场筛选状态：
 *   - search        ：模糊匹配 模型名/描述/平台/标签
 *   - categories    ：厂商/平台分类（claude/gpt/gemini/image/embedding/other）
 *   - capabilities  ：能力标签（vision/function_calling/reasoning/...）
 *   - priceRanges   ：价格区间桶 free / low / mid / high（按 input_price USD/MTok 划分）
 *
 * 全部 multi-select；空集 = 不过滤。
 */
export interface ModelFilterState {
  search: string
  categories: string[]
  capabilities: string[]
  priceRanges: string[]
}

const props = defineProps<{
  modelValue: ModelFilterState
  /** 可见的分类列表（来自实际数据，避免空选项；调用方计算）。 */
  availableCategories: string[]
  /** 可见的能力标签列表（来自实际数据）。 */
  availableCapabilities: string[]
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: ModelFilterState): void
}>()

const { t } = useI18n()

const categoryOptions = computed(() =>
  props.availableCategories.map((c) => ({
    value: c,
    label: t(`modelSquare.categories.${c}`, c),
  })),
)
const capabilityOptions = computed(() =>
  props.availableCapabilities.map((c) => ({
    value: c,
    label: t(`modelSquare.capabilities.${c}`, c),
  })),
)
const priceRangeOptions = computed(() => [
  { value: 'free', label: t('modelSquare.priceTier.free') },
  { value: 'low', label: t('modelSquare.priceTier.low') },
  { value: 'mid', label: t('modelSquare.priceTier.mid') },
  { value: 'high', label: t('modelSquare.priceTier.high') },
])

const hasActiveFilter = computed(
  () =>
    !!props.modelValue.search ||
    props.modelValue.categories.length > 0 ||
    props.modelValue.capabilities.length > 0 ||
    props.modelValue.priceRanges.length > 0,
)

function onSearchInput(e: Event) {
  emit('update:modelValue', {
    ...props.modelValue,
    search: (e.target as HTMLInputElement).value,
  })
}

function toggle(field: 'categories' | 'capabilities' | 'priceRanges', value: string) {
  const list = props.modelValue[field]
  const next = list.includes(value) ? list.filter((v) => v !== value) : [...list, value]
  emit('update:modelValue', { ...props.modelValue, [field]: next })
}

function clearAll() {
  emit('update:modelValue', {
    search: '',
    categories: [],
    capabilities: [],
    priceRanges: [],
  })
}
</script>
