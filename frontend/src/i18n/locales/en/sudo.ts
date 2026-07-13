// sudoapi: Local i18n overlay.

export default {
  admin: {
    accounts: {
      approve: "Approve",
      columns: {
        reviewStatus: "Review"
      },
      dataImportCompletedWithErrors: "Import completed with errors: account skipped {account_skipped}, account failed {account_failed}, proxy failed {proxy_failed}",
      dataImportResultSummary: "Proxies created {proxy_created}, reused {proxy_reused}, failed {proxy_failed}; Accounts created {account_created}, skipped {account_skipped}, failed {account_failed}",
      dataImportSuccess: "Import completed: accounts created {account_created}, skipped {account_skipped}, failed {account_failed}",
      enterCustomModelNames: "Enter custom model name, split spaces between multiple models",
      externalSubmission: "External submitter",
      failedToSave: "Failed to save account",
      oauth: {
        antigravity: {
          refreshTokenLabel: "Refresh Token"
        },
        copyUrl: "Copy URL",
        openai: {
          accessTokenAuth: "Manual AT Input",
          mobileRefreshTokenAuth: "Manual Mobile RT Input",
          refreshTokenLabel: "Refresh Token"
        },
        urlCopied: "URL copied to clipboard"
      },
      openai: {
        codexImageGenerationBridge: "Codex image-generation bridge",
        codexImageGenerationBridgeBadgeDisabled: "Account off",
        codexImageGenerationBridgeBadgeEnabled: "Account on",
        codexImageGenerationBridgeBadgeInherit: "Channel policy",
        codexImageGenerationBridgeDesc: "Account policy takes precedence over channel and global settings. Only controls whether Codex requests through the /responses text endpoint receive the image_generation tool; standalone image-generation endpoints are unaffected.",
        codexImageGenerationBridgeDisabled: "Force off",
        codexImageGenerationBridgeDisabledDesc: "Block image tool injection for Codex /responses requests.",
        codexImageGenerationBridgeEnabled: "Force on",
        codexImageGenerationBridgeEnabledDesc: "Allow image tool injection for Codex /responses requests.",
        codexImageGenerationBridgeInherit: "Follow channel",
        codexImageGenerationBridgeInheritDesc: "Do not write an account override; use the channel or global policy."
      },
      reject: "Reject",
      review: {
        approved: "Approved",
        pending: "Pending",
        rejected: "Rejected"
      },
      reviewApproved: "Account approved",
      reviewFilters: {
        approved: "Approved",
        external: "External Submissions",
        pending: "Pending Review",
        rejected: "Rejected"
      },
      reviewRejected: "Account rejected",
      reviewUpdateFailed: "Failed to update review status",
      testSuccess: "Account tested successfully"
    },
    channels: {
      endpointConfig: {
        addEndpoint: "Add endpoint",
        addModelType: "Add model type",
        description: "Rules are global per platform. Unmatched model types show no endpoints.",
        invalidKey: "Platform and model type may only contain lowercase letters, numbers, underscores, and hyphens.",
        invalidMethod: "Endpoint method must be GET or POST.",
        invalidPath: "Endpoint path must start with / and contain no spaces.",
        loadError: "Failed to load endpoint config",
        modelTypePlaceholder: "model_type, e.g. chat",
        noRules: "No endpoint rules. Unmatched models will not show endpoints.",
        platformRules: "Model type endpoint rules",
        saveError: "Failed to save endpoint config",
        saveSuccess: "Endpoint config saved",
        title: "Endpoint Config"
      },
      form: {
        cacheCreation1hPrice: "Cache Creation 1h",
        cacheCreation5mPrice: "Cache Creation 5m",
        cacheWriteFallback: "Fallback to Cache Write Price"
      },
      intervalValidation: {
        price: {
          cacheCreation1hPrice: "cache creation 1h price",
          cacheCreation5mPrice: "cache creation 5m price"
        }
      }
    },
    modelMetadata: {
      clear: "Clear override",
      clearConfirm: "Clear metadata override for \"{name}\"?",
      deleteError: "Failed to clear model metadata",
      deleteSuccess: "Model metadata override cleared",
      description: "Fill missing display metadata used by the /models page",
      edit: "Edit metadata",
      fields: {
        actions: "Actions",
        capabilities: "Capabilities",
        category: "Vendor",
        contextWindow: "Context",
        description: "Description",
        displayName: "Display name",
        featured: "Featured",
        iconUrl: "Icon URL",
        inputModalities: "Input support",
        maxOutput: "Max output",
        missing: "Missing",
        modelName: "Model",
        modelType: "Model type",
        outputModalities: "Output support",
        platforms: "Platforms",
        supportFlags: "Model tags"
      },
      form: {
        capabilitiesHint: "Selected capabilities override automatic inference. Leave empty to keep automatic data.",
        categoryPlaceholder: "Enter or select vendor",
        contextWindowPlaceholder: "e.g. 200000",
        descriptionPlaceholder: "Shown on /models cards and detail drawer",
        displayNamePlaceholder: "Defaults to model name",
        featuredHint: "Featured models are sorted first on /models.",
        iconUrlPlaceholder: "https://...",
        maxOutputPlaceholder: "e.g. 8192",
        modalitiesHint: "Leave empty to keep LiteLLM automatic data.",
        modelTypePlaceholder: "e.g. chat / image_generation",
        supportFlagsHint: "Model tags come from all true LiteLLM supports_* fields. Leave empty to keep automatic data."
      },
      loadError: "Failed to load model metadata",
      missingCount: "{count} missing",
      missingFields: {
        capabilities: "Capabilities",
        category: "Vendor",
        context_window: "Context",
        description: "Description",
        display_name: "Display name",
        icon_url: "Icon",
        input_modalities: "Input support",
        max_output: "Max output",
        model_type: "Model type",
        output_modalities: "Output support",
        support_flags: "Model tags"
      },
      missingOnly: "Missing only",
      noModels: "No maintainable models",
      noModelsDesc: "No displayable models are currently available from active channels",
      noResults: "No matching models",
      overrideActive: "Overridden",
      refresh: "Refresh",
      saveError: "Failed to save model metadata",
      saveSuccess: "Model metadata saved",
      searchPlaceholder: "Search models, platforms, or descriptions...",
      title: "Model Metadata"
    },
    modelSetting: {
      currentStatus: "Current Status",
      description: "Upload a CSV to control /model visibility and newest-first ordering.",
      duplicateRows: "Duplicate Rows",
      fileName: "File Name",
      filePath: "File Path",
      loadedRows: "Loaded Rows",
      loadFailed: "Failed to load model whitelist status",
      modelCount: "Whitelisted Models",
      parseSummary: "Parse Summary",
      skippedRows: "Skipped Blank Rows",
      source: "Source",
      title: "Model Whitelist Settings",
      totalRows: "Data Rows",
      updatedAt: "Updated At",
      upload: "Upload and Hot Reload",
      uploadFailed: "Upload failed",
      uploadHint: "CSV must contain serial_number and id columns. A successful upload replaces data/model_setting/models_grouped_id_desc.csv and takes effect immediately.",
      uploading: "Uploading...",
      uploadLabel: "Whitelist CSV",
      uploadSuccess: "Upload complete. Loaded {count} models."
    },
    proxies: {
      contributorLoginLinkCopied: "Contributor link copied",
      copyContributorLoginLink: "Contributor link",
      copyContributorLoginLinkTitle: "Copy contributor authorization link",
      countryCodeRequiredForContributorLink: "Detect country code first"
    },
    settings: {
      payment: {
        field_fuiouApiBaseHint: "Use https://hlwnets.fuioupay.com for production and https://hlwnets-test.fuioupay.com for sandbox/test. The base must match the environment your API keys were issued for.",
        field_fuiouPublicKey: "Fuiou Public Key",
        field_mchntCd: "Merchant ID (mchnt_cd)",
        field_merchantPrivateKey: "Merchant Private Key",
        fuiouGuideNote: "The merchant private key must be PKCS8 + Base64; the Fuiou public key is the PKIX + Base64 key downloaded from the Fuiou merchant portal. mchnt_cd is also used to route async callbacks, so it must match your Fuiou contract / portal value exactly.",
        fuiouGuideSummary: "Fuiou aggregate payment: accept Alipay and WeChat Pay through a single merchant account.",
        providerFuiou: "Fuiou Pay"
      }
    },
    users: {
      batch: {
        allFailed: "All rows failed. Check the error column.",
        columnsHint: "Supports comma / semicolon / tab. Header row is optional. Lines starting with # are ignored.",
        columnsTitle: "CSV column order (fixed)",
        errorCodes: {
          CREATE_FAILED: "Create failed",
          DUPLICATE_IN_PAYLOAD: "Email duplicates another row in this batch",
          EMAIL_EXISTS: "Email already exists",
          INVALID_BALANCE: "Balance must be a non-negative number",
          INVALID_CONCURRENCY: "Concurrency must be a non-negative integer",
          INVALID_EMAIL: "Invalid email format",
          INVALID_RPM: "RPM must be a non-negative integer",
          WEAK_PASSWORD: "Password must be at least 6 characters"
        },
        errorMsg: "Error",
        headerSkipped: "header row skipped",
        menuLabel: "Batch add users",
        pasteLabel: "Or paste CSV content here",
        previewTitle: "Preview",
        resultAborted: "aborted after first failure",
        resultSummary: "{total} rows: {created} created, {failed} failed",
        resultTitle: "Result",
        securityNotice: "Use a trusted network. The CSV contains plaintext passwords; they are not written to logs.",
        skipOnError: "Skip invalid rows and submit only valid ones",
        statsDup: "duplicates",
        statsError: "errors",
        statsValid: "valid",
        status: "Status",
        submitFailed: "Batch create failed",
        submitting: "Submitting...",
        submitWithCount: "Submit {count} rows",
        title: "Batch create users",
        uploadCsv: "Upload CSV"
      }
    }
  },
  contributor: {
    accounts: {
      add: "Add Account",
      authorizationMethod: "Authorization Method",
      authSubmitted: "{platform} account authorization submitted",
      authSubmittedReview: "{platform} account authorization submitted, pending admin review.",
      columns: {
        created: "Created"
      },
      created: "Account submitted for admin review",
      description: "Add and maintain model accounts you submitted. They are scheduled only after admin approval.",
      edit: "Edit Account",
      generateAuthUrlFailed: "Failed to generate authorization URL",
      loadFailed: "Failed to load accounts",
      loadProxiesFailed: "Failed to load proxies",
      missingOAuthState: "Missing OAuth state",
      openaiPartialSubmitted: "OpenAI account authorization partially submitted: {success} succeeded, {failed} failed",
      openaiRefreshTokenFailed: "OpenAI Refresh Token validation failed",
      openaiRefreshTokenRequired: "Please enter Refresh Token",
      openaiSubmittedCount: "{count} OpenAI account authorizations submitted",
      proxyLoading: "Loading proxy...",
      proxyUnavailable: "No proxy is currently available, so authorization cannot continue.",
      reauthorize: "Reauthorize",
      resetProxyReservationFailed: "Failed to reset proxy reservation",
      review: {
        approved: "Approved",
        pending: "Pending",
        rejected: "Rejected"
      },
      reviewStatus: "Review",
      saveFailed: "Failed to save account",
      search: "Search accounts...",
      selectPlatform: "Select Platform",
      title: "My Model Accounts",
      unknownError: "Unknown error",
      updated: "Account updated and returned to review"
    },
    authorization: {
      title: "Account Authorization"
    },
    login: {
      subtitle: "Sign in to submit model accounts for admin review",
      title: "Contributor Login"
    }
  },
  home: {
    modelSquare: {
      cta: "Browse all models & pricing",
      menuEntry: "Models",
      subtitle: "Browse every available model, API endpoint, and group pricing",
      title: "Model Square"
    }
  },
  modelQuote: {
    columns: {
      context: "Context",
      discount: "Discount",
      model: "Model",
      officialCacheRead: "Official cache read",
      officialCacheWrite: "Official cache write",
      officialImageOrRequest: "Official image/request",
      officialInput: "Official input",
      officialOutput: "Official output",
      platform: "Vendor",
      platformCacheRead: "Platform cache read",
      platformCacheWrite: "Platform cache write",
      platformImageOrRequest: "Platform image/request",
      platformInput: "Platform input",
      platformOutput: "Platform output",
      type: "Type"
    },
    discountValue: "{fold} off",
    empty: "No quote rows match your filters",
    filters: {
      allPlatforms: "All vendors",
      allTypes: "All types"
    },
    loadFailed: "Failed to load quote: {msg}",
    price: {
      officialShort: "Official",
      platformShort: "Platform"
    },
    resultCount: "Showing {count} of {total} quote rows",
    searchPlaceholder: "Search models, vendors, or types...",
    sort: {
      contextDesc: "Context long-short",
      discountAsc: "Best discount",
      discountDesc: "Smallest discount",
      modelAsc: "Model A-Z",
      modelDesc: "Model Z-A",
      platformAsc: "Vendor A-Z",
      priceAsc: "Platform price low-high",
      priceDesc: "Platform price high-low"
    },
    subtitle: "Public model prices compared with LiteLLM reference pricing",
    title: "Model Quote",
    units: {
      perMTok: "/M"
    },
    unmatchedOfficial: "-"
  },
  modelSquare: {
    capabilities: {
      assistant_prefill: "Assistant prefill",
      audio_input: "Audio input",
      audio_output: "Audio output",
      computer_use: "Computer use",
      function_calling: "Function calling",
      native_streaming: "Native streaming",
      parallel_function_calling: "Parallel function calling",
      parallel_tools: "Parallel tools",
      pdf_input: "PDF",
      prompt_caching: "Prompt caching",
      reasoning: "Reasoning",
      response_schema: "Response schema",
      service_tier: "Service tier",
      system_messages: "System messages",
      tool_choice: "Tool choice",
      url_context: "URL context",
      video_input: "Video input",
      vision: "Vision",
      web_search: "Web search"
    },
    categories: {
      anthropic: "Anthropic",
      antigravity: "Antigravity",
      audio: "Audio",
      claude: "Claude",
      embedding: "Embedding",
      gemini: "Gemini",
      gpt: "OpenAI",
      image: "Image generation",
      openai: "OpenAI",
      other: "Other"
    },
    clearFilters: "Clear filters",
    detail: {
      basicInfo: "Basic info",
      cacheCreation1hPrice: "Cache write 1h",
      cacheCreation5mPrice: "Cache write 5m",
      cacheReadPrice: "Cache read",
      cacheWritePrice: "Cache write",
      callChain: "{group} call chain",
      category: "Category",
      contextRange: "Context range",
      contextWindow: "Context window",
      discountValue: "{fold} off",
      endpoints: "API endpoints",
      endpointsHint: "Inbound paths and HTTP methods this model accepts.",
      exclusive: "Exclusive",
      groupPrices: "Group pricing",
      groupPricesHint: "Billing uses the pricing of the group bound to your API key.",
      imageOutputPrice: "Image output",
      inputModalities: "Input modalities",
      inputPrice: "Input",
      intervalPrices: "Context interval pricing",
      maxOutput: "Max output",
      modelType: "Model type",
      noDescription: "No description available",
      noPricing: "No pricing configured on this platform",
      outputModalities: "Output modalities",
      outputPrice: "Output",
      perRequestPrice: "Per request",
      priceItem: "Item",
      priceValue: "Price",
      rateMultiplier: "Effective multiplier",
      subscription: "Subscription",
      tier: "Tier"
    },
    empty: "No models match your filters",
    featured: "Featured",
    filter: {
      capability: "Capability",
      category: "Vendor",
      priceRange: "Price"
    },
    fromPrice: "From",
    inputPriceShort: "Input",
    loadFailed: "Failed to load: {msg}",
    modalities: {
      audio: "Audio",
      image: "Image",
      pdf: "PDF",
      text: "Text",
      video: "Video"
    },
    modelTypes: {
      audio: "Audio",
      audio_speech: "Audio speech",
      audio_transcription: "Audio transcription",
      chat: "Chat",
      completion: "Completion",
      embedding: "Embedding",
      image: "Image",
      image_generation: "Image generation",
      other: "Other",
      responses: "Responses"
    },
    outputPriceShort: "Output",
    priceTier: {
      free: "Free",
      high: "Premium (>$5)",
      low: "Low (<=$1/MTok)",
      mid: "Mid ($1-$5)"
    },
    resultCount: "Showing {count} of {total} models",
    searchPlaceholder: "Search by model name, description, or vendor...",
    sort: {
      contextDesc: "Context Long-Short",
      featured: "Featured first",
      label: "Sort",
      nameAsc: "Name A-Z",
      nameDesc: "Name Z-A",
      priceAsc: "Price Low-High",
      priceDesc: "Price High-Low"
    },
    subtitle: "Browse every available model, API endpoint, and group pricing",
    tabs: {
      all: "All"
    },
    title: "Model Square",
    viewAll: "View all"
  },
  nav: {
    modelMetadata: "Model Metadata",
    myModelAccounts: "My Model Accounts"
  },
  payment: {
    methods: {
      fuiou: "Fuiou Pay"
    }
  }
}
