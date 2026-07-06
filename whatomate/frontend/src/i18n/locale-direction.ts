const RTL_LOCALES = new Set<string>(['ar', 'fa', 'he', 'ur'])

export class LocaleDirectionManager {
  isRTL(locale: string): boolean {
    const normalized = String(locale || '').toLowerCase().split('-')[0]
    return RTL_LOCALES.has(normalized)
  }

  getDirection(locale: string): 'rtl' | 'ltr' {
    return this.isRTL(locale) ? 'rtl' : 'ltr'
  }

  applyLocaleDirection(locale: string): void {
    if (typeof document === 'undefined') {
      return
    }

    const direction = this.getDirection(locale)
    const root = document.documentElement

    root.setAttribute('lang', locale)
    root.setAttribute('dir', direction)
    document.body?.setAttribute('dir', direction)
  }
}

export const localeDirectionManager = new LocaleDirectionManager()
