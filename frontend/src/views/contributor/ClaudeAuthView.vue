<template>
  <div class="min-h-screen bg-gray-50 px-4 py-6 dark:bg-dark-950 sm:px-6 lg:px-8">
    <div class="mx-auto max-w-3xl">
      <div
        v-if="created"
        class="mb-5 rounded-lg border border-green-200 bg-green-50 p-4 text-sm text-green-800 dark:border-green-800/40 dark:bg-green-900/20 dark:text-green-200"
      >
        <div class="flex items-center justify-between gap-3">
          <span>Claude 账号授权已提交，等待管理员审核。</span>
          <button type="button" class="btn btn-secondary" @click="resetFlow">重新授权</button>
        </div>
      </div>

      <div
        class="mb-5 rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-700 dark:bg-blue-900/30"
      >
        <div class="space-y-4">
          <div>
            <label class="input-label">选择代理</label>
            <select
              v-model="selectedProxyId"
              class="input w-full"
              :disabled="true"
              @change="handleProxyChange"
            >
              <option v-if="proxyLoading" :value="null">正在加载代理...</option>
              <option
                v-for="proxy in proxies"
                :key="proxy.id"
                :value="proxy.id"
              >
                {{ formatProxyLabel(proxy) }}
              </option>
            </select>
          </div>

          <div
            v-if="proxyUnavailableMessage"
            class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-200"
          >
            {{ proxyUnavailableMessage }}
          </div>

        </div>
      </div>

      <OAuthAuthorizationFlow
        v-if="authUrl && sessionId"
        ref="oauthFlowRef"
        add-method="setup-token"
        :auth-url="authUrl"
        :session-id="sessionId"
        :loading="loading"
        :error="error"
        :show-help="true"
        :show-proxy-warning="false"
        :allow-multiple="false"
        :show-cookie-option="false"
        :show-refresh-token-option="false"
        :show-mobile-refresh-token-option="false"
        :show-session-token-option="false"
        :show-access-token-option="false"
        :show-codex-session-import-option="false"
        platform="anthropic"
        :show-project-id="false"
        @generate-url="generateAuthUrl"
      />

      <div v-if="authUrl && sessionId" class="mt-5 flex justify-end">
        <button
          type="button"
          class="btn btn-primary"
          data-testid="submit-auth-code"
          :disabled="!canSubmit"
          @click="submitAuthCode"
        >
          <svg
            v-if="loading"
            class="-ml-1 mr-2 h-4 w-4 animate-spin"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            />
          </svg>
          {{ loading ? t('common.loading') : t('common.submit') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import OAuthAuthorizationFlow from '@/components/account/OAuthAuthorizationFlow.vue'
import { contributorAPI } from '@/api/contributor'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import type { Proxy } from '@/types'

type OAuthFlowExposed = {
  authCode: string
  reset: () => void
}

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()

const oauthFlowRef = ref<OAuthFlowExposed | null>(null)
const authUrl = ref('')
const sessionId = ref('')
const accountName = ref(resolveAccountName())
const loading = ref(false)
const proxyLoading = ref(false)
const error = ref('')
const created = ref(false)
const proxies = ref<Proxy[]>([])
const selectedProxyId = ref<number | null>(null)
const proxyLoadFailed = ref(false)
const contributorCountry = computed(() => normalizeCountryParam(route.query.country ?? route.query.country_code))

const hasAvailableProxies = computed(() => proxies.value.length > 0)

const proxyUnavailableMessage = computed(() => {
  if (proxyLoading.value) return ''
  if (proxyLoadFailed.value || !hasAvailableProxies.value) {
    return '暂时没有可选代理，无法继续授权。'
  }
  return ''
})

const canSubmit = computed(() => {
  return (
    !loading.value &&
    hasAvailableProxies.value &&
    selectedProxyId.value !== null &&
    !!sessionId.value &&
    !!oauthFlowRef.value?.authCode?.trim()
  )
})

onMounted(() => {
  loadProxies()
})

function formatProxyLabel(proxy: Proxy): string {
  return `${proxy.name} (${proxy.protocol}://${proxy.host}:${proxy.port})`
}

function normalizeCountryParam(value: unknown): string {
  const raw = Array.isArray(value) ? value[0] : value
  return typeof raw === 'string' ? raw.trim().toUpperCase() : ''
}

async function loadProxies(): Promise<void> {
  proxyLoading.value = true
  proxyLoadFailed.value = false
  error.value = ''
  try {
    const result = await contributorAPI.accounts.getProxiesForCountry(contributorCountry.value)
    proxies.value = result
    selectedProxyId.value = result[0]?.id ?? null
    if (result.length > 0) {
      await generateAuthUrl()
    }
  } catch (err: any) {
    proxies.value = []
    selectedProxyId.value = null
    proxyLoadFailed.value = true
    error.value = err.response?.data?.detail || err.message || 'Failed to load proxies'
    appStore.showError(error.value)
  } finally {
    proxyLoading.value = false
  }
}

function resetAuthorizationState(): void {
  authUrl.value = ''
  sessionId.value = ''
  oauthFlowRef.value?.reset()
}

function handleProxyChange(): void {
  selectedProxyId.value = selectedProxyId.value === null ? null : Number(selectedProxyId.value)
  resetAuthorizationState()
}

function resolveAccountName(): string {
  const username = authStore.user?.username?.trim()
  if (username) return username
  const email = authStore.user?.email?.trim()
  if (email) return email
  return 'Claude OAuth'
}

function buildExtra(tokenInfo: Record<string, unknown>): Record<string, unknown> | undefined {
  const extra: Record<string, unknown> = {}
  for (const key of ['org_uuid', 'account_uuid', 'email_address']) {
    if (typeof tokenInfo[key] === 'string' && tokenInfo[key].trim()) {
      extra[key] = tokenInfo[key]
    }
  }
  return Object.keys(extra).length > 0 ? extra : undefined
}

async function generateAuthUrl(): Promise<void> {
  if (selectedProxyId.value === null || !hasAvailableProxies.value) {
    error.value = '暂时没有可选代理，无法继续授权。'
    return
  }

  loading.value = true
  error.value = ''
  authUrl.value = ''
  sessionId.value = ''
  try {
    const result = await contributorAPI.accounts.generateClaudeSetupTokenUrl({
      proxy_id: selectedProxyId.value
    })
    authUrl.value = result.auth_url
    sessionId.value = result.session_id
  } catch (err: any) {
    error.value = err.response?.data?.detail || err.message || 'Failed to generate auth URL'
    appStore.showError(error.value)
  } finally {
    loading.value = false
  }
}

async function submitAuthCode(): Promise<void> {
  const code = oauthFlowRef.value?.authCode?.trim() || ''
  if (!code || !sessionId.value) {
    return
  }

  loading.value = true
  error.value = ''
  try {
    const tokenInfo = await contributorAPI.accounts.exchangeClaudeSetupTokenCode({
      session_id: sessionId.value,
      code,
      proxy_id: selectedProxyId.value
    })
    await contributorAPI.accounts.create({
      name: accountName.value || resolveAccountName(),
      notes: null,
      platform: 'anthropic',
      type: 'setup-token',
      add_method: 'setup-token',
      credentials: tokenInfo,
      extra: buildExtra(tokenInfo),
      proxy_id: selectedProxyId.value,
      concurrency: 10,
      priority: 1,
      rate_multiplier: 1,
      group_ids: [],
      auto_pause_on_expired: true,
      country: contributorCountry.value || undefined
    })

    created.value = true
    appStore.showSuccess('Claude 账号授权已提交')
  } catch (err: any) {
    error.value = err.response?.data?.detail || err.message || t('admin.accounts.oauth.authFailed')
    appStore.showError(error.value)
  } finally {
    loading.value = false
  }
}

async function resetFlow(): Promise<void> {
  created.value = false
  accountName.value = resolveAccountName()
  resetAuthorizationState()
  loading.value = true
  try {
    await contributorAPI.accounts.releaseProxyReservation()
    await loadProxies()
  } catch (err: any) {
    error.value = err.response?.data?.detail || err.message || 'Failed to reset proxy reservation'
    appStore.showError(error.value)
  } finally {
    loading.value = false
  }
}
</script>
