import type { AnswerValue, Bundle, Questionnaire, QuestionnaireResponse, QuestionnaireResponseItem } from './api/fhir'
import { isQuestion, type Condition, type Item, type Kind } from './builder'
import { questionnaires as t } from './messages'
import { fromFHIR, type Version } from './questionnaire'
import { longDate, timeOf } from './time'

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
  title: string
  status: ResponseStatus
  authored?: string
  answers: Block[] // only for a submitted response
}

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

const calendarDate = new Intl.DateTimeFormat('en-GB', { timeZone: 'UTC', day: 'numeric', month: 'long', year: 'numeric' })

// valueText is one answer as the practitioner reads it.
export function valueText(v: AnswerValue): string {
  if (v.valueBoolean !== undefined) return v.valueBoolean ? t.yes : t.no
  if (v.valueDate !== undefined) return calendarDate.format(new Date(`${v.valueDate}T00:00:00Z`))
  return String(v.valueString ?? v.valueInteger ?? v.valueDecimal ?? '')
}

function requestOf(r: QuestionnaireResponse, v?: Version): Request {
  return {
    id: r.id,
    url: v?.url,
    title: v?.title ?? t.picker,
    status: r.status,
    authored: r.authored,
    answers: v && r.status === 'completed' ? answered(v.items, r.item) : [],
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
export function statusOf(r: Pick<Request, 'status' | 'authored'>, start: Date, now: Date, zone: string): string {
  if (r.status === 'stopped') return t.withdrawn
  if (r.status === 'completed') {
    const at = new Date(r.authored ?? '')
    return t.submitted(`${longDate(at, zone)}, ${timeOf(at, zone)}`)
  }
  return now >= start ? t.notSubmittedBefore : t.notSubmitted
}
