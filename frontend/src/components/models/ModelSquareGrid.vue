<template>
  <div class="space-y-5">
    <!-- Filter bar — 紧凑模式时折叠隐藏，避免首页内嵌时太拥挤 -->
    <ModelFilterBar
      v-if="!compact"
      v-model="filterState"
      :available-categories="availableCategories"
      :available-capabilities="availableCapabilities"
    />
    <div v-else class="relative w-full max-w-md">
      <Icon
        name="search"
        size="md"
        class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
      />
      <input
        v-model="filterState.search"
        type="text"
        :placeholder="t('modelSquare.searchPlaceholder')"
        class="input pl-10"
      />
    </div>

    <!-- Result summary -->
    <div class="flex items-center justify-between gap-2 text-xs text-gray-500 dark:text-dark-400">
      <span>
        {{ t('modelSquare.resultCount', { count: visibleCards.length, total: allCards.length }) }}
      </span>
      <button
        type="button"
        @click="reload"
        :disabled="loading"
        class="rounded-md p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-700 disabled:opacity-50 dark:hover:bg-dark-800 dark:hover:text-white"
        :title="t('common.refresh', 'Refresh')"
      >
        <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
      </button>
    </div>

    <!-- Loading / empty state -->
    <div v-if="loading && !allCards.length" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      <div
        v-for="i in compact ? 4 : 8"
        :key="i"
        class="h-44 animate-pulse rounded-2xl bg-gray-100 dark:bg-dark-800"
      ></div>
    </div>
    <div
      v-else-if="!loading && !visibleCards.length"
      class="rounded-2xl border border-dashed border-gray-200 bg-white/40 px-6 py-12 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800/40 dark:text-dark-400"
    >
      {{ t('modelSquare.empty') }}
    </div>

    <!-- Cards grid -->
    <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      <ModelCard
        v-for="card in displayedCards"
        :key="card.name"
        :card="card"
        @open="openDetail"
      />
    </div>

    <!-- View all link (compact mode) -->
    <div v-if="compact && visibleCards.length > maxItems" class="text-center">
      <router-link
        to="/models"
        class="inline-flex items-center gap-1 text-sm font-medium text-primary-600 hover:underline dark:text-primary-400"
      >
        {{ t('modelSquare.viewAll') }}
        <Icon name="arrowRight" size="sm" />
      </router-link>
    </div>

    <!-- Detail drawer -->
    <ModelDetailDrawer
      :open="detailOpen"
      :card="selectedCard"
      @close="closeDetail"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import modelSquareAPI, { type ModelSquareCard } from '@/api/models'
import Icon from '@/components/icons/Icon.vue'
import ModelCard from './ModelCard.vue'
import ModelFilterBar, { type ModelFilterState } from './ModelFilterBar.vue'
import ModelDetailDrawer from './ModelDetailDrawer.vue'

const props = withDefaults(
  defineProps<{
    /** 'public' 走未登录端点；'me' 走已登录端点。 */
    scope: 'public' | 'me'
    /** compact 模式只显示前 maxItems 张卡片 + 查看全部链接，用于首页内嵌。 */
    compact?: boolean
    maxItems?: number
  }>(),
  { compact: false, maxItems: 12 },
)

const { t } = useI18n()

const allCards = ref<ModelSquareCard[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
let abortCtl: AbortController | null = null

const filterState = ref<ModelFilterState>({
  search: '',
  categories: [],
  capabilities: [],
  priceRanges: [],
})

const detailOpen = ref(false)
const selectedCard = ref<ModelSquareCard | null>(null)

async function reload() {
  if (abortCtl) abortCtl.abort()
  abortCtl = new AbortController()
  loading.value = true
  error.value = null
  try {
    const list =
      props.scope === 'me'
        ? await modelSquareAPI.listMyModels({ signal: abortCtl.signal })
        : await modelSquareAPI.listPublicModels({ signal: abortCtl.signal })
    allCards.value = list
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    if (!/aborted|canceled/i.test(msg)) {
      error.value = msg
      console.error('[ModelSquare] load failed', e)
    }
  } finally {
    loading.value = false
  }
}

onMounted(reload)
watch(() => props.scope, reload)

// ── Filter logic (pure client-side) ─────────────────────────────────

const availableCategories = computed(() => {
  const set = new Set<string>()
  for (const c of allCards.value) set.add(c.category)
  return Array.from(set).sort()
})
const availableCapabilities = computed(() => {
  const set = new Set<string>()
  for (const c of allCards.value) (c.capabilities ?? []).forEach((cap) => set.add(cap))
  return Array.from(set).sort()
})

/**
 * 价格桶按「最便宜的 input_price_per_mtok_usd × effectiveMult」划分：
 *   free  : 0
 *   low   : (0, 1]   美元/MTok
 *   mid   : (1, 5]
 *   high  : (5, +∞)
 * 完全无 token 价的模型（如按次/embedding）暂归 mid 桶，方便也能被默认显示。
 */
function priceTier(card: ModelSquareCard): string {
  let min: number | null = null
  for (const platform of card.platforms ?? []) {
    for (const row of platform.group_prices ?? []) {
      if (row.input_price_per_mtok_usd != null) {
        const v = row.input_price_per_mtok_usd * row.base_rate_multiplier * (row.user_rate_multiplier ?? 1)
        if (min == null || v < min) min = v
      }
    }
  }
  if (min == null) return 'mid'
  if (min === 0) return 'free'
  if (min <= 1) return 'low'
  if (min <= 5) return 'mid'
  return 'high'
}

const visibleCards = computed(() => {
  const q = filterState.value.search.trim().toLowerCase()
  return allCards.value.filter((c) => {
    // search
    if (q) {
      const hay = [
        c.name,
        c.display_name,
        c.description,
        c.category,
        ...(c.capabilities ?? []),
        ...c.platforms.map((p) => p.platform),
      ]
        .join(' ')
        .toLowerCase()
      if (!hay.includes(q)) return false
    }
    if (filterState.value.categories.length && !filterState.value.categories.includes(c.category))
      return false
    if (filterState.value.capabilities.length) {
      const caps = new Set(c.capabilities ?? [])
      if (!filterState.value.capabilities.every((cap) => caps.has(cap))) return false
    }
    if (filterState.value.priceRanges.length) {
      if (!filterState.value.priceRanges.includes(priceTier(c))) return false
    }
    return true
  })
})

const displayedCards = computed(() =>
  props.compact ? visibleCards.value.slice(0, props.maxItems) : visibleCards.value,
)

function openDetail(card: ModelSquareCard) {
  selectedCard.value = card
  detailOpen.value = true
}
function closeDetail() {
  detailOpen.value = false
}
</script>
