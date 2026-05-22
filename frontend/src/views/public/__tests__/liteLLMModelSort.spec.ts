import { describe, expect, it } from 'vitest'
import type { LiteLLMModel } from '@/api/models'
import { sortLiteLLMModels } from '../liteLLMModelSort'

function model(name: string, serialNumber: number | null): LiteLLMModel {
  return {
    name,
    serial_number: serialNumber,
    provider: 'openai',
    mode: 'chat',
    category: 'gpt',
    max_tokens: 0,
    max_input_tokens: 0,
    max_output_tokens: 0,
    input_price_per_mtok_usd: null,
    input_price_priority_per_mtok_usd: null,
    output_price_per_mtok_usd: null,
    output_price_priority_per_mtok_usd: null,
    cache_creation_price_per_mtok_usd: null,
    cache_creation_above_1h_price_per_mtok_usd: null,
    cache_read_price_per_mtok_usd: null,
    cache_read_priority_price_per_mtok_usd: null,
    output_price_per_image_usd: null,
    output_price_per_image_mtok_usd: null,
    long_context_input_token_threshold: 0,
    long_context_input_cost_multiplier: 0,
    long_context_output_cost_multiplier: 0,
    supports_prompt_caching: false,
    supports_service_tier: false,
    supported_modalities: [],
    output_modalities: [],
    support_flags: [],
    capabilities: [],
    raw_fields: {},
  }
}

describe('sortLiteLLMModels', () => {
  it('sorts latestDesc by serial_number descending', () => {
    const rows = [
      model('gpt-c', 30),
      model('gpt-a', 10),
      model('gpt-b', 20),
    ]
    expect(sortLiteLLMModels(rows, 'latestDesc').map((row) => row.name)).toEqual([
      'gpt-c',
      'gpt-b',
      'gpt-a',
    ])
  })

  it('puts rows without serial_number last and ties by name', () => {
    const rows = [
      model('gpt-z', null),
      model('gpt-b', 1),
      model('gpt-a', 1),
    ]
    expect(sortLiteLLMModels(rows, 'latestDesc').map((row) => row.name)).toEqual([
      'gpt-a',
      'gpt-b',
      'gpt-z',
    ])
  })

  it('handles serial_number values defensively when API data is stringly typed', () => {
    const rows = [
      { ...model('gpt-10', null), serial_number: '10' as unknown as number },
      { ...model('gpt-2', null), serial_number: '2' as unknown as number },
    ]
    expect(sortLiteLLMModels(rows, 'latestDesc').map((row) => row.name)).toEqual([
      'gpt-10',
      'gpt-2',
    ])
  })
})
