// sudoapi: Account contributor review workflow.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ProxiesView from '../ProxiesView.vue'
import type { Proxy } from '@/types'
import { buildContributorLoginLink } from '@/utils/contributorLoginLink'

const { listProxies, copyToClipboard, showError, showSuccess } = vi.hoisted(() => ({
  listProxies: vi.fn(),
  copyToClipboard: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    proxies: {
      list: listProxies,
      getAll: vi.fn(),
      getAllWithCount: vi.fn(),
      getById: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      deleteProxy: vi.fn(),
      toggleStatus: vi.fn(),
      testProxy: vi.fn(),
      checkProxyQuality: vi.fn(),
      getStats: vi.fn(),
      getProxyAccounts: vi.fn(),
      batchCreate: vi.fn(),
      batchDelete: vi.fn(),
      exportData: vi.fn(),
      importData: vi.fn()
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showInfo: vi.fn()
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard
  })
}))

vi.mock('@/composables/useSwipeSelect', () => ({
  useSwipeSelect: vi.fn()
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const DataTableStub = {
  props: ['data'],
  template: `
    <table>
      <tbody>
        <tr v-for="row in data" :key="row.id">
          <td>
            <slot name="cell-address" :row="row" />
          </td>
          <td>
            <slot name="cell-actions" :row="row" />
          </td>
        </tr>
      </tbody>
    </table>
  `
}

function proxy(overrides: Partial<Proxy>): Proxy {
  return {
    id: 1,
    name: 'US default',
    protocol: 'http',
    host: 'proxy.example.com',
    port: 8080,
    username: 'user',
    password: 'pass',
    status: 'active',
    country_code: 'us',
    created_at: '2026-06-01T00:00:00Z',
    updated_at: '2026-06-01T00:00:00Z',
    ...overrides
  }
}

function mountView() {
  return mount(ProxiesView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
        },
        DataTable: DataTableStub,
        Pagination: true,
        BaseDialog: true,
        ConfirmDialog: true,
        EmptyState: true,
        ImportDataModal: true,
        Select: true,
        ProxyAdBanner: true,
        Icon: { props: ['name'], template: '<span :data-icon="name" />' },
        PlatformTypeBadge: true,
        Teleport: true
      }
    }
  })
}

describe('admin ProxiesView contributor login link', () => {
  beforeEach(() => {
    listProxies.mockReset()
    copyToClipboard.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
  })

  it('builds an absolute contributor login link with normalized country code', () => {
    expect(buildContributorLoginLink('us', 'https://example.test')).toBe(
      'https://example.test/contributor/login?country=US'
    )
  })

  it('copies the contributor login link from a proxy country code', async () => {
    listProxies.mockResolvedValue({
      items: [proxy({ id: 11, country_code: 'us' })],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="copy-contributor-login-link"]').trigger('click')

    expect(copyToClipboard).toHaveBeenCalledWith(
      `${window.location.origin}/contributor/login?country=US`,
      'admin.proxies.contributorLoginLinkCopied'
    )
  })

  it('disables contributor link copying when the country code is missing', async () => {
    listProxies.mockResolvedValue({
      items: [proxy({ id: 12, country_code: '' })],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const button = wrapper.get<HTMLButtonElement>('[data-test="copy-contributor-login-link"]')
    expect(button.element.disabled).toBe(true)

    await button.trigger('click')
    expect(copyToClipboard).not.toHaveBeenCalled()
  })

  it('keeps the original proxy URL copy action unchanged', async () => {
    listProxies.mockResolvedValue({
      items: [proxy({ id: 13 })],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="copy-proxy-url"]').trigger('click')

    expect(copyToClipboard).toHaveBeenCalledWith(
      'http://user:pass@proxy.example.com:8080',
      'admin.proxies.urlCopied'
    )
  })
})
