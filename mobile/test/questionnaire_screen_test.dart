import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'questionnaires.dart';
import 'support.dart';

// Now is Wednesday 4 November 2026, 09:00 UTC; the appointment starts the next day (support.dart).
const _start = '2026-11-05T08:00:00Z';
const _route = 'PUT /questionnaire-responses/r1/answers';
const _submit = 'POST /questionnaire-responses/r1/submit';

Map<String, Object?> _appointments() => bundle([appointment(appointmentId, _start, '2026-11-05T08:30:00Z')]);

/// The patient's appointments with [responses] of the golden version, and the list opened.
Future<Harness> _atAppointments(WidgetTester tester, List<Map<String, Object?>> responses, {DateTime? now}) async {
  final h = Harness(store: FakeTokenStore(token), now: now);
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.on('GET /fhir/R4/Appointment', (_) async => fhirResponse(200, _appointments()));
  h.api.answer('GET /me/consents', 200, {
    'consents': [consent('c1')],
  });
  h.api.on(
    'GET /fhir/R4/QuestionnaireResponse',
    (_) async => fhirResponse(200, bundle(responses, included: [everyType()])),
  );
  await h.pump(tester);
  await tapOutlined(tester, 'My appointments');
  return h;
}

/// The questionnaire opened from its card.
Future<Harness> _opened(WidgetTester tester, {Map<String, Object?>? r}) async {
  final h = await _atAppointments(tester, [r ?? response('r1')]);
  await tapFound(tester, find.widgetWithText(FilledButton, 'Fill in'));
  return h;
}

Finder _field(String linkId) => find.byKey(ValueKey('answer-$linkId'));

/// The page is a lazy list: a row is built once it is scrolled to. Scrolls down to [finder], from the top when
/// it is not built.
Future<void> _reveal(WidgetTester tester, Finder finder) async {
  final list = find.descendant(of: find.byType(ListView), matching: find.byType(Scrollable)).first;
  if (finder.evaluate().isEmpty) {
    tester.state<ScrollableState>(list).position.jumpTo(0);
    await tester.pump();
  }
  await tester.scrollUntilVisible(finder, 200, scrollable: list);
}

Future<void> _tapIn(WidgetTester tester, String linkId, String text) async {
  await _reveal(tester, _field(linkId));
  await tapFound(tester, find.descendant(of: _field(linkId), matching: find.text(text)));
}

Future<void> _submitTapped(WidgetTester tester) async {
  final button = find.widgetWithText(FilledButton, 'Submit');
  await _reveal(tester, button);
  await tapFound(tester, button);
}

/// The bodies of the saves, as the API received them.
List<Map<String, dynamic>> _saves(Harness h) => [
  for (final r in h.api.requests)
    if ('${r.method} ${r.url.path}' == _route) jsonDecode(r.body) as Map<String, dynamic>,
];

Future<void> _type(WidgetTester tester, String linkId, String text) async {
  await _reveal(tester, _field(linkId));
  await tester.enterText(find.descendant(of: _field(linkId), matching: find.byType(EditableText)), text);
  await tester.pump();
}

void _answerSaves(Harness h, {int from = 3}) {
  var version = from;
  h.api.on(_route, (_) async => jsonResponse(200, saved('r1', versionId: version++)));
}

/// Answers the boolean with Yes and the single choice with option B, so the section shows.
Future<void> _openSection(WidgetTester tester) async {
  await _tapIn(tester, 'smoker', 'Yes');
  await _tapIn(tester, 'freq', 'Placeholder option B');
}

/// The labels of the semantics nodes around [node], nearest first.
List<String> _ancestorLabels(SemanticsNode node) => [for (var p = node.parent; p != null; p = p.parent) p.label];

String _firstLine(String label) => label.split('\n').first;

void main() {
  group('the card in my appointments', () {
    testWidgets('each state of a request on its appointment', (tester) async {
      final h = Harness(store: FakeTokenStore(token));
      h.api.answer('GET /auth/me', 200, account());
      answerDirectory(h.api);
      h.api.on(
        'GET /fhir/R4/Appointment',
        (_) async => fhirResponse(
          200,
          bundle([
            appointment('a1', _start, '2026-11-05T08:30:00Z'),
            appointment('a2', '2026-11-06T08:00:00Z', '2026-11-06T08:30:00Z'),
            appointment('a3', '2026-11-07T08:00:00Z', '2026-11-07T08:30:00Z'),
            appointment('a4', '2026-11-04T08:50:00Z', '2026-11-04T09:20:00Z'), // started 10 minutes ago
            appointment('a5', '2026-11-08T08:00:00Z', '2026-11-08T08:30:00Z'),
          ]),
        ),
      );
      h.api.answer('GET /me/consents', 200, {
        'consents': [consent('c1')],
      });
      h.api.on(
        'GET /fhir/R4/QuestionnaireResponse',
        (_) async => fhirResponse(
          200,
          bundle(
            [
              response('r1', appointment: 'a1'),
              response(
                'r2',
                appointment: 'a2',
                items: [
                  {
                    'linkId': 'smoker',
                    'answer': [
                      {'valueBoolean': true},
                    ],
                  },
                ],
              ),
              response('r3', appointment: 'a3', status: 'completed', authored: '2026-11-03T08:00:00Z'),
              response('r4', appointment: 'a4'),
              response('r5', appointment: 'a5', status: 'stopped'),
            ],
            included: [everyType()],
          ),
        ),
      );
      await h.pump(tester);
      await tapOutlined(tester, 'My appointments');

      expect(find.text('Placeholder intake questionnaire'), findsNWidgets(4), reason: 'not the withdrawn one');
      expect(find.text('To fill in'), findsOneWidget);
      expect(find.text('In progress'), findsOneWidget);
      expect(find.text('Submitted'), findsOneWidget);
      expect(find.text('Closed'), findsOneWidget);
      expect(find.widgetWithText(FilledButton, 'Fill in'), findsOneWidget);
      expect(find.widgetWithText(FilledButton, 'Continue'), findsOneWidget);
      expect(find.widgetWithText(OutlinedButton, 'View answers'), findsOneWidget);
      expect(find.text('Please submit it before your appointment starts.'), findsNWidgets(2));
      expect(
        find.text('The appointment has started, so this questionnaire can no longer be filled in.'),
        findsOneWidget,
      );
      expect(h.api.last('GET /fhir/R4/QuestionnaireResponse').url.queryParametersAll, {
        'patient': ['Patient/$patientId'],
        '_include': ['QuestionnaireResponse:questionnaire'],
      });
    });

    testWidgets('a cancelled appointment shows no questionnaire', (tester) async {
      final h = Harness(store: FakeTokenStore(token));
      h.api.answer('GET /auth/me', 200, account());
      answerDirectory(h.api);
      h.api.on(
        'GET /fhir/R4/Appointment',
        (_) async => fhirResponse(
          200,
          bundle([appointment(appointmentId, _start, '2026-11-05T08:30:00Z', status: 'cancelled')]),
        ),
      );
      h.api.answer('GET /me/consents', 200, {
        'consents': [consent('c1')],
      });
      h.api.on(
        'GET /fhir/R4/QuestionnaireResponse',
        (_) async => fhirResponse(200, bundle([response('r1')], included: [everyType()])),
      );
      await h.pump(tester);
      await tapOutlined(tester, 'My appointments');

      expect(find.text('Cancelled'), findsOneWidget);
      expect(find.text('Placeholder intake questionnaire'), findsNothing);
    });
  });

  group('a failed questionnaire search', () {
    testWidgets('leaves the appointments shown, with Join, and no questionnaire lines', (tester) async {
      final h = Harness(store: FakeTokenStore(token), now: DateTime.utc(2026, 11, 5, 7, 55));
      h.api.answer('GET /auth/me', 200, account());
      answerDirectory(h.api);
      h.api.on('GET /fhir/R4/Appointment', (_) async => fhirResponse(200, _appointments()));
      h.api.answer('GET /me/consents', 200, {
        'consents': [consent('c1')],
      });
      h.api.on(
        'GET /fhir/R4/QuestionnaireResponse',
        (_) async => fhirResponse(500, {'resourceType': 'OperationOutcome'}),
      );
      await h.pump(tester);
      await tapOutlined(tester, 'My appointments');

      expect(find.text('Booked'), findsOneWidget);
      expect(find.widgetWithText(FilledButton, 'Join consultation'), findsOneWidget);
      expect(find.text('Placeholder intake questionnaire'), findsNothing);
      expect(find.text('Try again'), findsNothing);
    });
  });

  group('filling in', () {
    testWidgets('one page: the title, M5, sections as headings, help behind an icon, Required', (tester) async {
      _answerSaves(await _opened(tester));

      expect(find.text('Placeholder intake questionnaire'), findsOneWidget);
      expect(
        find.text('Your answers are saved as you go. $clinicName sees them only after you submit.'),
        findsOneWidget,
      );
      expect(find.text('Placeholder instructions'), findsOneWidget);
      expect(find.text('Placeholder section'), findsNothing, reason: 'hidden until option B');
      expect(find.text('Placeholder single choice'), findsNothing, reason: 'hidden until Yes');
      expect(find.text('Required'), findsOneWidget);
      expect(find.text('Placeholder help text'), findsNothing);

      await tapFound(tester, find.byTooltip('More information').first);
      expect(find.text('Placeholder help text'), findsOneWidget);

      await _openSection(tester);
      expect(find.text('Placeholder single choice'), findsOneWidget);
      expect(find.text('Select one.'), findsOneWidget);
      expect(find.text('Placeholder section'), findsOneWidget);
      expect(
        tester
            .widget<Semantics>(
              find.ancestor(of: find.text('Placeholder section'), matching: find.byType(Semantics)).first,
            )
            .properties
            .header,
        isTrue,
      );
      await _reveal(tester, find.text('Select all that apply.'));
      await _reveal(tester, find.text('Choose a date'));
      await _reveal(tester, _field('name'));
      expect(find.descendant(of: _field('name'), matching: find.text('Required')), findsOneWidget);
    });

    testWidgets('a condition shows its question and hides it again', (tester) async {
      _answerSaves((await _opened(tester)));

      await _tapIn(tester, 'smoker', 'Yes');
      expect(find.text('Placeholder single choice'), findsOneWidget);

      await _tapIn(tester, 'smoker', 'No');
      expect(find.text('Placeholder single choice'), findsNothing);
    });

    testWidgets('a choice saves at once; typing saves 2 s after it stops, not per keystroke', (tester) async {
      final h = await _opened(tester);
      _answerSaves(h);
      await _openSection(tester);

      expect(_saves(h), hasLength(2), reason: 'Yes, then option B');
      expect(_saves(h).last['answers'], [
        {
          'linkId': 'smoker',
          'values': [
            {'valueString': null, 'valueBoolean': true, 'valueInteger': null, 'valueDecimal': null, 'valueDate': null},
          ],
        },
        {
          'linkId': 'freq',
          'values': [
            {
              'valueString': 'Placeholder option B',
              'valueBoolean': null,
              'valueInteger': null,
              'valueDecimal': null,
              'valueDate': null,
            },
          ],
        },
      ]);
      expect(h.api.last(_route).headers['If-Match'], 'W/"3"', reason: 'the second save, after the first');

      await _type(tester, 'name', 'P');
      await tester.pump(const Duration(seconds: 1));
      await _type(tester, 'name', 'Pl');
      await tester.pump(const Duration(seconds: 1));
      await _type(tester, 'name', 'Placeholder');
      await tester.pump(const Duration(milliseconds: 1999));
      expect(_saves(h), hasLength(2), reason: 'still typing');

      await tester.pump(const Duration(milliseconds: 1));
      await tester.pumpAndSettle();
      expect(_saves(h), hasLength(3));
      expect((_saves(h).last['answers'] as List).last['linkId'], 'name');
      expect(find.text('Saved'), findsOneWidget);
      expectNoCsrfHeader(h.api);
    });

    testWidgets('leaving a field saves at once', (tester) async {
      final h = await _opened(tester);
      _answerSaves(h);
      await _openSection(tester);

      await _type(tester, 'count', '42');
      await _reveal(tester, _field('notes'));
      await tester.tap(find.descendant(of: _field('notes'), matching: find.byType(EditableText)));
      await tester.pump();

      expect(_saves(h), hasLength(3));
      expect((_saves(h).last['answers'] as List).last, {
        'linkId': 'count',
        'values': [
          {'valueString': null, 'valueBoolean': null, 'valueInteger': 42, 'valueDecimal': null, 'valueDate': null},
        ],
      });
      await tester.pump(const Duration(seconds: 3));
      expect(_saves(h), hasLength(3), reason: 'nothing changed since');
    });

    testWidgets("each typed type's check, shown on its field, and an invalid answer left out", (tester) async {
      final h = await _opened(tester);
      _answerSaves(h);
      await _openSection(tester);

      await _type(tester, 'count', '1.5');
      expect(find.text('Enter a whole number.'), findsOneWidget);
      await _type(tester, 'count', '2147483648');
      expect(find.text('This number is too large.'), findsOneWidget);
      await _type(tester, 'weight', 'abc');
      expect(find.text('Enter a number.'), findsOneWidget);
      await _type(tester, 'name', 'a' * 256);
      expect(find.text('Enter up to 255 characters.'), findsOneWidget);
      await _type(tester, 'notes', 'a' * 4001);
      expect(find.text('Enter up to 4000 characters.'), findsOneWidget);
      await tester.pump(const Duration(seconds: 2));
      await tester.pumpAndSettle();

      expect([for (final a in _saves(h).last['answers'] as List) a['linkId']], ['smoker', 'freq']);

      await _type(tester, 'weight', '72.5');
      await tester.pump(const Duration(seconds: 2));
      await tester.pumpAndSettle();
      expect(find.text('Enter a number.'), findsNothing);
      expect(
        (_saves(h).last['answers'] as List).firstWhere((a) => a['linkId'] == 'weight')['values'][0]['valueDecimal'],
        72.5,
      );
    });

    testWidgets('several choices and a date from the picker', (tester) async {
      final h = await _opened(tester);
      _answerSaves(h);
      await _openSection(tester);

      await _tapIn(tester, 'many', 'Placeholder option C');
      await _tapIn(tester, 'many', 'Placeholder option E');
      expect(
        (_saves(h).last['answers'] as List)
            .firstWhere((a) => a['linkId'] == 'many')['values']
            .map((v) => v['valueString']),
        ['Placeholder option C', 'Placeholder option E'],
      );

      await _tapIn(tester, 'since', 'Choose a date');
      expect(find.text('Choose a date'), findsWidgets, reason: "the picker's title");
      final picker = find.byType(DatePickerDialog);
      expect(tester.widget<DatePickerDialog>(picker).initialEntryMode, DatePickerEntryMode.calendarOnly);
      expect(find.descendant(of: picker, matching: find.byType(TextField)), findsNothing, reason: 'no typed entry');
      expect(find.descendant(of: picker, matching: find.byIcon(Icons.edit_outlined)), findsNothing);
      expect(find.descendant(of: picker, matching: find.text('OK')), findsNothing);
      expect(find.descendant(of: picker, matching: find.text('Cancel')), findsOneWidget);
      await tester.tap(find.text('15'));
      await tester.pumpAndSettle();
      await tester.tap(find.descendant(of: picker, matching: find.text('Done')));
      await tester.pumpAndSettle();

      expect(find.text('Sunday 15 November 2026'), findsOneWidget);
      expect(
        (_saves(h).last['answers'] as List).firstWhere((a) => a['linkId'] == 'since')['values'][0]['valueDate'],
        '2026-11-15',
      );
    });

    testWidgets('the saved answers are shown when it is opened again', (tester) async {
      await _atAppointments(tester, [
        response(
          'r1',
          items: [
            {
              'linkId': 'smoker',
              'answer': [
                {'valueBoolean': true},
              ],
            },
            {
              'linkId': 'freq',
              'answer': [
                {'valueString': 'Placeholder option B'},
              ],
            },
            {
              'linkId': 'details',
              'item': [
                {
                  'linkId': 'name',
                  'answer': [
                    {'valueString': 'Placeholder saved answer'},
                  ],
                },
              ],
            },
          ],
        ),
      ]);
      await tapFound(tester, find.widgetWithText(FilledButton, 'Continue'));

      await _reveal(tester, find.text('Placeholder section'));
      await _reveal(tester, find.text('Placeholder saved answer'));
    });

    testWidgets('a failed save says so and is tried again', (tester) async {
      final h = await _opened(tester);
      h.api.on(_route, (_) async => throw http.ClientException('network'));

      await _tapIn(tester, 'smoker', 'Yes');
      expect(find.text('Not saved. Check your connection. Saving will try again.'), findsOneWidget);
      final tries = _saves(h).length;

      _answerSaves(h);
      await tester.pump(const Duration(seconds: 5));
      await tester.pumpAndSettle();

      expect(_saves(h).length, tries + 1);
      expect(find.text('Saved'), findsOneWidget);
      expect(find.text('Not saved. Check your connection. Saving will try again.'), findsNothing);
    });

    for (final (code, status, text) in [
      ('consent_required', 409, 'Filling in this questionnaire needs your consent for this clinic.'),
      ('past_due', 409, 'The appointment has started, so this questionnaire can no longer be changed.'),
      ('withdrawn', 409, 'The clinic withdrew this questionnaire. You do not need to fill it in.'),
      ('submitted', 409, 'You have already submitted this questionnaire.'),
      // The appointment was cancelled (added on review of PR #24, owner).
      ('appointment_closed', 409, 'This questionnaire is not available.'),
    ]) {
      testWidgets('a refused save, $code, says so and stops saving', (tester) async {
        final h = await _opened(tester);
        h.api.answer(_route, status, error(code));

        await _tapIn(tester, 'smoker', 'Yes');

        expect(find.text(text), findsOneWidget);
        await tester.pump(const Duration(seconds: 10));
        await tester.pumpAndSettle();
        expect(_saves(h), hasLength(1), reason: 'not tried again');
        expect(find.widgetWithText(FilledButton, 'Submit'), findsNothing);
      });
    }

    testWidgets('a 412 reads the response again and shows the latest answers', (tester) async {
      final h = await _opened(tester);
      h.api.answer(_route, 412, error('version_conflict'));
      h.api.on(
        'GET /fhir/R4/QuestionnaireResponse/r1',
        (_) async => fhirResponse(
          200,
          response(
            'r1',
            versionId: 5,
            items: [
              {
                'linkId': 'smoker',
                'answer': [
                  {'valueBoolean': false},
                ],
              },
            ],
          ),
        ),
      );

      await _tapIn(tester, 'smoker', 'Yes');

      expect(find.text('These answers were changed on another device. The latest answers are shown.'), findsOneWidget);
      final no = tester.widget<RadioGroup<bool>>(
        find.descendant(of: _field('smoker'), matching: find.byType(RadioGroup<bool>)),
      );
      expect(no.groupValue, isFalse);

      _answerSaves(h, from: 6);
      await _tapIn(tester, 'smoker', 'Yes');
      expect(h.api.last(_route).headers['If-Match'], 'W/"5"');
    });

    testWidgets('a 412 whose response was withdrawn meanwhile says so (M16), and closes the form', (tester) async {
      final h = await _opened(tester);
      h.api.answer(_route, 412, error('version_conflict'));
      h.api.on(
        'GET /fhir/R4/QuestionnaireResponse/r1',
        (_) async => fhirResponse(200, response('r1', versionId: 5, status: 'stopped')),
      );

      await _tapIn(tester, 'smoker', 'Yes');

      expect(find.text('The clinic withdrew this questionnaire. You do not need to fill it in.'), findsOneWidget);
      expect(find.text('These answers were changed on another device. The latest answers are shown.'), findsNothing);
      expect(find.widgetWithText(FilledButton, 'Submit'), findsNothing);
    });

    // The question is the first line of each input's label (or of its options' group), so a screen reader says what
    // is asked on the field itself, not only the hint "Your answer".
    testWidgets('each answer is labelled with its question for TalkBack and VoiceOver', (tester) async {
      final semantics = tester.ensureSemantics();
      _answerSaves(await _opened(tester));
      await _openSection(tester);

      for (final (linkId, question) in [
        ('name', 'Placeholder short answer'),
        ('notes', 'Placeholder long answer'),
        ('count', 'Placeholder whole number'),
        ('weight', 'Placeholder decimal number'),
      ]) {
        await _reveal(tester, _field(linkId));
        final field = tester.getSemantics(find.descendant(of: _field(linkId), matching: find.byType(EditableText)));
        expect(field, isSemantics(isTextField: true), reason: linkId);
        expect(_firstLine(field.label), question, reason: linkId);
      }
      for (final (linkId, question, type) in [
        ('smoker', 'Placeholder yes or no question', RadioListTile<bool>),
        ('freq', 'Placeholder single choice', RadioListTile<String>),
        ('many', 'Placeholder multiple choice', CheckboxListTile),
      ]) {
        await _reveal(tester, _field(linkId));
        final option = tester.getSemantics(find.descendant(of: _field(linkId), matching: find.byType(type)).first);
        expect(_ancestorLabels(option).map(_firstLine), contains(question), reason: linkId);
      }
      await _reveal(tester, _field('since'));
      final date = tester.getSemantics(find.descendant(of: _field('since'), matching: find.byType(OutlinedButton)));
      expect(date, isSemantics(isButton: true));
      expect(_firstLine(date.label), 'Placeholder date');
      semantics.dispose();
    });

    testWidgets('a 422 marks the named questions', (tester) async {
      final h = await _opened(tester);
      h.api.answer(_route, 422, {
        'error': 'answers_invalid',
        'linkIds': ['smoker'],
      });

      await _tapIn(tester, 'smoker', 'Yes');

      expect(find.descendant(of: _field('smoker'), matching: find.text('Select one.')), findsOneWidget);
    });

    testWidgets('leaving while a save is pending asks first', (tester) async {
      final h = await _opened(tester);
      h.api.on(_route, (_) async => throw http.ClientException('network'));
      await _tapIn(tester, 'smoker', 'Yes');

      await tester.tap(find.byType(BackButton));
      await tester.pumpAndSettle();
      expect(find.text('Your last answer is not saved yet. Leave anyway?'), findsOneWidget);
      await tapLink(tester, 'Stay');
      expect(find.text('Placeholder intake questionnaire'), findsOneWidget);

      await tester.tap(find.byType(BackButton));
      await tester.pumpAndSettle();
      await tapLink(tester, 'Leave');
      expect(find.text('My appointments'), findsOneWidget);
      expect(find.text('To fill in'), findsOneWidget, reason: 'the list read again');
    });

    testWidgets('leaving with every answer saved does not ask', (tester) async {
      final h = await _opened(tester);
      _answerSaves(h);
      await _tapIn(tester, 'smoker', 'Yes');

      await tester.tap(find.byType(BackButton));
      await tester.pumpAndSettle();
      expect(find.text('My appointments'), findsOneWidget);
    });
  });

  group('submitting', () {
    testWidgets('submit with a required question unanswered says so on it and does not ask', (tester) async {
      final h = await _opened(tester);

      await _submitTapped(tester);

      expect(find.text('Some required questions are not answered.'), findsOneWidget);
      expect(find.descendant(of: _field('smoker'), matching: find.text('This question is required.')), findsOneWidget);
      expect(find.text('Submit your answers?'), findsNothing);
      expect(h.api.calls, isNot(contains(_submit)));
    });

    testWidgets('submit asks to confirm, then shows M13 and the read-only view', (tester) async {
      final h = await _opened(tester);
      _answerSaves(h);
      await _tapIn(tester, 'smoker', 'No');

      await _submitTapped(tester);
      expect(find.text('Submit your answers?'), findsOneWidget);
      expect(
        find.text('You cannot change them after you submit. Your practitioner at $clinicName will see them.'),
        findsOneWidget,
      );
      await tapLink(tester, 'Cancel');
      expect(h.api.calls, isNot(contains(_submit)));

      h.api.answer(
        _submit,
        200,
        saved(
          'r1',
          status: 'completed',
          versionId: 4,
          authored: '2026-11-04T09:00:00Z',
          answers: [
            {
              'linkId': 'smoker',
              'values': [
                {'valueBoolean': false},
              ],
            },
          ],
        ),
      );
      await _submitTapped(tester);
      await tester.tap(find.descendant(of: find.byType(AlertDialog), matching: find.text('Submit')));
      await tester.pumpAndSettle();

      expect(h.api.last(_submit).headers['If-Match'], 'W/"3"');
      expect(find.text('Answers submitted'), findsOneWidget);
      expect(
        find.text('Your practitioner can see them before your appointment. You can view them from My appointments.'),
        findsOneWidget,
      );
      expect(find.text('Your answers'), findsOneWidget);
      expect(find.text('Submitted Wednesday 4 November 2026, 10:00'), findsOneWidget);
      expect(find.text('Placeholder yes or no question'), findsOneWidget);
      expect(find.text('No'), findsOneWidget);
      expect(find.text('Placeholder single choice'), findsNothing, reason: 'hidden');
      expect(find.byType(EditableText), findsNothing);
      expect(find.widgetWithText(FilledButton, 'Submit'), findsNothing);
      expectNoCsrfHeader(h.api);
    });

    testWidgets('a 422 on submit marks the named questions', (tester) async {
      final h = await _opened(tester);
      _answerSaves(h);
      await _tapIn(tester, 'smoker', 'No');
      h.api.answer(_submit, 422, {
        'error': 'answers_invalid',
        'linkIds': ['smoker'],
      });

      await _submitTapped(tester);
      await tester.tap(find.descendant(of: find.byType(AlertDialog), matching: find.text('Submit')));
      await tester.pumpAndSettle();

      expect(find.text('Some required questions are not answered.'), findsOneWidget);
      expect(find.descendant(of: _field('smoker'), matching: find.text('This question is required.')), findsOneWidget);
    });

    testWidgets('a refused submit says so', (tester) async {
      final h = await _opened(tester);
      _answerSaves(h);
      await _tapIn(tester, 'smoker', 'No');
      h.api.answer(_submit, 409, error('past_due'));

      await _submitTapped(tester);
      await tester.tap(find.descendant(of: find.byType(AlertDialog), matching: find.text('Submit')));
      await tester.pumpAndSettle();

      expect(find.text('The appointment has started, so this questionnaire can no longer be changed.'), findsOneWidget);
    });

    testWidgets('a submit refused as the appointment was cancelled says it is not available, and closes', (
      tester,
    ) async {
      final h = await _opened(tester);
      _answerSaves(h);
      await _tapIn(tester, 'smoker', 'No');
      h.api.answer(_submit, 409, error('appointment_closed'));

      await _submitTapped(tester);
      await tester.tap(find.descendant(of: find.byType(AlertDialog), matching: find.text('Submit')));
      await tester.pumpAndSettle();

      expect(find.text('This questionnaire is not available.'), findsOneWidget);
      expect(find.widgetWithText(FilledButton, 'Submit'), findsNothing);
    });

    testWidgets('a submitted one opens read-only, every shown question with its answer or Not answered', (
      tester,
    ) async {
      await _atAppointments(tester, [
        response(
          'r1',
          status: 'completed',
          authored: '2026-11-03T15:30:00Z',
          items: [
            {
              'linkId': 'smoker',
              'answer': [
                {'valueBoolean': true},
              ],
            },
            {
              'linkId': 'freq',
              'answer': [
                {'valueString': 'Placeholder option B'},
              ],
            },
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
                  'linkId': 'many',
                  'answer': [
                    {'valueString': 'Placeholder option C'},
                    {'valueString': 'Placeholder option D'},
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
              ],
            },
          ],
        ),
      ]);
      await tapFound(tester, find.widgetWithText(OutlinedButton, 'View answers'));

      expect(find.text('Your answers'), findsOneWidget);
      expect(find.text('Submitted Tuesday 3 November 2026, 16:30'), findsOneWidget);
      expect(find.text('Placeholder section'), findsOneWidget);
      for (final shown in [
        'Yes',
        'Placeholder option B',
        'Placeholder answer',
        'Placeholder option C, Placeholder option D',
        '3',
        '72.5',
        'Friday 31 January 2020',
        'Not answered',
      ]) {
        await _reveal(tester, find.text(shown));
        expect(find.text(shown), findsOneWidget, reason: shown);
      }
      expect(find.text('Answers submitted'), findsNothing);
      expect(find.byType(EditableText), findsNothing);
    });
  });

  testWidgets('the screen prints nothing: no answer, question or title in a log line', (tester) async {
    final printed = <String?>[];
    final original = debugPrint;
    debugPrint = (String? message, {int? wrapWidth}) => printed.add(message);
    try {
      final h = await _opened(tester);
      _answerSaves(h);
      await _openSection(tester);
      await _type(tester, 'name', 'Placeholder secret answer');
      await tester.pump(const Duration(seconds: 2));
      await tester.pumpAndSettle();
      h.api.answer(_route, 422, {
        'error': 'answers_invalid',
        'linkIds': ['name'],
      });
      await _type(tester, 'name', 'Placeholder other answer');
      await tester.pump(const Duration(seconds: 2));
      await tester.pumpAndSettle();
    } finally {
      debugPrint = original;
    }
    expect(printed, isEmpty);
  });
}
