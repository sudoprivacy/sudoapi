<template>
  <button
    type="button"
    @click="$emit('open', card)"
    class="group flex h-full w-full flex-col rounded-2xl border border-gray-200/60 bg-white/80 p-4 text-left shadow-sm backdrop-blur-sm transition hover:-translate-y-0.5 hover:shadow-lg hover:shadow-primary-500/10 dark:border-dark-700/60 dark:bg-dark-800/60"
    :class="{ 'ring-2 ring-primary-400/40': card.featured }"
  >
    <!-- Header: icon + name + featured badge -->
    <div class="mb-3 flex items-start justify-between gap-2">
      <div class="flex min-w-0 flex-1 items-center gap-2">
        <img
          v-if="card.icon_url"
          :src="card.icon_url"
          alt=""
          class="h-8 w-8 flex-shrink-0 rounded-lg object-contain"
        />
        <div
          v-else
          class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg"
          :class="categoryGradient"
        >
          <ModelIcon :model="card.name" size="22px" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="truncate text-sm font-semibold text-gray-900 dark:text-white" :title="card.display_name || card.name">
            {{ card.display_name || card.name }}
          </div>
          <div class="truncate text-[11px] text-gray-500 dark:text-dark-400" :title="card.name">
            {{ card.name }}
          </div>
        </div>
      </div>
      <span
        v-if="card.featured"
        class="flex-shrink-0 rounded-full bg-amber-100 px-1.5 py-0.5 text-[10px] font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
      >
        {{ t('modelSquare.featured') }}
      </span>
    </div>

    <!-- Description -->
    <p
      v-if="card.description"
      class="mb-3 line-clamp-2 text-xs leading-relaxed text-gray-600 dark:text-dark-300"
    >
      {{ card.description }}
    </p>

    <!-- Capability tags -->
    <div v-if="card.capabilities && card.capabilities.length" class="mb-3 flex flex-wrap gap-1">
      <span
        v-for="cap in displayedCapabilities"
        :key="cap"
        class="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-600 dark:bg-dark-700 dark:text-dark-300"
      >
        {{ t(`modelSquare.capabilities.${cap}`, cap) }}
      </span>
      <span
        v-if="hiddenCapabilityCount > 0"
        class="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-500 dark:bg-dark-700 dark:text-dark-400"
      >
        +{{ hiddenCapabilityCount }}
      </span>
    </div>

    <!-- Footer: platforms + min price -->
    <div class="mt-auto flex items-end justify-between gap-2 pt-2">
      <div class="flex flex-wrap gap-1">
        <span
          v-for="p in card.platforms"
          :key="p.platform"
          :class="['rounded border px-1.5 py-0.5 text-[10px] font-medium', platformBadgeClass(p.platform)]"
        >
          {{ p.platform }}
        </span>
      </div>
      <div v-if="minInputPriceLabel" class="text-right">
        <div class="text-[10px] text-gray-500 dark:text-dark-400">
          {{ t('modelSquare.fromPrice') }}
        </div>
        <div class="text-xs font-semibold text-primary-600 dark:text-primary-400">
          {{ minInputPriceLabel }}
        </div>
      </div>
    </div>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelSquareCard } from '@/api/models'
import { platformBadgeClass } from '@/utils/platformColors'
import ModelIcon from '@/components/common/ModelIcon.vue'

const props = defineProps<{ card: ModelSquareCard }>()
defineEmits<{ (e: 'open', card: ModelSquareCard): void }>()

const { t } = useI18n()

const MAX_VISIBLE_TAGS = 4

const displayedCapabilities = computed(() =>
  (props.card.capabilities ?? []).slice(0, MAX_VISIBLE_TAGS),
)
const hiddenCapabilityCount = computed(() =>
  Math.max(0, (props.card.capabilities?.length ?? 0) - MAX_VISIBLE_TAGS),
)

const CATEGORY_GRADIENTS: Record<string, string> = {
  claude: 'bg-gradient-to-br from-orange-400 to-orange-500',
  gpt: 'bg-gradient-to-br from-emerald-500 to-emerald-600',
  gemini: 'bg-gradient-to-br from-blue-500 to-blue-600',
  image: 'bg-gradient-to-br from-pink-500 to-rose-600',
  embedding: 'bg-gradient-to-br from-teal-500 to-cyan-600',
  audio: 'bg-gradient-to-br from-violet-500 to-purple-600',
  other: 'bg-gradient-to-br from-slate-400 to-slate-500',
}
const categoryGradient = computed(
  () => CATEGORY_GRADIENTS[props.card.category] ?? CATEGORY_GRADIENTS.other,
)

/**
 * 「起步价」用于卡片右下角速览：取所有平台/分组中最便宜的 input_price_per_mtok_usd。
 * 若全为 null，则尝试 per_request_price_usd（按次模式）。完全无价时返回空串。
 */
const minInputPriceLabel = computed(() => {
  let minToken: number | null = null
  let minPerRequest: number | null = null
  for (const platform of props.card.platforms ?? []) {
    for (const row of platform.group_prices ?? []) {
      const effective = (row.user_rate_multiplier ?? 1) * row.base_rate_multiplier
      if (row.input_price_per_mtok_usd != null) {
        const scaled = row.input_price_per_mtok_usd * effective
        if (minToken == null || scaled < minToken) minToken = scaled
      }
      if (row.per_request_price_usd != null) {
        const scaled = row.per_request_price_usd * effective
        if (minPerRequest == null || scaled < minPerRequest) minPerRequest = scaled
      }
    }
  }
  if (minToken != null) {
    return `$${minToken.toFixed(2)} / MTok`
  }
  if (minPerRequest != null) {
    return `$${minPerRequest.toFixed(4)} / call`
  }
  return ''
})
</script>
