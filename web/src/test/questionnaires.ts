import type { Questionnaire } from '../api/fhir'
import type { components } from '../api/schema'
import type { FHIRQuestionnaire } from '../questionnaire'

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

// The same questionnaire written in Bosnian, with an English translation that lacks the multiple choice question
// and a German one with the title only, as the FHIR read carries them (the R4 translation extension) and as the
// JSON API answers them.
const translation = (lang: string, content: string) => ({
  url: 'http://hl7.org/fhir/StructureDefinition/translation',
  extension: [
    { url: 'lang', valueCode: lang },
    { url: 'content', valueString: content },
  ],
})
const inEnglish = (content: string) => ({ extension: [translation('en', content)] })

export function translatedFHIR(): FHIRQuestionnaire {
  const q = questionnaireFHIR()
  const [intro, smoker, sec] = q.item ?? []
  const [one, many, secHelp] = sec.item ?? []
  const [smokerHelp] = smoker.item ?? []
  return {
    ...q,
    language: 'bs',
    _title: { extension: [translation('en', 'Placeholder intake in English'), translation('de', 'Placeholder intake in German')] },
    item: [
      { ...intro, _text: inEnglish('Placeholder instructions in English') },
      { ...smoker, _text: inEnglish('Placeholder yes or no in English'), item: [{ ...smokerHelp, _text: inEnglish('Placeholder help in English') }] },
      {
        ...sec,
        _text: inEnglish('Placeholder section in English'),
        item: [
          {
            ...one,
            _text: inEnglish('Placeholder single in English'),
            answerOption: [
              { valueString: 'A', _valueString: inEnglish('A in English') },
              { valueString: 'B', _valueString: inEnglish('B in English') },
            ],
          },
          many,
          { ...secHelp, _text: inEnglish('Placeholder section help in English') },
        ],
      },
    ],
  }
}

export function translatedJSON(): JSONQuestionnaire {
  return {
    ...questionnaireJSON(),
    language: 'bs',
    translations: [
      {
        language: 'en',
        title: 'Placeholder intake in English',
        items: {
          intro: { text: 'Placeholder instructions in English' },
          smoker: { text: 'Placeholder yes or no in English', help: 'Placeholder help in English' },
          sec: { text: 'Placeholder section in English', help: 'Placeholder section help in English' },
          one: { text: 'Placeholder single in English', options: ['A in English', 'B in English'] },
        },
      },
      { language: 'de', title: 'Placeholder intake in German', items: {} },
    ],
  }
}
