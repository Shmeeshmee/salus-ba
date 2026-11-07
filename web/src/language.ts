import { matchLanguage, type Language } from './i18n/languages'
import { loadSavedLanguage } from './i18n/saved'

// languageNames is each staff language by its own name (docs/i18n/COPY.md L1). The names are the same in every UI
// language, so they are not in the catalogs.
export const languageNames: Record<Language, string> = {
  bs: 'Bosanski',
  hr: 'Hrvatski',
  'sr-Latn': 'Srpski (latinica)',
  'sr-Cyrl': 'Српски (ћирилица)',
  en: 'English',
}

// browserLanguages is the browser's languages in its order.
export function browserLanguages(): readonly string[] {
  return navigator.languages.length > 0 ? navigator.languages : [navigator.language]
}

// signInLanguages is what a sign-in or an invite acceptance sends (SPEC §3 "Picking and saving"): the language
// matched from the browser, and the one picked on the device before signing in, if any, which the account saves.
export function signInLanguages(browser: readonly string[] = browserLanguages()): { language?: Language; detectedLanguage: Language } {
  const language = loadSavedLanguage()
  const detectedLanguage = matchLanguage(browser)
  return language ? { language, detectedLanguage } : { detectedLanguage }
}
