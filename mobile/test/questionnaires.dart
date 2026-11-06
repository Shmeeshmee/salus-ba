// Synthetic questionnaires and responses as the API sends them over FHIR. Placeholder wording, not clinical
// content; the ids are opaque.
import 'support.dart';

const questionnaireId = 'a7000000-0000-4000-8000-0000000000a1';
const seriesId = 'a6000000-0000-4000-8000-0000000000a1';
const questionnaireUrl = 'https://salus.invalid/fhir/Questionnaire/$seriesId';
const appointmentId = 'ab000000-0000-4000-8000-0000000000ab';
const appointmentUrl = 'https://salus.invalid/fhir/StructureDefinition/questionnaireresponse-appointment';
const _helpUrl = 'http://hl7.org/fhir/StructureDefinition/questionnaire-itemControl';

Map<String, Object?> help(String linkId, String text) => {
  'extension': [
    {
      'url': _helpUrl,
      'valueCodeableConcept': {
        'coding': [
          {'system': 'http://hl7.org/fhir/questionnaire-item-control', 'code': 'help', 'display': 'Help-Button'},
        ],
      },
    },
  ],
  'linkId': linkId,
  'text': text,
  'type': 'display',
};

Map<String, Object?> questionnaire({
  String id = questionnaireId,
  String series = seriesId,
  String version = '1',
  String title = 'Placeholder intake questionnaire',
  required List<Map<String, Object?>> items,
}) => {
  'resourceType': 'Questionnaire',
  'id': id,
  'meta': {'lastUpdated': '2026-11-01T09:30:00Z', 'versionId': '4'},
  'url': 'https://salus.invalid/fhir/Questionnaire/$series',
  'version': version,
  'title': title,
  'status': 'active',
  'subjectType': ['Patient'],
  'date': '2026-11-01T09:30:00Z',
  'item': items,
};

/// The API's golden active version (internal/questionnaire/testdata/questionnaire_active.json): every type,
/// help text on a question, a decimal and a section, a condition on a boolean and on a choice.
Map<String, Object?> everyType({String title = 'Placeholder intake questionnaire'}) => questionnaire(
  title: title,
  items: [
    {'linkId': 'intro', 'text': 'Placeholder instructions', 'type': 'display'},
    {
      'linkId': 'smoker',
      'text': 'Placeholder yes or no question',
      'type': 'boolean',
      'item': [help('smoker.help', 'Placeholder help text')],
      'required': true,
    },
    {
      'linkId': 'freq',
      'text': 'Placeholder single choice',
      'type': 'choice',
      'enableWhen': [
        {'answerBoolean': true, 'question': 'smoker', 'operator': '='},
      ],
      'answerOption': [
        {'valueString': 'Placeholder option A'},
        {'valueString': 'Placeholder option B'},
      ],
    },
    {
      'linkId': 'details',
      'text': 'Placeholder section',
      'type': 'group',
      'enableWhen': [
        {'question': 'freq', 'operator': '=', 'answerString': 'Placeholder option B'},
      ],
      'item': [
        {'linkId': 'name', 'text': 'Placeholder short answer', 'type': 'string', 'required': true},
        {'linkId': 'notes', 'text': 'Placeholder long answer', 'type': 'text'},
        {
          'linkId': 'many',
          'text': 'Placeholder multiple choice',
          'type': 'choice',
          'answerOption': [
            {'valueString': 'Placeholder option C'},
            {'valueString': 'Placeholder option D'},
            {'valueString': 'Placeholder option E'},
          ],
          'repeats': true,
        },
        {'linkId': 'count', 'text': 'Placeholder whole number', 'type': 'integer'},
        {
          'linkId': 'weight',
          'text': 'Placeholder decimal number',
          'type': 'decimal',
          'item': [help('weight.help', 'Placeholder unit hint')],
        },
        {'linkId': 'since', 'text': 'Placeholder date', 'type': 'date'},
        help('details.help', 'Placeholder section help'),
      ],
    },
  ],
);

/// A QuestionnaireResponse of the patient's, answering version 1 of the series for [appointment].
Map<String, Object?> response(
  String id, {
  String appointment = appointmentId,
  String status = 'in-progress',
  String version = '1',
  String series = seriesId,
  int versionId = 2,
  String? authored,
  List<Map<String, Object?>> items = const [],
}) => {
  'resourceType': 'QuestionnaireResponse',
  'id': id,
  'meta': {'lastUpdated': '2026-11-02T09:30:00Z', 'versionId': '$versionId'},
  'extension': [
    {
      'url': appointmentUrl,
      'valueReference': {'reference': 'Appointment/$appointment'},
    },
  ],
  'questionnaire': 'https://salus.invalid/fhir/Questionnaire/$series|$version',
  'status': status,
  'subject': {'reference': 'Patient/$patientId'},
  'authored': ?authored,
  'author': {'reference': 'Patient/$patientId'},
  if (items.isNotEmpty) 'item': items,
};

/// The JSON API's answer to a save or a submit.
Map<String, Object?> saved(
  String id, {
  String status = 'in-progress',
  int versionId = 3,
  List<Map<String, Object?>> answers = const [],
  String? authored,
}) => {
  'id': id,
  'appointmentId': appointmentId,
  'questionnaireId': questionnaireId,
  'status': status,
  'source': 'default',
  'answers': answers,
  'dueAt': '2026-11-05T08:00:00Z',
  'authored': ?authored,
  'lastUpdated': '2026-11-04T09:00:00Z',
  'versionId': versionId,
};
