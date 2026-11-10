import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:salus/documents.dart';

import 'support.dart';

// SAL-17: the patient's signed documents and transcript copy per appointment (docs/consultation/SPEC.md "Patient
// documents"). Synthetic resources only. Now is Wednesday 4 November 2026, 09:00 UTC (support.dart).

const encounterId = 'ec000000-0000-4000-8000-0000000000a1';
const referenceDe = 'd0c00000-0000-4000-8000-0000000000de';
const referenceEn = 'd0c00000-0000-4000-8000-0000000000e1';
const compositionEn = 'c0000000-0000-4000-8000-0000000000e1';
const compositionDe = 'c0000000-0000-4000-8000-0000000000de';

Map<String, Object?> documentReference(String id, String composition, String title, String language, String signed) => {
  'resourceType': 'DocumentReference',
  'id': id,
  'status': 'current',
  'docStatus': 'final',
  'date': signed,
  'content': [
    {
      'attachment': {
        'contentType': 'application/pdf',
        'language': language,
        'url': 'http://api.test/document-references/$id/content',
        'title': title,
        'creation': signed,
        'size': 1234,
      },
    },
  ],
  'context': {
    'encounter': [
      {'reference': 'Encounter/$encounterId'},
    ],
    'related': [
      {'reference': 'Composition/$composition'},
    ],
  },
};

Map<String, Object?> composition() => {
  'resourceType': 'Composition',
  'id': compositionEn,
  'language': 'en',
  'status': 'final',
  'title': 'Placeholder findings report',
  'date': '2026-11-02T08:20:00Z',
  'section': [
    {
      'title': 'Placeholder findings',
      'text': {
        'status': 'generated',
        'div': '<div xmlns="http://www.w3.org/1999/xhtml"><p>Placeholder &lt;finding&gt; &amp; more</p><p>second line</p></div>',
      },
    },
    {
      'title': 'Placeholder tests',
      'text': {
        'status': 'generated',
        'div': '<div xmlns="http://www.w3.org/1999/xhtml"><ul><li>Placeholder one</li><li>Placeholder &#39;two&#39;</li></ul></div>',
      },
    },
    {
      'title': 'Placeholder control date',
      'text': {'status': 'empty', 'div': '<div xmlns="http://www.w3.org/1999/xhtml"><p>—</p></div>'},
      'emptyReason': {
        'coding': [
          {'system': 'http://terminology.hl7.org/CodeSystem/list-empty-reason', 'code': 'unavailable'},
        ],
      },
    },
  ],
};

Map<String, Object?> segment(String speaker, String original, String translation, String at) => {
  'encounter_id': encounterId,
  'segment_id': '51000000-0000-4000-8000-${at.substring(14, 16)}0000000000',
  'speaker': speaker,
  'lang': speaker == 'patient' ? 'en' : 'bs',
  'original': original,
  'translation': translation,
  'is_final': true,
  'ts_start': at,
  'ts_end': at,
};

/// The patient at their appointments, with one past (fulfilled) consultation whose encounter has [documents] and,
/// when [transcript] is set, a transcript the clinic shares (else 403).
Future<Harness> atDocuments(
  WidgetTester tester, {
  List<Map<String, Object?>>? documents,
  List<Map<String, Object?>>? transcript,
  bool encounter = true,
}) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.on(
    'GET /fhir/R4/Appointment',
    (_) async => fhirResponse(
      200,
      bundle([
        appointment('past', '2026-11-02T08:00:00Z', '2026-11-02T08:30:00Z', status: 'fulfilled'),
        appointment('later', '2026-11-09T08:00:00Z', '2026-11-09T08:30:00Z'),
      ]),
    ),
  );
  h.api.answer('GET /me/consents', 200, {
    'consents': [consent('c1')],
  });
  h.api.on(
    'GET /fhir/R4/Encounter',
    (_) async => fhirResponse(
      200,
      bundle([
        if (encounter) {'resourceType': 'Encounter', 'id': encounterId, 'status': 'finished'},
      ]),
    ),
  );
  h.api.on(
    'GET /fhir/R4/DocumentReference',
    (_) async => fhirResponse(
      200,
      bundle(
        documents ??
            [
              documentReference(referenceDe, compositionDe, 'Platzhalter Befund', 'de', '2026-11-02T08:10:00Z'),
              documentReference(
                referenceEn,
                compositionEn,
                'Placeholder findings report',
                'en',
                '2026-11-02T08:20:00Z',
              ),
            ],
      ),
    ),
  );
  h.api.on('GET /fhir/R4/Composition/$compositionEn', (_) async => fhirResponse(200, composition()));
  if (transcript == null) {
    h.api.answer('GET /encounters/$encounterId/transcript', 403, error('forbidden'));
  } else {
    h.api.answer('GET /encounters/$encounterId/transcript', 200, {'segments': transcript});
  }
  await h.pump(tester);
  await tapOutlined(tester, 'My appointments');
  return h;
}

Finder documentsButtons() => find.widgetWithText(OutlinedButton, 'Documents');

void main() {
  tests13();
  testWidgets('a past appointment has its documents; one that has not started has none yet', (tester) async {
    await atDocuments(tester);

    expect(documentsButtons(), findsOneWidget);
    final card = find.ancestor(of: documentsButtons(), matching: find.byType(Card));
    expect(find.descendant(of: card, matching: find.text('Attended')), findsOneWidget);
  });

  testWidgets("lists the encounter's signed documents, those in the app's language first", (tester) async {
    final h = await atDocuments(tester);

    await tapFound(tester, documentsButtons());

    expect(h.api.last('GET /fhir/R4/Encounter').url.queryParameters, {'appointment': 'past'});
    expect(h.api.last('GET /fhir/R4/DocumentReference').url.queryParameters, {'encounter': encounterId});
    expect(find.text('Placeholder findings report'), findsOneWidget);
    expect(find.text('Platzhalter Befund'), findsOneWidget);
    expect(
      tester.getTopLeft(find.text('Placeholder findings report')).dy,
      lessThan(tester.getTopLeft(find.text('Platzhalter Befund')).dy),
    );
    expect(find.text('Language: Deutsch'), findsOneWidget, reason: 'only a document in another language says so');
    expect(find.text('Signed on Monday 2 November 2026'), findsNWidgets(2));
    expect(find.text('Transcript'), findsNothing, reason: 'the clinic does not share it (403)');
  });

  testWidgets('says when there is no signed document yet, also before the consultation has an encounter', (
    tester,
  ) async {
    final h = await atDocuments(tester, documents: const [], encounter: false);

    await tapFound(tester, documentsButtons());

    expect(
      find.text(
        'There are no signed documents for this appointment yet. A document appears here once your practitioner signs it.',
      ),
      findsOneWidget,
    );
    expect(h.api.calls, isNot(contains('GET /fhir/R4/DocumentReference')));
  });

  testWidgets('opens a document as text, field by field, line by line', (tester) async {
    final h = await atDocuments(tester);
    await tapFound(tester, documentsButtons());

    await tapText(tester, 'Placeholder findings report');

    expect(h.api.calls, contains('GET /fhir/R4/Composition/$compositionEn'));
    for (final text in [
      'Placeholder findings',
      'Placeholder <finding> & more',
      'second line',
      'Placeholder tests',
      'Placeholder one',
      "Placeholder 'two'",
      'Placeholder control date',
      '—',
    ]) {
      expect(find.text(text), findsOneWidget, reason: text);
    }
  });

  testWidgets('hands the PDF to the share sheet, to open, save or send', (tester) async {
    final h = await atDocuments(tester);
    h.api.on(
      'GET /document-references/$referenceEn/content',
      (_) async => http.Response.bytes([0x25, 0x50, 0x44, 0x46], 200, headers: {'content-type': 'application/pdf'}),
    );
    await tapFound(tester, documentsButtons());
    await tapText(tester, 'Placeholder findings report');

    await tapButton(tester, 'Save or share the PDF');

    expect(h.files.shared, hasLength(1));
    expect(h.files.shared.single.$1, 'Placeholder findings report.pdf');
    expect(h.files.shared.single.$2, [0x25, 0x50, 0x44, 0x46]);
    expect(h.api.last('GET /document-references/$referenceEn/content').headers['Authorization'], 'Bearer $token');
  });

  testWidgets('says so when the PDF cannot be loaded', (tester) async {
    final h = await atDocuments(tester);
    h.api.answer('GET /document-references/$referenceEn/content', 500, error('internal'));
    await tapFound(tester, documentsButtons());
    await tapText(tester, 'Placeholder findings report');

    await tapButton(tester, 'Save or share the PDF');

    expect(h.files.shared, isEmpty);
    expect(find.text('Something went wrong. Try again.'), findsOneWidget);
  });

  testWidgets('shows the transcript copy when the clinic shares it, and shares it as text', (tester) async {
    final h = await atDocuments(
      tester,
      transcript: [
        segment('patient', 'Placeholder complaint', 'Placeholder prijevod', '2026-11-02T08:01:00Z'),
        segment('practitioner', 'Placeholder pitanje', 'Placeholder question', '2026-11-02T08:02:00Z'),
      ],
    );
    await tapFound(tester, documentsButtons());

    await tapOutlined(tester, 'Transcript');

    expect(find.text('You'), findsOneWidget);
    expect(find.text('Practitioner'), findsOneWidget);
    expect(find.text('Placeholder complaint'), findsOneWidget);
    expect(find.text('Translation: Placeholder question'), findsOneWidget);
    await tapButton(tester, 'Share the transcript');
    expect(h.files.sharedText.single, contains('Placeholder complaint'));
    expect(h.files.sharedText.single, contains('Practitioner'));
  });

  testWidgets('an empty transcript says so', (tester) async {
    await atDocuments(tester, transcript: const []);
    await tapFound(tester, documentsButtons());

    await tapOutlined(tester, 'Transcript');

    expect(find.text('Nothing was transcribed in this consultation.'), findsOneWidget);
    expect(find.widgetWithText(FilledButton, 'Share the transcript'), findsNothing);
  });

  test("the narrative's lines, unescaped", () {
    expect(narrativeLines('<div xmlns="x"><p>a &amp; b</p><p>&#34;c&#34;</p></div>'), ['a & b', '"c"']);
    expect(narrativeLines('<div xmlns="x"><ul><li>one</li><li>two</li></ul></div>'), ['one', 'two']);
    expect(narrativeLines('<div xmlns="x"></div>'), isEmpty);
  });

  test('a file name from a title', () {
    expect(pdfName('Placeholder: report / 2'), 'Placeholder- report - 2.pdf');
    expect(pdfName('  '), 'document.pdf');
  });
}

// SAL-13: a tagged term of a signed document or of the transcript copy is explained on a tap, in the app's language
// (else English), labelled informational and, until a doctor approved it, not reviewed yet. The text is shown as it
// was signed or said: an overlay only.
Map<String, Object?> explanation(String id, String text, {bool approved = false}) => {
  'termId': id,
  'language': 'en',
  'text': text,
  'reviewStatus': approved ? 'approved' : 'unreviewed',
};

Map<String, Object?> span(String id, int start, int end) => {'term_id': id, 'start': start, 'end': end};

void explainTerms(Harness h) {
  h.api.answer('GET /terminology/explanations', 200, {
    'explanations': [explanation('t-finding', 'Placeholder explanation of a finding.')],
  });
  h.api.answer('GET /compositions/$compositionEn/terms', 200, {
    'sections': [
      {
        'key': 'findings',
        'lines': [
          [span('t-finding', 13, 20)],
          [span('t-unexplained', 0, 6)],
        ],
      },
      {'key': 'tests', 'lines': []},
    ],
  });
}

void tests13() {
  testWidgets('explains a tagged term of a signed document on a tap, the document as it was signed', (tester) async {
    final h = await atDocuments(tester);
    explainTerms(h);
    await tapFound(tester, documentsButtons());
    await tapText(tester, 'Placeholder findings report');

    expect(h.api.calls, contains('GET /compositions/$compositionEn/terms'));
    expect(h.api.last('GET /terminology/explanations').url.queryParameters['language'], 'en');
    for (final text in ['Placeholder <finding> & more', 'second line', 'Placeholder one']) {
      expect(find.text(text), findsOneWidget, reason: text);
    }
    // The tap lands on the range's first box: it starts on the term (its own styled run), not on the "<" before it.
    await tester.tapOnText(find.textRange.ofSubstring('finding>'));
    await tester.pumpAndSettle();
    final dialog = find.byType(AlertDialog);
    expect(find.descendant(of: dialog, matching: find.text('finding')), findsOneWidget);
    expect(find.descendant(of: dialog, matching: find.text('Placeholder explanation of a finding.')), findsOneWidget);
    expect(find.descendant(of: dialog, matching: find.text('Informational, not a diagnosis')), findsOneWidget);
    expect(find.descendant(of: dialog, matching: find.text('Not reviewed yet')), findsOneWidget);
    await tapLink(tester, 'Close');
    expect(dialog, findsNothing);
  });

  testWidgets('shows the document as text when its terms cannot be read', (tester) async {
    final h = await atDocuments(tester);
    h.api.answer('GET /compositions/$compositionEn/terms', 500, error('internal'));
    await tapFound(tester, documentsButtons());
    await tapText(tester, 'Placeholder findings report');

    expect(find.text('Placeholder <finding> & more'), findsOneWidget);
    await tester.tap(find.text('Placeholder <finding> & more'));
    await tester.pumpAndSettle();
    expect(find.byType(AlertDialog), findsNothing);
  });

  testWidgets("explains a tagged term of the transcript copy's translation on a tap", (tester) async {
    final line = segment('practitioner', 'Placeholder pitanje', 'Placeholder question', '2026-11-02T08:02:00Z')
      ..['terms'] = {
        'translation': [span('t-question', 12, 20)],
      };
    final h = await atDocuments(tester, transcript: [line]);
    h.api.answer('GET /terminology/explanations', 200, {
      'explanations': [explanation('t-question', 'Placeholder explanation of a question.', approved: true)],
    });
    await tapFound(tester, documentsButtons());
    await tapOutlined(tester, 'Transcript');

    expect(find.text('Translation: Placeholder question'), findsOneWidget);
    await tester.tapOnText(find.textRange.ofSubstring('question'));
    await tester.pumpAndSettle();

    expect(find.text('Placeholder explanation of a question.'), findsOneWidget);
    expect(find.text('Informational, not a diagnosis'), findsOneWidget);
    expect(find.text('Not reviewed yet'), findsNothing);
  });
}
