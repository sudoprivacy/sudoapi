// sudoapi: Local i18n overlay.

export default {
  admin: {
    accounts: {
      dataImportResultSummary: 'Proxies created {proxy_created}, reused {proxy_reused}, failed {proxy_failed}; Accounts created {account_created}, skipped {account_skipped}, failed {account_failed}',
      dataImportSuccess: 'Import completed: accounts created {account_created}, skipped {account_skipped}, failed {account_failed}',
      dataImportCompletedWithErrors: 'Import completed with errors: account skipped {account_skipped}, account failed {account_failed}, proxy failed {proxy_failed}',
    },
    settings: {
      payment: {
        providerFuiou: 'Fuiou Pay',
        field_mchntCd: 'Merchant ID (mchnt_cd)',
        field_fuiouPublicKey: 'Fuiou Public Key',
        field_merchantPrivateKey: 'Merchant Private Key',
        field_fuiouApiBaseHint: 'Use https://hlwnets.fuioupay.com for production and https://hlwnets-test.fuioupay.com for sandbox/test. The base must match the environment your API keys were issued for.',
        fuiouGuideSummary: 'Fuiou aggregate payment: accept Alipay and WeChat Pay through a single merchant account.',
        fuiouGuideNote: 'The merchant private key must be PKCS8 + Base64; the Fuiou public key is the PKIX + Base64 key downloaded from the Fuiou merchant portal. mchnt_cd is also used to route async callbacks, so it must match your Fuiou contract / portal value exactly.',
      },
    },
    // sudoapi: Channel TTL-specific cache creation pricing.
    channels: {
      intervalValidation: {
        price: {
          cacheCreation5mPrice: '5-minute cache creation price',
          cacheCreation1hPrice: '1-hour cache creation price',
        },
      },
      form: {
        cacheCreation5mPrice: 'Cache Create 5m',
        cacheCreation1hPrice: 'Cache Create 1h',
        cacheCreation5mPriceShort: '5m',
        cacheCreation1hPriceShort: '1h',
      },
    },
  },
  payment: {
    methods: {
      fuiou: 'Fuiou Pay',
    },
  },
}
