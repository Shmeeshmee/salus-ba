import 'questionnaire.dart' show Answers, Item, flatten;

// Which items are shown (SAL-43, docs/clinic-cms/IMPLEMENTATION_PLAN.md "Q1, Q2, Q3"): FHIR R4 `enableWhen` with
// several conditions, `enableBehavior` all or any, and the operators exists, =, !=, >, <, >=, <=. A branch is a group
// carrying the condition; joining back is the next unconditional item. A condition names an earlier question, so one
// pass in document order decides every item. The server evaluates the same rules (internal/questionnaireresponse).

/// The R4 `enableWhen.operator` codes; one the app does not know is [unsupported] and never holds.
enum Operator { exists, equal, notEqual, greater, less, greaterOrEqual, lessOrEqual, unsupported }

/// How several conditions combine (`enableBehavior`): all of them or any of them.
enum EnableBehavior { all, any }

/// One `enableWhen`: [question]'s answers compared with [answer] by [operator]. The answer is in the question's
/// type, as an answer is (bool, int, num, a DateTime at UTC midnight, or a String option).
class Condition {
  const Condition(this.question, this.operator, this.answer);

  final String question;
  final Operator operator;
  final Object? answer;
}

Operator operatorOf(Object? code) => switch (code) {
  'exists' => Operator.exists,
  '=' => Operator.equal,
  '!=' => Operator.notEqual,
  '>' => Operator.greater,
  '<' => Operator.less,
  '>=' => Operator.greaterOrEqual,
  '<=' => Operator.lessOrEqual,
  _ => Operator.unsupported,
};

/// Which items are shown with [answers], by linkId: an item is enabled when its section is and its conditions hold
/// on the answers to enabled items. The answer to a hidden item counts as none (R4).
Map<String, bool> enabled(List<Item> items, Answers answers) {
  final on = <String, bool>{};
  void walk(List<Item> items, bool shown) {
    for (final it in items) {
      final e = shown && _holds(it, (q) => on[q] == true ? answers[q] ?? const [] : const []);
      on[it.linkId] = e;
      walk(it.items, e);
    }
  }

  walk(items, true);
  return on;
}

/// The items with an answer in [answers] that are hidden with them, in document order: what a change that led to
/// [answers] would discard.
List<String> discarded(List<Item> items, Answers answers) {
  final on = enabled(items, answers);
  return [
    for (final it in flatten(items))
      if (!on[it.linkId]! && (answers[it.linkId] ?? const []).isNotEmpty) it.linkId,
  ];
}

bool _holds(Item it, List<Object> Function(String question) answersOf) {
  if (it.enableWhen.isEmpty) return true;
  bool test(Condition c) => _meets(c, answersOf(c.question));
  return it.enableBehavior == EnableBehavior.any ? it.enableWhen.any(test) : it.enableWhen.every(test);
}

bool _meets(Condition c, List<Object> given) => switch (c.operator) {
  Operator.exists => given.isNotEmpty == c.answer,
  Operator.equal => given.any((v) => _compare(v, c.answer) == 0),
  // False on an unanswered or hidden question, as every operator but exists (test/testdata/enablewhen).
  Operator.notEqual => given.isNotEmpty && !given.any((v) => _compare(v, c.answer) == 0),
  Operator.greater => given.any((v) => (_compare(v, c.answer) ?? 0) > 0),
  Operator.less => given.any((v) => (_compare(v, c.answer) ?? 0) < 0),
  Operator.greaterOrEqual => given.any((v) => (_compare(v, c.answer) ?? -1) >= 0),
  Operator.lessOrEqual => given.any((v) => (_compare(v, c.answer) ?? 1) <= 0),
  Operator.unsupported => false,
};

/// [a] against [b]: negative, zero or positive, or null when they are not of one type. Numbers compare across int
/// and decimal; a yes or no answer only equals.
int? _compare(Object a, Object? b) => switch ((a, b)) {
  (final num x, final num y) => x.compareTo(y),
  (final DateTime x, final DateTime y) => x.compareTo(y),
  (final String x, final String y) => x == y ? 0 : null,
  (final bool x, final bool y) => x == y ? 0 : null,
  _ => null,
};
