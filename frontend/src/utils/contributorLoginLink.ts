// sudoapi: Account contributor review workflow.

export function normalizeContributorCountryCode(countryCode: string | null | undefined): string {
  return (countryCode || '').trim().toUpperCase()
}

export function buildContributorLoginLink(countryCode: string | null | undefined, origin: string): string | null {
  const normalizedCountry = normalizeContributorCountryCode(countryCode)
  if (!normalizedCountry) return null

  const url = new URL('/contributor/login', origin)
  url.searchParams.set('country', normalizedCountry)
  return url.toString()
}
