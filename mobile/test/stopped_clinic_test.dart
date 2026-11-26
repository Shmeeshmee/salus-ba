import 'package:flutter_test/flutter_test.dart';

import 'support.dart';

// A visit at a clinic the directory no longer lists, its membership suspended or cancelled (docs/clinic-cms
// PROGRESS.md, "Readiness follow-ups: suspended clinics"): the app reads the clinic and the visit's role by id, which
// the API answers for the clinic's own patients, and shows its name, its zone and the visit's address. Now is
// Wednesday 4 November 2026 (support.dart).

const _west = 'b2000000-0000-4000-8000-0000000000b2';

Future<Harness> _atAppointments(
  WidgetTester tester, {
  int clinicRead = 200,
}) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  // The directory's searches no longer show the clinic.
  h.api.on(
    'GET /fhir/R4/Organization',
    (_) async => fhirResponse(200, bundle(const [])),
  );
  h.api.on(
    'GET /fhir/R4/PractitionerRole',
    (_) async => fhirResponse(200, bundle(const [])),
  );
  h.api.on(
    'GET /fhir/R4/Organization/$clinicId',
    (_) async => fhirResponse(clinicRead, organization()),
  );
  h.api.on(
    'GET /fhir/R4/PractitionerRole/$roleId',
    (_) async => fhirResponse(200, practitionerRole()),
  );
  h.api.answer('GET /organizations/$clinicId/locations', 200, [
    location(
      _west,
      'Synthetic site West',
      addressLine: 'Placeholder road 2',
      postalCode: '71210',
    ),
  ]);
  h.api.on(
    'GET /fhir/R4/Appointment',
    (_) async => fhirResponse(
      200,
      bundle([
        appointment(
          'stopped',
          '2026-11-10T08:00:00Z',
          '2026-11-10T08:30:00Z',
          mode: 'in-person',
          location: _west,
        ),
      ]),
    ),
  );
  h.api.answer('GET /me/consents', 200, {
    'consents': [consent('c1')],
  });
  await h.pump(tester);
  await tapOutlined(tester, 'My appointments');
  return h;
}

void main() {
  testWidgets(
    'a visit at a clinic out of the directory shows its name, zone and address',
    (tester) async {
      final h = await _atAppointments(tester);

      expect(
        h.api.calls,
        containsAll([
          'GET /fhir/R4/Organization/$clinicId',
          'GET /fhir/R4/PractitionerRole/$roleId',
        ]),
      );
      expect(find.text(clinicName), findsOneWidget);
      expect(find.text('09:00 to 09:30 (Europe/Sarajevo)'), findsOneWidget);
      expect(find.text('Synthetic site West'), findsOneWidget);
      expect(find.text('Placeholder road 2, 71210'), findsOneWidget);
      expect(find.text('A clinic no longer listed'), findsNothing);
    },
  );

  testWidgets('a clinic that cannot be read either is no longer listed', (
    tester,
  ) async {
    await _atAppointments(tester, clinicRead: 404);

    expect(find.text('A clinic no longer listed'), findsOneWidget);
    expect(find.text('08:00 to 08:30 (UTC)'), findsOneWidget);
    expect(find.text('Placeholder road 2, 71210'), findsNothing);
  });

  testWidgets('a listed clinic is not read by id', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    answerDirectory(h.api);
    h.api.on(
      'GET /fhir/R4/Appointment',
      (_) async => fhirResponse(
        200,
        bundle([
          appointment('listed', '2026-11-10T08:00:00Z', '2026-11-10T08:30:00Z'),
        ]),
      ),
    );
    h.api.answer('GET /me/consents', 200, {
      'consents': [consent('c1')],
    });
    await h.pump(tester);
    await tapOutlined(tester, 'My appointments');

    expect(find.text('Cy Example, $clinicName'), findsOneWidget);
    expect(
      h.api.calls.where(
        (c) =>
            c.startsWith('GET /fhir/R4/Organization/') ||
            c.startsWith('GET /fhir/R4/PractitionerRole/'),
      ),
      isEmpty,
    );
  });
}
