import { describe, expect, it, vi, beforeEach } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import ClaudeAuthView from '../ClaudeAuthView.vue'

const generateClaudeAuthUrl = vi.fn()
const generateClaudeSetupTokenUrl = vi.fn()
const exchangeClaudeCode = vi.fn()
const exchangeClaudeSetupTokenCode = vi.fn()
const create = vi.fn()
const showSuccess = vi.fn()
const showError = vi.fn()

vi.mock('@/api/contributor', () => ({
  contributorAPI: {
    accounts: {
      generateClaudeAuthUrl: (...args: any[]) => generateClaudeAuthUrl(...args),
      generateClaudeSetupTokenUrl: (...args: any[]) => generateClaudeSetupTokenUrl(...args),
      exchangeClaudeCode: (...args: any[]) => exchangeClaudeCode(...args),
      exchangeClaudeSetupTokenCode: (...args: any[]) => exchangeClaudeSetupTokenCode(...args),
      create: (...args: any[]) => create(...args)
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess,
    showError
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: null
  })
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      t: (key: string) => key,
      locale: { value: 'en' },
      setLocaleMessage: vi.fn()
    }
  }),
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/components/account/OAuthAuthorizationFlow.vue', () => ({
  default: defineComponent({
    name: 'OAuthAuthorizationFlow',
    props: {
      addMethod: {
        type: String,
        required: true
      }
    },
    emits: ['generate-url'],
    setup(props, { expose, emit }) {
      const authCode = ref('claude-auth-code')
      expose({
        authCode,
        reset: vi.fn(() => {
          authCode.value = ''
        })
      })
      return () =>
        h(
          'button',
          { class: 'generate-url', 'data-add-method': props.addMethod, onClick: () => emit('generate-url') },
          'generate'
        )
    }
  })
}))

describe('ClaudeAuthView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    generateClaudeSetupTokenUrl.mockResolvedValue({
      auth_url: 'https://claude.example/oauth',
      session_id: 'session-123'
    })
    exchangeClaudeSetupTokenCode.mockResolvedValue({
      access_token: 'access-token',
      refresh_token: 'refresh-token',
      org_uuid: 'org-123',
      account_uuid: 'account-123',
      email_address: 'claude@example.com'
    })
    create.mockResolvedValue({ id: 1 })
  })

  it('auto-generates auth URL on mount', async () => {
    mount(ClaudeAuthView)
    await flushPromises()

    expect(generateClaudeSetupTokenUrl).toHaveBeenCalledWith({ proxy_id: null })
    expect(generateClaudeAuthUrl).not.toHaveBeenCalled()
  })

  it('defaults Claude authorization add method to setup token', async () => {
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    expect(wrapper.find('.generate-url').attributes('data-add-method')).toBe('setup-token')
  })

  it('exchanges auth code and creates contributor Anthropic setup token account', async () => {
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    await wrapper.find('.btn-primary').trigger('click')
    await flushPromises()

    expect(exchangeClaudeSetupTokenCode).toHaveBeenCalledWith({
      session_id: 'session-123',
      code: 'claude-auth-code',
      proxy_id: null
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
      proxy_id: null,
      concurrency: 10,
      priority: 1,
      rate_multiplier: 1,
      group_ids: [],
      auto_pause_on_expired: true
    }))
    expect(create.mock.calls[0][0].name).toBe('Claude OAuth')
    expect(showSuccess).toHaveBeenCalledWith('Claude 账号授权已提交')
  })
})
