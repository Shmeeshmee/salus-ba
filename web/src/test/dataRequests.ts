import type { components } from '../api/schema'
import { admin } from './fixtures'

type DataRequest = components['schemas']['DataRequest']

const patientId = 'aa000000-0000-4000-8000-0000000000aa'
// Synthetic requests targeting the clinic, as the API orders them: the open ones by due date, then the answered.
export const requests: DataRequest[] = [
  { id: 'f1000000-0000-4000-8000-000000000001', kind: 'erasure', target: 'clinic', organizationId: admin.organizationId, patientId, status: 'received',
    receivedAt: '2026-11-02T09:00:00Z', dueAt: '2026-12-02T09:00:00Z', filedByPatient: true, version: 1 },
  { id: 'f1000000-0000-4000-8000-000000000002', kind: 'access', target: 'clinic', organizationId: admin.organizationId, patientId, status: 'in_progress',
    receivedAt: '2026-10-01T09:00:00Z', dueAt: '2026-11-01T09:00:00Z', extendedUntil: '2027-01-01T09:00:00Z', extensionReason: 'complex', filedByPatient: false, version: 2 },
  { id: 'f1000000-0000-4000-8000-000000000003', kind: 'objection', target: 'clinic', organizationId: admin.organizationId, patientId, status: 'fulfilled',
    receivedAt: '2026-09-01T09:00:00Z', dueAt: '2026-10-01T09:00:00Z', answeredAt: '2026-09-03T09:00:00Z', answerCode: 'done', filedByPatient: true, version: 2 },
]
