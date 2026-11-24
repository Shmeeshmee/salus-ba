import type { components } from './api/schema'

export type DataRequest = components['schemas']['DataRequest']

// isOpen reports whether the request still waits for the clinic's answer (docs/clinic-cms/WAVE5.md D4).
export function isOpen(r: DataRequest): boolean {
  return r.status === 'received' || r.status === 'in_progress'
}
