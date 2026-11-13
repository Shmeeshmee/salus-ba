import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'support.dart';

// SAL-41: the allergies in the patient's profile (docs/plans/patient-history.md "SAL-41"). The resources have the
// shapes of internal/allergyintolerance/testdata. Synthetic data only.

const ownId = 'a1000000-0000-4000-8000-0000000000a1';
const clinicEntryId = 'a1000000-0000-4000-8000-0000000000a2';
const removedId = 'a1000000-0000-4000-8000-0000000000a3';
const termId = 'e7000000-0000-4000-8000-0000000000e7';

Map<String, Object?> _coding(String system, String code) => {
  'coding': [
    {'system': system, 'code': code},
  ],
};

Map<String, Object?> allergy({
  String id = ownId,
  String text = 'Amoxicillin',
  String version = '1',
  String verification = 'unconfirmed',
  String clinical = 'active',
  bool clinic = false,
  List<String> category = const ['medication'],
  String? reaction = 'Placeholder reaction',
  String? severity = 'moderate',
}) => {
  'resourceType': 'AllergyIntolerance',
  'id': id,
  'meta': {'lastUpdated': '2026-11-03T09:30:00Z', 'versionId': version},
  'clinicalStatus': _coding('http://terminology.hl7.org/CodeSystem/allergyintolerance-clinical', clinical),
  'verificationStatus': _coding('http://terminology.hl7.org/CodeSystem/allergyintolerance-verification', verification),
  'code': {'text': text},
  'patient': {'reference': 'Patient/$patientId'},
  'recordedDate': '2026-11-03T09:30:00Z',
  if (category.isNotEmpty) 'category': category,
  if (reaction != null)
    'reaction': [
      {
        'severity': ?severity,
        'manifestation': [
          {'text': reaction},
        ],
      },
    ],
  if (clinic) ...{
    'encounter': {'reference': 'Encounter/f1000000-0000-4000-8000-0000000000f1'},
    'recorder': {'reference': 'PractitionerRole/$roleId'},
  } else
    'asserter': {'reference': 'Patient/$patientId'},
};

Map<String, Object?> term() => {
  'id': termId,
  'category': 'drug',
  'reviewStatus': 'approved',
  'codes': {'atc': 'J01CA04'},
  'labels': [
    {'language': 'en', 'preferred': 'Amoxicillin', 'synonyms': <String>[]},
  ],
};

/// A signed-in patient with a profile, on the allergies screen with [entries].
Future<Harness> openAllergies(WidgetTester tester, List<Map<String, Object?>> entries) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  h.api.on('GET /fhir/R4/AllergyIntolerance', (_) async => fhirResponse(200, bundle(entries)));
  await h.pump(tester);
  await tapOutlined(tester, 'My profile');
  await tapOutlined(tester, 'Allergies');
  return h;
}

/// The JSON body of the last [route] request, without the null fields the generated client writes for absent ones.
Map<String, Object?> body(Harness h, String route) => sent(h.api.last(route).body);


Future<void> search(WidgetTester tester, String label, String text) async {
  await enter(tester, label, text);
  await tester.pump(const Duration(milliseconds: 400));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('lists the patient\'s allergies from the FHIR search, without a removed one', (tester) async {
    final h = await openAllergies(tester, [
      allergy(),
      allergy(
        id: clinicEntryId,
        text: 'Placeholder substance',
        verification: 'confirmed',
        clinic: true,
        category: ['food'],
        reaction: null,
      ),
      allergy(id: removedId, text: 'Placeholder removed', verification: 'entered-in-error'),
    ]);

    final search = h.api.last('GET /fhir/R4/AllergyIntolerance');
    expect(search.url.queryParameters, {'patient': 'Patient/$patientId'});
    expect(search.headers['Accept'], 'application/fhir+json');
    expect(find.text('Amoxicillin'), findsOneWidget);
    expect(find.text('Medication'), findsOneWidget);
    expect(find.text('Reaction: Placeholder reaction (Moderate)'), findsOneWidget);
    expect(find.text('Placeholder substance'), findsOneWidget);
    expect(find.text('Food'), findsOneWidget);
    expect(find.text('Confirmed by a clinician'), findsOneWidget);
    expect(find.text('Recorded by a clinic during one of your visits'), findsOneWidget);
    expect(find.text('Placeholder removed'), findsNothing);
    // The patient changes only their own entry; a clinic's can be added to the profile.
    expect(find.widgetWithText(OutlinedButton, 'Change'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, 'Add to my profile'), findsOneWidget);
  });

  testWidgets('says when there are none', (tester) async {
    await openAllergies(tester, const []);

    expect(find.text('You have not added any allergies.'), findsOneWidget);
  });

  testWidgets('adds an allergy picked from the terminology with its term and its label', (tester) async {
    final h = await openAllergies(tester, const []);
    h.api.answer('GET /terminology/search', 200, {
      'terms': [term()],
    });
    h.api.answer('POST /me/allergies', 201, {'id': ownId, 'versionId': '1'});
    await tapButton(tester, 'Add an allergy');

    await search(tester, 'Substance or food', 'amox');
    final query = h.api.last('GET /terminology/search').url.queryParameters;
    expect(query['q'], 'amox');
    expect(query['category'], 'drug');
    expect(query['languages'], 'en');
    h.api.on('GET /fhir/R4/AllergyIntolerance', (_) async => fhirResponse(200, bundle([allergy()])));
    await tapText(tester, 'Amoxicillin');
    await enter(tester, 'Reaction (optional)', '  Placeholder reaction  ');
    await tester.pumpAndSettle();
    await choose(tester, 'Severity', 'Severe');
    await tapButton(tester, 'Save');

    expect(body(h, 'POST /me/allergies'), {
      'termId': termId,
      'text': 'Amoxicillin',
      'reaction': 'Placeholder reaction',
      'severity': 'severe',
      'clinicalStatus': 'active',
    });
    expect(find.text('Saved to your profile.'), findsOneWidget);
    expect(find.text('Amoxicillin'), findsOneWidget);
    expectNoCsrfHeader(h.api);
  });

  testWidgets('adds a typed allergy as text only, its category kept and no severity without a reaction', (
    tester,
  ) async {
    final h = await openAllergies(tester, const []);
    h.api.answer('GET /terminology/search', 200, {
      'terms': [term()],
    });
    h.api.answer('POST /me/allergies', 201, {'id': ownId, 'versionId': '1'});
    await tapButton(tester, 'Add an allergy');

    await tapButton(tester, 'Save');
    expect(find.text('Enter a name, up to 200 characters.'), findsOneWidget);
    expect(h.api.calls, isNot(contains('POST /me/allergies')));

    await search(tester, 'Substance or food', ' Placeholder pollen ');
    await choose(tester, 'Type (optional)', 'Environment');
    await tapButton(tester, 'Save');

    expect(body(h, 'POST /me/allergies'), {
      'text': 'Placeholder pollen',
      'category': 'environment',
      'clinicalStatus': 'active',
    });
  });

  testWidgets('a refused allergy says so in the form', (tester) async {
    final h = await openAllergies(tester, const []);
    h.api.answer('GET /terminology/search', 200, {'terms': []});
    h.api.answer('POST /me/allergies', 400, error('invalid_request'));
    await tapButton(tester, 'Add an allergy');
    await search(tester, 'Substance or food', 'Placeholder');
    await tapButton(tester, 'Save');

    expect(find.text('Some details are not valid. Check them and try again.'), findsOneWidget);
  });

  testWidgets('changes a confirmed own entry at its version, without the term and with its text', (tester) async {
    final h = await openAllergies(tester, [allergy(version: '3', verification: 'confirmed')]);
    h.api.answer('PUT /allergies/$ownId', 200, {'id': ownId, 'versionId': '4'});
    await tapOutlined(tester, 'Change');

    expect(find.text('If you change this entry, it is no longer confirmed by a clinician.'), findsOneWidget);
    await choose(tester, 'Active', 'Resolved');
    await tapButton(tester, 'Save');

    final put = h.api.last('PUT /allergies/$ownId');
    expect(put.headers['If-Match'], 'W/"3"');
    expect(sent(put.body), {
      'text': 'Amoxicillin',
      'category': 'medication',
      'reaction': 'Placeholder reaction',
      'severity': 'moderate',
      'clinicalStatus': 'resolved',
    });
  });

  testWidgets('an entry changed elsewhere is read again and the patient told', (tester) async {
    final h = await openAllergies(tester, [allergy()]);
    h.api.answer('PUT /allergies/$ownId', 412);
    await tapOutlined(tester, 'Change');
    h.api.on(
      'GET /fhir/R4/AllergyIntolerance',
      (_) async => fhirResponse(200, bundle([allergy(text: 'Placeholder changed', version: '2')])),
    );
    await tapButton(tester, 'Save');

    expect(find.text('This entry was changed in the meantime. The latest version is shown.'), findsOneWidget);
    expect(find.text('Placeholder changed'), findsOneWidget);
  });

  testWidgets('removes an entry at its version after asking', (tester) async {
    final h = await openAllergies(tester, [allergy(version: '2')]);
    h.api.answer('DELETE /allergies/$ownId', 204);
    await tapLink(tester, 'Remove');

    expect(find.text('Remove this entry?'), findsOneWidget);
    h.api.on('GET /fhir/R4/AllergyIntolerance', (_) async => fhirResponse(200, bundle(const [])));
    await tapButton(tester, 'Remove');

    expect(h.api.last('DELETE /allergies/$ownId').headers['If-Match'], 'W/"2"');
    expect(find.text('The entry is removed.'), findsOneWidget);
    expect(find.text('Amoxicillin'), findsNothing);
  });

  testWidgets('keeps the entry when the removal is cancelled', (tester) async {
    final h = await openAllergies(tester, [allergy()]);
    await tapLink(tester, 'Remove');
    await tapLink(tester, 'Cancel');

    expect(h.api.calls, isNot(contains('DELETE /allergies/$ownId')));
    expect(find.text('Amoxicillin'), findsOneWidget);
  });

  testWidgets('adds a clinic\'s entry to the profile after saying who will see it', (tester) async {
    final h = await openAllergies(tester, [allergy(id: clinicEntryId, clinic: true, version: '5')]);
    h.api.answer('POST /allergies/$clinicEntryId/profile', 200, {'id': clinicEntryId, 'versionId': '6'});
    await tapOutlined(tester, 'Add to my profile');

    expect(
      find.text(
        'Every clinic you have given your consent to will then see it, without the name of the clinic that recorded it.',
      ),
      findsOneWidget,
    );
    await tapButton(tester, 'Add to my profile');

    expect(h.api.last('POST /allergies/$clinicEntryId/profile').headers['If-Match'], 'W/"5"');
    expect(find.text('It is now in your profile.'), findsOneWidget);
  });

  testWidgets('a refused action has a short message', (tester) async {
    final h = await openAllergies(tester, [allergy(id: clinicEntryId, clinic: true)]);
    h.api.answer('POST /allergies/$clinicEntryId/profile', 403, error('forbidden'));
    await tapOutlined(tester, 'Add to my profile');
    await tapButton(tester, 'Add to my profile');

    expect(find.text('You cannot change this entry.'), findsOneWidget);

    h.api.answer('DELETE /allergies/$clinicEntryId', 404, error('not_found'));
    await tapLink(tester, 'Remove');
    await tapButton(tester, 'Remove');

    expect(find.text('This entry is no longer available.'), findsOneWidget);
  });
}
