import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'support.dart';

// SAL-41: the current medications in the patient's profile (docs/plans/patient-history.md "SAL-41"). The
// resources have the shapes of internal/medicationstatement/testdata. Synthetic data only.

const ownId = 'b1000000-0000-4000-8000-0000000000b1';
const clinicEntryId = 'b1000000-0000-4000-8000-0000000000b2';
const termId = 'e7000000-0000-4000-8000-0000000000e8';
const _confirmed = 'https://salus.invalid/fhir/StructureDefinition/medicationstatement-confirmed';

Map<String, Object?> medication({
  String id = ownId,
  String text = 'Paracetamol',
  String version = '1',
  String status = 'active',
  bool clinic = false,
  bool confirmed = false,
  Map<String, Object?>? dosage,
}) => {
  'resourceType': 'MedicationStatement',
  'id': id,
  'meta': {'lastUpdated': '2026-11-03T09:30:00Z', 'versionId': version},
  'status': status,
  'dateAsserted': '2026-11-03T09:30:00Z',
  'subject': {'reference': 'Patient/$patientId'},
  'medicationCodeableConcept': {
    'text': text,
    'coding': [
      {'system': 'http://www.whocc.no/atc', 'code': 'N02BE01'},
    ],
  },
  if (confirmed)
    'extension': [
      {'url': _confirmed, 'valueBoolean': true},
    ],
  'dosage': [
    dosage ??
        {
          'timing': {
            'repeat': {'frequency': 2, 'period': 1, 'periodUnit': 'd'},
          },
          'doseAndRate': [
            {
              'doseQuantity': {'value': 500, 'unit': 'mg', 'system': 'http://unitsofmeasure.org', 'code': 'mg'},
            },
          ],
        },
  ],
  if (clinic) ...{
    'context': {'reference': 'Encounter/f1000000-0000-4000-8000-0000000000f1'},
    'informationSource': {'reference': 'PractitionerRole/$roleId'},
  } else
    'informationSource': {'reference': 'Patient/$patientId'},
};

Future<Harness> openMedications(WidgetTester tester, List<Map<String, Object?>> entries) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  h.api.on('GET /fhir/R4/MedicationStatement', (_) async => fhirResponse(200, bundle(entries)));
  await h.pump(tester);
  await tapOutlined(tester, 'My profile');
  await tapOutlined(tester, 'Current medications');
  return h;
}

void main() {
  testWidgets('lists the medications with their dose and how often', (tester) async {
    final h = await openMedications(tester, [
      medication(),
      medication(
        id: clinicEntryId,
        text: 'Placeholder medication',
        clinic: true,
        confirmed: true,
        status: 'stopped',
        dosage: {
          'timing': {
            'repeat': {'frequency': 1, 'period': 2, 'periodUnit': 'wk'},
          },
          'doseAndRate': [
            {
              'doseQuantity': {'value': 1, 'unit': '{tablet}', 'system': 'http://unitsofmeasure.org', 'code': '{tablet}'},
            },
          ],
        },
      ),
      medication(id: 'b1000000-0000-4000-8000-0000000000b3', text: 'Placeholder as needed', dosage: {'asNeededBoolean': true}),
      medication(id: 'b1000000-0000-4000-8000-0000000000b4', text: 'Placeholder removed', status: 'entered-in-error'),
    ]);

    expect(h.api.last('GET /fhir/R4/MedicationStatement').url.queryParameters, {'patient': 'Patient/$patientId'});
    expect(find.text('Paracetamol'), findsOneWidget);
    expect(find.text('Dose: 500 mg'), findsOneWidget);
    expect(find.text('2× a day'), findsOneWidget);
    expect(find.text('Taking it now'), findsWidgets);
    expect(find.text('Dose: 1 tablet'), findsOneWidget);
    expect(find.text('1× every 2 weeks'), findsOneWidget);
    expect(find.text('Stopped'), findsOneWidget);
    expect(find.text('Confirmed by a clinician'), findsOneWidget);
    expect(find.text('Recorded by a clinic during one of your visits'), findsOneWidget);
    expect(find.text('As needed'), findsOneWidget);
    expect(find.text('Placeholder removed'), findsNothing);
  });

  testWidgets('adds a medication picked from the terminology with its dose and how often', (tester) async {
    final h = await openMedications(tester, const []);
    expect(find.text('You have not added any medications.'), findsOneWidget);
    h.api.answer('GET /terminology/search', 200, {
      'terms': [
        {
          'id': termId,
          'category': 'drug',
          'reviewStatus': 'approved',
          'codes': {'atc': 'N02BE01'},
          'labels': [
            {'language': 'en', 'preferred': 'Paracetamol', 'synonyms': <String>[]},
          ],
        },
      ],
    });
    h.api.answer('POST /me/medications', 201, {'id': ownId, 'versionId': '1'});
    await tapButton(tester, 'Add a medication');

    await enter(tester, 'Medication', 'parac');
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpAndSettle();
    expect(h.api.last('GET /terminology/search').url.queryParameters['category'], 'drug');
    await tapText(tester, 'Paracetamol');
    await enter(tester, 'Amount', '0,5');
    await choose(tester, 'Unit', 'grams (g)');
    await enter(tester, 'Times', '3');
    await enter(tester, 'Every', '1');
    await choose(tester, 'Unit of time', 'days');
    await tapFound(tester, find.text('As needed'));
    h.api.on('GET /fhir/R4/MedicationStatement', (_) async => fhirResponse(200, bundle([medication()])));
    await tapButton(tester, 'Save');

    expect(sent(h.api.last('POST /me/medications').body), {
      'termId': termId,
      'text': 'Paracetamol',
      'status': 'active',
      'dosage': {'doseUnit': 'g', 'periodUnit': 'd', 'doseValue': 0.5, 'period': 1, 'frequency': 3, 'asNeeded': true},
    });
    expect(find.text('Saved to your profile.'), findsOneWidget);
  });

  testWidgets('asks for both parts of the dose and all three of how often', (tester) async {
    final h = await openMedications(tester, const []);
    h.api.answer('GET /terminology/search', 200, {'terms': []});
    h.api.answer('POST /me/medications', 201, {'id': ownId, 'versionId': '1'});
    await tapButton(tester, 'Add a medication');
    await enter(tester, 'Medication', 'Placeholder medication');
    await enter(tester, 'Amount', '10');
    await enter(tester, 'Times', '2');
    await tapButton(tester, 'Save');

    expect(find.text('Enter both the amount and the unit, or neither.'), findsOneWidget);
    expect(find.text('Fill in all three, or leave them empty.'), findsOneWidget);
    expect(h.api.calls, isNot(contains('POST /me/medications')));

    await enter(tester, 'Amount', '');
    await enter(tester, 'Times', '');
    await choose(tester, 'Taking it now', 'Stopped');
    await tapButton(tester, 'Save');

    expect(sent(h.api.last('POST /me/medications').body), {
      'text': 'Placeholder medication',
      'status': 'stopped',
      'dosage': <String, Object?>{},
    });
  });

  testWidgets('changes an own medication at its version with the full input', (tester) async {
    final h = await openMedications(tester, [medication(version: '2')]);
    h.api.answer('PUT /medications/$ownId', 200, {'id': ownId, 'versionId': '3'});
    await tapOutlined(tester, 'Change');

    expect(find.text('If you change this entry, it is no longer confirmed by a clinician.'), findsNothing);
    await enter(tester, 'Amount', '1000');
    await tapButton(tester, 'Save');

    final put = h.api.last('PUT /medications/$ownId');
    expect(put.headers['If-Match'], 'W/"2"');
    expect(sent(put.body), {
      'text': 'Paracetamol',
      'status': 'active',
      'dosage': {'doseUnit': 'mg', 'periodUnit': 'd', 'doseValue': 1000, 'period': 1, 'frequency': 2},
    });
  });

  testWidgets('adds a clinic\'s medication to the profile, and removes one after asking', (tester) async {
    final h = await openMedications(tester, [medication(id: clinicEntryId, clinic: true, version: '4')]);
    h.api.answer('POST /medications/$clinicEntryId/profile', 200, {'id': clinicEntryId, 'versionId': '5'});
    h.api.answer('DELETE /medications/$clinicEntryId', 204);
    expect(find.widgetWithText(OutlinedButton, 'Change'), findsNothing);

    await tapOutlined(tester, 'Add to my profile');
    await tapButton(tester, 'Add to my profile');
    expect(h.api.last('POST /medications/$clinicEntryId/profile').headers['If-Match'], 'W/"4"');
    expect(find.text('It is now in your profile.'), findsOneWidget);

    await tapLink(tester, 'Remove');
    await tapButton(tester, 'Remove');
    expect(h.api.last('DELETE /medications/$clinicEntryId').headers['If-Match'], 'W/"4"');
  });

  testWidgets('writes the dose and how often in the app\'s language, and searches in it', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, {...account(), 'language': 'bs'});
    h.api.answer('GET /terminology/search', 200, {'terms': []});
    h.api.on(
      'GET /fhir/R4/MedicationStatement',
      (_) async => fhirResponse(
        200,
        bundle([
          medication(
            dosage: {
              'timing': {
                'repeat': {'frequency': 1, 'period': 2, 'periodUnit': 'd'},
              },
              'doseAndRate': [
                {
                  'doseQuantity': {'value': 2, 'unit': '{tablet}', 'system': 'http://unitsofmeasure.org', 'code': '{tablet}'},
                },
              ],
            },
          ),
        ]),
      ),
    );
    await h.pump(tester);
    await tapOutlined(tester, 'Moj profil');
    await tapOutlined(tester, 'Trenutni lijekovi');

    expect(find.text('Doza: 2 tablete'), findsOneWidget);
    expect(find.text('1× svaka 2 dana'), findsOneWidget);

    await tapButton(tester, 'Dodaj lijek');
    await enter(tester, 'Lijek', 'para');
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpAndSettle();
    expect(h.api.last('GET /terminology/search').url.queryParameters['languages'], 'bs');
  });
}
