import type { Questionnaire, QuestionnaireItem } from './api/fhir'
import type { components } from './api/schema'
import { fileKinds, isChoice, isQuestion, type Condition, type FileKind, type Item, type Kind } from './builder'

type JSONQuestionnaire = components['schemas']['Questionnaire']
type JSONItem = components['schemas']['QuestionnaireItem']
export type SaveRequest = components['schemas']['SaveQuestionnaireRequest']
export type Status = Questionnaire['status']

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

function fromFHIRItem(f: QuestionnaireItem): Item {
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

export function fromFHIR(q: Questionnaire): Version {
  return {
    id: q.id,
    url: q.url,
    version: Number(q.version),
    status: q.status,
    title: q.title,
    description: q.description ?? '',
    items: (q.item ?? []).map(fromFHIRItem),
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

// toRequest is the body of a create or a save.
export function toRequest(title: string, description: string, items: Item[]): SaveRequest {
  const body: SaveRequest = { title, item: items.map(toJSONItem) }
  if (description.trim() !== '') body.description = description
  return body
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
