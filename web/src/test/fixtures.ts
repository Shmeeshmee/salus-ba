import type { Account } from '../api/client'
import { listedURL, timeZoneURL, type Appointment, type Bundle, type Organization } from '../api/fhir'
import { fhirJSON } from './fakeFetch'

// Synthetic accounts, clinic and staff shared by the screen tests. Invented names, example.test addresses.
export const org = 'a0000000-0000-4000-8000-00000000000a'
export const meta = { versionId: '1', lastUpdated: '2026-11-04T09:30:00Z' }
export const cy = { practitionerId: 'a3000000-0000-4000-8000-0000000000a1', roleId: 'a4000000-0000-4000-8000-0000000000a1' }
export const dee = { practitionerId: 'a3000000-0000-4000-8000-0000000000a2', roleId: 'a4000000-0000-4000-8000-0000000000a2' }

export const admin: Account = { id: 'a1000000-0000-4000-8000-000000000001', email: 'admin@example.test', role: 'admin', organizationId: org, mfaEnrolled: true }
export const practitioner: Account = {
  ...admin,
  id: 'a1000000-0000-4000-8000-000000000002',
  email: 'cy@example.test',
  role: 'practitioner',
  practitionerId: cy.practitionerId,
}

export const clinic: Organization = {
  resourceType: 'Organization',
  id: org,
  name: 'Synthetic clinic',
  active: true,
  meta,
  extension: [
    { url: listedURL, valueBoolean: true },
    { url: timeZoneURL, valueCode: 'Europe/Sarajevo' },
  ],
}

export const clinicRoute = { [`GET /fhir/R4/Organization/${org}`]: () => fhirJSON(clinic) }

// staffBundle is the clinic's two practitioners, Cy Example and Dee Sample, with their roles.
export const staffBundle: Bundle = {
  resourceType: 'Bundle',
  type: 'searchset',
  total: 2,
  entry: [
    ...[cy, dee].map(({ roleId, practitionerId }) => ({
      fullUrl: `https://api.example.test/fhir/R4/PractitionerRole/${roleId}`,
      resource: {
        resourceType: 'PractitionerRole' as const,
        id: roleId,
        meta,
        active: true,
        practitioner: { reference: `Practitioner/${practitionerId}` },
        organization: { reference: `Organization/${org}` },
      },
    })),
    ...[
      [cy.practitionerId, 'Cy', 'Example'],
      [dee.practitionerId, 'Dee', 'Sample'],
    ].map(([id, given, family]) => ({
      fullUrl: `https://api.example.test/fhir/R4/Practitioner/${id}`,
      resource: { resourceType: 'Practitioner' as const, id, meta, active: true, name: [{ given: [given], family }] },
    })),
  ],
}

export const staffRoute = { 'GET /fhir/R4/PractitionerRole': () => fhirJSON(staffBundle) }

export const patientId = 'aa000000-0000-4000-8000-0000000000aa'

// appointment is a booked appointment with Cy Example; the patient's name only when given.
export function appointment(id: string, start: string, end: string, extra: Partial<Appointment> = {}, patientName?: string): Appointment {
  return {
    resourceType: 'Appointment',
    id,
    meta,
    status: 'booked',
    start,
    end,
    participant: [
      { status: 'accepted', actor: { reference: `Patient/${patientId}`, display: patientName } },
      { status: 'accepted', actor: { reference: `Practitioner/${cy.practitionerId}` } },
      { status: 'accepted', actor: { reference: `PractitionerRole/${cy.roleId}` } },
    ],
    ...extra,
  }
}

export function searchset<R>(resources: R[]): Bundle<R> {
  return {
    resourceType: 'Bundle',
    type: 'searchset',
    total: resources.length,
    entry: resources.map((resource) => ({ fullUrl: 'https://api.example.test/fhir/R4/x', resource, search: { mode: 'match' } })),
  }
}
