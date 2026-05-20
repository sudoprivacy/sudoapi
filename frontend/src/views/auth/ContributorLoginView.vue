<template>
  <AuthLayout>
    <div class="space-y-6">
      <div class="text-center">
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ isRegisterMode ? t('auth.createAccount') : t('auth.welcomeBack') }}
        </h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ isRegisterMode ? '注册 Claude 账号贡献者' : 'Claude 账号贡献者登录' }}
        </p>
      </div>

      <form class="space-y-5" @submit.prevent="handleLogin">
        <div>
          <label for="email" class="input-label">{{ t('auth.emailLabel') }}</label>
          <div class="relative">
            <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5">
              <Icon name="mail" size="md" class="text-gray-400 dark:text-dark-500" />
            </div>
            <input
              id="email"
              v-model="formData.email"
              type="email"
              required
              autofocus
              autocomplete="email"
              :disabled="authActionDisabled"
              class="input pl-11"
              :class="{ 'input-error': errors.email }"
              :placeholder="t('auth.emailPlaceholder')"
            />
          </div>
        </div>

        <div>
          <label for="password" class="input-label">{{ t('auth.passwordLabel') }}</label>
          <div class="relative">
            <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5">
              <Icon name="lock" size="md" class="text-gray-400 dark:text-dark-500" />
            </div>
            <input
              id="password"
              v-model="formData.password"
              :type="showPassword ? 'text' : 'password'"
              required
              autocomplete="current-password"
              :disabled="authActionDisabled"
              class="input pl-11 pr-11"
              :class="{ 'input-error': errors.password }"
              :placeholder="t('auth.passwordPlaceholder')"
            />
            <button
              type="button"
              :disabled="authActionDisabled"
              class="absolute inset-y-0 right-0 flex items-center pr-3.5 text-gray-400 transition-colors hover:text-gray-600 dark:hover:text-dark-300"
              @click="showPassword = !showPassword"
            >
              <Icon v-if="showPassword" name="eyeOff" size="md" />
              <Icon v-else name="eye" size="md" />
            </button>
          </div>
        </div>

        <div v-if="turnstileEnabled && turnstileSiteKey">
          <TurnstileWidget
            ref="turnstileRef"
            :site-key="turnstileSiteKey"
            @verify="onTurnstileVerify"
            @expire="onTurnstileExpire"
            @error="onTurnstileError"
          />
        </div>

        <button
          type="submit"
          :disabled="authActionDisabled || (turnstileEnabled && !turnstileToken)"
          class="btn btn-primary w-full"
        >
          <svg
            v-if="isLoading"
            class="-ml-1 mr-2 h-4 w-4 animate-spin text-white"
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
          <Icon v-else name="login" size="md" class="mr-2" />
          {{ isLoading ? t('auth.signingIn') : isRegisterMode ? t('auth.createAccount') : t('auth.signIn') }}
        </button>

        <div v-if="showOAuthLogin" class="space-y-3 pt-1">
          <div class="flex items-center gap-3">
            <div class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></div>
            <span class="text-xs text-gray-500 dark:text-dark-400">
              {{ t('auth.oauthOrContinue') }}
            </span>
            <div class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></div>
          </div>

          <EmailOAuthButtons
            :disabled="authActionDisabled"
            :github-enabled="false"
            :google-enabled="googleOAuthEnabled"
            :redirect-to="contributorRedirect"
            contributor
            :show-divider="false"
          />
        </div>
      </form>
    </div>

    <template v-if="registrationEnabled" #footer>
      <p class="text-gray-500 dark:text-dark-400">
        {{ isRegisterMode ? t('auth.alreadyHaveAccount') : t('auth.dontHaveAccount') }}
        <button
          type="button"
          class="font-medium text-primary-600 transition-colors hover:text-primary-500 dark:text-primary-400 dark:hover:text-primary-300"
          @click="toggleAuthMode"
        >
          {{ isRegisterMode ? t('auth.signIn') : t('auth.signUp') }}
        </button>
      </p>
    </template>
  </AuthLayout>

  <TotpLoginModal
    v-if="show2FAModal"
    ref="totpModalRef"
    :temp-token="totpTempToken"
    :user-email-masked="totpUserEmailMasked"
    @verify="handle2FAVerify"
    @cancel="handle2FACancel"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { AuthLayout } from '@/components/layout'
import Icon from '@/components/icons/Icon.vue'
import TurnstileWidget from '@/components/TurnstileWidget.vue'
import TotpLoginModal from '@/components/auth/TotpLoginModal.vue'
import EmailOAuthButtons from '@/components/auth/EmailOAuthButtons.vue'
import { useAppStore, useAuthStore } from '@/stores'
import { getPublicSettings, isTotp2FARequired } from '@/api/auth'
import { extractI18nErrorMessage } from '@/utils/apiError'
import type { TotpLoginResponse } from '@/types'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()

const isLoading = ref(false)
const publicSettingsLoaded = ref(false)
const showPassword = ref(false)
const authMode = ref<'login' | 'register'>('login')
const turnstileEnabled = ref(false)
const turnstileSiteKey = ref('')
const turnstileToken = ref('')
const turnstileRef = ref<InstanceType<typeof TurnstileWidget> | null>(null)
const backendModeEnabled = ref(false)
const registrationEnabled = ref(false)
const googleOAuthEnabled = ref(false)
const contributorRedirect = '/contributor/claude-auth'

const show2FAModal = ref(false)
const totpTempToken = ref('')
const totpUserEmailMasked = ref('')
const totpModalRef = ref<InstanceType<typeof TotpLoginModal> | null>(null)

const formData = reactive({
  email: '',
  password: ''
})

const errors = reactive({
  email: '',
  password: '',
  turnstile: ''
})

const authActionDisabled = computed(() => isLoading.value || !publicSettingsLoaded.value)
const showOAuthLogin = computed(() => !backendModeEnabled.value && googleOAuthEnabled.value)
const isRegisterMode = computed(() => authMode.value === 'register')

watch(
  () => errors.email || errors.password || errors.turnstile,
  (message, previous) => {
    if (message && message !== previous) {
      appStore.showError(message)
    }
  }
)

onMounted(async () => {
  try {
    const settings = await getPublicSettings()
    turnstileEnabled.value = settings.turnstile_enabled
    turnstileSiteKey.value = settings.turnstile_site_key || ''
    backendModeEnabled.value = settings.backend_mode_enabled
    registrationEnabled.value = settings.registration_enabled
    googleOAuthEnabled.value = settings.google_oauth_enabled
  } catch (error) {
    console.error('Failed to load public settings:', error)
  } finally {
    publicSettingsLoaded.value = true
  }
})

function onTurnstileVerify(token: string): void {
  turnstileToken.value = token
  errors.turnstile = ''
}

function toggleAuthMode(): void {
  authMode.value = isRegisterMode.value ? 'login' : 'register'
  errors.email = ''
  errors.password = ''
  errors.turnstile = ''
}

function onTurnstileExpire(): void {
  turnstileToken.value = ''
  errors.turnstile = t('auth.turnstileExpired')
}

function onTurnstileError(): void {
  turnstileToken.value = ''
  errors.turnstile = t('auth.turnstileFailed')
}

function validateForm(): boolean {
  errors.email = ''
  errors.password = ''
  errors.turnstile = ''

  if (!formData.email.trim()) {
    errors.email = t('auth.emailRequired')
    return false
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
    errors.email = t('auth.invalidEmail')
    return false
  }
  if (!formData.password) {
    errors.password = t('auth.passwordRequired')
    return false
  }
  if (formData.password.length < 6) {
    errors.password = t('auth.passwordMinLength')
    return false
  }
  if (turnstileEnabled.value && !turnstileToken.value) {
    errors.turnstile = t('auth.completeVerification')
    return false
  }
  return true
}

async function handleLogin(): Promise<void> {
  if (!validateForm()) return

  isLoading.value = true
  try {
    const response = await authStore.contributorLogin({
      email: formData.email,
      password: formData.password,
      turnstile_token: turnstileEnabled.value ? turnstileToken.value : undefined
    })

    if (isTotp2FARequired(response)) {
      const totpResponse = response as TotpLoginResponse
      totpTempToken.value = totpResponse.temp_token || ''
      totpUserEmailMasked.value = totpResponse.user_email_masked || ''
      show2FAModal.value = true
      return
    }

    appStore.showSuccess(t('auth.loginSuccess'))
    await router.push('/contributor/claude-auth')
  } catch (error: unknown) {
    turnstileRef.value?.reset()
    turnstileToken.value = ''
    appStore.showError(extractI18nErrorMessage(error, t, 'auth.errors', t('auth.loginFailed')))
  } finally {
    isLoading.value = false
  }
}

async function handle2FAVerify(code: string): Promise<void> {
  totpModalRef.value?.setVerifying(true)
  try {
    await authStore.login2FA(totpTempToken.value, code)
    show2FAModal.value = false
    appStore.showSuccess(t('auth.loginSuccess'))
    await router.push('/contributor/claude-auth')
  } catch (error: unknown) {
    const err = error as { message?: string; response?: { data?: { message?: string } } }
    totpModalRef.value?.setError(err.response?.data?.message || err.message || t('profile.totp.loginFailed'))
    totpModalRef.value?.setVerifying(false)
  }
}

function handle2FACancel(): void {
  show2FAModal.value = false
  totpTempToken.value = ''
  totpUserEmailMasked.value = ''
}
</script>
