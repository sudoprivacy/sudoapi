<template>
  <div class="min-h-screen bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-gray-100">
    <header class="sticky top-0 z-30 w-full border-b border-gray-200 bg-white/95 px-4 py-3 backdrop-blur dark:border-dark-800 dark:bg-dark-900/95 sm:px-6">
      <nav class="flex w-full items-center justify-between gap-4">
        <router-link
          to="/home"
          class="inline-flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white"
        >
          <Icon name="arrowLeft" size="md" />
          <span class="hidden sm:inline">{{ siteName }}</span>
        </router-link>
        <div class="flex items-center gap-3">
          <LocaleSwitcher />
        </div>
      </nav>
    </header>

    <main class="mx-auto w-full max-w-[1440px] px-4 py-6 sm:px-6 lg:px-8">
      <div class="mb-6 flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
        <div class="max-w-3xl">
          <h1 class="text-3xl font-semibold tracking-normal text-gray-950 dark:text-white sm:text-4xl">
            {{ t('liteLLMModels.title') }}
          </h1>
          <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-dark-300">
            {{ t('liteLLMModels.subtitle') }}
          </p>
        </div>
        <button
          type="button"
          @click="reload"
          :disabled="loading"
          class="btn btn-secondary h-10 gap-2 px-4 text-sm shadow-sm"
        >
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          <span>{{ t('common.refresh', 'Refresh') }}</span>
        </button>
      </div>

      <div class="mb-5 grid gap-3 md:grid-cols-2">
        <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <div class="flex items-center gap-2 text-xs font-medium uppercase text-gray-500 dark:text-dark-400">
            <Icon name="cube" size="xs" />
            <span>{{ t('liteLLMModels.stats.models') }}</span>
          </div>
          <div class="mt-2 text-2xl font-semibold text-gray-950 dark:text-white">{{ formatNumber(models.length) }}</div>
        </div>
        <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <div class="flex items-center gap-2 text-xs font-medium uppercase text-gray-500 dark:text-dark-400">
            <Icon name="database" size="xs" />
            <span>{{ t('liteLLMModels.stats.providers') }}</span>
          </div>
          <div class="mt-2 flex flex-wrap gap-1.5">
            <span v-for="provider in providerSummary" :key="provider.key" :class="providerBadgeClass(provider.key)">
              {{ provider.label }} · {{ provider.count }}
            </span>
          </div>
        </div>
      </div>

      <div class="mb-4 rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="grid gap-3 xl:grid-cols-[minmax(260px,1fr)_180px_180px_230px_auto]">
          <SearchInput
            v-model="filters.search"
            :placeholder="t('liteLLMModels.searchPlaceholder')"
            :debounce-ms="0"
          />
          <select v-model="filters.provider" class="input h-10 text-sm">
            <option value="">{{ t('liteLLMModels.filters.allProviders') }}</option>
            <option v-for="provider in providers" :key="provider" :value="provider">
              {{ providerLabel(provider) }}
            </option>
          </select>
          <select v-model="filters.mode" class="input h-10 text-sm">
            <option value="">{{ t('liteLLMModels.filters.allModes') }}</option>
            <option v-for="mode in modes" :key="mode" :value="mode">
              {{ modeLabel(mode) }}
            </option>
          </select>
          <select v-model="sortKey" class="input h-10 text-sm">
            <option value="latestDesc">{{ t('liteLLMModels.sort.latestDesc') }}</option>
            <option value="nameAsc">{{ t('liteLLMModels.sort.nameAsc') }}</option>
            <option value="nameDesc">{{ t('liteLLMModels.sort.nameDesc') }}</option>
            <option value="providerAsc">{{ t('liteLLMModels.sort.providerAsc') }}</option>
            <option value="modeAsc">{{ t('liteLLMModels.sort.modeAsc') }}</option>
            <option value="inputAsc">{{ t('liteLLMModels.sort.inputAsc') }}</option>
            <option value="inputDesc">{{ t('liteLLMModels.sort.inputDesc') }}</option>
            <option value="outputAsc">{{ t('liteLLMModels.sort.outputAsc') }}</option>
            <option value="outputDesc">{{ t('liteLLMModels.sort.outputDesc') }}</option>
            <option value="contextDesc">{{ t('liteLLMModels.sort.contextDesc') }}</option>
          </select>
          <div class="inline-flex h-10 rounded-lg border border-gray-200 bg-gray-50 p-1 dark:border-dark-700 dark:bg-dark-950">
            <button
              type="button"
              :title="t('liteLLMModels.view.table')"
              :aria-label="t('liteLLMModels.view.table')"
              @click="viewMode = 'table'"
              :class="viewButtonClass(viewMode === 'table')"
            >
              <Icon name="menu" size="sm" />
            </button>
            <button
              type="button"
              :title="t('liteLLMModels.view.cards')"
              :aria-label="t('liteLLMModels.view.cards')"
              @click="viewMode = 'cards'"
              :class="viewButtonClass(viewMode === 'cards')"
            >
              <Icon name="grid" size="sm" />
            </button>
          </div>
        </div>
      </div>

      <div class="mb-3 flex flex-wrap items-center justify-between gap-2 text-xs text-gray-500 dark:text-dark-400">
        <span>{{ t('liteLLMModels.resultCount', { count: displayedModels.length, total: models.length }) }}</span>
        <span v-if="lastUpdated">{{ t('liteLLMModels.lastLoaded', { time: lastUpdated }) }}</span>
      </div>

      <div
        v-if="error && !loading"
        class="mb-4 rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700 dark:border-rose-900/40 dark:bg-rose-900/10 dark:text-rose-300"
      >
        {{ t('liteLLMModels.loadFailed', { msg: error }) }}
      </div>

      <div v-if="viewMode === 'table'" class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="overflow-x-auto">
          <table class="w-full min-w-[1280px] divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-100/80 dark:bg-dark-800">
              <tr>
                <th v-for="column in columns" :key="column.key" scope="col" class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-normal text-gray-500 dark:text-dark-400">
                  <button
                    v-if="column.sort"
                    type="button"
                    class="inline-flex items-center gap-1 hover:text-gray-800 dark:hover:text-dark-100"
                    @click="setColumnSort(column.sort)"
                  >
                    <span>{{ column.label }}</span>
                    <Icon name="sort" size="xs" />
                  </button>
                  <span v-else>{{ column.label }}</span>
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-800">
              <tr v-if="loading && !models.length">
                <td :colspan="columns.length" class="px-4 py-12 text-center text-sm text-gray-500 dark:text-dark-400">
                  {{ t('common.loading', 'Loading...') }}
                </td>
              </tr>
              <tr v-else-if="displayedModels.length === 0">
                <td :colspan="columns.length" class="px-4 py-12 text-center text-sm text-gray-500 dark:text-dark-400">
                  {{ t('liteLLMModels.empty') }}
                </td>
              </tr>
              <template v-else>
                <tr
                  v-for="model in displayedModels"
                  :key="model.name"
                  class="cursor-pointer transition-colors hover:bg-gray-50 focus:bg-gray-50 dark:hover:bg-dark-800 dark:focus:bg-dark-800"
                  tabindex="0"
                  @click="openDetail(model)"
                  @keydown.enter.prevent="openDetail(model)"
                  @keydown.space.prevent="openDetail(model)"
                >
                  <td class="max-w-[320px] px-4 py-3">
                    <div class="truncate text-sm font-medium text-gray-950 dark:text-white" :title="model.name">
                      {{ model.name }}
                    </div>
                    <div class="mt-1 flex flex-wrap gap-1">
                      <span v-for="cap in model.capabilities.slice(0, 3)" :key="cap" class="rounded bg-gray-100 px-1.5 py-0.5 text-[11px] text-gray-600 dark:bg-dark-800 dark:text-dark-300">
                        {{ capabilityLabel(cap) }}
                      </span>
                    </div>
                  </td>
                  <td class="px-4 py-3 text-sm">
                    <span :class="providerBadgeClass(model.provider, model.name)">{{ providerLabel(model.provider, model.name) }}</span>
                  </td>
                  <td class="px-4 py-3 text-sm">{{ modeLabel(model.mode) }}</td>
                  <td class="px-4 py-3 text-sm">{{ formatNumber(contextWindow(model)) }}</td>
                  <td class="px-4 py-3 text-sm">{{ formatNumber(model.max_output_tokens) }}</td>
                  <td class="px-4 py-3 text-sm"><span :class="priceClass(model.input_price_per_mtok_usd)">{{ formatTokenPrice(model.input_price_per_mtok_usd) }}</span></td>
                  <td class="px-4 py-3 text-sm"><span :class="priceClass(model.output_price_per_mtok_usd)">{{ formatTokenPrice(model.output_price_per_mtok_usd) }}</span></td>
                  <td class="px-4 py-3 text-sm"><span :class="priceClass(model.cache_creation_price_per_mtok_usd)">{{ formatTokenPrice(model.cache_creation_price_per_mtok_usd) }}</span></td>
                  <td class="px-4 py-3 text-sm"><span :class="priceClass(model.cache_read_price_per_mtok_usd)">{{ formatTokenPrice(model.cache_read_price_per_mtok_usd) }}</span></td>
                  <td class="px-4 py-3 text-sm">{{ formatImagePrice(model.output_price_per_image_usd) }}</td>
                  <td class="px-4 py-3 text-sm">
                    <div class="flex flex-wrap gap-1">
                      <span v-for="m in model.supported_modalities" :key="m" class="rounded bg-blue-50 px-1.5 py-0.5 text-[11px] text-blue-700 dark:bg-blue-900/20 dark:text-blue-300">
                        {{ modalityLabel(m) }}
                      </span>
                      <span v-if="model.supported_modalities.length === 0">-</span>
                    </div>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
      </div>

      <div v-else class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="model in displayedModels"
          :key="model.name"
          class="cursor-pointer rounded-lg border border-gray-200 bg-white p-4 shadow-sm transition-shadow hover:shadow-md focus:outline-none focus:ring-2 focus:ring-primary-500 dark:border-dark-700 dark:bg-dark-900"
          tabindex="0"
          @click="openDetail(model)"
          @keydown.enter.prevent="openDetail(model)"
          @keydown.space.prevent="openDetail(model)"
        >
          <div class="mb-3 flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="truncate font-medium text-gray-950 dark:text-white" :title="model.name">
                {{ model.name }}
              </div>
              <div class="mt-2 flex flex-wrap items-center gap-1.5">
                <span :class="providerBadgeClass(model.provider, model.name)">{{ providerLabel(model.provider, model.name) }}</span>
                <span class="rounded-md bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">
                  {{ modeLabel(model.mode) }}
                </span>
              </div>
            </div>
            <span class="shrink-0 rounded-md bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">
              {{ categoryLabel(model.category) }}
            </span>
          </div>

          <div class="grid grid-cols-2 gap-2 text-sm">
            <Metric :label="t('liteLLMModels.columns.context')" :value="formatNumber(contextWindow(model))" />
            <Metric :label="t('liteLLMModels.columns.maxOutput')" :value="formatNumber(model.max_output_tokens)" />
            <Metric :label="t('liteLLMModels.columns.input')" :value="formatTokenPrice(model.input_price_per_mtok_usd)" />
            <Metric :label="t('liteLLMModels.columns.output')" :value="formatTokenPrice(model.output_price_per_mtok_usd)" />
            <Metric :label="t('liteLLMModels.columns.cacheWrite')" :value="formatTokenPrice(model.cache_creation_price_per_mtok_usd)" />
            <Metric :label="t('liteLLMModels.columns.cacheRead')" :value="formatTokenPrice(model.cache_read_price_per_mtok_usd)" />
          </div>

          <div class="mt-3 flex flex-wrap gap-1">
            <span v-for="cap in model.capabilities" :key="cap" class="rounded bg-gray-100 px-1.5 py-0.5 text-[11px] text-gray-600 dark:bg-dark-800 dark:text-dark-300">
              {{ capabilityLabel(cap) }}
            </span>
            <span v-if="model.capabilities.length === 0" class="text-xs text-gray-400 dark:text-dark-500">-</span>
          </div>
        </div>
      </div>
    </main>

    <div
      v-if="selectedModel"
      class="fixed inset-0 z-50 bg-gray-950/40 backdrop-blur-sm"
      @click.self="closeDetail"
    >
      <aside class="ml-auto flex h-full w-full max-w-3xl flex-col overflow-hidden bg-white shadow-2xl dark:bg-dark-900 sm:border-l sm:border-gray-200 sm:dark:border-dark-700">
        <header class="flex items-start justify-between gap-4 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span :class="providerBadgeClass(selectedModel.provider, selectedModel.name)">
                {{ providerLabel(selectedModel.provider, selectedModel.name) }}
              </span>
              <span class="rounded-md bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">
                {{ modeLabel(selectedModel.mode) }}
              </span>
            </div>
            <h2 class="mt-2 break-words text-xl font-semibold text-gray-950 dark:text-white">
              {{ selectedModel.name }}
            </h2>
          </div>
          <button
            type="button"
            class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md text-gray-500 hover:bg-gray-100 hover:text-gray-900 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :aria-label="t('common.close', 'Close')"
            @click="closeDetail"
          >
            <Icon name="x" size="sm" />
          </button>
        </header>

        <div class="flex-1 overflow-y-auto px-5 py-5">
          <section class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <Metric :label="t('liteLLMModels.columns.context')" :value="formatNumber(contextWindow(selectedModel))" />
            <Metric :label="t('liteLLMModels.columns.maxOutput')" :value="formatNumber(selectedModel.max_output_tokens)" />
            <Metric :label="t('liteLLMModels.detail.maxTokens')" :value="formatNumber(selectedModel.max_tokens)" />
            <Metric :label="t('liteLLMModels.detail.category')" :value="categoryLabel(selectedModel.category)" />
            <Metric :label="t('liteLLMModels.detail.promptCaching')" :value="booleanLabel(selectedModel.supports_prompt_caching)" />
            <Metric :label="t('liteLLMModels.detail.serviceTier')" :value="booleanLabel(selectedModel.supports_service_tier)" />
          </section>

          <section class="mt-6">
            <h3 class="text-sm font-semibold text-gray-950 dark:text-white">{{ t('liteLLMModels.detail.capabilities') }}</h3>
            <div class="mt-3 flex flex-wrap gap-1.5">
              <span v-for="cap in selectedModel.capabilities" :key="cap" class="rounded-md bg-gray-100 px-2 py-1 text-xs text-gray-700 dark:bg-dark-800 dark:text-dark-200">
                {{ capabilityLabel(cap) }}
              </span>
              <span v-for="flag in selectedModel.support_flags" :key="`flag-${flag}`" class="rounded-md bg-emerald-50 px-2 py-1 text-xs text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300">
                {{ supportFlagLabel(flag) }}
              </span>
              <span v-if="selectedModel.capabilities.length === 0 && selectedModel.support_flags.length === 0" class="text-sm text-gray-400 dark:text-dark-500">-</span>
            </div>
          </section>

          <section class="mt-6 grid gap-4 md:grid-cols-2">
            <div>
              <h3 class="text-sm font-semibold text-gray-950 dark:text-white">{{ t('liteLLMModels.detail.inputModalities') }}</h3>
              <div class="mt-3 flex flex-wrap gap-1.5">
                <span v-for="item in selectedModel.supported_modalities" :key="item" class="rounded-md bg-sky-50 px-2 py-1 text-xs text-sky-700 dark:bg-sky-900/20 dark:text-sky-300">
                  {{ modalityLabel(item) }}
                </span>
                <span v-if="selectedModel.supported_modalities.length === 0" class="text-sm text-gray-400 dark:text-dark-500">-</span>
              </div>
            </div>
            <div>
              <h3 class="text-sm font-semibold text-gray-950 dark:text-white">{{ t('liteLLMModels.detail.outputModalities') }}</h3>
              <div class="mt-3 flex flex-wrap gap-1.5">
                <span v-for="item in selectedModel.output_modalities" :key="item" class="rounded-md bg-indigo-50 px-2 py-1 text-xs text-indigo-700 dark:bg-indigo-900/20 dark:text-indigo-300">
                  {{ modalityLabel(item) }}
                </span>
                <span v-if="selectedModel.output_modalities.length === 0" class="text-sm text-gray-400 dark:text-dark-500">-</span>
              </div>
            </div>
          </section>

          <section class="mt-6">
            <h3 class="text-sm font-semibold text-gray-950 dark:text-white">{{ t('liteLLMModels.detail.pricingRules') }}</h3>
            <div class="mt-3 space-y-4">
              <div
                v-for="section in pricingSections(selectedModel)"
                :key="section.key"
                class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700"
              >
                <div class="bg-gray-50 px-3 py-2 text-xs font-semibold uppercase text-gray-500 dark:bg-dark-800 dark:text-dark-400">
                  {{ section.title }}
                </div>
                <table class="w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
                  <tbody class="divide-y divide-gray-100 dark:divide-dark-800">
                    <tr v-for="row in section.rows" :key="row.key">
                      <td class="w-1/2 px-3 py-2 text-gray-600 dark:text-dark-300">{{ row.label }}</td>
                      <td class="px-3 py-2 text-right font-medium text-gray-950 dark:text-white">{{ row.value }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <div
                v-if="pricingSections(selectedModel).length === 0"
                class="rounded-lg border border-gray-200 px-3 py-6 text-center text-sm text-gray-400 dark:border-dark-700 dark:text-dark-500"
              >
                -
              </div>
            </div>
          </section>
        </div>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import modelSquareAPI, { type LiteLLMModel } from '@/api/models'
import SearchInput from '@/components/common/SearchInput.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { contextWindow, sortLiteLLMModels, type LiteLLMSortKey } from './liteLLMModelSort'

type ViewMode = 'table' | 'cards'
type SortKey = LiteLLMSortKey
type AllowedProvider = 'anthropic' | 'gemini' | 'openai'
type DetailRow = {
  key: string
  label: string
  value: string
}
type PricingSection = {
  key: string
  title: string
  rows: DetailRow[]
}

const ALLOWED_PROVIDERS: AllowedProvider[] = ['anthropic', 'gemini', 'openai']
const PROVIDER_LABELS: Record<AllowedProvider, string> = {
  anthropic: 'Anthropic',
  gemini: 'Gemini',
  openai: 'OpenAI',
}

const Metric = defineComponent({
  props: {
    label: { type: String, required: true },
    value: { type: String, required: true },
  },
  setup(props) {
    return () =>
      h('div', { class: 'rounded-md bg-gray-50 p-2 dark:bg-dark-800' }, [
        h('div', { class: 'text-[11px] text-gray-500 dark:text-dark-400' }, props.label),
        h('div', { class: 'mt-1 truncate font-medium text-gray-950 dark:text-white' }, props.value),
      ])
  },
})

const { t } = useI18n()
const appStore = useAppStore()

const models = ref<LiteLLMModel[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const viewMode = ref<ViewMode>('table')
const sortKey = ref<SortKey>('latestDesc')
const lastUpdated = ref('')
const selectedModel = ref<LiteLLMModel | null>(null)
let abortCtl: AbortController | null = null

const filters = ref({
  search: '',
  provider: '',
  mode: '',
})

const siteName = computed(
  () => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'SudoRouter',
)

const columns = computed(() => [
  { key: 'name', label: t('liteLLMModels.columns.model'), sort: 'nameAsc' as SortKey },
  { key: 'provider', label: t('liteLLMModels.columns.provider'), sort: 'providerAsc' as SortKey },
  { key: 'mode', label: t('liteLLMModels.columns.mode'), sort: 'modeAsc' as SortKey },
  { key: 'context', label: t('liteLLMModels.columns.context'), sort: 'contextDesc' as SortKey },
  { key: 'maxOutput', label: t('liteLLMModels.columns.maxOutput') },
  { key: 'input', label: t('liteLLMModels.columns.input'), sort: 'inputAsc' as SortKey },
  { key: 'output', label: t('liteLLMModels.columns.output'), sort: 'outputAsc' as SortKey },
  { key: 'cacheWrite', label: t('liteLLMModels.columns.cacheWrite') },
  { key: 'cacheRead', label: t('liteLLMModels.columns.cacheRead') },
  { key: 'image', label: t('liteLLMModels.columns.image') },
  { key: 'modalities', label: t('liteLLMModels.columns.modalities') },
])

const providers = computed(() =>
  ALLOWED_PROVIDERS.filter((provider) =>
    models.value.some((m) => normalizedProvider(m.provider, m.name) === provider),
  ),
)

const modes = computed(() =>
  Array.from(new Set(models.value.map((m) => m.mode).filter(Boolean))).sort((a, b) =>
    a.localeCompare(b),
  ),
)

const displayedModels = computed(() => sortLiteLLMModels(filterModels(models.value), sortKey.value))
const providerSummary = computed(() =>
  ALLOWED_PROVIDERS.map((key) => ({
    key,
    label: PROVIDER_LABELS[key],
    count: models.value.filter((m) => normalizedProvider(m.provider, m.name) === key).length,
  })).filter((item) => item.count > 0),
)

async function reload() {
  abortCtl?.abort()
  abortCtl = new AbortController()
  loading.value = true
  error.value = null
  try {
    const result = await modelSquareAPI.listLiteLLMModelsWithDiagnostics({ signal: abortCtl.signal })
    logCSVOnlyModels(result.diagnostics.csv_only_models)
    const rows = result.items
    models.value = rows.filter((row) => normalizedProvider(row.provider, row.name) != null)
    lastUpdated.value = new Intl.DateTimeFormat(undefined, {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    }).format(new Date())
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    if (!/aborted|canceled/i.test(msg)) {
      error.value = msg
      console.error('[LiteLLMModelView] load failed', e)
    }
  } finally {
    loading.value = false
  }
}

function logCSVOnlyModels(rows: Array<{ serial_number: number; id: string }>) {
  if (!rows.length) return
  console.warn(
    '[LiteLLMModelView] CSV whitelist models missing from LiteLLM pricing, not shown on /model:',
    rows.map((row) => ({
      serial_number: row.serial_number,
      id: row.id,
    })),
  )
}

onMounted(reload)

function filterModels(rows: LiteLLMModel[]) {
  const q = filters.value.search.trim().toLowerCase()
  return rows.filter((row) => {
    if (filters.value.provider && normalizedProvider(row.provider, row.name) !== filters.value.provider) return false
    if (filters.value.mode && row.mode !== filters.value.mode) return false
    if (!q) return true
    return [
      row.name,
      row.provider,
      row.mode,
      row.category,
      ...row.supported_modalities,
      ...row.output_modalities,
      ...row.support_flags,
      ...row.capabilities,
    ]
      .join(' ')
      .toLowerCase()
      .includes(q)
  })
}

function setColumnSort(next: SortKey) {
  const reverse: Partial<Record<SortKey, SortKey>> = {
    nameAsc: 'nameDesc',
    inputAsc: 'inputDesc',
    inputDesc: 'inputAsc',
    outputAsc: 'outputDesc',
    outputDesc: 'outputAsc',
  }
  sortKey.value = sortKey.value === next && reverse[next] ? reverse[next] : next
}

function openDetail(model: LiteLLMModel) {
  selectedModel.value = model
}

function closeDetail() {
  selectedModel.value = null
}

function formatNumber(value: number): string {
  if (!value) return '-'
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(value)
}

function formatTokenPrice(value: number | null): string {
  if (!hasPrice(value)) return '-'
  return `$${formatPriceNumber(value)}/M`
}

function formatImagePrice(value: number | null): string {
  if (!hasPrice(value)) return '-'
  return `$${formatPriceNumber(value)}`
}

function hasPrice(value: number | null): value is number {
  return value != null && Number.isFinite(value)
}

function formatPriceNumber(value: number): string {
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 0,
    maximumFractionDigits: value > 0 && value < 0.01 ? 6 : 4,
  }).format(value)
}

function modeLabel(mode: string): string {
  return mode ? t(`modelSquare.modelTypes.${mode}`, mode) : '-'
}

function categoryLabel(category: string): string {
  return category ? t(`modelSquare.categories.${category}`, category) : '-'
}

function normalizedProvider(provider: string, modelName = ''): AllowedProvider | null {
  const normalized = provider.trim().toLowerCase()
  const name = modelName.trim().toLowerCase()
  if (normalized === 'authropic') return 'anthropic'
  if (normalized === 'text-completion-openai') return 'openai'
  if ((ALLOWED_PROVIDERS as string[]).includes(normalized)) return normalized as AllowedProvider
  if (name.startsWith('claude-')) return 'anthropic'
  if (name.startsWith('gemini-')) return 'gemini'
  if (/^(gpt-|o\d|openai)/.test(name)) return 'openai'
  return null
}

function providerLabel(provider: string, modelName = ''): string {
  const normalized = normalizedProvider(provider, modelName)
  return normalized ? PROVIDER_LABELS[normalized] : provider || '-'
}

function providerBadgeClass(provider: string, modelName = '') {
  const normalized = normalizedProvider(provider, modelName)
  const base = 'inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium'
  switch (normalized) {
    case 'anthropic':
      return `${base} bg-amber-50 text-amber-700 ring-1 ring-amber-200 dark:bg-amber-900/20 dark:text-amber-300 dark:ring-amber-800/60`
    case 'gemini':
      return `${base} bg-sky-50 text-sky-700 ring-1 ring-sky-200 dark:bg-sky-900/20 dark:text-sky-300 dark:ring-sky-800/60`
    case 'openai':
      return `${base} bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-900/20 dark:text-emerald-300 dark:ring-emerald-800/60`
    default:
      return `${base} bg-gray-100 text-gray-600 ring-1 ring-gray-200 dark:bg-dark-800 dark:text-dark-300 dark:ring-dark-700`
  }
}

function priceClass(value: number | null) {
  return [
    'inline-flex min-w-[72px] justify-end rounded-md px-2 py-0.5 font-medium tabular-nums',
    hasPrice(value)
      ? 'bg-gray-100 text-gray-900 dark:bg-dark-800 dark:text-dark-100'
      : 'bg-gray-50 text-gray-400 dark:bg-dark-900 dark:text-dark-500',
  ]
}

function capabilityLabel(capability: string): string {
  return capability ? t(`modelSquare.capabilities.${capability}`, capability) : capability
}

function supportFlagLabel(flag: string): string {
  return flag ? t(`modelSquare.capabilities.${flag}`, humanizeKey(flag)) : flag
}

function modalityLabel(modality: string): string {
  return modality ? t(`modelSquare.modalities.${modality}`, modality) : modality
}

function booleanLabel(value: boolean): string {
  return value ? t('common.yes', 'Yes') : t('common.no', 'No')
}

function humanizeKey(key: string): string {
  return key
    .split('_')
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

function pricingSections(model: LiteLLMModel): PricingSection[] {
  const sections: PricingSection[] = [
    {
      key: 'token',
      title: t('liteLLMModels.pricingSections.token'),
      rows: [
        priceRow('input_cost_per_token', t('liteLLMModels.detail.inputPrice'), formatTokenPrice(model.input_price_per_mtok_usd)),
        priceRow('output_cost_per_token', t('liteLLMModels.detail.outputPrice'), formatTokenPrice(model.output_price_per_mtok_usd)),
        rawPriceRow(model, 'input_cost_per_token_above_128k_tokens'),
        rawPriceRow(model, 'output_cost_per_token_above_128k_tokens'),
        rawPriceRow(model, 'input_cost_per_token_above_200k_tokens'),
        rawPriceRow(model, 'output_cost_per_token_above_200k_tokens'),
      ],
    },
    {
      key: 'cache',
      title: t('liteLLMModels.pricingSections.cache'),
      rows: [
        priceRow('cache_creation_input_token_cost', t('liteLLMModels.detail.cacheWritePrice'), formatTokenPrice(model.cache_creation_price_per_mtok_usd)),
        priceRow('cache_creation_input_token_cost_above_1hr', t('liteLLMModels.detail.cacheWrite1hPrice'), formatTokenPrice(model.cache_creation_above_1h_price_per_mtok_usd)),
        priceRow('cache_read_input_token_cost', t('liteLLMModels.detail.cacheReadPrice'), formatTokenPrice(model.cache_read_price_per_mtok_usd)),
        rawPriceRow(model, 'cache_creation_input_token_cost_above_200k_tokens'),
        rawPriceRow(model, 'cache_read_input_token_cost_above_200k_tokens'),
        rawPriceRow(model, 'cache_creation_input_audio_token_cost'),
        rawPriceRow(model, 'cache_read_input_audio_token_cost'),
        rawPriceRow(model, 'cache_read_input_image_token_cost'),
      ],
    },
    {
      key: 'serviceTier',
      title: t('liteLLMModels.pricingSections.serviceTier'),
      rows: [
        priceRow('input_cost_per_token_priority', t('liteLLMModels.detail.inputPriorityPrice'), formatTokenPrice(model.input_price_priority_per_mtok_usd)),
        priceRow('output_cost_per_token_priority', t('liteLLMModels.detail.outputPriorityPrice'), formatTokenPrice(model.output_price_priority_per_mtok_usd)),
        priceRow('cache_read_input_token_cost_priority', t('liteLLMModels.detail.cacheReadPriorityPrice'), formatTokenPrice(model.cache_read_priority_price_per_mtok_usd)),
        rawPriceRow(model, 'input_cost_per_audio_token_priority'),
        rawPriceRow(model, 'input_cost_per_token_above_200k_tokens_priority'),
        rawPriceRow(model, 'output_cost_per_token_above_200k_tokens_priority'),
        rawPriceRow(model, 'cache_read_input_token_cost_above_200k_tokens_priority'),
        rawPriceRow(model, 'input_cost_per_token_flex'),
        rawPriceRow(model, 'output_cost_per_token_flex'),
        rawPriceRow(model, 'cache_read_input_token_cost_flex'),
      ],
    },
    {
      key: 'multimodal',
      title: t('liteLLMModels.pricingSections.multimodal'),
      rows: [
        rawPriceRow(model, 'input_cost_per_image'),
        rawPriceRow(model, 'input_cost_per_image_above_128k_tokens'),
        rawPriceRow(model, 'input_cost_per_image_token'),
        priceRow('output_cost_per_image', t('liteLLMModels.detail.imagePrice'), formatImagePrice(model.output_price_per_image_usd)),
        priceRow('output_cost_per_image_token', t('liteLLMModels.detail.imageTokenPrice'), formatTokenPrice(model.output_price_per_image_mtok_usd)),
        rawPriceRow(model, 'input_cost_per_audio_token'),
        rawPriceRow(model, 'output_cost_per_audio_token'),
        rawPriceRow(model, 'input_cost_per_audio_per_second'),
        rawPriceRow(model, 'input_cost_per_audio_per_second_above_128k_tokens'),
        rawPriceRow(model, 'input_cost_per_video_per_second'),
        rawPriceRow(model, 'input_cost_per_video_per_second_above_128k_tokens'),
        rawPriceRow(model, 'output_cost_per_second'),
      ],
    },
    {
      key: 'batchAndText',
      title: t('liteLLMModels.pricingSections.batchAndText'),
      rows: [
        rawPriceRow(model, 'input_cost_per_token_batches'),
        rawPriceRow(model, 'output_cost_per_token_batches'),
        rawPriceRow(model, 'input_cost_per_character'),
        rawPriceRow(model, 'output_cost_per_character'),
        rawPriceRow(model, 'input_cost_per_character_above_128k_tokens'),
        rawPriceRow(model, 'output_cost_per_character_above_128k_tokens'),
        rawPriceRow(model, 'search_context_cost_per_query'),
      ],
    },
    {
      key: 'longContext',
      title: t('liteLLMModels.pricingSections.longContext'),
      rows: [
        thresholdRow('long_context_input_token_threshold', t('liteLLMModels.detail.longContextThreshold'), model.long_context_input_token_threshold),
        multiplierRow('long_context_input_cost_multiplier', t('liteLLMModels.detail.longContextInputMultiplier'), model.long_context_input_cost_multiplier),
        multiplierRow('long_context_output_cost_multiplier', t('liteLLMModels.detail.longContextOutputMultiplier'), model.long_context_output_cost_multiplier),
      ],
    },
  ]

  return sections
    .map((section) => ({
      ...section,
      rows: section.rows.filter((row) => row.value !== '-'),
    }))
    .filter((section) => section.rows.length > 0)
}

function priceRow(key: string, label: string, value: string): DetailRow {
  return { key, label, value }
}

function rawPriceRow(model: LiteLLMModel, key: string): DetailRow {
  return priceRow(key, pricingFieldLabel(key), formatPricingFieldValue(key, model.raw_fields?.[key]))
}

function thresholdRow(key: string, label: string, value: number): DetailRow {
  return priceRow(key, label, value > 0 ? formatNumber(value) : '-')
}

function multiplierRow(key: string, label: string, value: number): DetailRow {
  return priceRow(key, label, value > 0 ? `${formatPriceNumber(value)}x` : '-')
}

function formatPricingFieldValue(key: string, value: unknown): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '-'
  if (/per_image$/.test(key)) return `$${formatPriceNumber(value)}`
  if (/per_(audio_|video_)?second/.test(key)) return `$${formatPriceNumber(value)}/s`
  if (/per_character/.test(key)) return `$${formatPriceNumber(value * 1_000_000)}/M chars`
  if (/per_query/.test(key)) return `$${formatPriceNumber(value)}/query`
  if (/(token_cost|per_token|token_batches|audio_token|image_token|reasoning_token)/.test(key)) {
    return `$${formatPriceNumber(value * 1_000_000)}/M`
  }
  return formatRawValue(value)
}

function pricingFieldLabel(key: string): string {
  const labels: Record<string, string> = {
    input_cost_per_token_above_128k_tokens: t('liteLLMModels.pricingLabels.inputAbove128k'),
    output_cost_per_token_above_128k_tokens: t('liteLLMModels.pricingLabels.outputAbove128k'),
    input_cost_per_token_above_200k_tokens: t('liteLLMModels.pricingLabels.inputAbove200k'),
    output_cost_per_token_above_200k_tokens: t('liteLLMModels.pricingLabels.outputAbove200k'),
    cache_creation_input_token_cost_above_200k_tokens: t('liteLLMModels.pricingLabels.cacheWriteAbove200k'),
    cache_read_input_token_cost_above_200k_tokens: t('liteLLMModels.pricingLabels.cacheReadAbove200k'),
    cache_creation_input_audio_token_cost: t('liteLLMModels.pricingLabels.audioCacheWrite'),
    cache_read_input_audio_token_cost: t('liteLLMModels.pricingLabels.audioCacheRead'),
    cache_read_input_image_token_cost: t('liteLLMModels.pricingLabels.imageCacheRead'),
    input_cost_per_audio_token_priority: t('liteLLMModels.pricingLabels.priorityAudioInput'),
    input_cost_per_token_above_200k_tokens_priority: t('liteLLMModels.pricingLabels.priorityInputAbove200k'),
    output_cost_per_token_above_200k_tokens_priority: t('liteLLMModels.pricingLabels.priorityOutputAbove200k'),
    cache_read_input_token_cost_above_200k_tokens_priority: t('liteLLMModels.pricingLabels.priorityCacheReadAbove200k'),
    input_cost_per_token_flex: t('liteLLMModels.pricingLabels.flexInput'),
    output_cost_per_token_flex: t('liteLLMModels.pricingLabels.flexOutput'),
    cache_read_input_token_cost_flex: t('liteLLMModels.pricingLabels.flexCacheRead'),
    input_cost_per_image: t('liteLLMModels.pricingLabels.imageInput'),
    input_cost_per_image_above_128k_tokens: t('liteLLMModels.pricingLabels.imageInputAbove128k'),
    input_cost_per_image_token: t('liteLLMModels.pricingLabels.imageInputToken'),
    input_cost_per_audio_token: t('liteLLMModels.pricingLabels.audioInputToken'),
    output_cost_per_audio_token: t('liteLLMModels.pricingLabels.audioOutputToken'),
    input_cost_per_audio_per_second: t('liteLLMModels.pricingLabels.audioInputSecond'),
    input_cost_per_audio_per_second_above_128k_tokens: t('liteLLMModels.pricingLabels.audioInputSecondAbove128k'),
    input_cost_per_video_per_second: t('liteLLMModels.pricingLabels.videoInputSecond'),
    input_cost_per_video_per_second_above_128k_tokens: t('liteLLMModels.pricingLabels.videoInputSecondAbove128k'),
    output_cost_per_second: t('liteLLMModels.pricingLabels.outputSecond'),
    input_cost_per_token_batches: t('liteLLMModels.pricingLabels.batchInput'),
    output_cost_per_token_batches: t('liteLLMModels.pricingLabels.batchOutput'),
    input_cost_per_character: t('liteLLMModels.pricingLabels.characterInput'),
    output_cost_per_character: t('liteLLMModels.pricingLabels.characterOutput'),
    input_cost_per_character_above_128k_tokens: t('liteLLMModels.pricingLabels.characterInputAbove128k'),
    output_cost_per_character_above_128k_tokens: t('liteLLMModels.pricingLabels.characterOutputAbove128k'),
    search_context_cost_per_query: t('liteLLMModels.pricingLabels.searchContext'),
  }
  return labels[key] || humanizeKey(key)
}

function formatRawValue(value: unknown): string {
  if (typeof value === 'number') {
    return Number.isFinite(value) ? String(value) : '-'
  }
  if (typeof value === 'string') return value || '-'
  if (typeof value === 'boolean') return booleanLabel(value)
  if (value == null) return '-'
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

function viewButtonClass(active: boolean) {
  return [
    'inline-flex h-8 w-9 items-center justify-center rounded-md transition-colors',
    active
      ? 'bg-gray-900 text-white dark:bg-dark-100 dark:text-dark-950'
      : 'text-gray-500 hover:bg-gray-100 hover:text-gray-800 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-dark-100',
  ]
}
</script>
