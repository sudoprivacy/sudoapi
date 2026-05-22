// sudoapi: Contributor account self-service authorization.

import { describe, expect, it, vi, beforeEach } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import ClaudeAuthView from '../ClaudeAuthView.vue'

const generateClaudeAuthUrl = vi.fn()
const generateClaudeSetupTokenUrl = vi.fn()
const exchangeClaudeCode = vi.fn()
const exchangeClaudeSetupTokenCode = vi.fn()
const generateOpenAIAuthUrl = vi.fn()
const exchangeOpenAICode = vi.fn()
const refreshOpenAIToken = vi.fn()
const create = vi.fn()
const getProxies = vi.fn()
const getProxiesForCountry = vi.fn()
const releaseProxyReservation = vi.fn()
const showSuccess = vi.fn()
const showError = vi.fn()
const showWarning = vi.fn()

const routeState = vi.hoisted(() => ({
  query: {} as Record<string, unknown>
}))

vi.mock('@/api/contributor', () => ({
  contributorAPI: {
    accounts: {
      generateClaudeAuthUrl: (...args: any[]) => generateClaudeAuthUrl(...args),
      generateClaudeSetupTokenUrl: (...args: any[]) => generateClaudeSetupTokenUrl(...args),
      exchangeClaudeCode: (...args: any[]) => exchangeClaudeCode(...args),
      exchangeClaudeSetupTokenCode: (...args: any[]) => exchangeClaudeSetupTokenCode(...args),
      generateOpenAIAuthUrl: (...args: any[]) => generateOpenAIAuthUrl(...args),
      exchangeOpenAICode: (...args: any[]) => exchangeOpenAICode(...args),
      refreshOpenAIToken: (...args: any[]) => refreshOpenAIToken(...args),
      create: (...args: any[]) => create(...args),
      getProxies: (...args: any[]) => getProxies(...args),
      getProxiesForCountry: (...args: any[]) => getProxiesForCountry(...args),
      releaseProxyReservation: (...args: any[]) => releaseProxyReservation(...args)
    }
  }
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess,
    showError,
    showWarning
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: null
  })
}))

vi.mock('vue-i18n', () => {
  const messages: Record<string, string> = {
    'contributor.accounts.reauthorize': '重新授权',
    'contributor.accounts.proxyLoading': '正在加载代理...',
    'contributor.accounts.proxyUnavailable': '暂时没有可选代理，无法继续授权。',
    'contributor.accounts.authSubmitted': '{platform} 账号授权已提交',
    'contributor.accounts.authSubmittedReview': '{platform} 账号授权已提交，等待管理员审核。',
    'contributor.accounts.openaiRefreshTokenRequired': '请输入 Refresh Token',
    'contributor.accounts.openaiPartialSubmitted': 'OpenAI 账号部分授权已提交，成功 {success} 个，失败 {failed} 个',
    'contributor.accounts.openaiSubmittedCount': 'OpenAI 账号授权已提交 {count} 个',
    'contributor.accounts.openaiRefreshTokenFailed': 'OpenAI Refresh Token 验证失败',
    'contributor.accounts.unknownError': '未知错误'
  }
  const translate = (key: string, params?: Record<string, unknown>) => {
    let text = messages[key] || key
    Object.entries(params || {}).forEach(([param, value]) => {
      text = text.replaceAll(`{${param}}`, String(value))
    })
    return text
  }
  return {
    createI18n: () => ({
      global: {
        t: translate,
        locale: { value: 'en' },
        setLocaleMessage: vi.fn()
      }
    }),
    useI18n: () => ({
      t: translate
    })
  }
})

vi.mock('@/components/account/OAuthAuthorizationFlow.vue', () => ({
  default: defineComponent({
    name: 'OAuthAuthorizationFlow',
    props: {
      addMethod: {
        type: String,
        required: true
      },
      showRefreshTokenOption: {
        type: Boolean,
        default: false
      },
      showMobileRefreshTokenOption: {
        type: Boolean,
        default: false
      },
      forceShowMethodSelection: {
        type: Boolean,
        default: false
      }
    },
    emits: ['generate-url', 'validate-refresh-token', 'validate-mobile-refresh-token', 'update:inputMethod'],
    setup(props, { expose, emit }) {
      const authCode = ref('claude-auth-code')
      const oauthState = ref('state-123')
      expose({
        authCode,
        oauthState,
        reset: vi.fn(() => {
          authCode.value = ''
          oauthState.value = ''
        })
      })
      return () =>
        h('div', [
          h('input', {
            'data-testid': 'auth-code-input',
            value: authCode.value,
            onInput: (event: Event) => {
              authCode.value = (event.target as HTMLInputElement).value
            }
          }),
          h(
            'button',
            {
              class: 'generate-url',
              'data-add-method': props.addMethod,
              'data-show-refresh-token': String(props.showRefreshTokenOption),
              'data-show-mobile-refresh-token': String(props.showMobileRefreshTokenOption),
              'data-force-method-selection': String(props.forceShowMethodSelection),
              onClick: () => emit('generate-url')
            },
            'generate'
          ),
          h(
            'button',
            {
              'data-testid': 'select-refresh-token-method',
              onClick: () => emit('update:inputMethod', 'refresh_token')
            },
            'refresh token method'
          ),
          h(
            'button',
            {
              'data-testid': 'validate-refresh-token',
              onClick: () => emit('validate-refresh-token', 'openai-refresh-token-input')
            },
            'validate refresh token'
          ),
          h(
            'button',
            {
              'data-testid': 'validate-mobile-refresh-token',
              onClick: () => emit('validate-mobile-refresh-token', 'openai-mobile-refresh-token-input')
            },
            'validate mobile refresh token'
          )
        ])
    }
  })
}))

const proxyA = {
  id: 10,
  name: 'proxy-a',
  protocol: 'http',
  host: 'proxy-a.example.com',
  port: 8080,
  username: null,
  status: 'active',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z'
}

const proxyB = {
  ...proxyA,
  id: 11,
  name: 'proxy-b',
  host: 'proxy-b.example.com'
}

describe('ClaudeAuthView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    routeState.query = {}
    getProxies.mockResolvedValue([proxyA])
    getProxiesForCountry.mockResolvedValue([proxyA])
    releaseProxyReservation.mockResolvedValue(undefined)
    generateClaudeSetupTokenUrl.mockResolvedValue({
      auth_url: 'https://claude.example/oauth',
      session_id: 'session-123'
    })
    generateOpenAIAuthUrl.mockResolvedValue({
      auth_url: 'https://openai.example/oauth?state=state-123',
      session_id: 'openai-session-123'
    })
    exchangeClaudeSetupTokenCode.mockResolvedValue({
      access_token: 'access-token',
      refresh_token: 'refresh-token',
      org_uuid: 'org-123',
      account_uuid: 'account-123',
      email_address: 'claude@example.com'
    })
    exchangeOpenAICode.mockResolvedValue({
      access_token: 'openai-access-token',
      refresh_token: 'openai-refresh-token',
      expires_at: 1767225600,
      email: 'openai@example.com',
      chatgpt_account_id: 'chatgpt-account-123'
    })
    refreshOpenAIToken.mockResolvedValue({
      access_token: 'openai-access-token',
      refresh_token: 'openai-refresh-token',
      expires_at: 1767225600,
      email: 'openai@example.com'
    })
    create.mockResolvedValue({ id: 1 })
  })

  it('loads one proxy and auto-generates setup token auth URL', async () => {
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    expect(wrapper.text()).toContain('Anthropic')
    expect(wrapper.text()).toContain('OpenAI')
    expect(getProxiesForCountry).toHaveBeenCalledWith('')
    expect(generateClaudeSetupTokenUrl).toHaveBeenCalledWith({ proxy_id: 10 })
    expect(generateClaudeAuthUrl).not.toHaveBeenCalled()
    expect(generateOpenAIAuthUrl).not.toHaveBeenCalled()
  })

  it('hides the proxy card generate button while auto-generating for available proxies', async () => {
    getProxies.mockResolvedValue([proxyA, proxyB])
    getProxiesForCountry.mockResolvedValue([proxyA, proxyB])
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    expect(generateClaudeSetupTokenUrl).toHaveBeenCalledWith({ proxy_id: 10 })
    expect(wrapper.find('[data-testid="generate-auth-url"]').exists()).toBe(false)
  })

  it('disables authorization flow when no proxies are available', async () => {
    getProxies.mockResolvedValue([])
    getProxiesForCountry.mockResolvedValue([])
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    expect(wrapper.text()).toContain('暂时没有可选代理，无法继续授权。')
    expect(wrapper.find('[data-testid="generate-auth-url"]').exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'OAuthAuthorizationFlow' }).exists()).toBe(false)
    expect(generateClaudeSetupTokenUrl).not.toHaveBeenCalled()
  })

  it('defaults Claude authorization add method to setup token', async () => {
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    expect(wrapper.find('.generate-url').attributes('data-add-method')).toBe('setup-token')
  })

  it('exchanges auth code and creates contributor Anthropic setup token account', async () => {
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    await wrapper.find('[data-testid="submit-auth-code"]').trigger('click')
    await flushPromises()

    expect(exchangeClaudeSetupTokenCode).toHaveBeenCalledWith({
      session_id: 'session-123',
      code: 'claude-auth-code',
      proxy_id: 10
    })
    expect(exchangeClaudeCode).not.toHaveBeenCalled()
    expect(create).toHaveBeenCalledWith(expect.objectContaining({
      platform: 'anthropic',
      type: 'setup-token',
      add_method: 'setup-token',
      credentials: expect.objectContaining({
        access_token: 'access-token',
        refresh_token: 'refresh-token'
      }),
      extra: {
        org_uuid: 'org-123',
        account_uuid: 'account-123',
        email_address: 'claude@example.com'
      },
      proxy_id: 10,
      concurrency: 10,
      priority: 1,
      rate_multiplier: 1,
      group_ids: [],
      auto_pause_on_expired: true
    }))
    expect(create.mock.calls[0][0].name).toBe('Claude OAuth')
    expect(showSuccess).toHaveBeenCalledWith('Claude 账号授权已提交')
  })

  it('hides the submit button for a submitted platform and shows it after switching to an unsubmitted platform', async () => {
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    await wrapper.find('[data-testid="submit-auth-code"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Claude 账号授权已提交，等待管理员审核。')
    expect(wrapper.find('[data-testid="submit-auth-code"]').exists()).toBe(false)

    const openAIButton = wrapper.findAll('button').find((button) => button.text() === 'OpenAI')
    expect(openAIButton).toBeTruthy()
    await openAIButton!.trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="submit-auth-code"]').exists()).toBe(true)
    expect(generateOpenAIAuthUrl).toHaveBeenCalledWith({ proxy_id: 10 })
    expect(releaseProxyReservation).not.toHaveBeenCalled()
  })

  it('switches to OpenAI without releasing or reserving another proxy', async () => {
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    getProxiesForCountry.mockClear()
    generateClaudeSetupTokenUrl.mockClear()

    const openAIButton = wrapper.findAll('button').find((button) => button.text() === 'OpenAI')
    expect(openAIButton).toBeTruthy()
    await openAIButton!.trigger('click')
    await flushPromises()

    expect(releaseProxyReservation).not.toHaveBeenCalled()
    expect(getProxiesForCountry).not.toHaveBeenCalled()
    expect(generateClaudeSetupTokenUrl).not.toHaveBeenCalled()
    expect(generateOpenAIAuthUrl).toHaveBeenCalledWith({ proxy_id: 10 })
  })

  it('ignores stale Anthropic auth URL responses after switching to OpenAI', async () => {
    let resolveClaudeAuthUrl: ((value: { auth_url: string; session_id: string }) => void) | undefined
    generateClaudeSetupTokenUrl.mockImplementationOnce(() => new Promise((resolve) => {
      resolveClaudeAuthUrl = resolve
    }))
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    const openAIButton = wrapper.findAll('button').find((button) => button.text() === 'OpenAI')
    expect(openAIButton).toBeTruthy()
    await openAIButton!.trigger('click')
    await flushPromises()

    resolveClaudeAuthUrl?.({
      auth_url: 'https://claude.example/late-oauth',
      session_id: 'late-claude-session'
    })
    await flushPromises()

    await wrapper.find('[data-testid="auth-code-input"]').setValue('openai-auth-code')
    await wrapper.find('[data-testid="submit-auth-code"]').trigger('click')
    await flushPromises()

    expect(exchangeOpenAICode).toHaveBeenCalledWith(expect.objectContaining({
      session_id: 'openai-session-123',
      code: 'openai-auth-code',
      proxy_id: 10
    }))
  })

  it('configures OpenAI OAuth authorization methods like the admin OAuth flow', async () => {
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    const openAIButton = wrapper.findAll('button').find((button) => button.text() === 'OpenAI')
    expect(openAIButton).toBeTruthy()
    await openAIButton!.trigger('click')
    await flushPromises()

    const flowButton = wrapper.find('.generate-url')
    expect(flowButton.attributes('data-add-method')).toBe('oauth')
    expect(flowButton.attributes('data-show-refresh-token')).toBe('true')
    expect(flowButton.attributes('data-show-mobile-refresh-token')).toBe('true')
    expect(flowButton.attributes('data-force-method-selection')).toBe('true')
  })

  it('creates OpenAI OAuth contributor account through refresh token method with the reserved proxy', async () => {
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    const openAIButton = wrapper.findAll('button').find((button) => button.text() === 'OpenAI')
    expect(openAIButton).toBeTruthy()
    await openAIButton!.trigger('click')
    await flushPromises()

    await wrapper.find('[data-testid="select-refresh-token-method"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="submit-auth-code"]').exists()).toBe(false)

    await wrapper.find('[data-testid="validate-refresh-token"]').trigger('click')
    await flushPromises()

    expect(refreshOpenAIToken).toHaveBeenCalledWith({
      refresh_token: 'openai-refresh-token-input',
      proxy_id: 10,
      client_id: undefined
    })
    expect(create).toHaveBeenCalledWith(expect.objectContaining({
      platform: 'openai',
      type: 'oauth',
      add_method: 'oauth',
      credentials: expect.objectContaining({
        access_token: 'openai-access-token',
        refresh_token: 'openai-refresh-token',
        expires_at: 1767225600,
        email: 'openai@example.com'
      }),
      extra: {
        email: 'openai@example.com'
      },
      proxy_id: 10
    }))
    expect(create.mock.calls[0][0].name).toBe('OpenAI OAuth')
    expect(showSuccess).toHaveBeenCalledWith('OpenAI 账号授权已提交')
  })

  it('exchanges auth code and creates contributor OpenAI OAuth account with the reserved proxy', async () => {
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    const openAIButton = wrapper.findAll('button').find((button) => button.text() === 'OpenAI')
    expect(openAIButton).toBeTruthy()
    await openAIButton!.trigger('click')
    await flushPromises()

    await wrapper.find('[data-testid="auth-code-input"]').setValue('openai-auth-code')
    await wrapper.find('[data-testid="submit-auth-code"]').trigger('click')
    await flushPromises()

    expect(exchangeOpenAICode).toHaveBeenCalledWith({
      session_id: 'openai-session-123',
      code: 'openai-auth-code',
      state: 'state-123',
      proxy_id: 10
    })
    expect(exchangeClaudeSetupTokenCode).not.toHaveBeenCalled()
    expect(create).toHaveBeenCalledWith(expect.objectContaining({
      platform: 'openai',
      type: 'oauth',
      add_method: 'oauth',
      credentials: expect.objectContaining({
        access_token: 'openai-access-token',
        refresh_token: 'openai-refresh-token',
        expires_at: 1767225600,
        email: 'openai@example.com',
        chatgpt_account_id: 'chatgpt-account-123'
      }),
      extra: {
        email: 'openai@example.com'
      },
      proxy_id: 10,
      concurrency: 10,
      priority: 1,
      rate_multiplier: 1,
      group_ids: [],
      auto_pause_on_expired: true
    }))
    expect(create.mock.calls[0][0].name).toBe('OpenAI OAuth')
    expect(showSuccess).toHaveBeenCalledWith('OpenAI 账号授权已提交')
  })

  it('loads the country-selected proxy and includes country when creating the account', async () => {
    routeState.query = { country: ' us ' }
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    expect(getProxiesForCountry).toHaveBeenCalledWith('US')
    expect(wrapper.find('select').attributes('disabled')).toBeDefined()

    await wrapper.find('[data-testid="submit-auth-code"]').trigger('click')
    await flushPromises()

    expect(create).toHaveBeenCalledWith(expect.objectContaining({
      proxy_id: 10,
      country: 'US'
    }))
  })

  it('releases the current reservation before restarting authorization', async () => {
    routeState.query = { country: 'US' }
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    await wrapper.find('[data-testid="submit-auth-code"]').trigger('click')
    await flushPromises()

    getProxiesForCountry.mockClear()
    generateClaudeSetupTokenUrl.mockClear()

    const resetButton = wrapper.findAll('button').find((button) => button.text() === '重新授权')
    expect(resetButton).toBeTruthy()
    await resetButton!.trigger('click')
    await flushPromises()

    expect(releaseProxyReservation).toHaveBeenCalledTimes(1)
    expect(getProxiesForCountry).toHaveBeenCalledWith('US')
    expect(generateClaudeSetupTokenUrl).toHaveBeenCalledWith({ proxy_id: 10 })
  })
})
