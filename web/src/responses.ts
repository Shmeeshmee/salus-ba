import type { AnswerValue, Bundle, Questionnaire, QuestionnaireResponse, QuestionnaireResponseItem } from './api/fhir'
import { find, isChoice, isQuestion, type Condition, type Item, type Kind } from './builder'
import { formatDate, formatTime, intlLocale } from './i18n/format'
import type { Language } from './i18n/languages'
import type { useT } from './i18n/useT'
import { fromFHIR, isContentLanguage, type ContentLanguage, type Translation, type Version } from './questionnaire'

type T = ReturnType<typeof useT>

// An appointment's questionnaire requests as the appointment detail shows them, read from the
// QuestionnaireResponse search with the versions included (SPEC §2 "What the practitioner sees").

export type ResponseStatus = QuestionnaireResponse['status']

// Question is an enabled question of a submitted response and its answers, none when it was not answered.
export type Question = { linkId: string; text: string; kind: Kind; values: AnswerValue[] }

// Block is the questions of one section, or a run of questions outside any section.
export type Block = { section?: string; questions: Question[] }

export type Request = {
  id: string
  url?: string // the series' canonical url
  title?: string // the version's title; none when the search did not include the version
  status: ResponseStatus
  authored?: string
  answers: Block[] // only for a submitted response
  language?: ContentLanguage // the language the patient answered in (B9); none before they started
  translated?: Block[] // the answers with the questions in that language, when it is not the original
}

// The FHIR response carries the language answered in (QuestionnaireResponse.language).
type FHIRResponse = QuestionnaireResponse & { language?: string }

function flatten(items: QuestionnaireResponseItem[], out: Map<string, AnswerValue[]>): Map<string, AnswerValue[]> {
  for (const it of items) {
    if (it.answer) out.set(it.linkId, it.answer)
    if (it.item) flatten(it.item, out)
  }
  return out
}

function matches(c: Condition, values: AnswerValue[]): boolean {
  return values.some((v) => (typeof c.answer === 'boolean' ? v.valueBoolean === c.answer : v.valueString === c.answer))
}

// answered lists the enabled questions of the version by section, with the response's answers. An item is
// enabled when its section is, and its condition's question is enabled and answered as the condition says
// (any answer of a choice with repeats); instructions and hidden items are left out, as on submit.
export function answered(items: Item[], response: QuestionnaireResponseItem[] = []): Block[] {
  const answers = flatten(response, new Map())
  const index = new Map<string, { item: Item; section?: Item }>()
  for (const it of items) {
    index.set(it.linkId, { item: it })
    for (const child of it.items) index.set(child.linkId, { item: child, section: it })
  }
  // Conditions point at earlier items only (the service refuses anything else), so this ends.
  const enabled = (linkId: string): boolean => {
    const at = index.get(linkId)
    if (!at || (at.section && !enabled(at.section.linkId))) return false
    const c = at.item.condition
    return !c || (enabled(c.question) && matches(c, answers.get(c.question) ?? []))
  }
  const question = (it: Item): Question => ({ linkId: it.linkId, text: it.text, kind: it.kind, values: answers.get(it.linkId) ?? [] })

  const blocks: Block[] = []
  for (const it of items) {
    if (!enabled(it.linkId)) continue
    if (it.kind === 'group') {
      const questions = it.items.filter((c) => isQuestion(c.kind) && enabled(c.linkId)).map(question)
      if (questions.length > 0) blocks.push({ section: it.text, questions })
    } else if (isQuestion(it.kind)) {
      const last = blocks.at(-1)
      if (last && last.section === undefined) last.questions.push(question(it))
      else blocks.push({ questions: [question(it)] })
    }
  }
  return blocks
}

// valueText is one answer as the practitioner reads it, in the UI language: a date on the calendar (no time
// zone), a number as written but with the language's decimal sign.
export function valueText(v: AnswerValue, t: T, language: Language): string {
  if (v.valueBoolean !== undefined) return t(v.valueBoolean ? 'questionnaire.yes' : 'questionnaire.no')
  const locale = intlLocale(language)
  if (v.valueDate !== undefined) {
    return new Intl.DateTimeFormat(locale, { timeZone: 'UTC', day: 'numeric', month: 'long', year: 'numeric' }).format(new Date(`${v.valueDate}T00:00:00Z`))
  }
  const n = v.valueInteger ?? v.valueDecimal
  if (n !== undefined) return new Intl.NumberFormat(locale, { maximumFractionDigits: 20, useGrouping: false }).format(n)
  return v.valueString ?? ''
}

const filled = (s?: string): s is string => s !== undefined && s.trim() !== ''

// inTranslation is the items with their texts in the translation, the original's where it lacks one. The
// options stay the original's: the answers and the conditions are written in them.
function inTranslation(items: Item[], t: Translation): Item[] {
  return items.map((it) => {
    const text = t.items[it.linkId]?.text
    return { ...it, text: filled(text) ? text : it.text, items: inTranslation(it.items, t) }
  })
}

// translatedValues gives a choice answer as the patient read it: the option at the same position in the translation.
function translatedValues(q: Question, items: Item[], t: Translation): Question {
  const item = find(items, q.linkId)
  if (!item || !isChoice(item.kind)) return q
  const options = t.items[q.linkId]?.options ?? []
  const values = q.values.map((v) => {
    const option = options[item.options.indexOf(v.valueString ?? '')]
    return filled(option) ? { valueString: option } : v
  })
  return { ...q, values }
}

// translatedAnswers is the answers as the patient read the questions (B9), when they answered in a translation.
function translatedAnswers(v: Version, language: ContentLanguage | undefined, r: QuestionnaireResponse): Block[] | undefined {
  const t = v.translations.find((x) => x.language === language)
  if (!t || language === v.language) return undefined
  return answered(inTranslation(v.items, t), r.item).map((b) => ({ ...b, questions: b.questions.map((q) => translatedValues(q, v.items, t)) }))
}

function requestOf(r: FHIRResponse, v?: Version): Request {
  const language = isContentLanguage(r.language) ? r.language : undefined
  const submitted = v && r.status === 'completed'
  return {
    id: r.id,
    url: v?.url,
    title: v?.title,
    status: r.status,
    authored: r.authored,
    answers: submitted ? answered(v.items, r.item) : [],
    language,
    translated: submitted ? translatedAnswers(v, language, r) : undefined,
  }
}

// requestsOf reads the search: each response with the title of the version it answers (`url|version`).
export function requestsOf(bundle: Bundle<QuestionnaireResponse | Questionnaire>): Request[] {
  const entries = (bundle.entry ?? []).map((e) => e.resource)
  const versions = new Map<string, Version>()
  for (const q of entries) {
    if (q.resourceType === 'Questionnaire') versions.set(`${q.url}|${q.version}`, fromFHIR(q))
  }
  return entries.flatMap((r) => (r.resourceType === 'QuestionnaireResponse' ? [requestOf(r, versions.get(r.questionnaire))] : []))
}

// requestOfVersion is a request just made: in progress, of the version picked.
export function requestOfVersion(id: string, v: Version): Request {
  return { id, url: v.url, title: v.title, status: 'in-progress', answers: [] }
}

// statusOf is W34: unsubmitted ones are closed once the appointment has started.
export function statusOf(r: Pick<Request, 'status' | 'authored'>, start: Date, now: Date, zone: string, t: T, language: Language): string {
  if (r.status === 'stopped') return t('questionnaire.withdrawn')
  if (r.status === 'completed') {
    const at = new Date(r.authored ?? '')
    return t('questionnaire.submitted', { date: formatDate(at, zone, language, 'long'), time: formatTime(at, zone, language) })
  }
  return t(now >= start ? 'questionnaire.notSubmittedBefore' : 'questionnaire.notSubmitted')
}
