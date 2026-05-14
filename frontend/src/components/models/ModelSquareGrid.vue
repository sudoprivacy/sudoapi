<template>
  <div class="space-y-5">
    <!-- 分类 tabs（仅 full 模式且数据已加载时显示，给用户一个不需展开完整 filter 的快捷入口） -->
    <div
      v-if="!compact && availableCategories.length > 1"
      class="-mx-1 flex flex-wrap items-center gap-1 overflow-x-auto"
    >
      <button
        type="button"
        @click="setCategoryTab('')"
        :class="categoryTabClass('')"
      >
        {{ t('modelSquare.tabs.all') }}
        <span class="ml-1 text-[10px] opacity-60">{{ allCards.length }}</span>
      </button>
      <button
        v-for="cat in availableCategories"
        :key="cat"
        type="button"
        @click="setCategoryTab(cat)"
        :class="categoryTabClass(cat)"
      >
        {{ t(`modelSquare.categories.${cat}`, cat) }}
        <span class="ml-1 text-[10px] opacity-60">{{ countByCategory[cat] ?? 0 }}</span>
      </button>
    </div>

    <!-- Filter bar — compact 时只显示一个搜索框 -->
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

    <!-- Result summary + sort + refresh -->
    <div class="flex flex-wrap items-center justify-between gap-3 text-xs text-gray-500 dark:text-dark-400">
      <span>
        {{ t('modelSquare.resultCount', { count: visibleCards.length, total: allCards.length }) }}
      </span>
      <div class="flex items-center gap-2">
        <label v-if="!compact" class="flex items-center gap-1.5">
          <span>{{ t('modelSquare.sort.label') }}</span>
          <select v-model="sortKey" class="input h-7 py-0 pl-2 pr-7 text-xs">
            <option value="featured">{{ t('modelSquare.sort.featured') }}</option>
            <option value="nameAsc">{{ t('modelSquare.sort.nameAsc') }}</option>
            <option value="nameDesc">{{ t('modelSquare.sort.nameDesc') }}</option>
            <option value="priceAsc">{{ t('modelSquare.sort.priceAsc') }}</option>
            <option value="priceDesc">{{ t('modelSquare.sort.priceDesc') }}</option>
            <option value="contextDesc">{{ t('modelSquare.sort.contextDesc') }}</option>
          </select>
        </label>
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
    </div>

    <!-- Error state with retry -->
    <div
      v-if="error && !loading"
      class="flex flex-col items-center gap-2 rounded-2xl border border-rose-200 bg-rose-50/60 px-6 py-8 text-center text-sm text-rose-700 dark:border-rose-900/40 dark:bg-rose-900/10 dark:text-rose-300"
    >
      <span>{{ t('modelSquare.loadFailed', { msg: error }) }}</span>
      <button type="button" @click="reload" class="btn btn-secondary h-7 px-3 text-xs">
        {{ t('common.retry', 'Retry') }}
      </button>
    </div>

    <!-- Loading skeleton -->
    <div v-else-if="loading && !allCards.length" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      <div
        v-for="i in compact ? 4 : 8"
        :key="i"
        class="h-44 animate-pulse rounded-2xl bg-gray-100 dark:bg-dark-800"
      ></div>
    </div>

    <!-- Empty state -->
    <div
      v-else-if="!visibleCards.length"
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
    <ModelDetailDrawer :open="detailOpen" :card="selectedCard" @close="closeDetail" />
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

/** 排序键：featured = 后端默认（featured → category → name）。 */
const sortKey = ref<'featured' | 'nameAsc' | 'nameDesc' | 'priceAsc' | 'priceDesc' | 'contextDesc'>(
  'featured',
)

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

// ── Filter / sort logic (pure client-side) ──────────────────────────

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

/** 按 category 统计计数，给分类 tab 显示数量。 */
const countByCategory = computed(() => {
  const m: Record<string, number> = {}
  for (const c of allCards.value) m[c.category] = (m[c.category] ?? 0) + 1
  return m
})

/**
 * 取卡片最低 input_price（USD/MTok）作为排序锚，用于 priceAsc/priceDesc 和价格桶。
 * 返回 Infinity 表示"无价 token 数据"，排序时沉到最后。
 */
function minInputPrice(card: ModelSquareCard): number {
  let min: number | null = null
  for (const platform of card.platforms ?? []) {
    for (const row of platform.group_prices ?? []) {
      if (row.input_price_per_mtok_usd != null) {
        const v =
          row.input_price_per_mtok_usd * row.base_rate_multiplier * (row.user_rate_multiplier ?? 1)
        if (min == null || v < min) min = v
      }
    }
  }
  return min ?? Number.POSITIVE_INFINITY
}

function priceTier(card: ModelSquareCard): string {
  const min = minInputPrice(card)
  if (!Number.isFinite(min)) return 'mid'
  if (min === 0) return 'free'
  if (min <= 1) return 'low'
  if (min <= 5) return 'mid'
  return 'high'
}

const filteredCards = computed(() => {
  const q = filterState.value.search.trim().toLowerCase()
  return allCards.value.filter((c) => {
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

const visibleCards = computed(() => {
  const list = [...filteredCards.value]
  switch (sortKey.value) {
    case 'nameAsc':
      list.sort((a, b) => a.name.localeCompare(b.name))
      break
    case 'nameDesc':
      list.sort((a, b) => b.name.localeCompare(a.name))
      break
    case 'priceAsc':
      list.sort((a, b) => minInputPrice(a) - minInputPrice(b))
      break
    case 'priceDesc':
      list.sort((a, b) => minInputPrice(b) - minInputPrice(a))
      break
    case 'contextDesc':
      list.sort((a, b) => (b.context_window || 0) - (a.context_window || 0))
      break
    case 'featured':
    default:
      // 后端已经按 featured → category → name 排序，保持原顺序。
      break
  }
  return list
})

const displayedCards = computed(() =>
  props.compact ? visibleCards.value.slice(0, props.maxItems) : visibleCards.value,
)

// ── Category tabs (单选语义，复用 filterState.categories) ────────────

function setCategoryTab(cat: string) {
  filterState.value = {
    ...filterState.value,
    categories: cat ? [cat] : [],
  }
}

function categoryTabClass(cat: string): string {
  const active =
    (cat === '' && filterState.value.categories.length === 0) ||
    (cat !== '' && filterState.value.categories.length === 1 && filterState.value.categories[0] === cat)
  return active
    ? 'rounded-full bg-primary-500 px-3 py-1.5 text-xs font-medium text-white shadow-sm'
    : 'rounded-full border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-600 hover:border-gray-300 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-300'
}

function openDetail(card: ModelSquareCard) {
  selectedCard.value = card
  detailOpen.value = true
}
function closeDetail() {
  detailOpen.value = false
}
</script>
