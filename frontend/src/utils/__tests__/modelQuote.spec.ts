import { describe, expect, it } from 'vitest'
import type { ModelSquareCard } from '@/api/models'
import {
  buildModelQuoteRows,
  calculateDiscount,
  filterModelQuoteRows,
  lowestPlatformPrice,
  sortModelQuoteRows,
  type ModelQuotePriceSet,
} from '../modelQuote'

const emptyPriceSet = (): ModelQuotePriceSet => ({
  input: null,
  output: null,
  cacheRead: null,
  cacheWrite: null,
  imageOutput: null,
  imageOrRequest: null,
})

describe('modelQuote', () => {
  it('flattens model square cards into one quote row per model platform', () => {
    const cards: ModelSquareCard[] = [
      {
        name: 'gpt-quote',
        display_name: 'GPT Quote',
        category: 'gpt',
        description: '',
        model_type: 'chat',
        context_window: 128000,
        max_output: 16000,
        input_modalities: [],
        output_modalities: [],
        support_flags: [],
        capabilities: [],
        featured: false,
        icon_url: null,
        official_price: {
          input_price_per_mtok_usd: 2,
          output_price_per_mtok_usd: 10,
          cache_read_price_per_mtok_usd: null,
          cache_write_price_per_mtok_usd: null,
          image_output_price_per_mtok_usd: null,
          image_price_usd: null,
        },
        platforms: [
          { platform: 'openai', endpoints: [], group_prices: [] },
          { platform: 'antigravity', endpoints: [], group_prices: [] },
        ],
      },
    ]

    const rows = buildModelQuoteRows(cards)

    expect(rows.map((row) => row.id)).toEqual(['gpt-quote::openai', 'gpt-quote::antigravity'])
    expect(rows[0]).toMatchObject({
      model: 'gpt-quote',
      displayName: 'GPT Quote',
      vendor: 'gpt',
      platform: 'openai',
      modelType: 'chat',
      contextWindow: 128000,
    })
  })

  it('selects the lowest effective platform price including multipliers and intervals', () => {
    const price = lowestPlatformPrice([
      {
        group_id: 1,
        group_name: 'standard',
        subscription_type: 'standard',
        is_exclusive: false,
        base_rate_multiplier: 0.5,
        user_rate_multiplier: null,
        billing_mode: 'token',
        input_price_per_mtok_usd: 4,
        output_price_per_mtok_usd: 20,
        cache_read_price_per_mtok_usd: 1,
        cache_write_price_per_mtok_usd: 2,
        cache_creation_5m_price_per_mtok_usd: null,
        cache_creation_1h_price_per_mtok_usd: null,
        image_output_price_per_mtok_usd: null,
        per_request_price_usd: null,
        intervals: [
          {
            min_tokens: 0,
            max_tokens: null,
            tier_label: '',
            input_price_per_mtok_usd: 1,
            output_price_per_mtok_usd: 8,
            cache_read_price_per_mtok_usd: null,
            cache_write_price_per_mtok_usd: null,
            cache_creation_5m_price_per_mtok_usd: null,
            cache_creation_1h_price_per_mtok_usd: null,
            per_request_price_usd: null,
            sort_order: 0,
          },
        ],
        channel_chain: ['hidden'],
      },
      {
        group_id: 2,
        group_name: 'backup',
        subscription_type: 'standard',
        is_exclusive: false,
        base_rate_multiplier: 1,
        user_rate_multiplier: null,
        billing_mode: 'per_request',
        input_price_per_mtok_usd: 3,
        output_price_per_mtok_usd: 9,
        cache_read_price_per_mtok_usd: null,
        cache_write_price_per_mtok_usd: null,
        cache_creation_5m_price_per_mtok_usd: null,
        cache_creation_1h_price_per_mtok_usd: null,
        image_output_price_per_mtok_usd: null,
        per_request_price_usd: 0.02,
        intervals: [],
        channel_chain: [],
      },
    ])

    expect(price.input).toBe(0.5)
    expect(price.output).toBe(4)
    expect(price.cacheRead).toBe(0.5)
    expect(price.cacheWrite).toBe(1)
    expect(price.imageOrRequest).toBe(0.02)
  })

  it('calculates discount using input before later price dimensions', () => {
    const official: ModelQuotePriceSet = {
      ...emptyPriceSet(),
      input: 2,
      output: 10,
    }
    const platformPrice: ModelQuotePriceSet = {
      ...emptyPriceSet(),
      input: 1,
      output: 2,
    }

    expect(calculateDiscount(official, platformPrice)).toEqual({
      discountRatio: 0.5,
      discountBasis: 'input',
    })
  })

  it('returns unmatched discount when LiteLLM official price is missing', () => {
    const result = calculateDiscount(emptyPriceSet(), {
      ...emptyPriceSet(),
      input: 1,
    })

    expect(result).toEqual({ discountRatio: null, discountBasis: null })
  })

  it('filters and sorts quote rows', () => {
    const rows = [
      {
        id: 'a',
        model: 'claude-sonnet',
        displayName: 'Claude Sonnet',
        vendor: 'claude',
        platform: 'anthropic',
        modelType: 'chat',
        contextWindow: 200000,
        official: emptyPriceSet(),
        platformPrice: { ...emptyPriceSet(), input: 3 },
        discountRatio: 1,
        discountBasis: 'input' as const,
      },
      {
        id: 'b',
        model: 'gpt-mini',
        displayName: 'GPT Mini',
        vendor: 'gpt',
        platform: 'openai',
        modelType: 'responses',
        contextWindow: 128000,
        official: emptyPriceSet(),
        platformPrice: { ...emptyPriceSet(), input: 0.6 },
        discountRatio: 0.3,
        discountBasis: 'input' as const,
      },
    ]

    const filtered = filterModelQuoteRows(rows, {
      search: 'gpt',
      platform: 'gpt',
      modelType: 'responses',
    })
    expect(filtered.map((row) => row.model)).toEqual(['gpt-mini'])

    expect(sortModelQuoteRows(rows, 'priceAsc').map((row) => row.model)).toEqual(['gpt-mini', 'claude-sonnet'])
    expect(sortModelQuoteRows(rows, 'discountAsc').map((row) => row.model)).toEqual(['gpt-mini', 'claude-sonnet'])
    expect(sortModelQuoteRows(rows, 'contextDesc').map((row) => row.model)).toEqual(['claude-sonnet', 'gpt-mini'])
  })
})
