import { ids } from './sections'

export const standards = {
  id: ids.standards,
  eyebrow: 'Standards',
  heading: 'Built on the standards your data deserves.',
  body: 'Records follow HL7 FHIR, so they move cleanly between systems. Processing is designed around GDPR from day one.',
  // The last two are placeholders until hosting and certification are settled.
  tiles: [
    { title: 'HL7 FHIR', detail: 'Interoperable records' },
    { title: 'GDPR', detail: 'Privacy by design' },
    { title: '[Hosting]', detail: '[Where data is stored]' },
    { title: '[Certification]', detail: '[Audit or standard]' },
  ],
} as const
