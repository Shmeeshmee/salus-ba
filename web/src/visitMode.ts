import type { Appointment } from './api/fhir'
import type { components } from './api/schema'

export type VisitMode = components['schemas']['VisitMode']

export const visitModes: VisitMode[] = ['video', 'in-person']

// The extension of a FHIR Appointment (and Slot) that says how the patient attends (ADR 0016).
export const visitModeURL = 'https://salus.invalid/fhir/StructureDefinition/visit-mode'

// visitModeOf is how the patient attends the appointment: at the clinic, or by video, which is also every appointment
// without the extension.
export function visitModeOf(appt: Appointment): VisitMode {
  return appt.extension?.find((e) => e.url === visitModeURL)?.valueCode === 'in-person' ? 'in-person' : 'video'
}
