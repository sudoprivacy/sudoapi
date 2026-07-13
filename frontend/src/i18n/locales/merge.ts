// sudoapi: Local i18n overlay.

type LocaleMessages = Record<string, any>

function isPlainMessageObject(value: unknown): value is LocaleMessages {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

export function mergeLocaleMessages<T extends LocaleMessages>(
  base: T,
  extension: LocaleMessages
): T {
  const target = base as LocaleMessages

  for (const [key, value] of Object.entries(extension)) {
    const current = target[key]

    if (isPlainMessageObject(current) && isPlainMessageObject(value)) {
      mergeLocaleMessages(current, value)
    } else {
      target[key] = value
    }
  }

  return base
}
