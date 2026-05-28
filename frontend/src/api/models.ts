// sudoapi: Model Square model catalog.

/**
 * Model Square API client.
 *
 * 两个入口对应后端两个 scope：
 *   - listPublicModels：未登录可见，仅 standard 非专属分组定价（GET /public/models）
 *   - listMyModels    ：登录后调用，叠加用户可访问的专属/订阅分组（GET /models）
 *
 * 价格字段全部归一为「USD per 1M tokens」（后端已乘 1e6），前端无需再换算单位。
 * per_request / image 模式时按「USD per call」展示，对应 per_request_price_usd 字段。
 */

import { apiClient } from './client'

/** 模型对外暴露的入站端点（来自后端 InboundEndpointsForPlatform）。 */
export interface ModelEndpoint {
  path: string
  method: string
}

export interface ModelPriceInterval {
  min_tokens: number
  max_tokens: number | null
  tier_label: string
  input_price_per_mtok_usd: number | null
  output_price_per_mtok_usd: number | null
  cache_read_price_per_mtok_usd: number | null
  cache_write_price_per_mtok_usd: number | null
  per_request_price_usd: number | null
  sort_order: number
  cache_creation_5m_price_per_mtok_usd: number | null
  cache_creation_1h_price_per_mtok_usd: number | null
}

/** 单个分组下的定价行。
 *
 *  调用方展示有效倍率：
 *    effective = base_rate_multiplier × (user_rate_multiplier ?? 1)
 *  effective × *_per_mtok_usd = 实际单价。
 */
export interface ModelGroupPrice {
  group_id: number
  group_name: string
  subscription_type: string
  is_exclusive: boolean
  base_rate_multiplier: number
  /** 仅登录态且用户在 /groups/rates 上有专属倍率时填值，否则 null。 */
  user_rate_multiplier: number | null
  /** 'token' | 'per_request' | 'image'，对应 backend BillingMode。 */
  billing_mode: string
  input_price_per_mtok_usd: number | null
  output_price_per_mtok_usd: number | null
  cache_read_price_per_mtok_usd: number | null
  cache_write_price_per_mtok_usd: number | null
  image_output_price_per_mtok_usd: number | null
  /** per_request / image 模式：每次调用价格（USD）。 */
  per_request_price_usd: number | null
  /** 上下文区间定价；token 模式为 USD/MTok，按次/图片模式为 USD/call。 */
  intervals: ModelPriceInterval[]
  /** 同模型同分组在多个渠道下都有定价时的渠道名链路（已按字典序去重排序）。 */
  channel_chain: string[]
  cache_creation_5m_price_per_mtok_usd: number | null
  cache_creation_1h_price_per_mtok_usd: number | null
}

export interface ModelPlatformSection {
  platform: string
  endpoints: ModelEndpoint[]
  group_prices: ModelGroupPrice[]
}

export interface ModelOfficialPrice {
  input_price_per_mtok_usd: number | null
  output_price_per_mtok_usd: number | null
  cache_read_price_per_mtok_usd: number | null
  cache_write_price_per_mtok_usd: number | null
  image_output_price_per_mtok_usd: number | null
  image_price_usd: number | null
}

/** 模型广场卡片。capabilities 是 i18n key（vision/function_calling/reasoning/...）。 */
export interface ModelSquareCard {
  name: string
  display_name: string
  /** claude / gpt / gemini / image / embedding / audio / other */
  category: string
  description: string
  /** LiteLLM mode, e.g. chat / responses / embedding / image_generation. */
  model_type: string
  context_window: number
  max_output: number
  input_modalities: string[]
  output_modalities: string[]
  support_flags: string[]
  capabilities: string[]
  featured: boolean
  icon_url: string | null
  official_price: ModelOfficialPrice | null
  platforms: ModelPlatformSection[]
}

/** 公开入口：未登录也可调用。 */
export async function listPublicModels(options?: {
  signal?: AbortSignal
}): Promise<ModelSquareCard[]> {
  const { data } = await apiClient.get<ModelSquareCard[]>('/public/models', {
    signal: options?.signal,
  })
  return data
}

/** 已登录入口：叠加用户可访问的专属/订阅分组定价。 */
export async function listMyModels(options?: {
  signal?: AbortSignal
}): Promise<ModelSquareCard[]> {
  const { data } = await apiClient.get<ModelSquareCard[]>('/models', {
    signal: options?.signal,
  })
  return data
}

export const modelSquareAPI = {
  listPublicModels,
  listMyModels,
}

export default modelSquareAPI
