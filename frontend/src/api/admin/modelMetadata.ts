/**
 * Admin model metadata API.
 *
 * These overrides only affect /models display metadata. Model availability,
 * platforms, groups, and pricing still come from channel configuration.
 */

import { apiClient } from '../client'

export interface ModelMetadataView {
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

export interface ModelMetadataOverride extends ModelMetadataView {
  id: number
  model_name: string
  created_at: string
  updated_at: string
}

export interface ModelMetadataListItem {
  model_name: string
  platforms: string[]
  metadata: ModelMetadataView
  override: ModelMetadataOverride | null
  missing_fields: string[]
}

export interface ListResponse {
  items: ModelMetadataListItem[]
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

export async function list(): Promise<ListResponse> {
  const { data } = await apiClient.get<ListResponse>('/admin/channels/model-metadata')
  return data
}

export async function upsert(params: UpsertParams): Promise<ModelMetadataOverride> {
  const { data } = await apiClient.post<ModelMetadataOverride>(
    '/admin/channels/model-metadata',
    params,
  )
  return data
}

export async function remove(modelName: string): Promise<void> {
  await apiClient.delete('/admin/channels/model-metadata', {
    params: { model_name: modelName },
  })
}

export const modelMetadataAPI = { list, upsert, remove }
export default modelMetadataAPI
