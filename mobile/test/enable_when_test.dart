import 'package:flutter_test/flutter_test.dart';
import 'package:salus/questionnaire.dart';

// The branching of SAL-43 as FHIR R4 has it: `enableWhen` with several conditions, `enableBehavior` (all, any), the
// operators exists, =, !=, >, <, >=, <=; a branch is a group carrying the condition and joining back is the next
// unconditional item. Synthetic placeholder questions only.

Item _q(
  String linkId,
  ItemType type, {
  List<Condition> when = const [],
  EnableBehavior behavior = EnableBehavior.all,
}) => Item(linkId: linkId, type: type, text: 'Placeholder question', enableWhen: when, enableBehavior: behavior);

Item _group(String linkId, List<Item> items, {required List<Condition> when}) =>
    Item(linkId: linkId, type: ItemType.group, text: 'Placeholder section', enableWhen: when, items: items);

/// The ticket's example as placeholders: a yes or no question; yes opens a branch of a month and an age, with a
/// nested branch when the age is under 18; then the main questions everyone answers.
List<Item> example() => [
  _q('had', ItemType.boolean),
  _group(
    'yes-branch',
    [
      _q('month', ItemType.integer),
      _q('age', ItemType.integer),
      _group('young', [_q('young-q', ItemType.string)], when: const [Condition('age', Operator.less, 18)]),
    ],
    when: const [Condition('had', Operator.equal, true)],
  ),
  _q('main', ItemType.string),
];

void main() {
  group('the ticket example: branch out and join back', () {
    test('yes asks the branch, then the main questions', () {
      final on = enabled(example(), {
        'had': [true],
      });
      expect([on['yes-branch'], on['month'], on['age'], on['main']], [true, true, true, true]);
      expect(on['young'], isFalse, reason: 'no age yet');
    });

    test('no goes straight to the main questions', () {
      final on = enabled(example(), {
        'had': [false],
        'month': [3],
      });
      expect([on['yes-branch'], on['month'], on['age'], on['young-q']], [false, false, false, false]);
      expect(on['main'], isTrue);
    });

    test('a nested branch opens on a number and joins back', () {
      final on = enabled(example(), {
        'had': [true],
        'age': [17],
      });
      expect([on['young'], on['young-q'], on['main']], [true, true, true]);
      expect(
        enabled(example(), {
          'had': [true],
          'age': [18],
        })['young'],
        isFalse,
      );
    });

    test('the answer to a hidden question counts as none', () {
      final on = enabled(example(), {
        'had': [false],
        'age': [17],
      });
      expect(on['young'], isFalse);
    });
  });

  group('each operator', () {
    final date = DateTime.utc(2026, 5, 1);
    final cases = <({String name, Condition c, List<Object> given, bool on})>[
      (name: 'exists true, answered', c: const Condition('x', Operator.exists, true), given: ['a'], on: true),
      (name: 'exists true, unanswered', c: const Condition('x', Operator.exists, true), given: [], on: false),
      (name: 'exists false, unanswered', c: const Condition('x', Operator.exists, false), given: [], on: true),
      (name: 'exists false, answered', c: const Condition('x', Operator.exists, false), given: ['a'], on: false),
      (name: '= boolean', c: const Condition('x', Operator.equal, false), given: [false], on: true),
      (name: '= choice, any of several', c: const Condition('x', Operator.equal, 'B'), given: ['A', 'B'], on: true),
      (name: '= choice, none of several', c: const Condition('x', Operator.equal, 'B'), given: ['A', 'C'], on: false),
      (name: '= unanswered', c: const Condition('x', Operator.equal, 'B'), given: [], on: false),
      (name: '!= another option', c: const Condition('x', Operator.notEqual, 'B'), given: ['A'], on: true),
      (name: '!= the option', c: const Condition('x', Operator.notEqual, 'B'), given: ['A', 'B'], on: false),
      (
        name: '!= unanswered: false, as every operator but exists',
        c: const Condition('x', Operator.notEqual, 'B'),
        given: [],
        on: false,
      ),
      (name: '> integer', c: const Condition('x', Operator.greater, 20), given: [21], on: true),
      (name: '> integer, equal', c: const Condition('x', Operator.greater, 20), given: [20], on: false),
      (name: '>= integer, equal', c: const Condition('x', Operator.greaterOrEqual, 20), given: [20], on: true),
      (name: '< decimal', c: const Condition('x', Operator.less, 37.5), given: [37.4], on: true),
      (name: '<= decimal, integer answer', c: const Condition('x', Operator.lessOrEqual, 37.0), given: [37], on: true),
      (name: '= decimal and integer', c: const Condition('x', Operator.equal, 2), given: [2.0], on: true),
      (name: '!= number', c: const Condition('x', Operator.notEqual, 2), given: [3], on: true),
      (name: '< unanswered', c: const Condition('x', Operator.less, 18), given: [], on: false),
      (name: 'before a date', c: Condition('x', Operator.less, date), given: [DateTime.utc(2026, 4, 30)], on: true),
      (name: 'after a date', c: Condition('x', Operator.greater, date), given: [date], on: false),
      (name: 'on a date', c: Condition('x', Operator.equal, date), given: [DateTime.utc(2026, 5, 1)], on: true),
      (name: 'a type that does not fit', c: const Condition('x', Operator.greater, 3), given: ['4'], on: false),
      (
        name: 'an operator the app does not know',
        c: const Condition('x', Operator.unsupported, 'a'),
        given: ['a'],
        on: false,
      ),
    ];
    for (final c in cases) {
      test(c.name, () {
        final items = [
          _q('x', ItemType.string),
          _q('y', ItemType.string, when: [c.c]),
        ];
        expect(enabled(items, {if (c.given.isNotEmpty) 'x': c.given})['y'], c.on);
      });
    }
  });

  group('several conditions', () {
    final both = [const Condition('a', Operator.equal, true), const Condition('n', Operator.less, 18)];
    List<Item> items(EnableBehavior b) => [
      _q('a', ItemType.boolean),
      _q('n', ItemType.integer),
      _q('y', ItemType.string, when: both, behavior: b),
    ];
    final cases = <({String name, EnableBehavior b, Answers given, bool on})>[
      (
        name: 'all, both hold',
        b: EnableBehavior.all,
        given: {
          'a': [true],
          'n': [12],
        },
        on: true,
      ),
      (
        name: 'all, one holds',
        b: EnableBehavior.all,
        given: {
          'a': [true],
          'n': [30],
        },
        on: false,
      ),
      (
        name: 'any, one holds',
        b: EnableBehavior.any,
        given: {
          'a': [false],
          'n': [12],
        },
        on: true,
      ),
      (
        name: 'any, none holds',
        b: EnableBehavior.any,
        given: {
          'a': [false],
          'n': [30],
        },
        on: false,
      ),
      (name: 'any, nothing answered', b: EnableBehavior.any, given: {}, on: false),
    ];
    for (final c in cases) {
      test(c.name, () => expect(enabled(items(c.b), c.given)['y'], c.on));
    }
  });

  group('the answers a change would discard', () {
    test('the answered items it hides, in document order', () {
      expect(
        discarded(example(), {
          'had': [false],
          'young-q': ['Placeholder answer'],
          'month': [3],
          'main': ['Placeholder answer'],
        }),
        ['month', 'young-q'],
      );
    });

    test('none while the branch stays shown', () {
      expect(
        discarded(example(), {
          'had': [true],
          'month': [3],
        }),
        isEmpty,
      );
    });
  });

  group('the FHIR form', () {
    test('several conditions, their behaviour, operators and answer types', () {
      final v = Version.fromFhir({
        'resourceType': 'Questionnaire',
        'item': [
          {'linkId': 'n', 'type': 'integer', 'text': 'Placeholder number'},
          {
            'linkId': 'y',
            'type': 'string',
            'text': 'Placeholder question',
            'enableBehavior': 'any',
            'enableWhen': [
              {'question': 'n', 'operator': '>=', 'answerInteger': 18},
              {'question': 'n', 'operator': '<', 'answerDecimal': 2.5},
              {'question': 'd', 'operator': '>', 'answerDate': '2026-05-01'},
              {'question': 'c', 'operator': '!=', 'answerString': 'Placeholder option'},
              {'question': 'b', 'operator': '=', 'answerBoolean': false},
              {'question': 'b', 'operator': 'exists', 'answerBoolean': true},
              {'question': 'b', 'operator': 'near', 'answerBoolean': true},
            ],
          },
        ],
      });
      final y = v.items[1];
      expect(y.enableBehavior, EnableBehavior.any);
      expect(
        [for (final c in y.enableWhen) (c.question, c.operator, c.answer)],
        [
          ('n', Operator.greaterOrEqual, 18),
          ('n', Operator.less, 2.5),
          ('d', Operator.greater, DateTime.utc(2026, 5, 1)),
          ('c', Operator.notEqual, 'Placeholder option'),
          ('b', Operator.equal, false),
          ('b', Operator.exists, true),
          ('b', Operator.unsupported, true),
        ],
      );
      expect(v.items[0].enableBehavior, EnableBehavior.all, reason: 'all when the resource names none');
    });
  });
}
