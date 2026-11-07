import { isStaffLanguage, matchLanguage, type Language } from './languages'

// The language a user picked before signing in is kept on the device (SPEC §3 "Picking and saving"). Storage
// can be missing or blocked; the app then follows the browser.
const key = 'salus.language'

export function loadSavedLanguage(): Language | null {
  try {
    const value = localStorage.getItem(key)
    return isStaffLanguage(value) ? value : null
  } catch {
    return null
  }
}

// saveLanguage keeps language on the device; null clears it, so the app follows the browser again.
export function saveLanguage(language: Language | null): void {
  try {
    if (language) localStorage.setItem(key, language)
    else localStorage.removeItem(key)
  } catch {
    // Without storage the choice lasts until the page is reloaded.
  }
}

// startLanguage is the language the app opens in: the one saved on the device, else the browser's.
export function startLanguage(browser: readonly string[]): Language {
  return loadSavedLanguage() ?? matchLanguage(browser)
}
