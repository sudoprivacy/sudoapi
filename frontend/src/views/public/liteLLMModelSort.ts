import type { LiteLLMModel } from '@/api/models'

export type LiteLLMSortKey =
  | 'latestDesc'
  | 'nameAsc'
  | 'nameDesc'
  | 'providerAsc'
  | 'modeAsc'
  | 'inputAsc'
  | 'inputDesc'
  | 'outputAsc'
  | 'outputDesc'
  | 'contextDesc'

export function sortLiteLLMModels(rows: LiteLLMModel[], key: LiteLLMSortKey): LiteLLMModel[] {
  const sorted = [...rows]
  const byText = (a: string, b: string) => a.localeCompare(b, undefined, { sensitivity: 'base', numeric: true })
  const byNumber = (a: number | null, b: number | null, dir: 'asc' | 'desc') => {
    const av = Number.isFinite(a) && a != null ? a : Number.POSITIVE_INFINITY
    const bv = Number.isFinite(b) && b != null ? b : Number.POSITIVE_INFINITY
    return dir === 'asc' ? av - bv : bv - av
  }
  sorted.sort((a, b) => {
    switch (key) {
      case 'latestDesc':
        return compareSerialNumberDesc(a, b) || byText(a.name, b.name)
      case 'nameDesc':
        return byText(b.name, a.name)
      case 'providerAsc':
        return byText(a.provider || a.category, b.provider || b.category) || byText(a.name, b.name)
      case 'modeAsc':
        return byText(a.mode, b.mode) || byText(a.name, b.name)
      case 'inputAsc':
        return byNumber(a.input_price_per_mtok_usd, b.input_price_per_mtok_usd, 'asc') || byText(a.name, b.name)
      case 'inputDesc':
        return byNumber(a.input_price_per_mtok_usd, b.input_price_per_mtok_usd, 'desc') || byText(a.name, b.name)
      case 'outputAsc':
        return byNumber(a.output_price_per_mtok_usd, b.output_price_per_mtok_usd, 'asc') || byText(a.name, b.name)
      case 'outputDesc':
        return byNumber(a.output_price_per_mtok_usd, b.output_price_per_mtok_usd, 'desc') || byText(a.name, b.name)
      case 'contextDesc':
        return contextWindow(b) - contextWindow(a) || byText(a.name, b.name)
      case 'nameAsc':
      default:
        return byText(a.name, b.name)
    }
  })
  return sorted
}

export function contextWindow(model: LiteLLMModel): number {
  return model.max_input_tokens || model.max_tokens || 0
}

function compareSerialNumberDesc(a: LiteLLMModel, b: LiteLLMModel): number {
  const av = serialRank(a.serial_number)
  const bv = serialRank(b.serial_number)
  if (av === bv) return 0
  if (av === Number.NEGATIVE_INFINITY) return 1
  if (bv === Number.NEGATIVE_INFINITY) return -1
  return bv - av
}

function serialRank(value: LiteLLMModel['serial_number']): number {
  if (value == null) return Number.NEGATIVE_INFINITY
  const numeric = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(numeric) ? numeric : Number.NEGATIVE_INFINITY
}
