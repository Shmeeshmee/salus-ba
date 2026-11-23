import type { components } from '../api/schema'
import { fhirJSON } from './fakeFetch'
import { cy, dee, org } from './fixtures'

// Synthetic desk data (clinic-cms F3): invented patients, today Monday 23 November 2026 in Sarajevo (UTC+1).
type ClinicPatient = components['schemas']['ClinicPatient']
type DeskVisit = components['schemas']['DeskVisit']

export const now = () => new Date('2026-11-23T08:20:00Z')
export const exam = 'c1000000-0000-4000-8000-000000000001'
export const videoOnly = 'c1000000-0000-4000-8000-000000000002'
export const retired = 'c1000000-0000-4000-8000-000000000003'

export const ana: ClinicPatient = {
  id: 'b0000000-0000-4000-8000-0000000000b1', givenName: 'Ana', familyName: 'Example', birthDate: '1980-03-02', phone: '+38761000001',
  registeredByClinic: false, versionId: 1, lastUpdated: '2026-11-01T09:00:00Z',
}
export const bo: ClinicPatient = {
  id: 'b0000000-0000-4000-8000-0000000000b2', givenName: 'Bo', familyName: 'Sample', birthDate: '1992-07-14',
  registeredByClinic: true, versionId: 1, lastUpdated: '2026-11-01T09:00:00Z',
}
export const cleo: ClinicPatient = {
  id: 'b0000000-0000-4000-8000-0000000000b3', givenName: 'Cleo', familyName: 'Test', birthDate: '2001-11-30',
  registeredByClinic: true, versionId: 1, lastUpdated: '2026-11-01T09:00:00Z',
}

export const services = [
  { id: exam, name: 'Placeholder examination', priceMinor: 5000, currency: 'BAM', modes: ['in-person', 'video'], active: true, translations: [], versionId: 1 },
  { id: videoOnly, name: 'Placeholder video follow-up', priceMinor: 3000, currency: 'BAM', modes: ['video'], active: true, translations: [], versionId: 1 },
  { id: retired, name: 'Placeholder retired item', priceMinor: 100, currency: 'BAM', modes: ['in-person'], active: false, translations: [], versionId: 1 },
]

function visit(id: string, roleId: string, start: string, end: string, patient: ClinicPatient, extra: Partial<DeskVisit['appointment']> = {}, serviceName?: string): DeskVisit {
  return {
    appointment: {
      id, organizationId: org, practitionerRoleId: roleId, slotId: id.replace(/^e/, 'f'), start, end, status: 'booked', versionId: 1,
      bookedVia: 'direct', rescheduleCount: 0, visitMode: 'in-person', ...extra,
    },
    patient,
    serviceName,
  }
}

export const anaVisit = visit('e0000000-0000-4000-8000-0000000000e1', cy.roleId, '2026-11-23T08:00:00Z', '2026-11-23T08:30:00Z', ana, { serviceId: exam }, 'Placeholder examination')
export const boVisit = visit('e0000000-0000-4000-8000-0000000000e2', cy.roleId, '2026-11-23T09:00:00Z', '2026-11-23T09:30:00Z', bo, { visitMode: 'video' })
export const cleoVisit = visit('e0000000-0000-4000-8000-0000000000e3', dee.roleId, '2026-11-23T08:30:00Z', '2026-11-23T09:00:00Z', cleo, {
  status: 'arrived', arrivedAt: '2026-11-23T08:25:00Z',
})

export const deskDay = { date: '2026-11-23', visits: [anaVisit, boVisit, cleoVisit] }

export const cySchedule = 'd0000000-0000-4000-8000-0000000000d1'
const slotModeURL = 'https://salus.invalid/fhir/StructureDefinition/visit-mode'

function slot(id: string, start: string, end: string, mode: 'in-person' | 'video') {
  return {
    fullUrl: `https://api.example.test/fhir/R4/Slot/${id}`,
    resource: {
      resourceType: 'Slot', id, status: 'free', start, end, schedule: { reference: `Schedule/${cySchedule}` },
      extension: [{ url: slotModeURL, valueCode: mode }], meta: { versionId: '1', lastUpdated: '2026-11-01T09:00:00Z' },
    },
  }
}

export const freeInPerson = 'f0000000-0000-4000-8000-0000000000f7'
export const freeVideo = 'f0000000-0000-4000-8000-0000000000f8'

export const scheduleRoute = () =>
  fhirJSON({
    resourceType: 'Bundle', type: 'searchset', total: 1,
    entry: [{ fullUrl: `https://api.example.test/fhir/R4/Schedule/${cySchedule}`, resource: { resourceType: 'Schedule', id: cySchedule, actor: [{ reference: `PractitionerRole/${cy.roleId}` }] } }],
  })

export const slotsRoute = () =>
  fhirJSON({
    resourceType: 'Bundle', type: 'searchset', total: 2,
    entry: [slot(freeInPerson, '2026-11-23T10:00:00Z', '2026-11-23T10:30:00Z', 'in-person'), slot(freeVideo, '2026-11-23T10:30:00Z', '2026-11-23T11:00:00Z', 'video')],
  })
