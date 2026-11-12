import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'documents_test.dart' show composition, compositionEn, documentReference, encounterId, referenceEn;
import 'questionnaires.dart' show everyType, questionnaireUrl, response;
import 'support.dart';

// SAL-39: the patient's history across every clinic (docs/plans/patient-history.md). Synthetic items only. Now is
// Wednesday 4 November 2026, 09:00 UTC (support.dart).

const otherClinic = 'b0000000-0000-4000-8000-00000000000b';
const fileId = 'f1000000-0000-4000-8000-0000000000f1';
const responseId = 'd1000000-0000-4000-8000-0000000000d1';

Map<String, Object?> item(String type, String reference, String date, {String clinic = clinicId, String? title, String? encounter}) => {
  'date': date,
  'clinic': {'id': clinic, 'name': clinic == clinicId ? clinicName : 'Synthetic unlisted clinic'},
  'type': type,
  'reference': reference,
  'encounterId': ?encounter,
  'specialty': '394579002',
  'title': ?title,
};

/// The four kinds of item, newest first: a visit at the other (unlisted) clinic, and at the listed one a document,
/// a questionnaire's answers and a file.
List<Map<String, Object?>> items() => [
  item('visit', 'Encounter/$encounterId', '2026-11-03T08:00:00Z', clinic: otherClinic, encounter: encounterId),
  item('document', 'Composition/$compositionEn', '2026-11-02T08:20:00Z', title: 'Placeholder findings report', encounter: encounterId),
  item('questionnaire_response', 'QuestionnaireResponse/$responseId', '2026-11-01T08:00:00Z'),
  item('attachment', 'Binary/$fileId', '2026-11-01T08:00:00Z', title: 'Placeholder scan.png'),
];

/// The signed-in patient on their history, the API answering [pages] in turn (the last again after).
Future<Harness> atHistory(WidgetTester tester, {List<Map<String, Object?>>? pages}) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.answer('GET /me/consents', 200, {
    'consents': [consent('c1'), consent('c2', org: otherClinic)],
  });
  final answers = pages ?? [{'items': items()}];
  var n = 0;
  h.api.on('GET /me/history', (_) async => jsonResponse(200, answers[n < answers.length ? n++ : answers.length - 1]));
  await h.pump(tester);
  await tapOutlined(tester, 'My history');
  return h;
}

List<http.Request> historyRequests(Harness h) => [for (final r in h.api.requests) if (r.url.path == '/me/history') r];

void main() {
  testWidgets('lists the history of every clinic, newest first, with its date, specialty and clinic', (tester) async {
    final h = await atHistory(tester);

    expect(historyRequests(h).single.url.queryParameters, isEmpty, reason: 'the first page, unfiltered');
    for (final text in ['Visit', 'Placeholder findings report', 'Questionnaire answers', 'Placeholder scan.png']) {
      expect(find.descendant(of: find.byType(Card), matching: find.text(text)), findsOneWidget, reason: text);
    }
    expect(
      tester.getTopLeft(find.text('Visit')).dy,
      lessThan(tester.getTopLeft(find.text('Placeholder findings report')).dy),
    );
    expect(find.text('Monday 2 November 2026'), findsOneWidget, reason: "in the listed clinic's zone");
    expect(find.text('Tuesday 3 November 2026'), findsOneWidget, reason: 'in UTC for a clinic no longer listed');
    expect(find.text('Cardiology, Synthetic unlisted clinic'), findsOneWidget);
    expect(find.text('Cardiology, $clinicName'), findsNWidgets(3));
  });

  testWidgets('says when there is nothing yet, and when the filters leave nothing', (tester) async {
    await atHistory(tester, pages: [{'items': <Object>[]}]);
    expect(find.text('There is nothing in your history yet. Your visits, documents and answers appear here.'), findsOneWidget);

    await tester.tap(find.widgetWithText(FilterChip, 'Visits'));
    await tester.pumpAndSettle();
    expect(find.text('Nothing in your history matches these filters.'), findsOneWidget);
  });

  testWidgets('loads the next page by its cursor as the list scrolls to its end', (tester) async {
    final first = [
      for (var i = 0; i < 20; i++) item('visit', 'Encounter/e$i', '2026-10-${(30 - i).toString().padLeft(2, '0')}T08:00:00Z'),
    ];
    final h = await atHistory(
      tester,
      pages: [
        {'items': first, 'next': 'c2'},
        {'items': [item('visit', 'Encounter/last', '2026-09-01T08:00:00Z')]},
      ],
    );
    expect(historyRequests(h), hasLength(1));

    await tester.dragUntilVisible(find.text('Tuesday 1 September 2026'), find.byType(SingleChildScrollView), const Offset(0, -300));
    await tester.pumpAndSettle();

    expect(historyRequests(h), hasLength(2));
    expect(historyRequests(h).last.url.queryParameters, {'cursor': 'c2'});
    expect(find.text('Show more'), findsNothing, reason: 'the last page has no next');
  });

  testWidgets('filters by type and clinic, each change from the first page, and picks dates', (tester) async {
    final h = await atHistory(tester);

    await tester.tap(find.widgetWithText(FilterChip, 'Files'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(FilterChip, 'Visits'));
    await tester.pumpAndSettle();
    expect(historyRequests(h).last.url.queryParametersAll, {
      'type': ['visit', 'attachment'],
    });

    await tester.tap(find.text('All clinics'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Synthetic unlisted clinic').last);
    await tester.pumpAndSettle();
    expect(historyRequests(h).last.url.queryParameters['organization'], otherClinic);

    await tester.tap(find.text('Any date'));
    await tester.pumpAndSettle();
    expect(find.byType(DateRangePickerDialog), findsOneWidget, reason: 'the dates are chosen in the picker');
  });

  testWidgets("opens a visit's documents and a signed document", (tester) async {
    final h = await atHistory(tester);
    h.api.on(
      'GET /fhir/R4/DocumentReference',
      (_) async => fhirResponse(200, bundle([documentReference(referenceEn, compositionEn, 'Placeholder findings report', 'en', '2026-11-02T08:20:00Z')])),
    );
    h.api.on('GET /fhir/R4/Composition/$compositionEn', (_) async => fhirResponse(200, composition()));
    h.api.answer('GET /encounters/$encounterId/transcript', 403, error('forbidden'));

    await tapText(tester, 'Placeholder findings report');
    expect(h.api.last('GET /fhir/R4/DocumentReference').url.queryParameters, {'encounter': encounterId});
    expect(find.text('Placeholder <finding> & more'), findsOneWidget, reason: 'the document opened as text');

    await tester.pageBack();
    await tester.pumpAndSettle();
    await tapText(tester, 'Visit');
    expect(find.text('Documents'), findsWidgets);
    expect(find.text('Placeholder findings report'), findsOneWidget, reason: "the visit's documents");
  });

  testWidgets('hands a file to the share sheet in its type, and says when an item cannot be opened', (tester) async {
    final h = await atHistory(tester);
    h.api.on(
      'GET /binaries/$fileId/content',
      (_) async => http.Response.bytes([1, 2, 3], 200, headers: {'content-type': 'image/png'}),
    );

    await tapText(tester, 'Placeholder scan.png');
    expect(h.files.sharedFiles.single.$1, 'Placeholder scan.png');
    expect(h.files.sharedFiles.single.$3, 'image/png');

    h.api.answer('GET /binaries/$fileId/content', 404, error('not_found'));
    await tapText(tester, 'Placeholder scan.png');
    expect(find.text('This item could not be opened. Try again.'), findsOneWidget);
  });

  testWidgets('a later page that fails says so, and Try again reads it again', (tester) async {
    final h = await atHistory(tester, pages: [{'items': items(), 'next': 'c2'}]);
    h.api.answer('GET /me/history', 500, error('internal error'));

    await tester.ensureVisible(find.text('Show more')); // scrolled to the end, the next page loads
    await tester.pumpAndSettle();
    expect(find.text('More of your history could not be loaded.'), findsOneWidget);

    h.api.answer('GET /me/history', 200, {'items': <Object>[]});
    await tapOutlined(tester, 'Try again');
    expect(find.text('More of your history could not be loaded.'), findsNothing);
    expect(historyRequests(h).last.url.queryParameters, {'cursor': 'c2'});
  });

  testWidgets("opens a questionnaire's answers by reading that response and its version only", (tester) async {
    final h = await atHistory(tester);
    h.api.on(
      'GET /fhir/R4/QuestionnaireResponse/$responseId',
      (_) async => fhirResponse(200, response(responseId, status: 'completed', authored: '2026-11-01T08:00:00Z')),
    );
    h.api.on('GET /fhir/R4/Questionnaire', (_) async => fhirResponse(200, bundle([everyType()])));

    await tapFound(tester, find.descendant(of: find.byType(Card), matching: find.text('Questionnaire answers')));
    expect(h.api.last('GET /fhir/R4/Questionnaire').url.queryParameters, {'url': questionnaireUrl, 'version': '1'});
    expect(
      [for (final r in h.api.requests) if (r.url.path == '/fhir/R4/QuestionnaireResponse') r],
      isEmpty,
      reason: 'not every response of the patient',
    );
    expect(find.text('Placeholder intake questionnaire'), findsWidgets);
  });

  testWidgets('a page left empty with a next page does not say the history is empty', (tester) async {
    await atHistory(tester, pages: [{'items': <Object>[], 'next': 'c2'}]);
    expect(find.text('There is nothing in your history yet. Your visits, documents and answers appear here.'), findsNothing);
    expect(find.text('Show more'), findsOneWidget);
  });
}
