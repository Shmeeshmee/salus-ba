import type { Fetch } from './client'
import { isStaffLanguage, type Language } from '../i18n/languages'

// Hand-written types for the FHIR R4 elements the web app uses; everything else is absent.
export interface Meta {
  versionId: string
  lastUpdated: string
}

export interface Reference {
  reference: string
  display?: string
}

export interface Coding {
  system: string
  code: string
  display?: string
}

export interface CodeableConcept {
  coding?: Coding[]
  text?: string
}

export interface Extension {
  url: string
  valueBoolean?: boolean
  valueCode?: string
}

// Address is the clinic's location (docs/plans/analytics.md O1): the leaf region's ISO 3166-2:BA code as the state,
// the city's Latin name, and the city's Wikidata id in the organization-city extension.
export interface Address {
  state: string
  city?: string
  extension: Extension[]
}

export interface Organization {
  resourceType: 'Organization'
  id: string
  meta: Meta
  name: string
  active: boolean
  extension?: Extension[]
  address?: Address[]
}

export interface HumanName {
  family?: string
  given?: string[]
}

export interface Practitioner {
  resourceType: 'Practitioner'
  id: string
  meta: Meta
  name?: HumanName[]
  active: boolean
}

export interface PractitionerRole {
  resourceType: 'PractitionerRole'
  id: string
  meta: Meta
  practitioner: Reference
  organization: Reference
  active: boolean
  specialty?: CodeableConcept[]
}

export interface Patient {
  resourceType: 'Patient'
  id: string
  meta: Meta
  name?: HumanName[]
  birthDate?: string
  telecom?: { system: string; value: string; use?: string }[]
}

export type AppointmentStatus = 'booked' | 'cancelled' | 'fulfilled' | 'noshow'

export interface Appointment {
  resourceType: 'Appointment'
  id: string
  meta: Meta
  status: AppointmentStatus
  start: string
  end: string
  participant: { status: string; actor: Reference }[]
}

export interface QuestionnaireItem {
  linkId: string
  text: string
  type: string
  required?: boolean
  repeats?: boolean
  answerOption?: { valueString: string }[]
  enableWhen?: { question: string; operator: string; answerBoolean?: boolean; answerString?: string }[]
  extension?: { url: string; valueCodeableConcept?: CodeableConcept; valueCode?: string; valueInteger?: number }[]
  item?: QuestionnaireItem[]
}

export interface Questionnaire {
  resourceType: 'Questionnaire'
  id: string
  meta: Meta
  url: string
  version: string
  title: string
  status: 'draft' | 'active' | 'retired'
  description?: string
  item?: QuestionnaireItem[]
}

// Attachment is a file attached to an answer (attachments SPEC §2 "FHIR"): its bytes at url, which ends in
// /binaries/<id>/content.
export interface Attachment {
  contentType: string
  size: number
  title?: string
  hash?: string
  url?: string
}

export interface AnswerValue {
  valueAttachment?: Attachment
  valueString?: string
  valueBoolean?: boolean
  valueInteger?: number
  valueDecimal?: number
  valueDate?: string
}

// QuestionnaireResponseItem is an answered question, or a section holding them (SPEC §2 "FHIR").
export interface QuestionnaireResponseItem {
  linkId: string
  answer?: AnswerValue[]
  item?: QuestionnaireResponseItem[]
}

export interface QuestionnaireResponse {
  resourceType: 'QuestionnaireResponse'
  id: string
  meta: Meta
  extension?: { url: string; valueReference?: Reference }[]
  questionnaire: string // the version's canonical url|version
  status: 'in-progress' | 'completed' | 'stopped'
  subject: Reference
  author: Reference
  authored?: string
  item?: QuestionnaireResponseItem[] // a Name view has none
}

export interface Bundle<R = PractitionerRole | Practitioner> {
  resourceType: 'Bundle'
  type: 'searchset'
  total?: number
  entry?: { fullUrl: string; resource: R; search?: { mode: 'match' | 'include' } }[]
}

export function nameOf(name?: HumanName): string {
  return [...(name?.given ?? []), name?.family ?? ''].join(' ').trim()
}

// participant is the appointment's participant of a resource type: its id and, where the caller may see it,
// its display (the patient's name).
export function participant(appt: Appointment, type: 'Patient' | 'Practitioner' | 'PractitionerRole') {
  const actor = appt.participant.find((p) => p.actor.reference.startsWith(`${type}/`))?.actor
  if (!actor) return undefined
  return { id: actor.reference.slice(type.length + 1), display: actor.display }
}

// The clinic's settings in Organization extensions (api/fhir/StructureDefinition-organization-*.json).
export const listedURL = 'https://salus.invalid/fhir/StructureDefinition/organization-listed'
export const timeZoneURL = 'https://salus.invalid/fhir/StructureDefinition/organization-timezone'
export const defaultLanguageURL = 'https://salus.invalid/fhir/StructureDefinition/organization-default-language'
export const transcriptCopyURL = 'https://salus.invalid/fhir/StructureDefinition/organization-patient-transcript-copy'
export const contributesURL = 'https://salus.invalid/fhir/StructureDefinition/organization-analytics-contributes'
export const cityURL = 'https://salus.invalid/fhir/StructureDefinition/organization-city'
export const defaultTimeZone = 'Europe/Sarajevo'

// defaultLanguage is a staff language (docs/i18n/SPEC.md §3 "The clinic's default language"), Bosnian unless set;
// transcriptCopy whether the patients get a copy of their consultations' transcripts (SAL-17), off unless set;
// region and city the clinic's location ('' for none), cityName the city's Latin name; contributes whether the
// clinic counts in the platform figures (SAL-21, O6), on unless the admin opts out.
export type ClinicSettings = {
  name: string
  listed: boolean
  timeZone: string
  defaultLanguage: Language
  transcriptCopy: boolean
  region: string
  city: string
  cityName: string
  contributes: boolean
}

export function settingsOf(org: Organization): ClinicSettings {
  const ext = (url: string) => org.extension?.find((e) => e.url === url)
  const language = ext(defaultLanguageURL)?.valueCode
  return {
    name: org.name,
    listed: ext(listedURL)?.valueBoolean ?? false,
    timeZone: ext(timeZoneURL)?.valueCode ?? defaultTimeZone,
    defaultLanguage: isStaffLanguage(language) ? language : 'bs',
    transcriptCopy: ext(transcriptCopyURL)?.valueBoolean ?? false,
    region: org.address?.[0]?.state ?? '',
    city: org.address?.[0]?.extension.find((e) => e.url === cityURL)?.valueCode ?? '',
    cityName: org.address?.[0]?.city ?? '',
    contributes: ext(contributesURL)?.valueBoolean ?? true,
  }
}

export function withSettings(org: Organization, s: ClinicSettings): Organization {
  const own = [listedURL, timeZoneURL, defaultLanguageURL, transcriptCopyURL, contributesURL]
  const others = (org.extension ?? []).filter((e) => !own.includes(e.url))
  return {
    ...org,
    name: s.name,
    extension: [
      ...others,
      { url: listedURL, valueBoolean: s.listed },
      { url: timeZoneURL, valueCode: s.timeZone },
      { url: defaultLanguageURL, valueCode: s.defaultLanguage },
      { url: transcriptCopyURL, valueBoolean: s.transcriptCopy },
      { url: contributesURL, valueBoolean: s.contributes },
    ],
    address: s.city ? [{ state: s.region, city: s.cityName, extension: [{ url: cityURL, valueCode: s.city }] }] : org.address,
  }
}

// FhirError is a FHIR call that did not succeed: the HTTP status and the first OperationOutcome issue code.
export class FhirError extends Error {
  readonly status: number
  readonly code?: string

  constructor(status: number, code?: string) {
    super(`FHIR request failed with ${status}`)
    this.status = status
    this.code = code
  }
}

export type FhirClient = ReturnType<typeof newFhirClient>

const fhirJSON = 'application/fhir+json'

// newFhirClient calls /fhir/R4 on the page's own origin with the session cookie. Writes carry the CSRF header
// and If-Match with the version that was read.
export function newFhirClient(fetchImpl: Fetch = (r) => fetch(r)) {
  async function send<T>(method: string, path: string, body?: unknown, version?: string): Promise<T> {
    const headers = new Headers({ Accept: fhirJSON })
    if (body !== undefined) {
      headers.set('Content-Type', fhirJSON)
      headers.set('X-Salus-CSRF', '1')
    }
    if (version !== undefined) headers.set('If-Match', `W/"${version}"`)
    const url = new URL(`/fhir/R4/${path}`, globalThis.location.origin)
    const res = await fetchImpl(
      new Request(url, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) }),
    )
    if (!res.ok) throw new FhirError(res.status, await issueCode(res))
    return (await res.json()) as T
  }

  return {
    get: <T>(path: string) => send<T>('GET', path),
    put: <T>(path: string, resource: unknown, version: string) => send<T>('PUT', path, resource, version),
  }
}

async function issueCode(res: Response): Promise<string | undefined> {
  try {
    const body = (await res.json()) as { resourceType?: string; issue?: { code?: string }[] }
    return body.resourceType === 'OperationOutcome' ? body.issue?.[0]?.code : undefined
  } catch {
    return undefined
  }
}
