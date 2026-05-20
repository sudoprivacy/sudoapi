<!-- sudoapi: Model market. -->

<template>
  <div class="min-h-screen bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-gray-100">
    <header class="w-full border-b border-gray-200 bg-white/90 px-4 py-4 dark:border-dark-800 dark:bg-dark-900/90 sm:px-6">
      <nav class="flex w-full items-center justify-between gap-4">
        <router-link
          to="/models"
          class="inline-flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white"
        >
          <Icon name="arrowLeft" size="md" />
          <span>{{ t('modelSquare.title') }}</span>
        </router-link>
        <LocaleSwitcher />
      </nav>
    </header>

    <main class="w-full px-4 py-6 sm:px-6">
      <div class="mb-5 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-2xl font-semibold tracking-normal text-gray-950 dark:text-white">
            {{ t('modelQuote.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            {{ t('modelQuote.subtitle') }}
          </p>
        </div>
        <button
          type="button"
          @click="reload"
          :disabled="loading"
          class="btn btn-secondary h-9 gap-2 px-3 text-sm"
        >
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          <span>{{ t('common.refresh', 'Refresh') }}</span>
        </button>
      </div>

      <div class="mb-4 grid gap-3 lg:grid-cols-[minmax(260px,1fr)_180px_180px_220px]">
        <SearchInput
          v-model="filters.search"
          :placeholder="t('modelQuote.searchPlaceholder')"
          :debounce-ms="0"
        />
        <select v-model="filters.platform" class="input h-10 text-sm">
          <option value="">{{ t('modelQuote.filters.allPlatforms') }}</option>
          <option v-for="vendor in availableVendors" :key="vendor" :value="vendor">
            {{ vendorLabel(vendor) }}
          </option>
        </select>
        <select v-model="filters.modelType" class="input h-10 text-sm">
          <option value="">{{ t('modelQuote.filters.allTypes') }}</option>
          <option v-for="type in availableModelTypes" :key="type" :value="type">
            {{ modelTypeLabel(type) }}
          </option>
        </select>
        <select v-model="sortKey" class="input h-10 text-sm">
          <option value="modelAsc">{{ t('modelQuote.sort.modelAsc') }}</option>
          <option value="modelDesc">{{ t('modelQuote.sort.modelDesc') }}</option>
          <option value="platformAsc">{{ t('modelQuote.sort.platformAsc') }}</option>
          <option value="priceAsc">{{ t('modelQuote.sort.priceAsc') }}</option>
          <option value="priceDesc">{{ t('modelQuote.sort.priceDesc') }}</option>
          <option value="discountAsc">{{ t('modelQuote.sort.discountAsc') }}</option>
          <option value="discountDesc">{{ t('modelQuote.sort.discountDesc') }}</option>
          <option value="contextDesc">{{ t('modelQuote.sort.contextDesc') }}</option>
        </select>
      </div>

      <div class="mb-3 text-xs text-gray-500 dark:text-dark-400">
        {{ t('modelQuote.resultCount', { count: displayedRows.length, total: quoteRows.length }) }}
      </div>

      <div
        v-if="error && !loading"
        class="mb-4 rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700 dark:border-rose-900/40 dark:bg-rose-900/10 dark:text-rose-300"
      >
        {{ t('modelQuote.loadFailed', { msg: error }) }}
      </div>

      <DataTable
        :columns="columns"
        :data="displayedRows"
        :loading="loading && !quoteRows.length"
        :row-key="(row) => row.id"
        :sticky-actions-column="false"
        :estimate-row-height="64"
      >
        <template #empty>
          <div class="py-4 text-sm text-gray-500 dark:text-dark-400">
            {{ t('modelQuote.empty') }}
          </div>
        </template>

        <template #cell-model="{ row }">
          <div class="max-w-[260px]">
            <div class="truncate font-medium text-gray-950 dark:text-white" :title="row.displayName">
              {{ row.displayName }}
            </div>
            <div class="truncate text-xs text-gray-500 dark:text-dark-400" :title="row.model">
              {{ row.model }}
            </div>
          </div>
        </template>

        <template #cell-platform="{ row }">
          <span class="inline-flex items-center gap-1.5 rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-800 dark:text-dark-200">
            {{ vendorLabel(row.vendor) }}
          </span>
        </template>

        <template #cell-officialInput="{ row }">
          {{ formatTokenPrice(row.official.input) }}
        </template>
        <template #cell-platformInput="{ row }">
          {{ formatTokenPrice(row.platformPrice.input) }}
        </template>
        <template #cell-officialOutput="{ row }">
          {{ formatTokenPrice(row.official.output) }}
        </template>
        <template #cell-officialCacheRead="{ row }">
          {{ formatTokenPrice(row.official.cacheRead) }}
        </template>
        <template #cell-officialCacheWrite="{ row }">
          {{ formatTokenPrice(row.official.cacheWrite) }}
        </template>
        <template #cell-officialImageOrRequest="{ row }">
          {{ formatCallPrice(row.official.imageOrRequest) }}
        </template>
        <template #cell-platformOutput="{ row }">
          {{ formatTokenPrice(row.platformPrice.output) }}
        </template>
        <template #cell-platformCacheRead="{ row }">
          {{ formatTokenPrice(row.platformPrice.cacheRead) }}
        </template>
        <template #cell-platformCacheWrite="{ row }">
          {{ formatTokenPrice(row.platformPrice.cacheWrite) }}
        </template>
        <template #cell-platformImageOrRequest="{ row }">
          {{ formatCallPrice(row.platformPrice.imageOrRequest) }}
        </template>
        <template #cell-discount="{ row }">
          <span
            v-if="row.discountRatio != null"
            :class="[
              'inline-flex rounded-md px-2 py-1 text-xs font-semibold',
              row.discountRatio <= 1
                ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300'
                : 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300'
            ]"
          >
            {{ formatDiscount(row.discountRatio) }}
          </span>
          <span v-else class="text-xs text-gray-400 dark:text-dark-500">
            -
          </span>
        </template>
      </DataTable>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import modelSquareAPI, { type ModelSquareCard } from '@/api/models'
import DataTable from '@/components/common/DataTable.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import {
  buildModelQuoteRows,
  filterModelQuoteRows,
  sortModelQuoteRows,
  type ModelQuoteFilters,
  type ModelQuoteSortKey,
} from '@/utils/modelQuote'

const { t } = useI18n()

const cards = ref<ModelSquareCard[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
let abortCtl: AbortController | null = null

const filters = ref<ModelQuoteFilters>({
  search: '',
  platform: '',
  modelType: '',
})
const sortKey = ref<ModelQuoteSortKey>('modelAsc')

const columns = computed<Column[]>(() => [
  { key: 'model', label: t('modelQuote.columns.model') },
  { key: 'platform', label: t('modelQuote.columns.platform') },
  { key: 'officialInput', label: t('modelQuote.columns.officialInput') },
  { key: 'officialOutput', label: t('modelQuote.columns.officialOutput') },
  { key: 'officialCacheRead', label: t('modelQuote.columns.officialCacheRead') },
  { key: 'officialCacheWrite', label: t('modelQuote.columns.officialCacheWrite') },
  { key: 'officialImageOrRequest', label: t('modelQuote.columns.officialImageOrRequest') },
  { key: 'platformInput', label: t('modelQuote.columns.platformInput') },
  { key: 'platformOutput', label: t('modelQuote.columns.platformOutput') },
  { key: 'platformCacheRead', label: t('modelQuote.columns.platformCacheRead') },
  { key: 'platformCacheWrite', label: t('modelQuote.columns.platformCacheWrite') },
  { key: 'platformImageOrRequest', label: t('modelQuote.columns.platformImageOrRequest') },
  { key: 'discount', label: t('modelQuote.columns.discount') },
])

async function reload() {
  abortCtl?.abort()
  abortCtl = new AbortController()
  loading.value = true
  error.value = null
  try {
    cards.value = await modelSquareAPI.listPublicModels({ signal: abortCtl.signal })
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    if (!/aborted|canceled/i.test(msg)) {
      error.value = msg
      console.error('[ModelQuote] load failed', e)
    }
  } finally {
    loading.value = false
  }
}

onMounted(reload)

const quoteRows = computed(() => buildModelQuoteRows(cards.value))
const availableVendors = computed(() =>
  Array.from(new Set(quoteRows.value.map((row) => row.vendor))).sort(),
)
const availableModelTypes = computed(() =>
  Array.from(new Set(quoteRows.value.map((row) => row.modelType).filter(Boolean))).sort(),
)
const displayedRows = computed(() =>
  sortModelQuoteRows(filterModelQuoteRows(quoteRows.value, filters.value), sortKey.value),
)

function vendorLabel(vendor: string): string {
  return vendor ? t(`modelSquare.categories.${vendor}`, vendor) : '-'
}

function modelTypeLabel(type: string): string {
  return type ? t(`modelSquare.modelTypes.${type}`, type) : '-'
}

function formatTokenPrice(value: number | null): string {
  return formatUSD(value, t('modelQuote.units.perMTok'))
}

function formatCallPrice(value: number | null): string {
  return formatUSD(value, '')
}

function formatUSD(value: number | null, suffix: string): string {
  if (value == null || !Number.isFinite(value)) return '-'
  const maximumFractionDigits = value > 0 && value < 0.01 ? 6 : 4
  const amount = new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 0,
    maximumFractionDigits,
  }).format(value)
  const text = `$${amount}`
  return suffix ? `${text} ${suffix}` : text
}

function formatDiscount(ratio: number): string {
  const fold = (ratio * 10).toFixed(1)
  return t('modelQuote.discountValue', { fold })
}
</script>
