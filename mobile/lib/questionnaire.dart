import 'package:salus_api/api.dart' as api;

import 'messages.dart';

// The questionnaires the patient fills in (docs/questionnaire/SPEC.md §2 "Item model", "Patient flow"): the FHIR
// Questionnaire and QuestionnaireResponse as the app reads them, the conditions evaluated as the server evaluates
// them (internal/questionnaireresponse/enabled.go), and each typed answer checked against the server's limits.

/// The item types of SPEC §2 "Item model". A type the app does not know (the reserved `attachment`) is
/// [unsupported] and not shown: the list of types is not assumed closed.
enum ItemType { group, display, string, text, boolean, choice, integer, decimal, date, unsupported }

/// An item's `enableWhen`: shown only when [question] was answered with [boolean], or with the option [string].
class Condition {
  const Condition(this.question, {this.boolean, this.string});

  final String question;
  final bool? boolean;
  final String? string;
}

class Item {
  const Item({
    required this.linkId,
    required this.type,
    this.text = '',
    this.required = false,
    this.repeats = false,
    this.options = const [],
    this.help,
    this.condition,
    this.items = const [],
  });

  final String linkId;
  final ItemType type;
  final String text;
  final bool required;

  /// A choice with several answers.
  final bool repeats;
  final List<String> options;

  /// The text of the item's `help` display child.
  final String? help;
  final Condition? condition;

  /// A section's questions and instructions.
  final List<Item> items;

  bool get isQuestion => !const {ItemType.group, ItemType.display, ItemType.unsupported}.contains(type);
}

/// A Questionnaire version.
class Version {
  const Version({required this.id, required this.url, required this.version, required this.title, required this.items});

  final String id;
  final String url;
  final String version;
  final String title;
  final List<Item> items;

  /// How a response names it: `<url>|<version>`.
  String get canonical => '$url|$version';

  factory Version.fromFhir(Map<String, dynamic> r) => Version(
    id: _string(r['id']),
    url: _string(r['url']),
    version: _string(r['version']),
    title: _string(r['title']),
    items: _items(r['item']),
  );
}

/// The answers by linkId. Each value is in its item's type: a String (string, text, choice), bool, int, num
/// (decimal), or a DateTime at UTC midnight (date).
typedef Answers = Map<String, List<Object>>;

enum ResponseStatus { inProgress, completed, stopped }

/// The patient's QuestionnaireResponse.
class Response {
  const Response({
    required this.id,
    required this.appointmentId,
    required this.questionnaire,
    required this.status,
    required this.versionId,
    required this.answers,
    this.authored,
  });

  final String id;
  final String appointmentId;

  /// The version answered, `<url>|<version>`.
  final String questionnaire;
  final ResponseStatus status;
  final int versionId;
  final Answers answers;
  final DateTime? authored;

  factory Response.fromFhir(Map<String, dynamic> r) {
    final answers = <String, List<Object>>{};
    void walk(Object? items) {
      for (final it in (items as List?) ?? const []) {
        final values = [for (final a in (it['answer'] as List?) ?? const []) ?_value(a as Map)];
        if (values.isNotEmpty) answers[_string(it['linkId'])] = values;
        walk(it['item']);
      }
    }

    walk(r['item']);
    final authored = r['authored'];
    return Response(
      id: _string(r['id']),
      appointmentId: _appointment(r['extension']),
      questionnaire: _string(r['questionnaire']),
      status: switch (r['status']) {
        'completed' => ResponseStatus.completed,
        'stopped' => ResponseStatus.stopped,
        _ => ResponseStatus.inProgress,
      },
      versionId: int.tryParse(_string(r['meta']?['versionId'])) ?? 0,
      answers: answers,
      authored: authored is String ? DateTime.parse(authored).toUtc() : null,
    );
  }
}

/// Which items are shown with [answers], by linkId: an item is enabled when its section is and its condition
/// holds on the answer to an enabled item. A condition names an earlier item, so one pass in document order
/// decides every item.
Map<String, bool> enabled(List<Item> items, Answers answers) {
  final on = <String, bool>{};
  void walk(List<Item> items, bool shown) {
    for (final it in items) {
      final e = shown && _holds(it.condition, on, answers);
      on[it.linkId] = e;
      walk(it.items, e);
    }
  }

  walk(items, true);
  return on;
}

bool _holds(Condition? c, Map<String, bool> on, Answers answers) {
  if (c == null) return true;
  if (on[c.question] != true) return false;
  return (answers[c.question] ?? const []).any((v) => c.boolean != null ? v == c.boolean : v == c.string);
}

/// Every item in document order, sections before their items.
Iterable<Item> flatten(List<Item> items) sync* {
  for (final it in items) {
    yield it;
    yield* flatten(it.items);
  }
}

/// The shown required questions without an answer, in document order, as the server names them on submit.
List<String> missing(List<Item> items, Answers answers) {
  final on = enabled(items, answers);
  return [
    for (final it in flatten(items))
      if (it.isQuestion && it.required && on[it.linkId]! && (answers[it.linkId] ?? const []).isEmpty) it.linkId,
  ];
}

/// The answers of the shown items, in document order: what the server keeps on submit.
Answers shownAnswers(List<Item> items, Answers answers) {
  final on = enabled(items, answers);
  return {
    for (final it in flatten(items))
      if (on[it.linkId]! && answers[it.linkId] != null) it.linkId: answers[it.linkId]!,
  };
}

// The answer limits of SPEC §2 "Limits".
const maxString = 255;
const maxText = 4000;
const _maxDecimals = 15;
const _minInt = -2147483648;
const _maxInt = 2147483647;
final _integer = RegExp(r'^-?\d+$');
final _decimal = RegExp(r'^-?(0|[1-9][0-9]*)(\.[0-9]+)?$');

/// The answer typed into a field of [type]: its value, or the M10 error; neither when the field is empty.
({Object? value, String? error}) readTyped(ItemType type, String typed) {
  if (type == ItemType.string || type == ItemType.text) {
    final limit = type == ItemType.string ? maxString : maxText;
    if (typed.trim().isEmpty) return (value: null, error: null);
    return typed.runes.length > limit ? (value: null, error: upToCharacters(limit)) : (value: typed, error: null);
  }
  final s = typed.trim();
  if (s.isEmpty) return (value: null, error: null);
  if (type == ItemType.integer) {
    if (!_integer.hasMatch(s)) return (value: null, error: enterWholeNumber);
    final n = int.tryParse(s);
    return n == null || n < _minInt || n > _maxInt ? (value: null, error: numberTooLarge) : (value: n, error: null);
  }
  return _readDecimal(s);
}

/// A plainly written decimal of at most 15 significant digits, as the server takes it. The API client writes a
/// decimal as a Dart number, which has an exponent below 0.000001, which the server refuses: such a number is
/// refused here.
({Object? value, String? error}) _readDecimal(String s) {
  if (!_decimal.hasMatch(s)) return (value: null, error: enterNumber);
  final whole = s.replaceFirst('-', '').split('.').first;
  if (whole.length > _maxDecimals) return (value: null, error: numberTooLarge);
  final digits = s.replaceFirst('-', '').replaceFirst('.', '').replaceFirst(RegExp('^0+'), '');
  final n = num.parse(s);
  if (digits.length > _maxDecimals || n.toString().contains('e')) return (value: null, error: enterNumber);
  return (value: n, error: null);
}

/// The answers as the JSON API takes them, in document order, each value in its item's type.
List<api.Answer> toApi(List<Item> items, Answers answers) => [
  for (final it in flatten(items))
    if (it.isQuestion && (answers[it.linkId] ?? const []).isNotEmpty)
      api.Answer(linkId: it.linkId, values: [for (final v in answers[it.linkId]!) _toValue(it.type, v)]),
];

api.AnswerValue _toValue(ItemType type, Object v) => switch (type) {
  ItemType.boolean => api.AnswerValue(valueBoolean: v as bool),
  ItemType.integer => api.AnswerValue(valueInteger: v as int),
  ItemType.decimal => api.AnswerValue(valueDecimal: v as num),
  ItemType.date => api.AnswerValue(valueDate: v as DateTime),
  _ => api.AnswerValue(valueString: v as String),
};

/// The JSON API's answers as [Answers], read by their items' types.
Answers fromApi(List<Item> items, List<api.Answer> answers) {
  final types = {for (final it in flatten(items)) it.linkId: it.type};
  return {
    for (final a in answers)
      a.linkId: [
        for (final v in a.values)
          ?switch (types[a.linkId]) {
            ItemType.boolean => v.valueBoolean,
            ItemType.integer => v.valueInteger,
            ItemType.decimal => v.valueDecimal,
            ItemType.date =>
              v.valueDate == null ? null : DateTime.utc(v.valueDate!.year, v.valueDate!.month, v.valueDate!.day),
            _ => v.valueString,
          },
      ],
  };
}

const _helpControl = 'http://hl7.org/fhir/StructureDefinition/questionnaire-itemControl';

List<Item> _items(Object? raw) => [
  for (final it in (raw as List?) ?? const [])
    if (!_isHelp(it as Map)) _item(it),
];

Item _item(Map it) {
  final children = (it['item'] as List?) ?? const [];
  final help = children.cast<Map>().where(_isHelp).firstOrNull;
  final when = (it['enableWhen'] as List?)?.firstOrNull as Map?;
  return Item(
    linkId: _string(it['linkId']),
    type: ItemType.values.firstWhere((t) => t.name == it['type'], orElse: () => ItemType.unsupported),
    text: _string(it['text']),
    required: it['required'] == true,
    repeats: it['repeats'] == true,
    options: [for (final o in (it['answerOption'] as List?) ?? const []) _string(o['valueString'])],
    help: help == null ? null : _string(help['text']),
    condition: when == null
        ? null
        : Condition(
            _string(when['question']),
            boolean: when['answerBoolean'] as bool?,
            string: when['answerString'] as String?,
          ),
    items: _items(children),
  );
}

bool _isHelp(Map it) =>
    it['type'] == 'display' &&
    ((it['extension'] as List?) ?? const []).any(
      (e) =>
          e['url'] == _helpControl && (e['valueCodeableConcept']?['coding'] as List?)?.firstOrNull?['code'] == 'help',
    );

const _appointmentExtension = 'https://salus.invalid/fhir/StructureDefinition/questionnaireresponse-appointment';

String _appointment(Object? extensions) {
  for (final e in (extensions as List?) ?? const []) {
    if (e['url'] == _appointmentExtension) {
      final ref = _string(e['valueReference']?['reference']);
      return ref.startsWith('Appointment/') ? ref.substring('Appointment/'.length) : '';
    }
  }
  return '';
}

Object? _value(Map a) {
  if (a['valueBoolean'] is bool) return a['valueBoolean'];
  if (a['valueInteger'] is int) return a['valueInteger'];
  if (a['valueDecimal'] is num) return a['valueDecimal'];
  if (a['valueDate'] is String) return parseDate(a['valueDate'] as String);
  return a['valueString'] as String?;
}

String _string(Object? v) => v is String ? v : '';
