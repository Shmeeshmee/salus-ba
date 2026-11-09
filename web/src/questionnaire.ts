import type { Questionnaire, QuestionnaireItem } from './api/fhir'
import type { components } from './api/schema'
import { all, fileKinds, isChoice, isQuestion, type Condition, type FileKind, type Item, type Kind } from './builder'
import { languageNames } from './language'

type JSONQuestionnaire = components['schemas']['Questionnaire']
type JSONItem = components['schemas']['QuestionnaireItem']
export type SaveRequest = components['schemas']['SaveQuestionnaireRequest']
export type Status = Questionnaire['status']

// ContentLanguage is a language a questionnaire is written or translated in: any of the ten patient languages
// (docs/i18n/SPEC.md §3 "Tags"), in the database enum's order.
export type ContentLanguage = components['schemas']['Language']
export const contentLanguages: ContentLanguage[] = ['bs', 'hr', 'sr-Latn', 'sr-Cyrl', 'en', 'de', 'ar', 'tr', 'it', 'fr']

// contentLanguageNames is each by its own name (COPY.md L1): the staff languages' names, and the other five.
export const contentLanguageNames: Record<ContentLanguage, string> = {
  ...languageNames,
  de: 'Deutsch',
  ar: 'العربية',
  tr: 'Türkçe',
  it: 'Italiano',
  fr: 'Français',
}

export const isContentLanguage = (x: unknown): x is ContentLanguage => contentLanguages.includes(x as ContentLanguage)

// Translation is a version's texts in another language: the title, the description, and by linkId each item's
// text, help text and option texts (options by position). A text not yet translated is absent or blank.
export type Translation = components['schemas']['QuestionnaireTranslation']
export type ItemTranslation = components['schemas']['QuestionnaireItemTranslation']

// The FHIR read carries the translations as the R4 translation extension on each translated primitive.
const translationURL = 'http://hl7.org/fhir/StructureDefinition/translation'
type Element = { extension?: { url: string; extension?: { url: string; valueCode?: string; valueString?: string }[] }[] }
export type FHIRItem = Omit<QuestionnaireItem, 'answerOption' | 'item'> & { _text?: Element; answerOption?: { valueString: string; _valueString?: Element }[]; item?: FHIRItem[] }
export type FHIRQuestionnaire = Omit<Questionnaire, 'item'> & { language?: string; _title?: Element; _description?: Element; item?: FHIRItem[] }

// Version is one version of a questionnaire as the library and the builder use it, read over FHIR or from
// the JSON API's answer to a write.
export type Version = {
  id: string
  url: string // the series' canonical url, the same for every version
  version: number
  status: Status
  title: string
  description: string
  items: Item[]
  language: ContentLanguage // the original language
  translations: Translation[]
  versionId: number // for If-Match
  lastUpdated: string
}

const helpControl = 'http://hl7.org/fhir/StructureDefinition/questionnaire-itemControl'
const mimeType = 'http://hl7.org/fhir/StructureDefinition/mimeType'
const maxOccurs = 'http://hl7.org/fhir/StructureDefinition/questionnaire-maxOccurs'

// The content types of each kind of file (attachments SPEC §2 "Item model").
const kindOfType: Record<string, FileKind> = { 'image/jpeg': 'photo', 'image/png': 'photo', 'application/pdf': 'document', 'video/mp4': 'video' }

// ordered lists the kinds once each, in the builder's order.
function ordered(accept: FileKind[]): FileKind[] {
  return fileKinds.filter((k) => accept.includes(k))
}

// filesOfFHIR reads a file upload's kinds from its mimeType extensions and its number of files from maxOccurs.
function filesOfFHIR(f: QuestionnaireItem): Partial<Item> {
  if (f.type !== 'attachment') return {}
  const ext = f.extension ?? []
  const accept = ext.flatMap((e) => (e.url === mimeType && e.valueCode && kindOfType[e.valueCode] ? [kindOfType[e.valueCode]] : []))
  return { accept: ordered(accept), maxFiles: ext.find((e) => e.url === maxOccurs)?.valueInteger ?? 1 }
}

// isHelp tells a help item (SPEC §2 "Item model": a child display item with the help item control).
function isHelp(f: QuestionnaireItem): boolean {
  return f.type === 'display' && (f.extension ?? []).some((e) => e.url === helpControl && e.valueCodeableConcept?.coding?.some((c) => c.code === 'help'))
}

function kindOf(type: string, repeats?: boolean): Kind {
  if (type === 'choice') return repeats ? 'multiple' : 'single'
  return type as Kind
}

function conditionOf(c?: { question: string; answerBoolean?: boolean; answerString?: string }): Condition | undefined {
  if (!c) return undefined
  return { question: c.question, answer: c.answerBoolean ?? c.answerString ?? '' }
}

function item(linkId: string, kind: Kind, text: string, rest: Partial<Item>): Item {
  return { linkId, kind, text, required: false, help: '', options: [], items: [], ...rest }
}

function fromFHIRItem(f: FHIRItem): Item {
  const children = f.item ?? []
  return item(f.linkId, kindOf(f.type, f.repeats), f.text, {
    required: f.required ?? false,
    help: children.find(isHelp)?.text ?? '',
    options: (f.answerOption ?? []).map((o) => o.valueString),
    condition: conditionOf(f.enableWhen?.[0]),
    items: children.filter((c) => !isHelp(c)).map(fromFHIRItem),
    ...filesOfFHIR(f),
  })
}

// translationsOf reads each language's text of a primitive from its translation extensions.
function translationsOf(e?: Element): Map<ContentLanguage, string> {
  const out = new Map<ContentLanguage, string>()
  for (const x of e?.extension ?? []) {
    if (x.url !== translationURL) continue
    const lang = x.extension?.find((p) => p.url === 'lang')?.valueCode
    const content = x.extension?.find((p) => p.url === 'content')?.valueString
    if (isContentLanguage(lang) && content !== undefined) out.set(lang, content)
  }
  return out
}

// translationsOfFHIR gathers the translation extensions of the whole questionnaire into one Translation per
// language that has any text, in the language list's order.
function translationsOfFHIR(q: FHIRQuestionnaire): Translation[] {
  const found = new Map<ContentLanguage, Translation>()
  const of = (lang: ContentLanguage): Translation => {
    const t = found.get(lang) ?? { language: lang, title: '', items: {} }
    found.set(lang, t)
    return t
  }
  const entry = (lang: ContentLanguage, linkId: string) => (of(lang).items[linkId] ??= {})
  for (const [lang, text] of translationsOf(q._title)) of(lang).title = text
  for (const [lang, text] of translationsOf(q._description)) of(lang).description = text
  const visit = (f: FHIRItem) => {
    for (const [lang, text] of translationsOf(f._text)) entry(lang, f.linkId).text = text
    const help = (f.item ?? []).find(isHelp) as FHIRItem | undefined
    for (const [lang, text] of translationsOf(help?._text)) entry(lang, f.linkId).help = text
    const options = f.answerOption ?? []
    options.forEach((o, i) => {
      for (const [lang, text] of translationsOf(o._valueString)) {
        const e = entry(lang, f.linkId)
        e.options ??= options.map(() => '')
        e.options[i] = text
      }
    })
    for (const child of f.item ?? []) if (!isHelp(child)) visit(child)
  }
  for (const f of q.item ?? []) visit(f)
  return contentLanguages.flatMap((lang) => found.get(lang) ?? [])
}

export function fromFHIR(q: FHIRQuestionnaire): Version {
  return {
    id: q.id,
    url: q.url,
    version: Number(q.version),
    status: q.status,
    title: q.title,
    description: q.description ?? '',
    items: (q.item ?? []).map(fromFHIRItem),
    language: isContentLanguage(q.language) ? q.language : 'bs',
    translations: translationsOfFHIR(q),
    versionId: Number(q.meta.versionId),
    lastUpdated: q.meta.lastUpdated,
  }
}

function fromJSONItem(j: JSONItem): Item {
  return item(j.linkId, kindOf(j.type, j.repeats), j.text, {
    required: j.required ?? false,
    help: j.help ?? '',
    options: (j.answerOption ?? []).map((o) => o.valueString),
    condition: conditionOf(j.enableWhen),
    items: (j.item ?? []).map(fromJSONItem),
    ...(j.type === 'attachment' ? { accept: ordered(j.accept ?? []), maxFiles: j.maxFiles ?? 1 } : {}),
  })
}

export function fromJSON(q: JSONQuestionnaire): Version {
  return {
    id: q.id,
    url: q.url,
    version: q.version,
    status: q.status,
    title: q.title,
    description: q.description ?? '',
    items: q.item.map(fromJSONItem),
    language: q.language ?? 'bs',
    translations: q.translations ?? [],
    versionId: q.versionId,
    lastUpdated: q.lastUpdated,
  }
}

function toJSONItem(it: Item): JSONItem {
  const out: JSONItem = { linkId: it.linkId, text: it.text, type: isChoice(it.kind) ? 'choice' : (it.kind as JSONItem['type']) }
  if (isQuestion(it.kind) && it.required) out.required = true
  if (it.kind === 'multiple') out.repeats = true
  if (it.kind !== 'display' && it.help.trim() !== '') out.help = it.help
  if (isChoice(it.kind)) out.answerOption = it.options.map((valueString) => ({ valueString }))
  if (it.kind === 'attachment') {
    out.accept = ordered(it.accept ?? [])
    // A blank field is NaN, which JSON writes as null; left out, the API names the item (422) instead of a 400.
    if (Number.isFinite(it.maxFiles)) out.maxFiles = it.maxFiles
  }
  if (it.condition) {
    const { question, answer } = it.condition
    out.enableWhen = typeof answer === 'boolean' ? { question, answerBoolean: answer } : { question, answerString: answer }
  }
  if (it.kind === 'group') out.item = it.items.map(toJSONItem)
  return out
}

const filled = (s?: string) => s !== undefined && s.trim() !== ''

// itemTranslation is what of an item's translation the API takes: the texts the original has, blanks left
// out, and a choice's options as many as the item has (a blank one not yet translated).
function itemTranslation(it: Item, t: ItemTranslation): ItemTranslation {
  const out: ItemTranslation = {}
  if (filled(t.text)) out.text = t.text
  if (it.kind !== 'display' && filled(it.help) && filled(t.help)) out.help = t.help
  if (isChoice(it.kind) && (t.options ?? []).some(filled)) out.options = it.options.map((_, i) => t.options?.[i] ?? '')
  return out
}

// translationRequest holds a translation to the draft as it is sent: the entries of items no longer in it are
// left out, and a description only when the original has one.
function translationRequest(t: Translation, description: string, items: Item[]): Translation {
  const out: Translation = { language: t.language, title: t.title, items: {} }
  if (filled(description) && filled(t.description)) out.description = t.description
  for (const it of all(items)) {
    const entry = itemTranslation(it, t.items[it.linkId] ?? {})
    if (Object.keys(entry).length > 0) out.items[it.linkId] = entry
  }
  return out
}

// toRequest is the body of a create or a save; with a language, also the original language and every
// translation (an empty list removes them).
export function toRequest(title: string, description: string, items: Item[], language?: ContentLanguage, translations: Translation[] = []): SaveRequest {
  const body: SaveRequest = { title, item: items.map(toJSONItem) }
  if (description.trim() !== '') body.description = description
  if (language === undefined) return body
  return { ...body, language, translations: translations.map((t) => translationRequest(t, description, items)) }
}

const rank: Record<Status, number> = { active: 0, draft: 1, retired: 2 }

// libraryOf picks what the library lists of the clinic's versions: of each questionnaire its draft and its
// active version, or, when it has neither, its latest retired one. By title, then version.
export function libraryOf(all: Questionnaire[]): Version[] {
  const bySeries = new Map<string, Version[]>()
  for (const q of all) {
    const v = fromFHIR(q)
    bySeries.set(v.url, [...(bySeries.get(v.url) ?? []), v])
  }
  const shown: Version[] = []
  for (const versions of bySeries.values()) {
    const live = versions.filter((v) => v.status !== 'retired')
    if (live.length > 0) shown.push(...live)
    else shown.push(versions.reduce((a, b) => (b.version > a.version ? b : a)))
  }
  return shown.sort((a, b) => a.title.localeCompare(b.title) || rank[a.status] - rank[b.status] || a.version - b.version)
}
