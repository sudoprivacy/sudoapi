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

      <OAuthAuthorizationFlow
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

      <div class="mt-5 flex justify-end">
        <button type="button" class="btn btn-primary" :disabled="!canSubmit" @click="submitAuthCode">
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
import OAuthAuthorizationFlow from '@/components/account/OAuthAuthorizationFlow.vue'
import { contributorAPI } from '@/api/contributor'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'

type OAuthFlowExposed = {
  authCode: string
  reset: () => void
}

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const oauthFlowRef = ref<OAuthFlowExposed | null>(null)
const authUrl = ref('')
const sessionId = ref('')
const accountName = ref(resolveAccountName())
const loading = ref(false)
const error = ref('')
const created = ref(false)

const canSubmit = computed(() => {
  return !loading.value && !!sessionId.value && !!oauthFlowRef.value?.authCode?.trim()
})

onMounted(() => {
  generateAuthUrl()
})

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
  loading.value = true
  error.value = ''
  authUrl.value = ''
  sessionId.value = ''
  try {
    const result = await contributorAPI.accounts.generateClaudeSetupTokenUrl({ proxy_id: null })
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
      proxy_id: null
    })
    await contributorAPI.accounts.create({
      name: accountName.value || resolveAccountName(),
      notes: null,
      platform: 'anthropic',
      type: 'setup-token',
      add_method: 'setup-token',
      credentials: tokenInfo,
      extra: buildExtra(tokenInfo),
      proxy_id: null,
      concurrency: 10,
      priority: 1,
      rate_multiplier: 1,
      group_ids: [],
      auto_pause_on_expired: true
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

function resetFlow(): void {
  created.value = false
  accountName.value = resolveAccountName()
  oauthFlowRef.value?.reset()
  generateAuthUrl()
}
</script>
