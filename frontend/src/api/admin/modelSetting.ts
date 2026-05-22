import { apiClient } from '../client'

export interface ModelSettingSummary {
  total_rows: number
  loaded_rows: number
  duplicate_rows: number
  skipped_rows: number
  header_row_count: number
}

export interface ModelSettingStatus {
  file_path: string
  file_name: string
  source: string
  model_count: number
  updated_at: string
  summary: ModelSettingSummary
}

export async function getStatus(): Promise<ModelSettingStatus> {
  const { data } = await apiClient.get<ModelSettingStatus>('/admin/model_setting')
  return data
}

export async function upload(file: File): Promise<ModelSettingStatus> {
  const formData = new FormData()
  formData.append('file', file)
  const { data } = await apiClient.post<ModelSettingStatus>('/admin/model_setting', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return data
}

export const modelSettingAPI = { getStatus, upload }
export default modelSettingAPI
