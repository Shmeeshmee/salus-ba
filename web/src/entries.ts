import { useEffect, useState } from 'react'
import type { ApiClient } from './api/client'
import { participant, type Appointment, type Bundle, type CodeableConcept, type Extension, type FhirClient, type Meta, type Reference } from './api/fhir'
import { attempt } from './api/result'
import type { components } from './api/schema'

// The patient's allergies and current medications (SAL-41, ADR 0014): read as FHIR AllergyIntolerance and
// MedicationStatement searches, recorded and confirmed through the JSON API (api/openapi.yaml, tag history). The API
// returns only what the practitioner may see; an entry entered in error never comes back.

export type AllergyCategory = components['schemas']['AllergyCategory']
export type AllergySeverity = components['schemas']['AllergySeverity']
export type ClinicalStatus = components['schemas']['AllergyClinicalStatus']
export type MedicationStatus = components['schemas']['MedicationStatus']
export type DoseUnit = components['schemas']['DoseUnit']
export type PeriodUnit = components['schemas']['PeriodUnit']
export type AllergyInput = components['schemas']['AllergyIntoleranceInput']
export type MedicationInput = components['schemas']['MedicationStatementInput']

export const allergyCategories: AllergyCategory[] = ['food', 'medication', 'environment', 'biologic']
export const severities: AllergySeverity[] = ['mild', 'moderate', 'severe']
export const clinicalStatuses: ClinicalStatus[] = ['active', 'inactive', 'resolved']
export const medicationStatuses: MedicationStatus[] = ['active', 'stopped']
export const doseUnits: DoseUnit[] = ['mg', 'g', 'ug', 'mL', '[iU]', '{tablet}', '{capsule}', '{drop}', '{puff}', '{patch}']
export const periodUnits: PeriodUnit[] = ['h', 'd', 'wk', 'mo']

export interface AllergyIntolerance {
  resourceType: 'AllergyIntolerance'
  id: string
  meta: Meta
  patient: Reference
  code: CodeableConcept
  category?: string[]
  reaction?: { severity?: string; manifestation?: CodeableConcept[] }[]
  clinicalStatus?: CodeableConcept
  verificationStatus?: CodeableConcept
}

export interface MedicationStatement {
  resourceType: 'MedicationStatement'
  id: string
  meta: Meta
  status: string
  subject: Reference
  medicationCodeableConcept: CodeableConcept
  extension?: Extension[]
  dosage?: {
    asNeededBoolean?: boolean
    timing?: { repeat?: { frequency?: number; period?: number; periodUnit?: string } }
    doseAndRate?: { doseQuantity?: { value?: number; unit?: string; code?: string; system?: string } }[]
  }[]
}

export type Allergy = {
  id: string
  version: string
  text: string
  category?: AllergyCategory
  reaction?: string
  severity?: AllergySeverity
  clinicalStatus?: ClinicalStatus
  confirmed: boolean
}

export type Medication = {
  id: string
  version: string
  text: string
  status: MedicationStatus
  dose?: { value: number; unit: DoseUnit }
  timing?: { frequency: number; period: number; periodUnit: PeriodUnit }
  asNeeded: boolean
  confirmed: boolean
}

export type Entries = { allergies: Allergy[]; medications: Medication[] }

export const confirmedURL = 'https://salus.invalid/fhir/StructureDefinition/medicationstatement-confirmed'

export function oneOf<T extends string>(value: string | undefined, set: readonly T[]): T | undefined {
  return set.find((x) => x === value)
}

function allergyOf(r: AllergyIntolerance): Allergy {
  const reaction = r.reaction?.[0]
  const a: Allergy = {
    id: r.id,
    version: r.meta.versionId,
    text: r.code.text ?? '',
    category: oneOf(r.category?.[0], allergyCategories),
    reaction: reaction?.manifestation?.[0]?.text,
    severity: oneOf(reaction?.severity, severities),
    clinicalStatus: oneOf(r.clinicalStatus?.coding?.[0]?.code, clinicalStatuses),
    confirmed: r.verificationStatus?.coding?.[0]?.code === 'confirmed',
  }
  return strip(a)
}

function medicationOf(r: MedicationStatement): Medication {
  const dosage = r.dosage?.[0]
  const quantity = dosage?.doseAndRate?.[0]?.doseQuantity
  const repeat = dosage?.timing?.repeat
  const unit = oneOf(quantity?.code, doseUnits)
  const periodUnit = oneOf(repeat?.periodUnit, periodUnits)
  const m: Medication = {
    id: r.id,
    version: r.meta.versionId,
    text: r.medicationCodeableConcept.text ?? '',
    status: oneOf(r.status, medicationStatuses) ?? 'active',
    dose: quantity?.value !== undefined && unit ? { value: quantity.value, unit } : undefined,
    timing: repeat?.frequency && repeat.period && periodUnit ? { frequency: repeat.frequency, period: repeat.period, periodUnit } : undefined,
    asNeeded: dosage?.asNeededBoolean === true,
    confirmed: r.extension?.some((e) => e.url === confirmedURL && e.valueBoolean === true) === true,
  }
  return strip(m)
}

// strip leaves out the fields that are undefined, so an entry holds only what the resource has.
function strip<T extends object>(o: T): T {
  return Object.fromEntries(Object.entries(o).filter(([, v]) => v !== undefined)) as T
}

export function allergiesOf(b: Bundle<AllergyIntolerance>): Allergy[] {
  return (b.entry ?? []).map((e) => allergyOf(e.resource))
}

export function medicationsOf(b: Bundle<MedicationStatement>): Medication[] {
  return (b.entry ?? []).map((e) => medicationOf(e.resource))
}

// activeAllergies are the allergies the warning lists: those whose clinical status is active.
export function activeAllergies(list: Allergy[]): Allergy[] {
  return list.filter((a) => a.clinicalStatus === 'active')
}

export async function readEntries(fhir: Pick<FhirClient, 'get'>, patientId: string): Promise<Entries> {
  const q = `?patient=${encodeURIComponent(patientId)}`
  const [allergies, medications] = await Promise.all([
    fhir.get<Bundle<AllergyIntolerance>>(`AllergyIntolerance${q}`),
    fhir.get<Bundle<MedicationStatement>>(`MedicationStatement${q}`),
  ])
  return { allergies: allergiesOf(allergies), medications: medicationsOf(medications) }
}

// encounterOf is the appointment's encounter, undefined while there is none (before the consultation started).
export async function encounterOf(fhir: Pick<FhirClient, 'get'>, appointmentId: string): Promise<string | undefined> {
  const b = await fhir.get<Bundle<{ id: string }>>(`Encounter?appointment=${encodeURIComponent(appointmentId)}`)
  return b.entry?.[0]?.resource.id
}

// Written is how a write ended: saved, changed since it was read (412), not allowed (403), refused as invalid (400),
// or failed otherwise.
export type Written = 'ok' | 'changed' | 'forbidden' | 'invalid' | 'failed'

function written(res: { ok: boolean; response?: Response }): Written {
  if (res.ok) return 'ok'
  const status = res.response?.status
  if (status === 412) return 'changed'
  if (status === 403) return 'forbidden'
  if (status === 400) return 'invalid'
  return 'failed'
}

export type EntryKind = 'allergy' | 'medication'

export async function confirmEntry(api: ApiClient, kind: EntryKind, id: string, version: string): Promise<Written> {
  const params = { params: { path: { id }, header: { 'If-Match': `W/"${version}"` } } }
  const res = await attempt(kind === 'allergy' ? api.POST('/allergies/{id}/confirm', params) : api.POST('/medications/{id}/confirm', params))
  return written(res)
}

export async function recordAllergy(api: ApiClient, encounterId: string, body: AllergyInput): Promise<Written> {
  return written(await attempt(api.POST('/encounters/{id}/allergies', { params: { path: { id: encounterId } }, body })))
}

export async function recordMedication(api: ApiClient, encounterId: string, body: MedicationInput): Promise<Written> {
  return written(await attempt(api.POST('/encounters/{id}/medications', { params: { path: { id: encounterId } }, body })))
}

// unitKey is the catalog key of a UCUM unit: its code without the brackets UCUM puts around arbitrary units.
export function unitKey(unit: string): string {
  return unit.replace(/[[\]{}]/g, '')
}

type Fhir = Pick<FhirClient, 'get'>

export type PatientEntries = { entries?: Entries; failed: boolean; reload: () => void }

// refreshEvery is how often the call page reads the lists again, so an entry another clinician or the patient adds
// during the visit is shown.
export const refreshEvery = 60_000

// usePatientEntries reads the appointment's patient, then their allergies and medications, again every refresh
// milliseconds and on reload. The last list read stays shown while it is read again.
export function usePatientEntries(fhir: Fhir | undefined, appointmentId: string, refresh = refreshEvery): PatientEntries {
  const [state, setState] = useState<{ entries?: Entries; failed: boolean }>({ failed: false })
  const [reads, setReads] = useState(0)

  useEffect(() => {
    const timer = setInterval(() => setReads((n) => n + 1), refresh)
    return () => clearInterval(timer)
  }, [refresh])

  useEffect(() => {
    if (!fhir) return
    let open = true
    async function load(f: Fhir): Promise<Entries> {
      const appt = await f.get<Appointment>(`Appointment/${encodeURIComponent(appointmentId)}`)
      const patient = participant(appt, 'Patient')
      if (!patient) return { allergies: [], medications: [] }
      return readEntries(f, patient.id)
    }
    load(fhir).then(
      (entries) => open && setState({ entries, failed: false }),
      () => open && setState((s) => ({ ...s, failed: true })),
    )
    return () => {
      open = false
    }
  }, [fhir, appointmentId, reads])

  return { ...state, reload: () => setReads((n) => n + 1) }
}

