<!-- sudoapi: Contributor account self-service authorization. -->

<template>
  <ContributorLayout content-class="px-4 py-6 sm:px-6 lg:px-8">
    <div class="mx-auto max-w-3xl">
      <div
        v-if="submittedCurrentPlatform"
        class="mb-5 rounded-lg border border-green-200 bg-green-50 p-4 text-sm text-green-800 dark:border-green-800/40 dark:bg-green-900/20 dark:text-green-200"
      >
        <div class="flex items-center justify-between gap-3">
          <span>{{ successMessage }}</span>
          <button type="button" class="btn btn-secondary" @click="resetFlow">
            {{ t('contributor.accounts.reauthorize') }}
          </button>
        </div>
      </div>

      <div
        class="mb-5 rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900"
      >
        <label class="input-label">{{ t('contributor.accounts.selectPlatform') }}</label>
        <div class="grid grid-cols-2 gap-2">
          <button
            v-for="option in platformOptions"
            :key="option.value"
            type="button"
            class="rounded-md border px-3 py-2 text-sm font-medium transition-colors"
            :class="selectedPlatform === option.value
              ? 'border-blue-500 bg-blue-50 text-blue-700 dark:border-blue-400 dark:bg-blue-900/30 dark:text-blue-200'
              : 'border-gray-200 bg-white text-gray-700 hover:bg-gray-50 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-200 dark:hover:bg-dark-800'"
            @click="selectPlatform(option.value)"
          >
            {{ option.label }}
          </button>
        </div>
      </div>

      <div
        class="mb-5 rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-700 dark:bg-blue-900/30"
      >
        <div class="space-y-4">
          <div>
            <select
              v-model="selectedProxyId"
              class="input w-full"
              :disabled="true"
              @change="handleProxyChange"
            >
              <option v-if="proxyLoading" :value="null">
                {{ t('contributor.accounts.proxyLoading') }}
              </option>
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
        v-if="!submittedCurrentPlatform && authUrl && sessionId"
        ref="oauthFlowRef"
        :add-method="selectedPlatform === 'openai' ? 'oauth' : 'setup-token'"
        :auth-url="authUrl"
        :session-id="sessionId"
        :loading="loading"
        :error="error"
        :show-help="selectedPlatform === 'anthropic'"
        :show-proxy-warning="false"
        :allow-multiple="false"
        :show-cookie-option="false"
        :show-refresh-token-option="selectedPlatform === 'openai'"
        :show-mobile-refresh-token-option="selectedPlatform === 'openai'"
        :show-session-token-option="false"
        :show-access-token-option="false"
        :show-codex-session-import-option="false"
        :force-show-method-selection="selectedPlatform === 'openai'"
        :method-label="t('contributor.accounts.authorizationMethod')"
        :platform="selectedPlatform"
        :show-project-id="false"
        @generate-url="generateAuthUrl"
        @validate-refresh-token="handleOpenAIValidateRefreshToken"
        @validate-mobile-refresh-token="handleOpenAIValidateMobileRT"
        @update:input-method="updateInputMethod"
      />

      <div v-if="!submittedCurrentPlatform && isManualInputMethod && authUrl && sessionId" class="mt-5 flex justify-end">
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
  </ContributorLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import OAuthAuthorizationFlow from '@/components/account/OAuthAuthorizationFlow.vue'
import ContributorLayout from '@/components/layout/ContributorLayout.vue'
import { contributorAPI } from '@/api/contributor'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import type { Proxy } from '@/types'

type OAuthFlowExposed = {
  authCode: string
  reset: () => void
  // sudoapi: Contributor account OpenAI OAuth self-service authorization.
  oauthState?: string
  inputMethod?: AuthInputMethod
}

// sudoapi: Contributor account OpenAI OAuth self-service authorization.
type ContributorPlatform = 'anthropic' | 'openai'
type AuthInputMethod = 'manual' | 'cookie' | 'refresh_token' | 'mobile_refresh_token' | 'session_token' | 'access_token' | 'codex_session'

// sudoapi: Contributor account OpenAI OAuth self-service authorization.
const OPENAI_MOBILE_RT_CLIENT_ID = 'app_LlGpXReQgckcGGUo2JrYvtJK'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()

const oauthFlowRef = ref<OAuthFlowExposed | null>(null)
const authUrl = ref('')
const sessionId = ref('')
const openaiOAuthState = ref('')
const authRequestSeq = ref(0)
const selectedPlatform = ref<ContributorPlatform>('anthropic')
const currentInputMethod = ref<AuthInputMethod>('manual')
const accountName = ref(resolveAccountName())
const loading = ref(false)
const proxyLoading = ref(false)
const error = ref('')
const proxies = ref<Proxy[]>([])
const selectedProxyId = ref<number | null>(null)
const proxyLoadFailed = ref(false)
const submittedPlatforms = ref<Record<ContributorPlatform, boolean>>({
  anthropic: false,
  openai: false
})
const platformOptions: Array<{ value: ContributorPlatform; label: string }> = [
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' }
]
const contributorCountry = computed(() => normalizeCountryParam(route.query.country ?? route.query.country_code))

const hasAvailableProxies = computed(() => proxies.value.length > 0)

const proxyUnavailableMessage = computed(() => {
  if (proxyLoading.value) return ''
  if (proxyLoadFailed.value || !hasAvailableProxies.value) {
    return t('contributor.accounts.proxyUnavailable')
  }
  return ''
})

const platformDisplayName = computed(() => selectedPlatform.value === 'openai' ? 'OpenAI' : 'Claude')
const successMessage = computed(() => t('contributor.accounts.authSubmittedReview', {
  platform: platformDisplayName.value
}))
const submittedCurrentPlatform = computed(() => submittedPlatforms.value[selectedPlatform.value])
const isManualInputMethod = computed(() => currentInputMethod.value === 'manual')

const canSubmit = computed(() => {
  return (
    !loading.value &&
    hasAvailableProxies.value &&
    selectedProxyId.value !== null &&
    !submittedCurrentPlatform.value &&
    isManualInputMethod.value &&
    !!sessionId.value &&
    !!oauthFlowRef.value?.authCode?.trim()
  )
})

onMounted(() => {
  loadProxies()
})

watch(selectedPlatform, () => {
  resetAuthorizationState()
  if (!submittedCurrentPlatform.value && hasAvailableProxies.value && selectedProxyId.value !== null) {
    void generateAuthUrl()
  }
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
    error.value = err.response?.data?.detail || err.message || t('contributor.accounts.loadProxiesFailed')
    appStore.showError(error.value)
  } finally {
    proxyLoading.value = false
  }
}

function resetAuthorizationState(): void {
  authRequestSeq.value++
  authUrl.value = ''
  sessionId.value = ''
  openaiOAuthState.value = ''
  currentInputMethod.value = 'manual'
  oauthFlowRef.value?.reset()
}

function handleProxyChange(): void {
  selectedProxyId.value = selectedProxyId.value === null ? null : Number(selectedProxyId.value)
  resetAuthorizationState()
}

function selectPlatform(platform: ContributorPlatform): void {
  if (selectedPlatform.value === platform) return
  const currentDefaultName = defaultAccountName(selectedPlatform.value)
  selectedPlatform.value = platform
  if (!accountName.value || accountName.value === currentDefaultName) {
    accountName.value = resolveAccountName()
  }
}

function defaultAccountName(platform: ContributorPlatform = selectedPlatform.value): string {
  return platform === 'openai' ? 'OpenAI OAuth' : 'Claude OAuth'
}

function resolveAccountName(): string {
  const username = authStore.user?.username?.trim()
  if (username) return username
  const email = authStore.user?.email?.trim()
  if (email) return email
  return defaultAccountName()
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

function buildOpenAICredentials(tokenInfo: Record<string, unknown>): Record<string, unknown> {
  const credentials: Record<string, unknown> = {
    access_token: tokenInfo.access_token,
    expires_at: tokenInfo.expires_at
  }
  for (const key of [
    'refresh_token',
    'id_token',
    'email',
    'chatgpt_account_id',
    'chatgpt_user_id',
    'organization_id',
    'plan_type',
    'subscription_expires_at',
    'client_id'
  ]) {
    if (typeof tokenInfo[key] === 'string' && tokenInfo[key].trim()) {
      credentials[key] = tokenInfo[key]
    }
  }
  return credentials
}

function buildOpenAIExtra(tokenInfo: Record<string, unknown>): Record<string, unknown> | undefined {
  const extra: Record<string, unknown> = {}
  for (const key of ['email', 'name', 'privacy_mode']) {
    if (typeof tokenInfo[key] === 'string' && tokenInfo[key].trim()) {
      extra[key] = tokenInfo[key]
    }
  }
  return Object.keys(extra).length > 0 ? extra : undefined
}

function parseOAuthState(url: string): string {
  try {
    return new URL(url).searchParams.get('state') || ''
  } catch {
    return ''
  }
}

function parseRefreshTokens(input: string): string[] {
  return input
    .split('\n')
    .map((rt) => rt.trim())
    .filter((rt) => rt)
}

function markCurrentPlatformSubmitted(): void {
  submittedPlatforms.value = {
    ...submittedPlatforms.value,
    [selectedPlatform.value]: true
  }
}

function updateInputMethod(method: AuthInputMethod): void {
  currentInputMethod.value = method
}

async function generateAuthUrl(): Promise<void> {
  const platform = selectedPlatform.value
  const proxyId = selectedProxyId.value
  const requestSeq = ++authRequestSeq.value
  if (proxyId === null || !hasAvailableProxies.value) {
    error.value = t('contributor.accounts.proxyUnavailable')
    return
  }

  loading.value = true
  error.value = ''
  authUrl.value = ''
  sessionId.value = ''
  try {
    const result = platform === 'openai'
      ? await contributorAPI.accounts.generateOpenAIAuthUrl({
        proxy_id: proxyId
      })
      : await contributorAPI.accounts.generateClaudeSetupTokenUrl({
        proxy_id: proxyId
      })
    if (requestSeq !== authRequestSeq.value || selectedPlatform.value !== platform || selectedProxyId.value !== proxyId) {
      return
    }
    authUrl.value = result.auth_url
    sessionId.value = result.session_id
    openaiOAuthState.value = platform === 'openai' ? parseOAuthState(result.auth_url) : ''
  } catch (err: any) {
    if (requestSeq !== authRequestSeq.value) {
      return
    }
    error.value = err.response?.data?.detail || err.message || t('contributor.accounts.generateAuthUrlFailed')
    appStore.showError(error.value)
  } finally {
    if (requestSeq === authRequestSeq.value) {
      loading.value = false
    }
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
    if (selectedPlatform.value === 'openai') {
      const state = (oauthFlowRef.value?.oauthState || openaiOAuthState.value || '').trim()
      if (!state) {
        error.value = t('contributor.accounts.missingOAuthState')
        appStore.showError(error.value)
        return
      }
      const tokenInfo = await contributorAPI.accounts.exchangeOpenAICode({
        session_id: sessionId.value,
        code,
        state,
        proxy_id: selectedProxyId.value
      })
      await contributorAPI.accounts.create({
        name: accountName.value || resolveAccountName(),
        notes: null,
        platform: 'openai',
        type: 'oauth',
        add_method: 'oauth',
        credentials: buildOpenAICredentials(tokenInfo),
        extra: buildOpenAIExtra(tokenInfo),
        proxy_id: selectedProxyId.value,
        concurrency: 10,
        priority: 1,
        rate_multiplier: 1,
        group_ids: [],
        auto_pause_on_expired: true,
        country: contributorCountry.value || undefined
      })

      markCurrentPlatformSubmitted()
      appStore.showSuccess(t('contributor.accounts.authSubmitted', { platform: 'OpenAI' }))
      return
    }

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

    markCurrentPlatformSubmitted()
    appStore.showSuccess(t('contributor.accounts.authSubmitted', { platform: 'Claude' }))
  } catch (err: any) {
    error.value = err.response?.data?.detail || err.message || t('admin.accounts.oauth.authFailed')
    appStore.showError(error.value)
  } finally {
    loading.value = false
  }
}

async function handleOpenAIRefreshTokenInput(refreshTokenInput: string, clientId?: string): Promise<void> {
  if (selectedPlatform.value !== 'openai') return
  const refreshTokens = parseRefreshTokens(refreshTokenInput)
  if (refreshTokens.length === 0) {
    error.value = t('contributor.accounts.openaiRefreshTokenRequired')
    appStore.showError(error.value)
    return
  }

  loading.value = true
  error.value = ''
  let successCount = 0
  let failedCount = 0
  const errors: string[] = []

  try {
    for (let i = 0; i < refreshTokens.length; i++) {
      try {
        const tokenInfo = await contributorAPI.accounts.refreshOpenAIToken({
          refresh_token: refreshTokens[i],
          proxy_id: selectedProxyId.value,
          client_id: clientId
        })
        const credentials = buildOpenAICredentials(tokenInfo)
        if (clientId) {
          credentials.client_id = clientId
        }
        const baseName = accountName.value || (typeof tokenInfo.email === 'string' && tokenInfo.email.trim()) || resolveAccountName()
        const name = refreshTokens.length > 1 ? `${baseName} #${i + 1}` : baseName
        await contributorAPI.accounts.create({
          name,
          notes: null,
          platform: 'openai',
          type: 'oauth',
          add_method: 'oauth',
          credentials,
          extra: buildOpenAIExtra(tokenInfo),
          proxy_id: selectedProxyId.value,
          concurrency: 10,
          priority: 1,
          rate_multiplier: 1,
          group_ids: [],
          auto_pause_on_expired: true,
          country: contributorCountry.value || undefined
        })
        successCount++
      } catch (err: any) {
        failedCount++
        errors.push(`#${i + 1}: ${err.response?.data?.detail || err.message || t('contributor.accounts.unknownError')}`)
      }
    }

    if (successCount > 0) {
      if (failedCount > 0) {
        error.value = errors.join('\n')
        appStore.showWarning(t('contributor.accounts.openaiPartialSubmitted', {
          success: successCount,
          failed: failedCount
        }))
      } else {
        markCurrentPlatformSubmitted()
        appStore.showSuccess(refreshTokens.length > 1
          ? t('contributor.accounts.openaiSubmittedCount', { count: successCount })
          : t('contributor.accounts.authSubmitted', { platform: 'OpenAI' }))
      }
      return
    }

    error.value = errors.join('\n') || t('contributor.accounts.openaiRefreshTokenFailed')
    appStore.showError(error.value)
  } finally {
    loading.value = false
  }
}

function handleOpenAIValidateRefreshToken(refreshTokenInput: string): void {
  void handleOpenAIRefreshTokenInput(refreshTokenInput)
}

function handleOpenAIValidateMobileRT(refreshTokenInput: string): void {
  void handleOpenAIRefreshTokenInput(refreshTokenInput, OPENAI_MOBILE_RT_CLIENT_ID)
}

async function resetFlow(): Promise<void> {
  submittedPlatforms.value = {
    ...submittedPlatforms.value,
    [selectedPlatform.value]: false
  }
  accountName.value = resolveAccountName()
  resetAuthorizationState()
  loading.value = true
  try {
    await contributorAPI.accounts.releaseProxyReservation()
    await loadProxies()
  } catch (err: any) {
    error.value = err.response?.data?.detail || err.message || t('contributor.accounts.resetProxyReservationFailed')
    appStore.showError(error.value)
  } finally {
    loading.value = false
  }
}
</script>
