import { describe, expect, it } from 'vitest'
import {
  PAYMENT_CURRENCY_OPTIONS,
  PROVIDER_CALLBACK_PATHS,
  PROVIDER_CONFIG_FIELDS,
  PROVIDER_SUPPORTED_TYPES,
  WEBHOOK_PATHS,
} from '@/components/payment/providerConfig'

function findField(providerKey: string, key: string) {
  const fields = PROVIDER_CONFIG_FIELDS[providerKey] || []
  return fields.find(field => field.key === key)
}

describe('PROVIDER_CONFIG_FIELDS.wxpay', () => {
  it('keeps admin form validation aligned with backend-required credentials', () => {
    expect(findField('wxpay', 'publicKeyId')?.optional).toBeFalsy()
    expect(findField('wxpay', 'certSerial')?.optional).toBeFalsy()
  })

  it('only keeps the simplified visible credential set in the admin form', () => {
    expect(findField('wxpay', 'mpAppId')).toBeUndefined()
    expect(findField('wxpay', 'h5AppName')).toBeUndefined()
    expect(findField('wxpay', 'h5AppUrl')).toBeUndefined()
  })
})

describe('PROVIDER_CONFIG_FIELDS.airwallex', () => {
  it('adds currency config with CNY as the default', () => {
    const currency = findField('airwallex', 'currency')

    expect(currency?.defaultValue).toBe('CNY')
    expect(currency?.hintKey).toBe('admin.settings.payment.field_paymentCurrencyHint')
    expect(currency?.options).toBe(PAYMENT_CURRENCY_OPTIONS)
  })

  it('marks accountId as optional and explains when it can be left blank', () => {
    const accountId = findField('airwallex', 'accountId')

    expect(accountId?.optional).toBe(true)
    expect(accountId?.clearable).toBe(true)
    expect(accountId?.hintKey).toBe('admin.settings.payment.field_accountIdHint')
  })

  it('explains that apiBase must match the Airwallex key environment', () => {
    expect(findField('airwallex', 'apiBase')?.hintKey).toBe('admin.settings.payment.field_airwallexApiBaseHint')
  })
})

describe('PROVIDER_CONFIG_FIELDS.stripe', () => {
  it('adds currency config with CNY as the default', () => {
    const currency = findField('stripe', 'currency')

    expect(currency?.defaultValue).toBe('CNY')
    expect(currency?.hintKey).toBe('admin.settings.payment.field_paymentCurrencyHint')
    expect(currency?.options).toBe(PAYMENT_CURRENCY_OPTIONS)
  })
})

describe('PROVIDER_CONFIG_FIELDS.fuiou', () => {
  it('exposes mchnt_cd as a non-sensitive identity field', () => {
    const mchnt = findField('fuiou', 'mchntCd')
    expect(mchnt).toBeDefined()
    expect(mchnt?.sensitive).toBe(false)
  })

  it('marks the RSA key pair fields as sensitive', () => {
    expect(findField('fuiou', 'fuiouPublicKey')?.sensitive).toBe(true)
    expect(findField('fuiou', 'merchantPrivateKey')?.sensitive).toBe(true)
  })

  it('defaults apiBase to the Fuiou production gateway with an environment hint', () => {
    const apiBase = findField('fuiou', 'apiBase')
    expect(apiBase?.defaultValue).toBe('https://hlwnets.fuioupay.com')
    expect(apiBase?.hintKey).toBe('admin.settings.payment.field_fuiouApiBaseHint')
  })

  it('defaults currency to CNY', () => {
    const currency = findField('fuiou', 'currency')
    expect(currency?.defaultValue).toBe('CNY')
    expect(currency?.options).toBe(PAYMENT_CURRENCY_OPTIONS)
  })

  it('supports both alipay and wxpay payment types', () => {
    expect(PROVIDER_SUPPORTED_TYPES.fuiou).toEqual(['alipay', 'wxpay'])
  })

  it('registers a dedicated webhook path', () => {
    expect(WEBHOOK_PATHS.fuiou).toBe('/api/v1/payment/webhook/fuiou')
  })

  it('exposes notify + return callback paths for the admin form', () => {
    expect(PROVIDER_CALLBACK_PATHS.fuiou?.notifyUrl).toBe(WEBHOOK_PATHS.fuiou)
    expect(PROVIDER_CALLBACK_PATHS.fuiou?.returnUrl).toBe('/payment/result')
  })
})
