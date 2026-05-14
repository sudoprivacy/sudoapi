import { apiClient } from '../client'
import type { Account, CreateAccountRequest, PaginatedResponse, UpdateAccountRequest } from '@/types'

export async function list(
  page = 1,
  pageSize = 20,
  filters?: {
    platform?: string
    type?: string
    status?: string
    search?: string
    sort_by?: string
    sort_order?: 'asc' | 'desc'
  }
): Promise<PaginatedResponse<Account>> {
  const { data } = await apiClient.get<PaginatedResponse<Account>>('/contributor/accounts', {
    params: { page, page_size: pageSize, ...filters }
  })
  return data
}

export async function getById(id: number): Promise<Account> {
  const { data } = await apiClient.get<Account>(`/contributor/accounts/${id}`)
  return data
}

export async function create(payload: CreateAccountRequest): Promise<Account> {
  const { data } = await apiClient.post<Account>('/contributor/accounts', payload)
  return data
}

export async function update(id: number, payload: UpdateAccountRequest): Promise<Account> {
  const { data } = await apiClient.put<Account>(`/contributor/accounts/${id}`, payload)
  return data
}

export async function testAccount(id: number): Promise<void> {
  await apiClient.post(`/contributor/accounts/${id}/test`)
}

export const contributorAccountsAPI = {
  list,
  getById,
  create,
  update,
  testAccount
}

export default contributorAccountsAPI
