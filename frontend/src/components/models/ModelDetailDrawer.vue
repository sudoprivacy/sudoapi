<template>
  <Teleport to="body">
    <transition name="fade">
      <div
        v-if="open"
        class="fixed inset-0 z-[99998] bg-black/40 backdrop-blur-sm"
        @click="close"
      />
    </transition>
    <transition name="slide-in-right">
      <aside
        v-if="open && card"
        class="fixed right-0 top-0 z-[99999] h-full w-full overflow-y-auto bg-white shadow-2xl dark:bg-dark-900 sm:max-w-2xl"
        role="dialog"
        aria-modal="true"
      >
        <!-- Header -->
        <header
          class="sticky top-0 z-10 flex items-center justify-between gap-3 border-b border-gray-200 bg-white/95 px-5 py-4 backdrop-blur dark:border-dark-700 dark:bg-dark-900/95"
        >
          <div class="flex min-w-0 flex-1 items-center gap-3">
            <img
              v-if="card.icon_url"
              :src="card.icon_url"
              alt=""
              class="h-10 w-10 flex-shrink-0 rounded-xl object-contain"
            />
            <div
              v-else
              class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl"
              :class="categoryGradient"
            >
              <ModelIcon :model="card.name" size="26px" />
            </div>
            <div class="min-w-0">
              <h2 class="truncate text-lg font-semibold text-gray-900 dark:text-white">
                {{ card.display_name || card.name }}
              </h2>
              <p class="truncate text-xs text-gray-500 dark:text-dark-400">{{ card.name }}</p>
            </div>
          </div>
          <button
            type="button"
            @click="close"
            class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :aria-label="t('common.close')"
          >
            <Icon name="x" size="md" />
          </button>
        </header>

        <div class="space-y-6 px-5 py-5">
          <!-- Section 1: 基本信息 -->
          <section>
            <h3 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('modelSquare.detail.basicInfo') }}
            </h3>
            <p class="mb-3 text-sm text-gray-600 dark:text-dark-300">
              {{ card.description || t('modelSquare.detail.noDescription') }}
            </p>
            <dl class="grid grid-cols-2 gap-3 text-xs sm:grid-cols-3">
              <div v-if="card.context_window > 0">
                <dt class="text-gray-500 dark:text-dark-400">
                  {{ t('modelSquare.detail.contextWindow') }}
                </dt>
                <dd class="font-medium text-gray-900 dark:text-white">
                  {{ formatTokens(card.context_window) }}
                </dd>
              </div>
              <div v-if="card.max_output > 0">
                <dt class="text-gray-500 dark:text-dark-400">
                  {{ t('modelSquare.detail.maxOutput') }}
                </dt>
                <dd class="font-medium text-gray-900 dark:text-white">
                  {{ formatTokens(card.max_output) }}
                </dd>
              </div>
              <div>
                <dt class="text-gray-500 dark:text-dark-400">
                  {{ t('modelSquare.detail.category') }}
                </dt>
                <dd class="font-medium text-gray-900 dark:text-white">
                  {{ t(`modelSquare.categories.${card.category}`, card.category) }}
                </dd>
              </div>
              <div v-if="card.model_type">
                <dt class="text-gray-500 dark:text-dark-400">
                  {{ t('modelSquare.detail.modelType') }}
                </dt>
                <dd class="font-medium text-gray-900 dark:text-white">
                  {{ modelTypeLabel(card.model_type) }}
                </dd>
              </div>
              <div v-if="card.input_modalities && card.input_modalities.length">
                <dt class="text-gray-500 dark:text-dark-400">
                  {{ t('modelSquare.detail.inputModalities') }}
                </dt>
                <dd class="font-medium text-gray-900 dark:text-white">
                  {{ card.input_modalities.map(modalityLabel).join(' / ') }}
                </dd>
              </div>
              <div v-if="card.output_modalities && card.output_modalities.length">
                <dt class="text-gray-500 dark:text-dark-400">
                  {{ t('modelSquare.detail.outputModalities') }}
                </dt>
                <dd class="font-medium text-gray-900 dark:text-white">
                  {{ card.output_modalities.map(modalityLabel).join(' / ') }}
                </dd>
              </div>
            </dl>
            <div v-if="supportFlags.length" class="mt-3 flex flex-wrap gap-1.5">
              <span
                v-for="cap in supportFlags"
                :key="cap"
                class="rounded-full bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
              >
                {{ supportFlagLabel(cap) }}
              </span>
            </div>
          </section>

          <!-- Section 2: API 端点 -->
          <section>
            <h3 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('modelSquare.detail.endpoints') }}
            </h3>
            <p class="mb-3 text-xs text-gray-500 dark:text-dark-400">
              {{ t('modelSquare.detail.endpointsHint') }}
            </p>
            <div class="space-y-3">
              <div
                v-for="platform in card.platforms"
                :key="platform.platform"
                class="rounded-lg border border-gray-200 p-3 dark:border-dark-700"
              >
                <div class="mb-2 flex items-center gap-2">
                  <span
                    :class="['rounded border px-1.5 py-0.5 text-[11px] font-medium', platformBadgeClass(platform.platform)]"
                  >
                    {{ platform.platform }}
                  </span>
                </div>
                <ul class="space-y-1.5">
                  <li
                    v-for="ep in platform.endpoints"
                    :key="`${platform.platform}-${ep.path}-${ep.method}`"
                    class="flex items-center gap-2 font-mono text-xs"
                  >
                    <span class="rounded bg-green-500/10 px-1.5 py-0.5 text-[10px] font-semibold text-green-700 dark:text-green-300">
                      {{ ep.method }}
                    </span>
                    <code class="text-gray-700 dark:text-dark-200">{{ ep.path }}</code>
                  </li>
                </ul>
              </div>
            </div>
          </section>

          <!-- Section 3: 分组价格 -->
          <section>
            <h3 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('modelSquare.detail.groupPrices') }}
            </h3>
            <p class="mb-3 text-xs text-gray-500 dark:text-dark-400">
              {{ t('modelSquare.detail.groupPricesHint') }}
            </p>
            <div class="space-y-4">
              <div
                v-for="platform in card.platforms"
                :key="platform.platform"
              >
                <div class="mb-1.5 flex items-center gap-2 text-xs text-gray-600 dark:text-dark-300">
                  <span
                    :class="['rounded border px-1.5 py-0.5 text-[11px] font-medium', platformBadgeClass(platform.platform)]"
                  >
                    {{ platform.platform }}
                  </span>
                </div>
                <div v-if="!platform.group_prices.length" class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('modelSquare.detail.noPricing') }}
                </div>
                <div
                  v-else
                  v-for="row in platform.group_prices"
                  :key="`${platform.platform}-${row.group_id}`"
                  class="mb-3 rounded-lg border border-gray-200 bg-white px-3 py-2 dark:border-dark-700 dark:bg-dark-800"
                >
                  <!-- 调用链路（仅多渠道时展示） -->
                  <div
                    v-if="row.channel_chain && row.channel_chain.length > 1"
                    class="mb-2 flex items-center gap-1 text-[11px] text-gray-500 dark:text-dark-400"
                  >
                    <span class="font-medium">{{ t('modelSquare.detail.callChain', { group: row.group_name }) }}</span>
                    <span class="font-mono">{{ row.channel_chain.join(' → ') }}</span>
                  </div>

                  <!-- 分组头 -->
                  <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
                    <div class="flex items-center gap-2">
                      <span class="text-sm font-semibold text-gray-900 dark:text-white">
                        {{ row.group_name }}
                      </span>
                      <span
                        v-if="row.is_exclusive"
                        class="rounded bg-violet-500/10 px-1.5 py-0.5 text-[10px] font-medium text-violet-700 dark:text-violet-300"
                      >
                        {{ t('modelSquare.detail.exclusive') }}
                      </span>
                      <span
                        v-if="row.subscription_type === 'subscription'"
                        class="rounded bg-amber-500/10 px-1.5 py-0.5 text-[10px] font-medium text-amber-700 dark:text-amber-300"
                      >
                        {{ t('modelSquare.detail.subscription') }}
                      </span>
                    </div>
                    <div class="text-[11px] text-gray-500 dark:text-dark-400">
                      {{ t('modelSquare.detail.rateMultiplier') }}:
                      <span class="font-mono text-gray-700 dark:text-dark-200">
                        {{ effectiveMultiplier(row).toFixed(2) }}×
                      </span>
                      <span v-if="row.user_rate_multiplier != null" class="ml-1">
                        ({{ row.base_rate_multiplier }} × {{ row.user_rate_multiplier }})
                      </span>
                    </div>
                  </div>

                  <!-- 价格表 -->
                  <table class="w-full text-xs">
                    <thead>
                      <tr class="border-b border-gray-100 text-left text-[11px] uppercase text-gray-400 dark:border-dark-700">
                        <th class="py-1">{{ t('modelSquare.detail.priceItem') }}</th>
                        <th class="py-1 text-right">{{ t('modelSquare.detail.priceValue') }}</th>
                      </tr>
                    </thead>
                    <tbody>
                      <PricingRow
                        v-if="row.billing_mode === 'token'"
                        :label="t('modelSquare.detail.inputPrice')"
                        :value="row.input_price_per_mtok_usd"
                        :mult="effectiveMultiplier(row)"
                        unit="MTok"
                      />
                      <PricingRow
                        v-if="row.billing_mode === 'token'"
                        :label="t('modelSquare.detail.outputPrice')"
                        :value="row.output_price_per_mtok_usd"
                        :mult="effectiveMultiplier(row)"
                        unit="MTok"
                      />
                      <PricingRow
                        v-if="row.billing_mode === 'token' && row.cache_read_price_per_mtok_usd != null"
                        :label="t('modelSquare.detail.cacheReadPrice')"
                        :value="row.cache_read_price_per_mtok_usd"
                        :mult="effectiveMultiplier(row)"
                        unit="MTok"
                      />
                      <PricingRow
                        v-if="row.billing_mode === 'token' && row.cache_write_price_per_mtok_usd != null"
                        :label="t('modelSquare.detail.cacheWritePrice')"
                        :value="row.cache_write_price_per_mtok_usd"
                        :mult="effectiveMultiplier(row)"
                        unit="MTok"
                      />
                      <PricingRow
                        v-if="row.billing_mode === 'token' && row.cache_creation_5m_price_per_mtok_usd != null"
                        :label="t('modelSquare.detail.cacheCreation5mPrice')"
                        :value="row.cache_creation_5m_price_per_mtok_usd"
                        :mult="effectiveMultiplier(row)"
                        unit="MTok"
                      />
                      <PricingRow
                        v-if="row.billing_mode === 'token' && row.cache_creation_1h_price_per_mtok_usd != null"
                        :label="t('modelSquare.detail.cacheCreation1hPrice')"
                        :value="row.cache_creation_1h_price_per_mtok_usd"
                        :mult="effectiveMultiplier(row)"
                        unit="MTok"
                      />
                      <PricingRow
                        v-if="row.billing_mode === 'token' && row.image_output_price_per_mtok_usd != null"
                        :label="t('modelSquare.detail.imageOutputPrice')"
                        :value="row.image_output_price_per_mtok_usd"
                        :mult="effectiveMultiplier(row)"
                        unit="MTok"
                      />
                      <PricingRow
                        v-if="row.billing_mode !== 'token' && row.per_request_price_usd != null"
                        :label="t('modelSquare.detail.perRequestPrice')"
                        :value="row.per_request_price_usd"
                        :mult="effectiveMultiplier(row)"
                        unit="call"
                      />
                    </tbody>
                  </table>

                  <div
                    v-if="row.intervals && row.intervals.length > 0"
                    class="mt-3 overflow-x-auto rounded-md border border-gray-100 dark:border-dark-700"
                  >
                    <div class="border-b border-gray-100 px-2 py-1.5 text-[11px] font-medium text-gray-500 dark:border-dark-700 dark:text-dark-300">
                      {{ t('modelSquare.detail.intervalPrices') }}
                    </div>
                    <table v-if="row.billing_mode === 'token'" class="min-w-[720px] w-full text-xs">
                      <thead>
                        <tr class="border-b border-gray-100 text-left text-[11px] text-gray-400 dark:border-dark-700">
                          <th class="px-2 py-1">{{ t('modelSquare.detail.contextRange') }}</th>
                          <th class="px-2 py-1 text-right">{{ t('modelSquare.detail.inputPrice') }}</th>
                          <th class="px-2 py-1 text-right">{{ t('modelSquare.detail.outputPrice') }}</th>
                          <th class="px-2 py-1 text-right">{{ t('modelSquare.detail.cacheReadPrice') }}</th>
                          <th class="px-2 py-1 text-right">{{ t('modelSquare.detail.cacheWritePrice') }}</th>
                          <th v-if="hasIntervalCacheCreation5m(row)" class="px-2 py-1 text-right">{{ t('modelSquare.detail.cacheCreation5mPrice') }}</th>
                          <th v-if="hasIntervalCacheCreation1h(row)" class="px-2 py-1 text-right">{{ t('modelSquare.detail.cacheCreation1hPrice') }}</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr
                          v-for="iv in sortedIntervals(row)"
                          :key="`${platform.platform}-${row.group_id}-${iv.min_tokens}-${iv.max_tokens ?? 'inf'}-${iv.sort_order}`"
                          class="border-b border-gray-100 last:border-0 dark:border-dark-700/60"
                        >
                          <td class="whitespace-nowrap px-2 py-1.5 font-mono text-gray-600 dark:text-dark-300">
                            {{ formatIntervalRange(iv) }}
                          </td>
                          <td class="px-2 py-1.5 text-right font-mono text-gray-900 dark:text-white">
                            {{ formatIntervalPrice(iv.input_price_per_mtok_usd, row, 'MTok') }}
                          </td>
                          <td class="px-2 py-1.5 text-right font-mono text-gray-900 dark:text-white">
                            {{ formatIntervalPrice(iv.output_price_per_mtok_usd, row, 'MTok') }}
                          </td>
                          <td class="px-2 py-1.5 text-right font-mono text-gray-900 dark:text-white">
                            {{ formatIntervalPrice(iv.cache_read_price_per_mtok_usd, row, 'MTok') }}
                          </td>
                          <td class="px-2 py-1.5 text-right font-mono text-gray-900 dark:text-white">
                            {{ formatIntervalPrice(iv.cache_write_price_per_mtok_usd, row, 'MTok') }}
                          </td>
                          <td v-if="hasIntervalCacheCreation5m(row)" class="px-2 py-1.5 text-right font-mono text-gray-900 dark:text-white">
                            {{ formatIntervalPrice(iv.cache_creation_5m_price_per_mtok_usd, row, 'MTok') }}
                          </td>
                          <td v-if="hasIntervalCacheCreation1h(row)" class="px-2 py-1.5 text-right font-mono text-gray-900 dark:text-white">
                            {{ formatIntervalPrice(iv.cache_creation_1h_price_per_mtok_usd, row, 'MTok') }}
                          </td>
                        </tr>
                      </tbody>
                    </table>
                    <table v-else class="min-w-[420px] w-full text-xs">
                      <thead>
                        <tr class="border-b border-gray-100 text-left text-[11px] text-gray-400 dark:border-dark-700">
                          <th class="px-2 py-1">{{ t('modelSquare.detail.tier') }}</th>
                          <th class="px-2 py-1">{{ t('modelSquare.detail.contextRange') }}</th>
                          <th class="px-2 py-1 text-right">{{ t('modelSquare.detail.perRequestPrice') }}</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr
                          v-for="iv in sortedIntervals(row)"
                          :key="`${platform.platform}-${row.group_id}-${iv.tier_label}-${iv.min_tokens}-${iv.max_tokens ?? 'inf'}-${iv.sort_order}`"
                          class="border-b border-gray-100 last:border-0 dark:border-dark-700/60"
                        >
                          <td class="whitespace-nowrap px-2 py-1.5 text-gray-600 dark:text-dark-300">
                            {{ iv.tier_label || '-' }}
                          </td>
                          <td class="whitespace-nowrap px-2 py-1.5 font-mono text-gray-600 dark:text-dark-300">
                            {{ formatIntervalRange(iv) }}
                          </td>
                          <td class="px-2 py-1.5 text-right font-mono text-gray-900 dark:text-white">
                            {{ formatIntervalPrice(iv.per_request_price_usd, row, 'call') }}
                          </td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </div>
      </aside>
    </transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelSquareCard, ModelGroupPrice, ModelPriceInterval } from '@/api/models'
import { platformBadgeClass } from '@/utils/platformColors'
import Icon from '@/components/icons/Icon.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import PricingRow from './PricingRow.vue'

const props = defineProps<{
  open: boolean
  card: ModelSquareCard | null
}>()
const emit = defineEmits<{ (e: 'close'): void }>()

const { t } = useI18n()

function close() {
  emit('close')
}

const CATEGORY_GRADIENTS: Record<string, string> = {
  anthropic: 'bg-gradient-to-br from-orange-400 to-orange-500',
  openai: 'bg-gradient-to-br from-emerald-500 to-emerald-600',
  antigravity: 'bg-gradient-to-br from-purple-500 to-purple-600',
  claude: 'bg-gradient-to-br from-orange-400 to-orange-500',
  gpt: 'bg-gradient-to-br from-emerald-500 to-emerald-600',
  gemini: 'bg-gradient-to-br from-blue-500 to-blue-600',
  image: 'bg-gradient-to-br from-pink-500 to-rose-600',
  embedding: 'bg-gradient-to-br from-teal-500 to-cyan-600',
  audio: 'bg-gradient-to-br from-violet-500 to-purple-600',
  other: 'bg-gradient-to-br from-slate-400 to-slate-500',
}
const categoryGradient = computed(
  () => CATEGORY_GRADIENTS[props.card?.category ?? 'other'] ?? CATEGORY_GRADIENTS.other,
)

const supportFlags = computed(() => {
  const card = props.card
  if (!card) return []
  return card.support_flags?.length ? card.support_flags : (card.capabilities ?? [])
})

function humanizeKey(key: string): string {
  return key
    .split('_')
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

function supportFlagLabel(key: string): string {
  return t(`modelSquare.capabilities.${key}`, humanizeKey(key))
}

function modalityLabel(key: string): string {
  return t(`modelSquare.modalities.${key}`, humanizeKey(key))
}

function modelTypeLabel(key: string): string {
  return t(`modelSquare.modelTypes.${key}`, humanizeKey(key))
}

/** 有效倍率 = base × user（user 缺失时按 1 计）。 */
function effectiveMultiplier(row: ModelGroupPrice): number {
  return row.base_rate_multiplier * (row.user_rate_multiplier ?? 1)
}

function sortedIntervals(row: ModelGroupPrice): ModelPriceInterval[] {
  return [...(row.intervals ?? [])].sort((a, b) => {
    if (a.sort_order !== b.sort_order) return a.sort_order - b.sort_order
    if (a.min_tokens !== b.min_tokens) return a.min_tokens - b.min_tokens
    return (a.max_tokens ?? Number.MAX_SAFE_INTEGER) - (b.max_tokens ?? Number.MAX_SAFE_INTEGER)
  })
}

function hasIntervalCacheCreation5m(row: ModelGroupPrice): boolean {
  return (row.intervals ?? []).some((iv) => iv.cache_creation_5m_price_per_mtok_usd != null)
}

function hasIntervalCacheCreation1h(row: ModelGroupPrice): boolean {
  return (row.intervals ?? []).some((iv) => iv.cache_creation_1h_price_per_mtok_usd != null)
}

function formatIntervalRange(iv: ModelPriceInterval): string {
  const min = formatTokenBoundary(iv.min_tokens)
  const max = iv.max_tokens == null ? '∞' : formatTokenBoundary(iv.max_tokens)
  return `(${min}, ${max}]`
}

function formatIntervalPrice(value: number | null, row: ModelGroupPrice, unit: 'MTok' | 'call'): string {
  if (value == null) return '-'
  return `$${formatMoney(value * effectiveMultiplier(row))} / ${unit}`
}

function formatMoney(v: number): string {
  if (v >= 1) return v.toFixed(2)
  if (v >= 0.01) return v.toFixed(4)
  return v.toFixed(6)
}

function formatTokenBoundary(n: number): string {
  if (n === 0) return '0'
  return formatTokens(n)
}

function formatTokens(n: number): string {
  if (!n || n <= 0) return '-'
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(0)}K`
  return String(n)
}

// Esc 关闭 + 锁滚动
function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.open) close()
}
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
watch(
  () => props.open,
  (v) => {
    document.body.style.overflow = v ? 'hidden' : ''
  },
)
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
.slide-in-right-enter-active,
.slide-in-right-leave-active {
  transition: transform 0.25s ease;
}
.slide-in-right-enter-from,
.slide-in-right-leave-to {
  transform: translateX(100%);
}
</style>
