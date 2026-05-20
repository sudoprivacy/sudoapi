import { describe, expect, it, vi, beforeEach } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import ClaudeAuthView from '../ClaudeAuthView.vue'

const generateClaudeAuthUrl = vi.fn()
const exchangeClaudeCode = vi.fn()
const create = vi.fn()
const showSuccess = vi.fn()
const showError = vi.fn()

vi.mock('@/api/contributor', () => ({
  contributorAPI: {
    accounts: {
      generateClaudeAuthUrl: (...args: any[]) => generateClaudeAuthUrl(...args),
      exchangeClaudeCode: (...args: any[]) => exchangeClaudeCode(...args),
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

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/components/account/OAuthAuthorizationFlow.vue', () => ({
  default: defineComponent({
    name: 'OAuthAuthorizationFlow',
    emits: ['generate-url'],
    setup(_props, { expose, emit }) {
      const authCode = ref('claude-auth-code')
      expose({
        authCode,
        reset: vi.fn(() => {
          authCode.value = ''
        })
      })
      return () => h('button', { class: 'generate-url', onClick: () => emit('generate-url') }, 'generate')
    }
  })
}))

describe('ClaudeAuthView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    generateClaudeAuthUrl.mockResolvedValue({
      auth_url: 'https://claude.example/oauth',
      session_id: 'session-123'
    })
    exchangeClaudeCode.mockResolvedValue({
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

    expect(generateClaudeAuthUrl).toHaveBeenCalledWith({ proxy_id: null })
  })

  it('exchanges auth code and creates contributor Anthropic OAuth account', async () => {
    const wrapper = mount(ClaudeAuthView)
    await flushPromises()

    await wrapper.find('.btn-primary').trigger('click')
    await flushPromises()

    expect(exchangeClaudeCode).toHaveBeenCalledWith({
      session_id: 'session-123',
      code: 'claude-auth-code',
      proxy_id: null
    })
    expect(create).toHaveBeenCalledWith(expect.objectContaining({
      platform: 'anthropic',
      type: 'oauth',
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
    expect(create.mock.calls[0][0].name).toMatch(/^claude-\d+-[a-z0-9]+$/)
    expect(showSuccess).toHaveBeenCalledWith('Claude 账号授权已提交')
  })
})
