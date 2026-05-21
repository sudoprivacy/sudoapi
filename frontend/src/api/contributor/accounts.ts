// sudoapi: Account contributor review workflow.

import { apiClient } from '../client'
import type { Account, CreateAccountRequest, PaginatedResponse, Proxy, UpdateAccountRequest } from '@/types'

export interface ContributorClaudeAuthURLRequest {
  proxy_id?: number | null
}

export interface ContributorClaudeAuthURLResponse {
  auth_url: string
  session_id: string
}

export interface ContributorClaudeExchangeCodeRequest {
  session_id: string
  code: string
  proxy_id?: number | null
}

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

export async function generateClaudeAuthUrl(
  payload: ContributorClaudeAuthURLRequest = {}
): Promise<ContributorClaudeAuthURLResponse> {
  const { data } = await apiClient.post<ContributorClaudeAuthURLResponse>(
    '/contributor/accounts/generate-auth-url',
    payload
  )
  return data
}

export async function generateClaudeSetupTokenUrl(
  payload: ContributorClaudeAuthURLRequest = {}
): Promise<ContributorClaudeAuthURLResponse> {
  const { data } = await apiClient.post<ContributorClaudeAuthURLResponse>(
    '/contributor/accounts/generate-setup-token-url',
    payload
  )
  return data
}

export async function exchangeClaudeCode(
  payload: ContributorClaudeExchangeCodeRequest
): Promise<Record<string, unknown>> {
  const { data } = await apiClient.post<Record<string, unknown>>(
    '/contributor/accounts/exchange-code',
    payload
  )
  return data
}

export async function exchangeClaudeSetupTokenCode(
  payload: ContributorClaudeExchangeCodeRequest
): Promise<Record<string, unknown>> {
  const { data } = await apiClient.post<Record<string, unknown>>(
    '/contributor/accounts/exchange-setup-token-code',
    payload
  )
  return data
}

export async function update(id: number, payload: UpdateAccountRequest): Promise<Account> {
  const { data } = await apiClient.put<Account>(`/contributor/accounts/${id}`, payload)
  return data
}

export async function testAccount(id: number): Promise<void> {
  await apiClient.post(`/contributor/accounts/${id}/test`)
}

export async function getProxies(): Promise<Proxy[]> {
  const { data } = await apiClient.get<Proxy[]>('/contributor/proxies/all')
  return data
}

export async function getProxiesForCountry(country?: string): Promise<Proxy[]> {
  const normalized = country?.trim().toUpperCase()
  const { data } = await apiClient.get<Proxy[]>('/contributor/proxies/all', {
    params: normalized ? { country: normalized } : undefined
  })
  return data
}

export async function releaseProxyReservation(): Promise<void> {
  await apiClient.post('/contributor/proxies/reservation/release')
}

export const contributorAccountsAPI = {
  list,
  getById,
  create,
  generateClaudeAuthUrl,
  generateClaudeSetupTokenUrl,
  exchangeClaudeCode,
  exchangeClaudeSetupTokenCode,
  update,
  testAccount,
  getProxies,
  getProxiesForCountry,
  releaseProxyReservation
}

export default contributorAccountsAPI
