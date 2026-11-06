import 'package:flutter_test/flutter_test.dart';
import 'package:salus/questionnaire.dart';
import 'package:salus_api/api.dart' as api;

import 'questionnaires.dart';

// The conditions are evaluated on the device as the server evaluates them (internal/questionnaireresponse/
// enabled.go), against the same cases as its TestSubmit_EvaluatesTheConditions: missing names the enabled
// required questions without an answer, kept the answers left once hidden items' answers are removed.

Item _question(String linkId, ItemType type, {Condition? when, bool required = false, List<String>? options}) => Item(
  linkId: linkId,
  text: 'Placeholder question',
  type: type,
  required: required,
  condition: when,
  options: options ?? const [],
  repeats: false,
);

/// The server test's `conditional()`: a boolean, a single choice, a choice with repeats, a section shown on the
/// boolean, and a chain of two (x1 on s2, in the section). Every conditional question is required. Synthetic.
List<Item> conditional() => [
  _question('b', ItemType.boolean),
  _question('b1', ItemType.string, when: const Condition('b', boolean: true), required: true),
  _question('c', ItemType.choice, options: const ['Option A', 'Option B']),
  _question('c1', ItemType.string, when: const Condition('c', string: 'Option B'), required: true),
  const Item(
    linkId: 'm',
    text: 'Placeholder question',
    type: ItemType.choice,
    repeats: true,
    options: ['Option A', 'Option B', 'Option C'],
  ),
  _question('m1', ItemType.string, when: const Condition('m', string: 'Option B'), required: true),
  Item(
    linkId: 'sec',
    text: 'Placeholder section',
    type: ItemType.group,
    condition: const Condition('b', boolean: false),
    items: [_question('s1', ItemType.string, required: true), _question('s2', ItemType.boolean)],
  ),
  _question('x1', ItemType.string, when: const Condition('s2', boolean: true), required: true),
];

void main() {
  group('the conditions, as on the server', () {
    const v = 'Placeholder';
    final cases = <({String name, Answers given, List<String> missing, List<String> kept})>[
      (
        name: 'a boolean shows its question',
        given: {
          'b': [true],
        },
        missing: ['b1'],
        kept: [],
      ),
      (
        name: "a boolean's question answered",
        given: {
          'b': [true],
          'b1': [v],
        },
        missing: [],
        kept: ['b', 'b1'],
      ),
      (
        name: 'a boolean hides its question, shows the section',
        given: {
          'b': [false],
          'b1': [v],
        },
        missing: ['s1'],
        kept: [],
      ),
      (
        name: 'a chain through the section',
        given: {
          'b': [false],
          's1': [v],
          's2': [true],
        },
        missing: ['x1'],
        kept: [],
      ),
      (
        name: 'the chain answered',
        given: {
          'b': [false],
          's1': [v],
          's2': [true],
          'x1': [v],
        },
        missing: [],
        kept: ['b', 's1', 's2', 'x1'],
      ),
      (
        name: 'the chain off',
        given: {
          'b': [false],
          's1': [v],
          's2': [false],
          'x1': [v],
        },
        missing: [],
        kept: ['b', 's1', 's2'],
      ),
      (
        name: 'a hidden section hides its items and the chain',
        given: {
          'b': [true],
          'b1': [v],
          's1': [v],
          's2': [true],
          'x1': [v],
        },
        missing: [],
        kept: ['b', 'b1'],
      ),
      (
        name: 'a single choice shows its question',
        given: {
          'b': [true],
          'b1': [v],
          'c': ['Option B'],
        },
        missing: ['c1'],
        kept: [],
      ),
      (
        name: 'a single choice hides its question',
        given: {
          'b': [true],
          'b1': [v],
          'c': ['Option A'],
          'c1': [v],
        },
        missing: [],
        kept: ['b', 'b1', 'c'],
      ),
      (
        name: 'any of several choices shows its question',
        given: {
          'b': [true],
          'b1': [v],
          'm': ['Option A', 'Option B'],
        },
        missing: ['m1'],
        kept: [],
      ),
      (
        name: 'several choices without it hide it',
        given: {
          'b': [true],
          'b1': [v],
          'm': ['Option A', 'Option C'],
          'm1': [v],
        },
        missing: [],
        kept: ['b', 'b1', 'm'],
      ),
      (
        name: 'an unanswered question hides its own',
        given: {
          'c1': [v],
          'm1': [v],
          'b1': [v],
        },
        missing: [],
        kept: [],
      ),
      (
        name: 'the missing ones in document order',
        given: {
          'm': ['Option B'],
          'c': ['Option B'],
          'b': [false],
        },
        missing: ['c1', 'm1', 's1'],
        kept: [],
      ),
    ];
    for (final c in cases) {
      test(c.name, () {
        final items = conditional();
        expect(missing(items, c.given), c.missing);
        if (c.missing.isEmpty) expect(shownAnswers(items, c.given).keys, c.kept);
      });
    }

    test('a hidden section hides every item in it', () {
      final on = enabled(conditional(), {
        'b': [true],
      });
      expect([on['sec'], on['s1'], on['s2'], on['x1']], [false, false, false, false]);
      expect([on['b'], on['b1'], on['c'], on['m']], [true, true, true, true]);
    });
  });

  group('a typed answer, checked as the server checks it', () {
    final cases = <({ItemType type, String typed, Object? value, String? error})>[
      (type: ItemType.string, typed: 'a' * 255, value: 'a' * 255, error: null),
      (type: ItemType.string, typed: 'é' * 255, value: 'é' * 255, error: null),
      (type: ItemType.string, typed: 'a' * 256, value: null, error: 'Enter up to 255 characters.'),
      (type: ItemType.string, typed: '   ', value: null, error: null),
      (type: ItemType.string, typed: '', value: null, error: null),
      (type: ItemType.text, typed: 'a' * 4000, value: 'a' * 4000, error: null),
      (type: ItemType.text, typed: 'a' * 4001, value: null, error: 'Enter up to 4000 characters.'),
      (type: ItemType.integer, typed: '2147483647', value: 2147483647, error: null),
      (type: ItemType.integer, typed: '-2147483648', value: -2147483648, error: null),
      (type: ItemType.integer, typed: ' 42 ', value: 42, error: null),
      (type: ItemType.integer, typed: '2147483648', value: null, error: 'This number is too large.'),
      (type: ItemType.integer, typed: '-2147483649', value: null, error: 'This number is too large.'),
      (type: ItemType.integer, typed: '99999999999999999999', value: null, error: 'This number is too large.'),
      (type: ItemType.integer, typed: '1.5', value: null, error: 'Enter a whole number.'),
      (type: ItemType.integer, typed: 'abc', value: null, error: 'Enter a whole number.'),
      (type: ItemType.decimal, typed: '123456789012345', value: 123456789012345, error: null),
      (type: ItemType.decimal, typed: '-123456789012.345', value: -123456789012.345, error: null),
      (type: ItemType.decimal, typed: '1.50', value: 1.5, error: null),
      (type: ItemType.decimal, typed: '0', value: 0, error: null),
      (type: ItemType.decimal, typed: '0.000001', value: 0.000001, error: null),
      (type: ItemType.decimal, typed: '1234567890123456', value: null, error: 'This number is too large.'),
      (type: ItemType.decimal, typed: '1234567890123.456', value: null, error: 'Enter a number.'),
      (type: ItemType.decimal, typed: '0.0000001', value: null, error: 'Enter a number.'),
      (type: ItemType.decimal, typed: '1e3', value: null, error: 'Enter a number.'),
      (type: ItemType.decimal, typed: '01', value: null, error: 'Enter a number.'),
      (type: ItemType.decimal, typed: '.5', value: null, error: 'Enter a number.'),
      (type: ItemType.decimal, typed: '1,5', value: null, error: 'Enter a number.'),
    ];
    for (final c in cases) {
      final shown = c.typed.length > 20 ? '${c.typed.substring(0, 12)}… (${c.typed.length})' : c.typed;
      test('${c.type.name} "$shown"', () {
        final read = readTyped(c.type, c.typed);
        expect(read.error, c.error);
        expect(read.value, c.value);
      });
    }
  });

  group('the FHIR resources', () {
    test('a version: items, sections, help text, conditions and options', () {
      final v = Version.fromFhir(everyType());

      expect(v.title, 'Placeholder intake questionnaire');
      expect(v.canonical, '$questionnaireUrl|1');
      expect([for (final i in v.items) i.linkId], ['intro', 'smoker', 'freq', 'details']);
      final smoker = v.items[1];
      expect((smoker.type, smoker.required, smoker.help), (ItemType.boolean, true, 'Placeholder help text'));
      expect(smoker.items, isEmpty, reason: 'the help item is the help text, not a child');
      final freq = v.items[2];
      expect((freq.condition!.question, freq.condition!.boolean), ('smoker', true));
      expect(freq.options, ['Placeholder option A', 'Placeholder option B']);
      final details = v.items[3];
      expect(
        (details.type, details.help, details.condition!.string),
        (ItemType.group, 'Placeholder section help', 'Placeholder option B'),
      );
      expect(
        [for (final i in details.items) i.type],
        [ItemType.string, ItemType.text, ItemType.choice, ItemType.integer, ItemType.decimal, ItemType.date],
      );
      expect(details.items[2].repeats, isTrue);
    });

    test('an item type the app does not know is kept as unsupported', () {
      final v = Version.fromFhir(
        questionnaire(
          items: [
            {'linkId': 'att', 'text': 'Placeholder upload', 'type': 'attachment'},
          ],
        ),
      );
      expect(v.items.single.type, ItemType.unsupported);
    });

    test("a response: its appointment, version, status and the answers flattened out of the sections", () {
      final r = Response.fromFhir(
        response(
          'r1',
          status: 'completed',
          authored: '2026-11-04T08:00:00Z',
          items: [
            {
              'linkId': 'details',
              'item': [
                {
                  'linkId': 'name',
                  'answer': [
                    {'valueString': 'Placeholder answer'},
                  ],
                },
                {
                  'linkId': 'count',
                  'answer': [
                    {'valueInteger': 3},
                  ],
                },
                {
                  'linkId': 'weight',
                  'answer': [
                    {'valueDecimal': 72.5},
                  ],
                },
                {
                  'linkId': 'since',
                  'answer': [
                    {'valueDate': '2020-01-31'},
                  ],
                },
                {
                  'linkId': 'many',
                  'answer': [
                    {'valueString': 'Placeholder option C'},
                    {'valueString': 'Placeholder option E'},
                  ],
                },
              ],
            },
            {
              'linkId': 'smoker',
              'answer': [
                {'valueBoolean': false},
              ],
            },
          ],
        ),
      );

      expect(
        (r.id, r.appointmentId, r.questionnaire, r.status, r.versionId),
        ('r1', appointmentId, '$questionnaireUrl|1', ResponseStatus.completed, 2),
      );
      expect(r.authored, DateTime.utc(2026, 11, 4, 8));
      expect(r.answers, {
        'name': ['Placeholder answer'],
        'count': [3],
        'weight': [72.5],
        'since': [DateTime.utc(2020, 1, 31)],
        'many': ['Placeholder option C', 'Placeholder option E'],
        'smoker': [false],
      });
    });

    test('the answers as the JSON API takes them, each value in its type', () {
      final items = Version.fromFhir(everyType()).items;
      final out = toApi(items, {
        'smoker': [true],
        'freq': ['Placeholder option B'],
        'name': ['Placeholder answer'],
        'many': ['Placeholder option C', 'Placeholder option D'],
        'count': [3],
        'weight': [72.5],
        'since': [DateTime.utc(2020, 1, 31)],
      });

      expect(
        [for (final a in out) a.toJson()],
        [
          {
            'linkId': 'smoker',
            'values': [api.AnswerValue(valueBoolean: true)],
          },
          {
            'linkId': 'freq',
            'values': [api.AnswerValue(valueString: 'Placeholder option B')],
          },
          {
            'linkId': 'name',
            'values': [api.AnswerValue(valueString: 'Placeholder answer')],
          },
          {
            'linkId': 'many',
            'values': [
              api.AnswerValue(valueString: 'Placeholder option C'),
              api.AnswerValue(valueString: 'Placeholder option D'),
            ],
          },
          {
            'linkId': 'count',
            'values': [api.AnswerValue(valueInteger: 3)],
          },
          {
            'linkId': 'weight',
            'values': [api.AnswerValue(valueDecimal: 72.5)],
          },
          {
            'linkId': 'since',
            'values': [api.AnswerValue(valueDate: DateTime.utc(2020, 1, 31))],
          },
        ],
      );
      expect(fromApi(items, out), {
        'smoker': [true],
        'freq': ['Placeholder option B'],
        'name': ['Placeholder answer'],
        'many': ['Placeholder option C', 'Placeholder option D'],
        'count': [3],
        'weight': [72.5],
        'since': [DateTime.utc(2020, 1, 31)],
      });
    });
  });
}
