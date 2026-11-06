import type { Bundle, Questionnaire, QuestionnaireResponse } from '../api/fhir'

// Synthetic QuestionnaireResponses as the FHIR search answers them, answering the synthetic questionnaire of
// ./questionnaires (version 1 of series s1) unless told otherwise. Placeholder answers only.
export const appointmentRef = 'Appointment/e1000000-0000-4000-8000-000000000001'

export function responseFHIR(id: string, status: QuestionnaireResponse['status'], extra: Partial<QuestionnaireResponse> = {}): QuestionnaireResponse {
  return {
    resourceType: 'QuestionnaireResponse',
    id,
    meta: { versionId: '2', lastUpdated: '2026-11-06T09:30:00Z' },
    extension: [{ url: 'https://salus.invalid/fhir/StructureDefinition/questionnaireresponse-appointment', valueReference: { reference: appointmentRef } }],
    questionnaire: 'https://salus.invalid/fhir/Questionnaire/s1|1',
    status,
    subject: { reference: 'Patient/aa000000-0000-4000-8000-0000000000aa' },
    author: { reference: 'Patient/aa000000-0000-4000-8000-0000000000aa' },
    ...extra,
  }
}

// responsesBundle is the search answer: the responses as matches, the versions as includes.
export function responsesBundle(responses: QuestionnaireResponse[], versions: Questionnaire[]): Bundle<QuestionnaireResponse | Questionnaire> {
  return {
    resourceType: 'Bundle',
    type: 'searchset',
    entry: [
      ...responses.map((resource) => ({ fullUrl: `https://api.example.test/fhir/R4/QuestionnaireResponse/${resource.id}`, resource, search: { mode: 'match' as const } })),
      ...versions.map((resource) => ({ fullUrl: `https://api.example.test/fhir/R4/Questionnaire/${resource.id}`, resource, search: { mode: 'include' as const } })),
    ],
  }
}
