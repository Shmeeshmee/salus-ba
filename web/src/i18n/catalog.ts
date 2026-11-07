import type { Language } from './languages'

// The catalogs (SPEC §3 "Catalogs"): web/src/i18n/<area>/<language>.json, English the source. A message is an
// ICU string, or an object with the string as defaultMessage and a description (the `legal` tag) beside it.
export type Message = string | { defaultMessage: string; description?: string }
export type Catalogs = Record<string, Partial<Record<Language, Record<string, Message>>>>

const files = import.meta.glob<Record<string, Message>>('./*/*.json', { eager: true, import: 'default' })

export const catalogs: Catalogs = {}
for (const [path, messages] of Object.entries(files)) {
  const [, area, language] = /^\.\/([^/]+)\/([^/]+)\.json$/.exec(path)!
  catalogs[area] = { ...catalogs[area], [language]: messages }
}

export const areas = Object.keys(catalogs)

function text(message: Message): string {
  return typeof message === 'string' ? message : message.defaultMessage
}

// messagesFor is every area's messages in language, each key prefixed with its area ("shell.signOut"). A key
// the language is missing shows its English text; scripts/i18n-check.sh fails the build on it instead.
export function messagesFor(language: Language, from: Catalogs = catalogs): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [area, byLanguage] of Object.entries(from)) {
    for (const source of [byLanguage.en, byLanguage[language]]) {
      for (const [key, message] of Object.entries(source ?? {})) out[`${area}.${key}`] = text(message)
    }
  }
  return out
}
