// sudoapi: Model catalog.

/**
 * Admin model catalog metadata API.
 *
 * These overrides only affect /models display metadata. Model availability,
 * platforms, groups, and pricing still come from channel configuration.
 */

import { apiClient } from '../client'

export interface ModelCatalogEndpoint {
  path: string
  method: 'GET' | 'POST'
}

export interface ModelCatalogEndpointConfig {
  platforms: Record<string, Record<string, ModelCatalogEndpoint[]>>
}

export interface ModelCatalogMetadataView {
  display_name: string
  description: string
  category: string
  model_type: string
  context_window: number
  max_output: number
  capabilities: string[]
  input_modalities: string[]
  output_modalities: string[]
  support_flags: string[]
  featured: boolean
  icon_url: string
}

export interface ModelCatalogMetadataOverride extends ModelCatalogMetadataView {
  id: number
  model_name: string
  created_at: string
  updated_at: string
}

export interface ModelCatalogMetadataListItem {
  model_name: string
  platforms: string[]
  metadata: ModelCatalogMetadataView
  override: ModelCatalogMetadataOverride | null
  missing_fields: string[]
}

export interface ListResponse {
  items: ModelCatalogMetadataListItem[]
}

export interface UpsertParams {
  model_name: string
  display_name?: string
  description?: string
  category?: string
  model_type?: string
  context_window?: number
  max_output?: number
  capabilities?: string[]
  input_modalities?: string[]
  output_modalities?: string[]
  support_flags?: string[]
  featured?: boolean
  icon_url?: string
}

export async function getEndpointConfig(): Promise<ModelCatalogEndpointConfig> {
  const { data } = await apiClient.get<ModelCatalogEndpointConfig>('/admin/model-catalog/endpoint-config')
  return data
}

export async function updateEndpointConfig(config: ModelCatalogEndpointConfig): Promise<ModelCatalogEndpointConfig> {
  const { data } = await apiClient.put<ModelCatalogEndpointConfig>('/admin/model-catalog/endpoint-config', config)
  return data
}

export async function listMetadata(): Promise<ListResponse> {
  const { data } = await apiClient.get<ListResponse>('/admin/model-catalog/metadata')
  return data
}

export async function upsertMetadata(params: UpsertParams): Promise<ModelCatalogMetadataOverride> {
  const { data } = await apiClient.post<ModelCatalogMetadataOverride>('/admin/model-catalog/metadata', params)
  return data
}

export async function removeMetadata(modelName: string): Promise<void> {
  await apiClient.delete('/admin/model-catalog/metadata', {
    params: { model_name: modelName },
  })
}

const modelCatalogAPI = {
  getEndpointConfig,
  updateEndpointConfig,
  listMetadata,
  upsertMetadata,
  removeMetadata,
}
export default modelCatalogAPI
