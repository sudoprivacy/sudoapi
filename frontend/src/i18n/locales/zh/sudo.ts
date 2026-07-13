// sudoapi: Local i18n overlay.

export default {
  admin: {
    accounts: {
      approve: "通过",
      columns: {
        reviewStatus: "审核状态"
      },
      dataImportCompletedWithErrors: "导入完成但有错误：账号跳过 {account_skipped}，账号失败 {account_failed}，代理失败 {proxy_failed}",
      dataImportResultSummary: "代理创建 {proxy_created}，复用 {proxy_reused}，失败 {proxy_failed}；账号创建 {account_created}，跳过 {account_skipped}，失败 {account_failed}",
      dataImportSuccess: "导入完成：账号创建 {account_created}，跳过 {account_skipped}，失败 {account_failed}",
      enterCustomModelNames: "输入自定义模型名称，多个模型之间空格分割",
      externalSubmission: "外部提交者",
      oauth: {
        antigravity: {
          refreshTokenLabel: "Refresh Token"
        },
        copyUrl: "复制链接",
        openai: {
          accessTokenAuth: "手动输入 AT",
          mobileRefreshTokenAuth: "手动输入 Mobile RT",
          refreshTokenLabel: "Refresh Token"
        },
        urlCopied: "链接已复制到剪贴板"
      },
      openai: {
        codexImageGenerationBridge: "Codex 图片生成桥接",
        codexImageGenerationBridgeBadgeDisabled: "账号关闭",
        codexImageGenerationBridgeBadgeEnabled: "账号开启",
        codexImageGenerationBridgeBadgeInherit: "渠道策略",
        codexImageGenerationBridgeDesc: "账号级策略优先于渠道和全局配置。仅控制 Codex 走 /responses 文本端点时是否注入 image_generation 工具；不影响独立图片生成接口。",
        codexImageGenerationBridgeDisabled: "强制关闭",
        codexImageGenerationBridgeDisabledDesc: "阻断 Codex /responses 的图片工具注入。",
        codexImageGenerationBridgeEnabled: "强制开启",
        codexImageGenerationBridgeEnabledDesc: "允许 Codex /responses 请求获得图片工具注入。",
        codexImageGenerationBridgeInherit: "跟随渠道",
        codexImageGenerationBridgeInheritDesc: "不写入账号覆盖，继续使用渠道或全局策略。"
      },
      reject: "拒绝",
      review: {
        approved: "已通过",
        pending: "待审核",
        rejected: "已拒绝"
      },
      reviewApproved: "账号已审核通过",
      reviewFilters: {
        approved: "已通过",
        external: "外部提交",
        pending: "待审核",
        rejected: "已拒绝"
      },
      reviewRejected: "账号已拒绝",
      reviewUpdateFailed: "更新审核状态失败"
    },
    channels: {
      endpointConfig: {
        addEndpoint: "添加端点",
        addModelType: "添加模型类型",
        description: "规则按平台全局生效。未匹配的模型类型不会展示端点。",
        invalidKey: "平台和模型类型只能包含小写字母、数字、下划线和连字符。",
        invalidMethod: "端点方法必须是 GET 或 POST。",
        invalidPath: "端点路径必须以 / 开头且不能包含空格。",
        loadError: "加载端点配置失败",
        modelTypePlaceholder: "模型类型，例如 chat",
        noRules: "暂无端点规则。未匹配模型不会展示端点。",
        platformRules: "模型类型端点规则",
        saveError: "保存端点配置失败",
        saveSuccess: "端点配置已保存",
        title: "端点配置"
      },
      form: {
        cacheCreation1hPrice: "缓存1h",
        cacheCreation5mPrice: "缓存5m",
        cacheWriteFallback: "回退缓存写入"
      },
      intervalValidation: {
        price: {
          cacheCreation1hPrice: "缓存 1h 创建价格",
          cacheCreation5mPrice: "缓存 5m 创建价格"
        }
      }
    },
    modelMetadata: {
      clear: "清除覆盖",
      clearConfirm: "确定要清除「{name}」的模型元数据覆盖吗？",
      deleteError: "清除模型元数据失败",
      deleteSuccess: "模型元数据覆盖已清除",
      description: "补齐 /models 页面缺失的模型展示信息",
      edit: "编辑元数据",
      fields: {
        actions: "操作",
        capabilities: "能力",
        category: "厂商",
        contextWindow: "上下文",
        description: "描述",
        displayName: "显示名",
        featured: "推荐",
        iconUrl: "图标 URL",
        inputModalities: "输入支持",
        maxOutput: "最大输出",
        missing: "缺失项",
        modelName: "模型名",
        modelType: "模型类型",
        outputModalities: "输出支持",
        platforms: "平台",
        supportFlags: "模型标签"
      },
      form: {
        capabilitiesHint: "能力标签会覆盖自动推断结果；不选则沿用自动数据。",
        categoryPlaceholder: "输入或选择厂商",
        contextWindowPlaceholder: "例如 200000",
        descriptionPlaceholder: "用于 /models 卡片和详情页展示",
        displayNamePlaceholder: "不填则使用模型名",
        featuredHint: "推荐模型会在 /models 排序中靠前。",
        iconUrlPlaceholder: "https://...",
        maxOutputPlaceholder: "例如 8192",
        modalitiesHint: "不选则沿用 LiteLLM 自动数据。",
        modelTypePlaceholder: "例如 chat / image_generation",
        supportFlagsHint: "模型标签来自 LiteLLM 中所有为 true 的 supports_* 字段；不选则沿用自动数据。"
      },
      loadError: "加载模型元数据失败",
      missingCount: "缺 {count} 项",
      missingFields: {
        capabilities: "能力",
        category: "厂商",
        context_window: "上下文",
        description: "描述",
        display_name: "显示名",
        icon_url: "图标",
        input_modalities: "输入支持",
        max_output: "最大输出",
        model_type: "模型类型",
        output_modalities: "输出支持",
        support_flags: "模型标签"
      },
      missingOnly: "仅看缺失",
      noModels: "暂无可维护模型",
      noModelsDesc: "当前没有来自活跃渠道的可展示模型",
      noResults: "没有匹配的模型",
      overrideActive: "已覆盖",
      refresh: "刷新",
      saveError: "保存模型元数据失败",
      saveSuccess: "模型元数据已保存",
      searchPlaceholder: "搜索模型、平台或描述...",
      title: "模型元数据"
    },
    modelSetting: {
      currentStatus: "当前状态",
      description: "上传 CSV 控制 /model 页面展示范围与最新优先排序。",
      duplicateRows: "重复行",
      fileName: "文件名",
      filePath: "文件路径",
      loadedRows: "已加载行",
      loadFailed: "加载模型白名单状态失败",
      modelCount: "白名单模型数",
      parseSummary: "解析摘要",
      skippedRows: "跳过空行",
      source: "来源",
      title: "模型白名单设置",
      totalRows: "数据行",
      updatedAt: "更新时间",
      upload: "上传并热更新",
      uploadFailed: "上传失败",
      uploadHint: "CSV 必须包含 serial_number 和 id 列；上传成功后会覆盖 data/model_setting/models_grouped_id_desc.csv 并立即生效。",
      uploading: "上传中...",
      uploadLabel: "白名单 CSV",
      uploadSuccess: "上传成功，已加载 {count} 个模型"
    },
    proxies: {
      contributorLoginLinkCopied: "贡献者链接已复制",
      copyContributorLoginLink: "贡献者链接",
      copyContributorLoginLinkTitle: "复制贡献者授权链接",
      countryCodeRequiredForContributorLink: "请先检测国家代码"
    },
    settings: {
      payment: {
        field_fuiouApiBaseHint: "生产环境填写 https://hlwnets.fuioupay.com，沙箱/测试环境填写 https://hlwnets-test.fuioupay.com。请确保 API Key 与环境一致。",
        field_fuiouPublicKey: "富友公钥",
        field_mchntCd: "商户号",
        field_merchantPrivateKey: "商户私钥",
        fuiouGuideNote: "商户私钥需 PKCS8 + Base64 编码；富友公钥为对接方在富友后台下载的 PKIX + Base64 公钥；商户号 mchnt_cd 同时用于回调归属，请确保与富友合同/后台一致。",
        fuiouGuideSummary: "富友聚合支付：通过同一个商户号同时受理支付宝和微信支付。",
        providerFuiou: "富友支付"
      }
    },
    users: {
      batch: {
        allFailed: "全部失败，请检查错误原因",
        columnsHint: "支持逗号 / 分号 / 制表符分隔；可带或不带表头；# 开头的行被忽略",
        columnsTitle: "CSV 列顺序（固定）",
        errorCodes: {
          CREATE_FAILED: "创建失败",
          DUPLICATE_IN_PAYLOAD: "与批次内其他行邮箱重复",
          EMAIL_EXISTS: "该邮箱在系统中已存在",
          INVALID_BALANCE: "余额必须为非负数",
          INVALID_CONCURRENCY: "并发数必须为非负整数",
          INVALID_EMAIL: "邮箱格式无效",
          INVALID_RPM: "RPM 必须为非负整数",
          WEAK_PASSWORD: "密码长度不足 6 位"
        },
        errorMsg: "错误原因",
        headerSkipped: "已跳过表头",
        menuLabel: "批量添加用户",
        pasteLabel: "或直接粘贴 CSV 内容",
        previewTitle: "预览",
        resultAborted: "已在首个错误后中止",
        resultSummary: "共 {total} 条：成功 {created} 条，失败 {failed} 条",
        resultTitle: "执行结果",
        securityNotice: "请在受信网络下操作。CSV 中包含明文密码，提交后不会被记录到日志。",
        skipOnError: "跳过错误行继续提交（已勾选时仅提交有效行）",
        statsDup: "重复",
        statsError: "错误",
        statsValid: "有效",
        status: "状态",
        submitFailed: "批量创建失败",
        submitting: "提交中...",
        submitWithCount: "提交 {count} 条",
        title: "批量创建用户",
        uploadCsv: "上传 CSV"
      }
    }
  },
  contributor: {
    accounts: {
      add: "添加账号",
      authorizationMethod: "授权方式",
      authSubmitted: "{platform} 账号授权已提交",
      authSubmittedReview: "{platform} 账号授权已提交，等待管理员审核。",
      columns: {
        created: "创建时间"
      },
      created: "账号已提交，等待管理员审核",
      description: "添加和维护你提交的大模型账号，管理员审核通过后才会参与调度。",
      edit: "编辑账号",
      generateAuthUrlFailed: "生成授权链接失败",
      loadFailed: "加载账号失败",
      loadProxiesFailed: "加载代理失败",
      missingOAuthState: "缺少 OAuth state",
      openaiPartialSubmitted: "OpenAI 账号部分授权已提交，成功 {success} 个，失败 {failed} 个",
      openaiRefreshTokenFailed: "OpenAI Refresh Token 验证失败",
      openaiRefreshTokenRequired: "请输入 Refresh Token",
      openaiSubmittedCount: "OpenAI 账号授权已提交 {count} 个",
      proxyLoading: "正在加载代理...",
      proxyUnavailable: "暂时没有可选代理，无法继续授权。",
      reauthorize: "重新授权",
      resetProxyReservationFailed: "重置代理预占失败",
      review: {
        approved: "已通过",
        pending: "待审核",
        rejected: "已拒绝"
      },
      reviewStatus: "审核状态",
      saveFailed: "保存账号失败",
      search: "搜索账号...",
      selectPlatform: "选择平台",
      title: "我的大模型账号",
      unknownError: "未知错误",
      updated: "账号已更新，等待管理员重新审核"
    },
    authorization: {
      title: "账号授权"
    },
    login: {
      subtitle: "登录后提交大模型账号，等待管理员审核",
      title: "贡献者登录"
    }
  },
  home: {
    modelSquare: {
      cta: "浏览全部模型与价格",
      menuEntry: "模型广场",
      subtitle: "浏览所有可用模型、API 端点和分组定价",
      title: "模型广场"
    }
  },
  modelQuote: {
    columns: {
      context: "上下文",
      discount: "平台折扣",
      model: "模型",
      officialCacheRead: "官方缓存读取",
      officialCacheWrite: "官方缓存写入",
      officialImageOrRequest: "官方图片/按次",
      officialInput: "官方输入价",
      officialOutput: "官方输出价",
      platform: "厂商",
      platformCacheRead: "平台缓存读取",
      platformCacheWrite: "平台缓存写入",
      platformImageOrRequest: "平台图片/按次",
      platformInput: "平台输入价",
      platformOutput: "平台输出价",
      type: "类型"
    },
    discountValue: "{fold} 折",
    empty: "没有符合条件的报价行",
    filters: {
      allPlatforms: "全部厂商",
      allTypes: "全部类型"
    },
    loadFailed: "报价加载失败：{msg}",
    price: {
      officialShort: "官方",
      platformShort: "平台"
    },
    resultCount: "显示 {count} / {total} 条报价",
    searchPlaceholder: "搜索模型、厂商或类型...",
    sort: {
      contextDesc: "上下文 长→短",
      discountAsc: "折扣最优",
      discountDesc: "折扣最小",
      modelAsc: "模型 A-Z",
      modelDesc: "模型 Z-A",
      platformAsc: "厂商 A-Z",
      priceAsc: "平台价 低→高",
      priceDesc: "平台价 高→低"
    },
    subtitle: "公开模型价格与 LiteLLM 官方原价对比",
    title: "模型报价单",
    units: {
      perMTok: "/M"
    },
    unmatchedOfficial: "-"
  },
  modelSquare: {
    capabilities: {
      assistant_prefill: "助手预填",
      audio_input: "音频输入",
      audio_output: "音频输出",
      computer_use: "电脑使用",
      function_calling: "函数调用",
      native_streaming: "原生流式",
      parallel_function_calling: "并行函数调用",
      parallel_tools: "并行工具",
      pdf_input: "PDF",
      prompt_caching: "提示缓存",
      reasoning: "推理",
      response_schema: "响应 Schema",
      service_tier: "服务层级",
      system_messages: "系统消息",
      tool_choice: "工具选择",
      url_context: "URL 上下文",
      video_input: "视频输入",
      vision: "多模态",
      web_search: "联网搜索"
    },
    categories: {
      anthropic: "Anthropic",
      antigravity: "Antigravity",
      audio: "语音",
      claude: "Claude",
      embedding: "向量嵌入",
      gemini: "Gemini",
      gpt: "OpenAI",
      image: "图像生成",
      openai: "OpenAI",
      other: "其它"
    },
    clearFilters: "清空筛选",
    detail: {
      basicInfo: "基本信息",
      cacheCreation1hPrice: "缓存创建 1h",
      cacheCreation5mPrice: "缓存创建 5m",
      cacheReadPrice: "缓存读取",
      cacheWritePrice: "缓存创建",
      callChain: "{group} 分组调用链路",
      category: "类别",
      contextRange: "上下文区间",
      contextWindow: "上下文窗口",
      discountValue: "{fold} 折",
      endpoints: "API 端点",
      endpointsHint: "该模型支持的入站接口与方法。",
      exclusive: "专属",
      groupPrices: "分组价格",
      groupPricesHint: "实际扣费以 API Key 绑定的分组价格为准。",
      imageOutputPrice: "图片输出",
      inputModalities: "输入模态",
      inputPrice: "输入价格",
      intervalPrices: "上下文区间定价",
      maxOutput: "最大输出",
      modelType: "模型类型",
      noDescription: "暂无模型描述",
      noPricing: "该平台暂未配置价格",
      outputModalities: "输出模态",
      outputPrice: "补全价格",
      perRequestPrice: "每次请求",
      priceItem: "项目",
      priceValue: "价格",
      rateMultiplier: "有效倍率",
      subscription: "订阅",
      tier: "阶梯"
    },
    empty: "没有符合条件的模型",
    featured: "推荐",
    filter: {
      capability: "能力",
      category: "厂商",
      priceRange: "价格区间"
    },
    fromPrice: "起步价",
    inputPriceShort: "输入",
    loadFailed: "加载失败：{msg}",
    modalities: {
      audio: "音频",
      image: "图片",
      pdf: "PDF",
      text: "文本",
      video: "视频"
    },
    modelTypes: {
      audio: "语音",
      audio_speech: "语音生成",
      audio_transcription: "语音转写",
      chat: "对话",
      completion: "补全",
      embedding: "向量",
      image: "图像",
      image_generation: "图像生成",
      other: "其它",
      responses: "Responses"
    },
    outputPriceShort: "输出",
    priceTier: {
      free: "免费",
      high: "高端 (>$5)",
      low: "低价 (<=$1/MTok)",
      mid: "中等 ($1-$5)"
    },
    resultCount: "显示 {count} / {total} 个模型",
    searchPlaceholder: "搜索模型名称、描述或厂商...",
    sort: {
      contextDesc: "上下文 长-短",
      featured: "推荐优先",
      label: "排序",
      nameAsc: "名称 A-Z",
      nameDesc: "名称 Z-A",
      priceAsc: "价格 低-高",
      priceDesc: "价格 高-低"
    },
    subtitle: "浏览所有可用模型、API 端点和分组定价",
    tabs: {
      all: "全部"
    },
    title: "模型广场",
    viewAll: "查看全部"
  },
  nav: {
    modelMetadata: "模型元数据",
    myModelAccounts: "我的大模型账号"
  },
  payment: {
    methods: {
      fuiou: "富友支付"
    }
  }
}
