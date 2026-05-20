// sudoapi: Model market.

import type { ModelGroupPrice, ModelOfficialPrice, ModelSquareCard } from '@/api/models'

export type ModelQuoteSortKey =
  | 'modelAsc'
  | 'modelDesc'
  | 'platformAsc'
  | 'priceAsc'
  | 'priceDesc'
  | 'discountAsc'
  | 'discountDesc'
  | 'contextDesc'

export interface ModelQuotePriceSet {
  input: number | null
  output: number | null
  cacheRead: number | null
  cacheWrite: number | null
  imageOutput: number | null
  imageOrRequest: number | null
}

export interface ModelQuoteRow {
  id: string
  model: string
  displayName: string
  vendor: string
  platform: string
  modelType: string
  contextWindow: number
  official: ModelQuotePriceSet
  platformPrice: ModelQuotePriceSet
  discountRatio: number | null
  discountBasis: keyof ModelQuotePriceSet | null
}

export interface ModelQuoteFilters {
  search: string
  platform: string
  modelType: string
}

const pricePriority: Array<keyof ModelQuotePriceSet> = [
  'input',
  'output',
  'cacheRead',
  'cacheWrite',
  'imageOutput',
  'imageOrRequest',
]

export function buildModelQuoteRows(cards: ModelSquareCard[]): ModelQuoteRow[] {
  const rows: ModelQuoteRow[] = []
  for (const card of cards) {
    for (const platform of card.platforms ?? []) {
      const platformPrice = lowestPlatformPrice(platform.group_prices ?? [])
      rows.push({
        id: `${card.name}::${platform.platform}`,
        model: card.name,
        displayName: card.display_name || card.name,
        vendor: card.category || 'other',
        platform: platform.platform,
        modelType: card.model_type || '',
        contextWindow: card.context_window || 0,
        official: officialPriceSet(card.official_price),
        platformPrice,
        ...calculateDiscount(officialPriceSet(card.official_price), platformPrice),
      })
    }
  }
  return rows
}

export function lowestPlatformPrice(groupPrices: ModelGroupPrice[]): ModelQuotePriceSet {
  const out = emptyPriceSet()
  for (const row of groupPrices) {
    const multiplier = finitePositive(row.base_rate_multiplier) ?? 1
    takeLowest(out, 'input', applyMultiplier(row.input_price_per_mtok_usd, multiplier))
    takeLowest(out, 'output', applyMultiplier(row.output_price_per_mtok_usd, multiplier))
    takeLowest(out, 'cacheRead', applyMultiplier(row.cache_read_price_per_mtok_usd, multiplier))
    takeLowest(out, 'cacheWrite', applyMultiplier(row.cache_write_price_per_mtok_usd, multiplier))
    takeLowest(out, 'imageOutput', applyMultiplier(row.image_output_price_per_mtok_usd, multiplier))
    takeLowest(out, 'imageOrRequest', applyMultiplier(row.per_request_price_usd, multiplier))

    for (const interval of row.intervals ?? []) {
      takeLowest(out, 'input', applyMultiplier(interval.input_price_per_mtok_usd, multiplier))
      takeLowest(out, 'output', applyMultiplier(interval.output_price_per_mtok_usd, multiplier))
      takeLowest(out, 'cacheRead', applyMultiplier(interval.cache_read_price_per_mtok_usd, multiplier))
      takeLowest(out, 'cacheWrite', applyMultiplier(interval.cache_write_price_per_mtok_usd, multiplier))
      takeLowest(out, 'imageOrRequest', applyMultiplier(interval.per_request_price_usd, multiplier))
    }
  }
  return out
}

export function calculateDiscount(
  official: ModelQuotePriceSet,
  platformPrice: ModelQuotePriceSet,
): Pick<ModelQuoteRow, 'discountRatio' | 'discountBasis'> {
  for (const key of pricePriority) {
    const officialValue = official[key]
    const platformValue = platformPrice[key]
    if (officialValue != null && officialValue > 0 && platformValue != null && platformValue >= 0) {
      return {
        discountRatio: platformValue / officialValue,
        discountBasis: key,
      }
    }
  }
  return { discountRatio: null, discountBasis: null }
}

export function filterModelQuoteRows(rows: ModelQuoteRow[], filters: ModelQuoteFilters): ModelQuoteRow[] {
  const query = filters.search.trim().toLowerCase()
  return rows.filter((row) => {
    if (filters.platform && row.vendor !== filters.platform) return false
    if (filters.modelType && row.modelType !== filters.modelType) return false
    if (!query) return true
    return [row.model, row.displayName, row.vendor, row.platform, row.modelType]
      .join(' ')
      .toLowerCase()
      .includes(query)
  })
}

export function sortModelQuoteRows(rows: ModelQuoteRow[], sortKey: ModelQuoteSortKey): ModelQuoteRow[] {
  const list = [...rows]
  list.sort((a, b) => {
    switch (sortKey) {
      case 'modelDesc':
        return compareText(b.model, a.model)
      case 'platformAsc':
        return compareText(a.vendor, b.vendor) || compareText(a.model, b.model)
      case 'priceAsc':
        return compareNullableNumbers(sortablePrice(a), sortablePrice(b)) || compareText(a.model, b.model)
      case 'priceDesc':
        return compareNullableNumbers(sortablePrice(b), sortablePrice(a)) || compareText(a.model, b.model)
      case 'discountAsc':
        return compareNullableNumbers(a.discountRatio, b.discountRatio) || compareText(a.model, b.model)
      case 'discountDesc':
        return compareNullableNumbers(b.discountRatio, a.discountRatio) || compareText(a.model, b.model)
      case 'contextDesc':
        return compareNullableNumbers(b.contextWindow, a.contextWindow) || compareText(a.model, b.model)
      case 'modelAsc':
      default:
        return compareText(a.model, b.model) || compareText(a.vendor, b.vendor) || compareText(a.platform, b.platform)
    }
  })
  return list
}

export function priceValueForBasis(row: ModelQuoteRow, source: 'official' | 'platform'): number | null {
  if (!row.discountBasis) return null
  const prices = source === 'official' ? row.official : row.platformPrice
  return prices[row.discountBasis]
}

function officialPriceSet(price: ModelOfficialPrice | null): ModelQuotePriceSet {
  return {
    input: price?.input_price_per_mtok_usd ?? null,
    output: price?.output_price_per_mtok_usd ?? null,
    cacheRead: price?.cache_read_price_per_mtok_usd ?? null,
    cacheWrite: price?.cache_write_price_per_mtok_usd ?? null,
    imageOutput: price?.image_output_price_per_mtok_usd ?? null,
    imageOrRequest: price?.image_price_usd ?? null,
  }
}

function emptyPriceSet(): ModelQuotePriceSet {
  return {
    input: null,
    output: null,
    cacheRead: null,
    cacheWrite: null,
    imageOutput: null,
    imageOrRequest: null,
  }
}

function applyMultiplier(value: number | null, multiplier: number): number | null {
  if (value == null || !Number.isFinite(value)) return null
  return value * multiplier
}

function takeLowest(set: ModelQuotePriceSet, key: keyof ModelQuotePriceSet, value: number | null) {
  if (value == null) return
  if (set[key] == null || value < set[key]!) {
    set[key] = value
  }
}

function finitePositive(value: number): number | null {
  return Number.isFinite(value) && value >= 0 ? value : null
}

function sortablePrice(row: ModelQuoteRow): number | null {
  for (const key of pricePriority) {
    const value = row.platformPrice[key]
    if (value != null) return value
  }
  return null
}

function compareNullableNumbers(a: number | null, b: number | null): number {
  const aMissing = a == null || !Number.isFinite(a)
  const bMissing = b == null || !Number.isFinite(b)
  if (aMissing && bMissing) return 0
  if (aMissing) return 1
  if (bMissing) return -1
  return a! - b!
}

function compareText(a: string, b: string): number {
  return a.localeCompare(b, undefined, { numeric: true, sensitivity: 'base' })
}
