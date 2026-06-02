// sudoapi: Model Square rate helpers.

import type { ModelGroupPrice } from '@/api/models'

/**
 * 用户专属分组倍率是覆盖值，不是叠加倍率。这里保持与网关计费路径一致：
 * user_rate_multiplier 非空时直接使用；否则使用分组默认倍率。
 */
export function effectiveModelRateMultiplier(
  row: Pick<ModelGroupPrice, 'base_rate_multiplier' | 'user_rate_multiplier'>,
): number {
  if (typeof row.user_rate_multiplier === 'number' && Number.isFinite(row.user_rate_multiplier)) {
    return row.user_rate_multiplier
  }
  if (Number.isFinite(row.base_rate_multiplier)) {
    return row.base_rate_multiplier
  }
  return 1
}
