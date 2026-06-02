// sudoapi: Model Square rate helpers.

import { describe, expect, it } from 'vitest'
import { effectiveModelRateMultiplier } from '../modelRate'

describe('effectiveModelRateMultiplier', () => {
  it('uses the user group rate as an override instead of multiplying it by the group rate', () => {
    expect(
      effectiveModelRateMultiplier({
        base_rate_multiplier: 0.5,
        user_rate_multiplier: 0.8,
      }),
    ).toBe(0.8)
  })

  it('falls back to the group rate when no user override is present', () => {
    expect(
      effectiveModelRateMultiplier({
        base_rate_multiplier: 0.5,
        user_rate_multiplier: null,
      }),
    ).toBe(0.5)
  })
})
