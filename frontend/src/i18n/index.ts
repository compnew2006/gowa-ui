import { createI18n } from 'vue-i18n'
import en from './locales/en.json'

export type MessageSchema = typeof en

// RTL languages (written right-to-left). Used by setLocale to flip the
// document direction so Arabic/Urdu render correctly. Any locale not listed
// here defaults to LTR.
const RTL_LOCALES = new Set(['ar', 'ur', 'he', 'fa'])

// Auto-discover available locales from the locales folder
// Vite imports all JSON files at build time
// Using import: 'default' to get JSON content directly (required for Vite 5+)
const localeModules = import.meta.glob('./locales/*.json', { eager: true, import: 'default' }) as Record<string, MessageSchema>

// Build supported locales list from available files
const localeNames: Record<string, { name: string; nativeName: string }> = {
  en: { name: 'English', nativeName: 'English' },
  es: { name: 'Spanish', nativeName: 'Español' },
  fr: { name: 'French', nativeName: 'Français' },
  de: { name: 'German', nativeName: 'Deutsch' },
  hi: { name: 'Hindi', nativeName: 'हिन्दी' },
  pt: { name: 'Portuguese', nativeName: 'Português' },
  zh: { name: 'Chinese', nativeName: '中文' },
  ja: { name: 'Japanese', nativeName: '日本語' },
  ko: { name: 'Korean', nativeName: '한국어' },
  ar: { name: 'Arabic', nativeName: 'العربية' },
  ru: { name: 'Russian', nativeName: 'Русский' },
  it: { name: 'Italian', nativeName: 'Italiano' },
  nl: { name: 'Dutch', nativeName: 'Nederlands' },
  tr: { name: 'Turkish', nativeName: 'Türkçe' },
  vi: { name: 'Vietnamese', nativeName: 'Tiếng Việt' },
  th: { name: 'Thai', nativeName: 'ไทย' },
  id: { name: 'Indonesian', nativeName: 'Bahasa Indonesia' },
  ms: { name: 'Malay', nativeName: 'Bahasa Melayu' },
  pl: { name: 'Polish', nativeName: 'Polski' },
  uk: { name: 'Ukrainian', nativeName: 'Українська' },
  ta: { name: 'Tamil', nativeName: 'தமிழ்' },
  bn: { name: 'Bengali', nativeName: 'বাংলা' },
  ur: { name: 'Urdu', nativeName: 'اُردُو' },

}

// Auto-generate SUPPORTED_LOCALES from available files
export const SUPPORTED_LOCALES = Object.keys(localeModules).map(path => {
  const code = path.replace('./locales/', '').replace('.json', '')
  const names = localeNames[code] || { name: code, nativeName: code }
  return { code, ...names }
})

// Build messages object from all locale files
const messages: Record<string, MessageSchema> = {}
for (const path in localeModules) {
  const code = path.replace('./locales/', '').replace('.json', '')
  messages[code] = localeModules[path]
}

// Get saved locale or detect from browser
function getDefaultLocale(): string {
  // Check localStorage first
  const saved = localStorage.getItem('locale')
  if (saved && messages[saved]) {
    return saved
  }

  // Detect from browser
  const browserLang = navigator.language.split('-')[0]
  if (messages[browserLang]) {
    return browserLang
  }

  return 'en'
}

export const i18n = createI18n({
  legacy: false, // Use Composition API
  locale: getDefaultLocale(),
  fallbackLocale: 'en',
  messages,
})

// Apply the document direction on first load so an RTL locale saved in
// localStorage (e.g. Arabic/Urdu) renders correctly before any switch.
document.documentElement.setAttribute('dir', RTL_LOCALES.has(i18n.global.locale.value) ? 'rtl' : 'ltr')

// Helper to change locale
export function setLocale(locale: string) {
  if (!messages[locale]) {
    console.warn(`Locale '${locale}' not available`)
    return
  }
  i18n.global.locale.value = locale
  localStorage.setItem('locale', locale)
  document.documentElement.setAttribute('lang', locale)
  // Flip the document direction for RTL languages (Arabic, Urdu, ...).
  // Logical CSS utilities (ms-, me-, start, end) follow this automatically.
  document.documentElement.setAttribute('dir', RTL_LOCALES.has(locale) ? 'rtl' : 'ltr')
}
