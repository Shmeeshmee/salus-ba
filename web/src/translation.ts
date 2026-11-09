import { all, isChoice, type Item } from './builder'
import { contentLanguages, type ContentLanguage, type ItemTranslation, type Translation } from './questionnaire'

// A translation's completeness as the API judges it at publishing (docs/i18n/PROGRESS.md 1.4 "Complete means"):
// a title; a description when the original has one; for every item its text, its help text when the original
// has one, and every option.

const filled = (s?: string) => s !== undefined && s.trim() !== ''

// Progress is n of total texts translated, and the items that lack any, in document order.
export type Progress = { n: number; total: number; missing: string[] }

function itemTexts(it: Item, t: ItemTranslation = {}): { n: number; total: number } {
  const texts: (string | undefined)[] = [t.text]
  if (it.kind !== 'display' && filled(it.help)) texts.push(t.help)
  if (isChoice(it.kind)) texts.push(...it.options.map((_, i) => t.options?.[i]))
  return { n: texts.filter(filled).length, total: texts.length }
}

export function progress(t: Translation, description: string, items: Item[]): Progress {
  const head = filled(description) ? [t.title, t.description] : [t.title]
  const out: Progress = { n: head.filter(filled).length, total: head.length, missing: [] }
  for (const it of all(items)) {
    const c = itemTexts(it, t.items[it.linkId])
    out.n += c.n
    out.total += c.total
    if (c.n < c.total) out.missing.push(it.linkId)
  }
  return out
}

const byList = (a: Translation, b: Translation) => contentLanguages.indexOf(a.language) - contentLanguages.indexOf(b.language)

// firstIncomplete is what a publish would be refused for: the first incomplete translation in the language
// list's order, and its items that lack a text.
export function firstIncomplete(translations: Translation[], description: string, items: Item[]): { language: ContentLanguage; linkIds: string[] } | undefined {
  for (const t of [...translations].sort(byList)) {
    const p = progress(t, description, items)
    if (p.n < p.total) return { language: t.language, linkIds: p.missing }
  }
  return undefined
}

// unused is the languages a translation can be added in: neither the original nor translated already.
export function unused(original: ContentLanguage, translations: Translation[]): ContentLanguage[] {
  return contentLanguages.filter((l) => l !== original && !translations.some((t) => t.language === l))
}

// withItem changes some of one item's translated texts.
export function withItem(t: Translation, linkId: string, patch: ItemTranslation): Translation {
  return { ...t, items: { ...t.items, [linkId]: { ...t.items[linkId], ...patch } } }
}
