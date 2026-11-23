import type { Appointment } from './api/fhir'

// The extension of a FHIR Appointment that names the price-list service the patient picked when booking
// (api/fhir/StructureDefinition-appointment-service.json, clinic-cms E).
export const serviceURL = 'https://salus.invalid/fhir/StructureDefinition/appointment-service'

// serviceOf is the id of the service the booking named, or undefined for none.
export function serviceOf(appt: Appointment): string | undefined {
  const ref = appt.extension?.find((e) => e.url === serviceURL)?.valueReference?.reference
  return ref?.startsWith('ChargeItemDefinition/') ? ref.slice('ChargeItemDefinition/'.length) : undefined
}
