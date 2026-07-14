// sudoapi: Local i18n overlay.

export default {
  admin: {
    accounts: {
      dataImportResultSummary: '代理创建 {proxy_created}，复用 {proxy_reused}，失败 {proxy_failed}；账号创建 {account_created}，跳过 {account_skipped}，失败 {account_failed}',
      dataImportSuccess: '导入完成：账号创建 {account_created}，跳过 {account_skipped}，失败 {account_failed}',
      dataImportCompletedWithErrors: '导入完成但有错误：账号跳过 {account_skipped}，账号失败 {account_failed}，代理失败 {proxy_failed}',
    },
    settings: {
      payment: {
        providerFuiou: '富友支付',
        field_mchntCd: '商户号',
        field_fuiouPublicKey: '富友公钥',
        field_merchantPrivateKey: '商户私钥',
        field_fuiouApiBaseHint: '生产环境填写 https://hlwnets.fuioupay.com，沙箱/测试环境填写 https://hlwnets-test.fuioupay.com。请确保 API Key 与环境一致。',
        fuiouGuideSummary: '富友聚合支付：通过同一个商户号同时受理支付宝和微信支付。',
        fuiouGuideNote: '商户私钥需 PKCS8 + Base64 编码；富友公钥为对接方在富友后台下载的 PKIX + Base64 公钥；商户号 mchnt_cd 同时用于回调归属，请确保与富友合同/后台一致。',
      },
    },
    // sudoapi: Channel TTL-specific cache creation pricing.
    channels: {
      intervalValidation: {
        price: {
          cacheCreation5mPrice: '5分钟缓存创建价格',
          cacheCreation1hPrice: '1小时缓存创建价格',
        },
      },
      form: {
        cacheCreation5mPrice: '缓存创建 5m',
        cacheCreation1hPrice: '缓存创建 1h',
        cacheCreation5mPriceShort: '5m',
        cacheCreation1hPriceShort: '1h',
      },
    },
  },
  payment: {
    methods: {
      fuiou: '富友支付',
    },
  },
}
