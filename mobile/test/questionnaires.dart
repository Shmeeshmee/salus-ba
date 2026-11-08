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
  String language = 'en',
  required List<Map<String, Object?>> items,
}) => {
  'resourceType': 'Questionnaire',
  'id': id,
  'meta': {'lastUpdated': '2026-11-01T09:30:00Z', 'versionId': '4'},
  'language': language,
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

const _translationUrl = 'http://hl7.org/fhir/StructureDefinition/translation';

/// [q] written in [language] and translated into [into], as the API's FHIR handler writes a patient's read
/// (internal/questionnaire/testdata/questionnaire_translated.json): the R4 `translation` extension on the title,
/// every item's text (help included) and every option. Each translated text is `<tag>: <original>`.
Map<String, Object?> translated(Map<String, Object?> q, {String language = 'bs', List<String> into = const ['en']}) {
  Map<String, Object?> ext(Object? text) => {
    'extension': [
      for (final l in into)
        {
          'url': _translationUrl,
          'extension': [
            {'url': 'lang', 'valueCode': l},
            {'url': 'content', 'valueString': '$l: $text'},
          ],
        },
    ],
  };
  Map<String, Object?> item(Map<String, Object?> it) => {
    ...it,
    if (it['text'] != null) '_text': ext(it['text']),
    if (it['answerOption'] != null)
      'answerOption': [
        for (final o in (it['answerOption'] as List).cast<Map<String, Object?>>())
          {...o, '_valueString': ext(o['valueString'])},
      ],
    if (it['item'] != null) 'item': [for (final c in (it['item'] as List).cast<Map<String, Object?>>()) item(c)],
  };
  if (into.isEmpty) return {...q, 'language': language};
  return {
    ...q,
    'language': language,
    '_title': ext(q['title']),
    'item': [for (final it in (q['item'] as List).cast<Map<String, Object?>>()) item(it)],
  };
}

/// A QuestionnaireResponse of the patient's, answering version 1 of the series for [appointment], started in
/// [language] (null: not started yet).
Map<String, Object?> response(
  String id, {
  String? language = 'en',
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
  'language': ?language,
  'questionnaire': 'https://salus.invalid/fhir/Questionnaire/$series|$version',
  'status': status,
  'subject': {'reference': 'Patient/$patientId'},
  'authored': ?authored,
  'author': {'reference': 'Patient/$patientId'},
  if (items.isNotEmpty) 'item': items,
};

/// The JSON API's answer to a start, a save or a submit.
Map<String, Object?> saved(
  String id, {
  String? language,
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
  'language': ?language,
  'lastUpdated': '2026-11-04T09:00:00Z',
  'versionId': versionId,
};

const _mimeType = 'http://hl7.org/fhir/StructureDefinition/mimeType';
const _maxOccurs = 'http://hl7.org/fhir/StructureDefinition/questionnaire-maxOccurs';

/// The content types of each kind, as the API's FHIR handler writes an attachment item's `mimeType` extensions.
const contentTypes = {
  'photo': ['image/jpeg', 'image/png'],
  'document': ['application/pdf'],
  'video': ['video/mp4'],
};

/// A file upload item accepting [accept] (`photo`, `document`, `video`), at most [maxFiles] files.
Map<String, Object?> attachmentItem(
  String linkId, {
  List<String> accept = const ['photo', 'document', 'video'],
  int maxFiles = 1,
  bool required = false,
  String text = 'Placeholder file upload',
}) => {
  'linkId': linkId,
  'text': text,
  'type': 'attachment',
  if (required) 'required': true,
  if (maxFiles > 1) 'repeats': true,
  'extension': [
    for (final kind in accept)
      for (final t in contentTypes[kind]!) {'url': _mimeType, 'valueCode': t},
    {'url': _maxOccurs, 'valueInteger': maxFiles},
  ],
};

/// A file answer as the API's FHIR handler writes it: `valueAttachment` with the download URL. Synthetic title.
Map<String, Object?> fileAnswer(
  String binaryId, {
  String contentType = 'image/jpeg',
  int size = 245000,
  String? title,
}) => {
  'valueAttachment': {
    'contentType': contentType,
    'size': size,
    'title': title ?? 'synthetic-$binaryId.jpg',
    'hash': 'c3ludGhldGljIGhhc2ggMjA=',
    'url': 'http://api.test/binaries/$binaryId/content',
  },
};
