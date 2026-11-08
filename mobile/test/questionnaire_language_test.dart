import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salus/l10n/app_localizations.dart';

import 'questionnaires.dart';
import 'support.dart';

// The questionnaire's language (docs/i18n/SPEC.md §3 "The patient's language", COPY.md M1 to M4): it opens in the
// app's language when it has it; otherwise the patient is told which languages it has and chooses one, English
// preselected when it is one of them, else the original. The choice starts the response. Synthetic placeholders;
// each translated text is `<tag>: <original>` (questionnaires.dart).

const _start = 'POST /questionnaire-responses/r1/start';
const _save = 'PUT /questionnaire-responses/r1/answers';

/// The patient's appointment with [r] answering [version], the app in the account's [language], the list opened.
Future<Harness> _atAppointments(
  WidgetTester tester, {
  required Map<String, Object?> version,
  required Map<String, Object?> r,
  String? language,
}) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, {...account(), 'language': ?language});
  answerDirectory(h.api);
  h.api.on(
    'GET /fhir/R4/Appointment',
    (_) async =>
        fhirResponse(200, bundle([appointment(appointmentId, '2026-11-05T08:00:00Z', '2026-11-05T08:30:00Z')])),
  );
  h.api.answer('GET /me/consents', 200, {
    'consents': [consent('c1')],
  });
  h.api.on('GET /fhir/R4/QuestionnaireResponse', (_) async => fhirResponse(200, bundle([r], included: [version])));
  await h.pump(tester);
  await tapOutlined(tester, _t(language).myAppointments);
  return h;
}

AppLocalizations _t(String? language) => lookupAppLocalizations(Locale(language ?? 'en'));

/// The questionnaire opened from its card.
Future<Harness> _opened(
  WidgetTester tester, {
  required Map<String, Object?> version,
  Map<String, Object?>? r,
  String? language,
  void Function(Harness h)? answer,
}) async {
  final h = await _atAppointments(tester, version: version, r: r ?? response('r1', language: null), language: language);
  answer?.call(h);
  await tapFound(tester, find.widgetWithText(FilledButton, _t(language).fillIn));
  return h;
}

/// Bosnian, translated into English.
Map<String, Object?> _bsEn() => translated(everyType());

List<Object?> _starts(Harness h) => [
  for (final r in h.api.requests)
    if ('${r.method} ${r.url.path}' == _start) (jsonDecode(r.body) as Map)['language'],
];

void _answerStart(Harness h, {int versionId = 3}) => h.api.on(_start, (request) async {
  final language = (jsonDecode(request.body) as Map)['language'] as String;
  return jsonResponse(200, saved('r1', versionId: versionId, language: language));
});

String? _picked(WidgetTester tester) => tester.widget<RadioGroup<String>>(find.byType(RadioGroup<String>)).groupValue;

void main() {
  testWidgets("the card names the questionnaire in the app's language when it has it", (tester) async {
    await _atAppointments(tester, version: _bsEn(), r: response('r1', language: null));

    expect(find.text('en: Placeholder intake questionnaire'), findsOneWidget);
  });

  testWidgets("opens in the app's language when the questionnaire has it, and starts the response in it", (
    tester,
  ) async {
    final h = await _opened(tester, version: _bsEn(), answer: _answerStart);

    expect(_starts(h), ['en']);
    expect(find.text('en: Placeholder intake questionnaire'), findsOneWidget);
    expect(find.text('en: Placeholder yes or no question'), findsOneWidget);
    expect(find.text('Placeholder yes or no question'), findsNothing);
  });

  testWidgets('a translated option is shown translated and answered as written, at the started version', (
    tester,
  ) async {
    final h = await _opened(tester, version: _bsEn(), answer: _answerStart);
    h.api.on(_save, (_) async => jsonResponse(200, saved('r1', versionId: 4)));

    await tapText(tester, 'Yes');
    await tapText(tester, 'en: Placeholder option B');

    final saves = [
      for (final r in h.api.requests)
        if ('${r.method} ${r.url.path}' == _save) r,
    ];
    expect(saves.first.headers['If-Match'], 'W/"3"', reason: 'the version the start answered');
    final answers = (jsonDecode(saves.last.body) as Map)['answers'] as List;
    final freq = answers.cast<Map>().singleWhere((a) => a['linkId'] == 'freq');
    expect((freq['values'] as List).single['valueString'], 'Placeholder option B');
    expect(find.text('en: Placeholder short answer'), findsOneWidget, reason: 'the condition holds on the value');
  });

  testWidgets('a questionnaire not in the app\'s language: its languages are named, English preselected', (
    tester,
  ) async {
    final t = _t('de');
    final h = await _opened(tester, version: _bsEn(), language: 'de', answer: _answerStart);

    expect(_starts(h), isEmpty, reason: 'nothing is started before the patient chooses');
    expect(find.text(t.questionnaireNotInLanguage('Deutsch')), findsOneWidget);
    expect(find.text(t.questionnaireLanguages('Bosanski, English')), findsOneWidget);
    expect(_picked(tester), 'en');

    await tapButton(tester, t.continueInLanguage('English'));

    expect(_starts(h), ['en']);
    expect(find.text('en: Placeholder yes or no question'), findsOneWidget);
  });

  testWidgets('the patient may choose another of its languages', (tester) async {
    final t = _t('de');
    final h = await _opened(tester, version: _bsEn(), language: 'de', answer: _answerStart);

    await tapText(tester, 'Bosanski');
    await tapButton(tester, t.continueInLanguage('Bosanski'));

    expect(_starts(h), ['bs']);
    expect(find.text('Placeholder yes or no question'), findsOneWidget);
  });

  testWidgets('a questionnaire only in its original language: the original is preselected', (tester) async {
    final t = _t('it');
    final h = await _opened(
      tester,
      version: translated(everyType(), into: const []),
      language: 'it',
      answer: _answerStart,
    );

    expect(find.text(t.questionnaireNotInLanguage('Italiano')), findsOneWidget);
    expect(find.text(t.questionnaireLanguages('Bosanski')), findsOneWidget);
    expect(_picked(tester), 'bs');

    await tapButton(tester, t.continueInLanguage('Bosanski'));

    expect(_starts(h), ['bs']);
    expect(find.text('Placeholder yes or no question'), findsOneWidget);
  });

  testWidgets('a language no longer available is said, and no longer offered (M4)', (tester) async {
    final t = _t('de');
    final h = await _opened(tester, version: _bsEn(), language: 'de');
    h.api.answer(_start, 422, error('language_unavailable'));

    await tapButton(tester, t.continueInLanguage('English'));

    expect(find.text(t.languageUnavailable), findsOneWidget);
    expect(find.text('English'), findsNothing);
    expect(_picked(tester), 'bs');

    _answerStart(h);
    await tapButton(tester, t.continueInLanguage('Bosanski'));

    expect(_starts(h), ['en', 'bs']);
    expect(find.text('Placeholder yes or no question'), findsOneWidget);
  });

  testWidgets("the app's language refused: the patient chooses among the others", (tester) async {
    final t = _t('en');
    await _opened(tester, version: _bsEn(), answer: (h) => h.api.answer(_start, 422, error('language_unavailable')));

    expect(find.text(t.languageUnavailable), findsOneWidget);
    expect(find.text(t.questionnaireNotInLanguage('English')), findsOneWidget);
    expect(_picked(tester), 'bs');
  });

  testWidgets('a start that fails says so, and is tried again with the same choice', (tester) async {
    final t = _t('en');
    final h = await _opened(tester, version: _bsEn(), answer: (h) => h.api.answer(_start, 500, error('internal')));

    expect(find.text(t.somethingWrong), findsOneWidget);
    expect(_picked(tester), 'en');

    _answerStart(h);
    await tapButton(tester, t.continueInLanguage('English'));

    expect(_starts(h), ['en', 'en']);
    expect(find.text('en: Placeholder yes or no question'), findsOneWidget);
  });

  testWidgets('a language already locked by a saved answer: the response is read and opens in its language', (
    tester,
  ) async {
    final h = await _opened(
      tester,
      version: _bsEn(),
      answer: (h) {
        h.api.answer(_start, 409, error('language_locked'));
        h.api.on(
          'GET /fhir/R4/QuestionnaireResponse/r1',
          (_) async => fhirResponse(
            200,
            response(
              'r1',
              language: 'bs',
              versionId: 5,
              items: [
                {
                  'linkId': 'smoker',
                  'answer': [
                    {'valueBoolean': true},
                  ],
                },
              ],
            ),
          ),
        );
      },
    );

    expect(_starts(h), ['en']);
    expect(find.text('Placeholder yes or no question'), findsOneWidget);
    expect(find.text('Placeholder single choice'), findsOneWidget, reason: 'the saved answer is read');
  });

  testWidgets('a response already started opens in its language, without starting again', (tester) async {
    final h = await _opened(
      tester,
      version: _bsEn(),
      r: response('r1', language: 'bs'),
    );

    expect(_starts(h), isEmpty);
    expect(find.text('Placeholder yes or no question'), findsOneWidget);
  });

  testWidgets('submitted answers are read in the language they were answered in', (tester) async {
    await _atAppointments(
      tester,
      version: _bsEn(),
      r: response(
        'r1',
        status: 'completed',
        authored: '2026-11-03T08:00:00Z',
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
        ],
      ),
    );
    await tapOutlined(tester, 'View answers');

    expect(find.text('en: Placeholder single choice'), findsOneWidget);
    expect(find.text('en: Placeholder option B'), findsOneWidget);
  });
}
