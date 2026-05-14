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
            </dl>
            <div v-if="card.capabilities && card.capabilities.length" class="mt-3 flex flex-wrap gap-1.5">
              <span
                v-for="cap in card.capabilities"
                :key="cap"
                class="rounded-full bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
              >
                {{ t(`modelSquare.capabilities.${cap}`, cap) }}
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
import type { ModelSquareCard, ModelGroupPrice } from '@/api/models'
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

/** 有效倍率 = base × user（user 缺失时按 1 计）。 */
function effectiveMultiplier(row: ModelGroupPrice): number {
  return row.base_rate_multiplier * (row.user_rate_multiplier ?? 1)
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
