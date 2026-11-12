import type { AllergyIntolerance, MedicationStatement } from '../entries'
import type { Bundle } from '../api/fhir'

// Synthetic FHIR AllergyIntolerance and MedicationStatement resources shaped as the API's golden files
// (internal/allergyintolerance/testdata, internal/medicationstatement/testdata). Every text is a placeholder.

export const patientID = 'd1000000-0000-4000-8000-0000000000d1'
const meta = (versionId: string) => ({ versionId, lastUpdated: '2026-11-03T09:30:00Z' })
const clinical = (code: string) => ({ coding: [{ system: 'http://terminology.hl7.org/CodeSystem/allergyintolerance-clinical', code }] })
const verification = (code: string) => ({ coding: [{ system: 'http://terminology.hl7.org/CodeSystem/allergyintolerance-verification', code }] })
export const confirmedURL = 'https://salus.invalid/fhir/StructureDefinition/medicationstatement-confirmed'

export function ownAllergy(p: Partial<AllergyIntolerance> = {}): AllergyIntolerance {
  return {
    resourceType: 'AllergyIntolerance',
    id: 'a1000000-0000-4000-8000-0000000000a1',
    meta: meta('1'),
    patient: { reference: `Patient/${patientID}` },
    code: { text: 'Placeholder substance A', coding: [{ system: 'http://www.whocc.no/atc', code: 'J01CA04' }] },
    category: ['medication'],
    reaction: [{ severity: 'moderate', manifestation: [{ text: 'Placeholder reaction' }] }],
    clinicalStatus: clinical('active'),
    verificationStatus: verification('unconfirmed'),
    ...p,
  }
}

export function clinicAllergy(p: Partial<AllergyIntolerance> = {}): AllergyIntolerance {
  return {
    resourceType: 'AllergyIntolerance',
    id: 'a1000000-0000-4000-8000-0000000000a2',
    meta: meta('3'),
    patient: { reference: `Patient/${patientID}` },
    code: { text: 'Placeholder substance B' },
    category: ['food'],
    clinicalStatus: clinical('active'),
    verificationStatus: verification('confirmed'),
    ...p,
  }
}

export function ownMedication(p: Partial<MedicationStatement> = {}): MedicationStatement {
  return {
    resourceType: 'MedicationStatement',
    id: 'b1000000-0000-4000-8000-0000000000b1',
    meta: meta('1'),
    status: 'active',
    subject: { reference: `Patient/${patientID}` },
    medicationCodeableConcept: { text: 'Placeholder medication A', coding: [{ system: 'http://www.whocc.no/atc', code: 'N02BE01' }] },
    dosage: [
      {
        timing: { repeat: { frequency: 2, period: 1, periodUnit: 'd' } },
        doseAndRate: [{ doseQuantity: { value: 500, unit: 'mg', system: 'http://unitsofmeasure.org', code: 'mg' } }],
      },
    ],
    ...p,
  }
}

export function clinicMedication(p: Partial<MedicationStatement> = {}): MedicationStatement {
  return {
    resourceType: 'MedicationStatement',
    id: 'b1000000-0000-4000-8000-0000000000b2',
    meta: meta('2'),
    status: 'stopped',
    subject: { reference: `Patient/${patientID}` },
    medicationCodeableConcept: { text: 'Placeholder medication B' },
    extension: [{ url: confirmedURL, valueBoolean: true }],
    dosage: [{ asNeededBoolean: true }],
    ...p,
  }
}

export function allergyBundle(...list: AllergyIntolerance[]): Bundle<AllergyIntolerance> {
  return { resourceType: 'Bundle', type: 'searchset', entry: list.map((r) => ({ fullUrl: `AllergyIntolerance/${r.id}`, resource: r })) }
}

export function medicationBundle(...list: MedicationStatement[]): Bundle<MedicationStatement> {
  return { resourceType: 'Bundle', type: 'searchset', entry: list.map((r) => ({ fullUrl: `MedicationStatement/${r.id}`, resource: r })) }
}
