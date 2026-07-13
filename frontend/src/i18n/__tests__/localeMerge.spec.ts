// sudoapi: Local i18n overlay.

import { describe, expect, it } from 'vitest'

import { mergeLocaleMessages } from '../locales/merge'

describe('mergeLocaleMessages', () => {
  it('deep merges extension messages without replacing sibling keys', () => {
    const base = {
      payment: {
        methods: {
          stripe: 'Stripe'
        },
        status: {
          paid: 'Paid'
        }
      }
    }

    const result = mergeLocaleMessages(base, {
      payment: {
        methods: {
          fuiou: 'Fuiou Pay'
        }
      }
    })

    expect(result.payment.methods).toEqual({
      stripe: 'Stripe',
      fuiou: 'Fuiou Pay'
    })
    expect(result.payment.status.paid).toBe('Paid')
  })
})
