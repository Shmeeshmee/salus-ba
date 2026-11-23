import 'package:flutter_test/flutter_test.dart';

import 'support.dart';

/// A listed clinic's profile for patients abroad (docs/clinic-cms, package X2).
Map<String, Object?> profile({bool welcomes = true}) => {
  'languages': ['bs', 'de', 'en'],
  'description': 'Synthetic description in Bosnian',
  'translations': [
    {'language': 'en', 'description': 'Synthetic description in English'},
  ],
  'welcomesInternational': welcomes,
};

Future<Harness> atClinics(WidgetTester tester) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  await h.pump(tester);
  await tapButton(tester, 'Book an appointment');
  return h;
}

void main() {
  testWidgets('filters the clinic list by a language spoken, and back to every clinic', (tester) async {
    final h = await atClinics(tester);
    expect(h.api.last('GET /fhir/R4/Organization').url.queryParametersAll, {
      'listed': ['true'],
    });

    await choose(tester, 'Any language', 'Deutsch');

    expect(h.api.last('GET /fhir/R4/Organization').url.queryParametersAll, {
      'listed': ['true'],
      'language': ['de'],
    });
    expect(find.text(clinicName), findsOneWidget);

    await choose(tester, 'Deutsch', 'Any language');
    expect(h.api.last('GET /fhir/R4/Organization').url.queryParametersAll, {
      'listed': ['true'],
    });
  });

  testWidgets('says when no clinic speaks the language picked', (tester) async {
    final h = await atClinics(tester);
    h.api.on('GET /fhir/R4/Organization', (_) async => fhirResponse(200, bundle([])));
    await choose(tester, 'Any language', 'Türkçe');
    expect(find.text('No listed clinic speaks this language yet.'), findsOneWidget);
  });

  testWidgets("shows the clinic's profile in the app's language, with its services and prices", (tester) async {
    final h = await atClinics(tester);
    h.api.answer('GET /organizations/$clinicId/profile', 200, profile());
    h.api.answer('GET /organizations/$clinicId/services', 200, [
      {'id': 'd1', 'name': 'Placeholder exam', 'priceMinor': 5000, 'currency': 'BAM', 'durationMinutes': 30},
    ]);

    await tapText(tester, clinicName);

    expect(find.text('Synthetic description in English'), findsOneWidget);
    expect(find.text('Synthetic description in Bosnian'), findsNothing);
    expect(find.text('Welcomes patients from abroad'), findsOneWidget);
    expect(find.text('Languages spoken: Bosanski, English, Deutsch'), findsOneWidget);
    expect(find.text('Placeholder exam'), findsOneWidget);
    expect(find.text('50.00 BAM'), findsOneWidget);
    expect(find.text('Cy Example'), findsOneWidget);
    expect(h.api.last('GET /organizations/$clinicId/services').url.queryParameters['language'], 'en');
  });

  testWidgets('falls back to the description as written, and shows only the practitioners without a profile', (
    tester,
  ) async {
    final h = await atClinics(tester);
    h.api.answer('GET /organizations/$clinicId/profile', 200, {
      ...profile(welcomes: false),
      'translations': <Object?>[],
    });
    await tapText(tester, clinicName);
    expect(find.text('Synthetic description in Bosnian'), findsOneWidget);
    expect(find.text('Welcomes patients from abroad'), findsNothing);

    await tester.pageBack();
    await tester.pumpAndSettle();
    h.api.answer('GET /organizations/$clinicId/profile', 404, {'error': 'not_found'});
    await tapText(tester, clinicName);
    expect(find.textContaining('Languages spoken'), findsNothing);
    expect(find.text('Cy Example'), findsOneWidget);
  });
}
