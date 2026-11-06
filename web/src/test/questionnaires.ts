import type { Questionnaire } from '../api/fhir'
import type { components } from '../api/schema'

// A synthetic questionnaire in both forms the web app reads: FHIR (the library and the builder's read) and
// the JSON API's answer (after a save). Placeholder wording, no clinical content.
const helpControl = {
  url: 'http://hl7.org/fhir/StructureDefinition/questionnaire-itemControl',
  valueCodeableConcept: { coding: [{ system: 'http://hl7.org/fhir/questionnaire-item-control', code: 'help', display: 'Help-Button' }] },
}

export function questionnaireFHIR(): Questionnaire {
  return {
    resourceType: 'Questionnaire',
    id: 'q1',
    meta: { versionId: '3', lastUpdated: '2026-11-06T09:30:00Z' },
    url: 'https://salus.invalid/fhir/Questionnaire/s1',
    version: '1',
    title: 'Placeholder intake',
    status: 'draft',
    item: [
      { linkId: 'intro', text: 'Placeholder instructions', type: 'display' },
      {
        linkId: 'smoker',
        text: 'Placeholder yes or no',
        type: 'boolean',
        required: true,
        item: [{ extension: [helpControl], linkId: 'smoker.help', text: 'Placeholder help', type: 'display' }],
      },
      {
        linkId: 'sec',
        text: 'Placeholder section',
        type: 'group',
        enableWhen: [{ question: 'smoker', operator: '=', answerBoolean: true }],
        item: [
          { linkId: 'one', text: 'Placeholder single', type: 'choice', answerOption: [{ valueString: 'A' }, { valueString: 'B' }] },
          {
            linkId: 'many',
            text: 'Placeholder multiple',
            type: 'choice',
            repeats: true,
            enableWhen: [{ question: 'one', operator: '=', answerString: 'B' }],
            answerOption: [{ valueString: 'C' }, { valueString: 'D' }],
          },
          { extension: [helpControl], linkId: 'sec.help', text: 'Placeholder section help', type: 'display' },
        ],
      },
    ],
  }
}

type JSONQuestionnaire = components['schemas']['Questionnaire']

export function questionnaireJSON(): JSONQuestionnaire {
  return {
    id: 'q1',
    url: 'https://salus.invalid/fhir/Questionnaire/s1',
    status: 'draft',
    title: 'Placeholder intake',
    lastUpdated: '2026-11-06T09:30:00Z',
    version: 1,
    versionId: 3,
    item: [
      { linkId: 'intro', text: 'Placeholder instructions', type: 'display' },
      { linkId: 'smoker', text: 'Placeholder yes or no', type: 'boolean', required: true, help: 'Placeholder help' },
      {
        linkId: 'sec',
        text: 'Placeholder section',
        type: 'group',
        help: 'Placeholder section help',
        enableWhen: { question: 'smoker', answerBoolean: true },
        item: [
          { linkId: 'one', text: 'Placeholder single', type: 'choice', answerOption: [{ valueString: 'A' }, { valueString: 'B' }] },
          {
            linkId: 'many',
            text: 'Placeholder multiple',
            type: 'choice',
            repeats: true,
            answerOption: [{ valueString: 'C' }, { valueString: 'D' }],
            enableWhen: { question: 'one', answerString: 'B' },
          },
        ],
      },
    ],
  }
}
