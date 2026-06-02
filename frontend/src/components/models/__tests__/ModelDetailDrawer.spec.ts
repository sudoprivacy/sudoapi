// sudoapi: Model Square model catalog.

import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import type { ModelSquareCard } from '@/api/models'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (key === 'modelSquare.detail.discountValue') return `${params?.fold} 折`
      if (key === 'modelSquare.detail.rateMultiplier') return '有效倍率'
      return key
    },
  }),
}))

import ModelDetailDrawer from '../ModelDetailDrawer.vue'

function makeCard(): ModelSquareCard {
  return {
    name: 'claude-test',
    display_name: 'Claude Test',
    category: 'claude',
    description: 'test model',
    model_type: 'chat',
    context_window: 200000,
    max_output: 4096,
    input_modalities: ['text'],
    output_modalities: ['text'],
    support_flags: [],
    capabilities: [],
    featured: false,
    icon_url: null,
    official_price: null,
    platforms: [
      {
        platform: 'anthropic',
        endpoints: [],
        group_prices: [
          {
            group_id: 1,
            group_name: 'discount-group',
            subscription_type: 'standard',
            is_exclusive: false,
            base_rate_multiplier: 0.8,
            user_rate_multiplier: null,
            billing_mode: 'token',
            input_price_per_mtok_usd: 10,
            output_price_per_mtok_usd: 20,
            cache_read_price_per_mtok_usd: null,
            cache_write_price_per_mtok_usd: null,
            image_output_price_per_mtok_usd: null,
            per_request_price_usd: null,
            intervals: [],
            channel_chain: [],
            cache_creation_5m_price_per_mtok_usd: null,
            cache_creation_1h_price_per_mtok_usd: null,
          },
          {
            group_id: 2,
            group_name: 'full-price-group',
            subscription_type: 'standard',
            is_exclusive: false,
            base_rate_multiplier: 1,
            user_rate_multiplier: null,
            billing_mode: 'token',
            input_price_per_mtok_usd: 10,
            output_price_per_mtok_usd: 20,
            cache_read_price_per_mtok_usd: null,
            cache_write_price_per_mtok_usd: null,
            image_output_price_per_mtok_usd: null,
            per_request_price_usd: null,
            intervals: [],
            channel_chain: [],
            cache_creation_5m_price_per_mtok_usd: null,
            cache_creation_1h_price_per_mtok_usd: null,
          },
        ],
      },
    ],
  }
}

describe('ModelDetailDrawer', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    document.body.style.overflow = ''
  })

  it('shows discount badge and hides the old effective multiplier copy', () => {
    const wrapper = mount(ModelDetailDrawer, {
      attachTo: document.body,
      props: {
        open: true,
        card: makeCard(),
      },
      global: {
        stubs: {
          Icon: { template: '<span />' },
          ModelIcon: { template: '<span />' },
        },
      },
    })

    const text = document.body.textContent ?? ''
    expect(text).toContain('8 折')
    expect(text).not.toContain('10 折')
    expect(text).not.toContain('有效倍率')
    expect(text).toContain('$8.00')
    expect(text).toContain('$10.00')
    expect(text).toContain('/ MTok')
    wrapper.unmount()
  })
})
