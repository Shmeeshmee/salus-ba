import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'support.dart';

// The location of an in-person visit (docs/clinic-cms/WAVE6.md U1, U2; package LC3): the patient picks it before
// the time and sees its address on the visit. Now is Wednesday 4 November 2026, 10:00 in Europe/Sarajevo.

const _centre = 'b1000000-0000-4000-8000-0000000000b1';
const _west = 'b2000000-0000-4000-8000-0000000000b2';
const _appointment = 'e1000000-0000-4000-8000-0000000000b1';

// Invented sites and addresses.
final _locations = [
  location(_centre, 'Synthetic site Centre', addressLine: 'Placeholder street 1', postalCode: '71000'),
  location(_west, 'Synthetic site West', addressLine: 'Placeholder road 2', postalCode: '71210'),
];

/// Wednesday's free times: 14:00 at the centre, 15:00 in the west, both at the clinic.
List<Map<String, Object?>> _twoSites() => [
  slot('centre', '2026-11-04T13:00:00Z', '2026-11-04T13:30:00Z', mode: 'in-person', location: _centre),
  slot('west', '2026-11-04T14:00:00Z', '2026-11-04T14:30:00Z', mode: 'in-person', location: _west),
];

Harness _harness() {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.on('GET /fhir/R4/Schedule', (_) async => fhirResponse(200, bundle([schedule()])));
  h.api.answer('GET /organizations/$clinicId/locations', 200, _locations);
  return h;
}

Future<Harness> _atSlots(WidgetTester tester, List<Map<String, Object?>> slots, {int locations = 200}) async {
  final h = _harness();
  h.api.on('GET /fhir/R4/Slot', (_) async => fhirResponse(200, bundle(slots)));
  if (locations != 200) h.api.answer('GET /organizations/$clinicId/locations', locations, error('internal'));
  await h.pump(tester);
  await tapButton(tester, 'Book an appointment');
  await tapText(tester, clinicName);
  await tapText(tester, 'Cy Example');
  return h;
}

Finder _chip(String name) => find.widgetWithText(ChoiceChip, name);

void main() {
  testWidgets('a day at two sites asks where before the time and books the time at the site picked', (tester) async {
    final h = await _atSlots(tester, _twoSites());

    expect(find.text('Where would you like to come?'), findsOneWidget);
    expect(tester.widget<ChoiceChip>(_chip('Synthetic site Centre')).selected, isTrue);
    expect(tester.widget<ChoiceChip>(_chip('Synthetic site West')).selected, isFalse);
    expect(find.text('Placeholder street 1, 71000'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, '14:00'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, '15:00'), findsNothing);

    await tapFound(tester, _chip('Synthetic site West'));
    expect(find.text('Placeholder road 2, 71210'), findsOneWidget);
    expect(find.text('Placeholder street 1, 71000'), findsNothing);
    expect(find.widgetWithText(OutlinedButton, '14:00'), findsNothing);
    await tapOutlined(tester, '15:00');

    expect(find.text('Book this time'), findsOneWidget);
    expect(find.text('Synthetic site West'), findsOneWidget);
    expect(find.text('Placeholder road 2, 71210'), findsOneWidget);

    h.api.answer('POST /appointments', 201, booked(_appointment));
    await tapButton(tester, 'Book');
    expect(jsonDecode(h.api.last('POST /appointments').body), containsPair('slotId', 'west'));
  });

  testWidgets('long site names wrap at 390 px without overflowing', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final h = _harness();
    h.api.answer('GET /organizations/$clinicId/locations', 200, [
      location(_centre, 'Synthetic site with a long placeholder name in the old town', addressLine: 'Placeholder street 1'),
      location(_west, 'Synthetic site with a long placeholder name by the river', addressLine: 'Placeholder road 2'),
    ]);
    h.api.on('GET /fhir/R4/Slot', (_) async => fhirResponse(200, bundle(_twoSites())));
    await h.pump(tester);
    await tapButton(tester, 'Book an appointment');
    await tapText(tester, clinicName);
    await tapText(tester, 'Cy Example');

    expect(find.text('Where would you like to come?'), findsOneWidget);
    await tapFound(tester, _chip('Synthetic site with a long placeholder name by the river'));
    expect(find.widgetWithText(OutlinedButton, '15:00'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('the site picked stays picked on the next day that has it', (tester) async {
    final h = await _atSlots(tester, _twoSites());
    await tapFound(tester, _chip('Synthetic site West'));
    h.api.on(
      'GET /fhir/R4/Slot',
      (_) async => fhirResponse(
        200,
        bundle([
          slot('c2', '2026-11-05T08:00:00Z', '2026-11-05T08:30:00Z', mode: 'in-person', location: _centre),
          slot('w2', '2026-11-05T09:00:00Z', '2026-11-05T09:30:00Z', mode: 'in-person', location: _west),
        ]),
      ),
    );

    await tapLink(tester, 'Next day');

    expect(tester.widget<ChoiceChip>(_chip('Synthetic site West')).selected, isTrue);
    expect(find.widgetWithText(OutlinedButton, '10:00'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, '09:00'), findsNothing);
  });

  testWidgets('a day at one site names it and its address and asks nothing', (tester) async {
    await _atSlots(tester, [_twoSites().first]);

    expect(find.text('Where would you like to come?'), findsNothing);
    expect(find.text('Synthetic site Centre'), findsOneWidget);
    expect(find.text('Placeholder street 1, 71000'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, '14:00'), findsOneWidget);
  });

  testWidgets('a video day names no site', (tester) async {
    await _atSlots(tester, [slot('video', '2026-11-04T13:00:00Z', '2026-11-04T13:30:00Z')]);

    expect(find.text('Where would you like to come?'), findsNothing);
    expect(find.text('Synthetic site Centre'), findsNothing);
    expect(find.widgetWithText(OutlinedButton, '14:00'), findsOneWidget);
  });

  testWidgets('without the sites the times still show, every one', (tester) async {
    await _atSlots(tester, _twoSites(), locations: 500);

    expect(find.text('Where would you like to come?'), findsNothing);
    expect(find.widgetWithText(OutlinedButton, '14:00'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, '15:00'), findsOneWidget);
  });

  group('the visit', () {
    Future<Harness> atAppointments(WidgetTester tester, {String mode = 'in-person'}) async {
      final h = _harness();
      h.api.on(
        'GET /fhir/R4/Appointment',
        (_) async => fhirResponse(
          200,
          bundle([
            appointment(_appointment, '2026-11-10T08:00:00Z', '2026-11-10T08:30:00Z', mode: mode, location: _west),
          ]),
        ),
      );
      h.api.answer('GET /me/consents', 200, {
        'consents': [consent('c1')],
      });
      h.api.answer('GET /appointments/$_appointment/actions', 200, {
        'confirm': false,
        'cancel': true,
        'cancel_until': '2026-11-09T08:00:00Z',
        'reschedule': true,
        'reschedule_until': '2026-11-09T08:00:00Z',
        'reschedules_left': 2,
      });
      h.api.on('GET /fhir/R4/Slot', (_) async => fhirResponse(200, bundle(_twoSites())));
      await h.pump(tester);
      await tapOutlined(tester, 'My appointments');
      return h;
    }

    testWidgets('at the clinic shows the site and its address', (tester) async {
      final h = await atAppointments(tester);

      expect(h.api.calls, contains('GET /organizations/$clinicId/locations'));
      expect(find.text('Synthetic site West'), findsOneWidget);
      expect(find.text('Placeholder road 2, 71210'), findsOneWidget);
    });

    testWidgets('a new time is offered only at the same site', (tester) async {
      await atAppointments(tester);

      await tapOutlined(tester, 'Change the time');

      expect(find.widgetWithText(OutlinedButton, '15:00'), findsOneWidget);
      expect(find.widgetWithText(OutlinedButton, '14:00'), findsNothing, reason: 'the visit stays at its site');
    });

    testWidgets('without the sites the visit still shows, without an address', (tester) async {
      final h = _harness();
      h.api.answer('GET /organizations/$clinicId/locations', 500, error('internal'));
      h.api.on(
        'GET /fhir/R4/Appointment',
        (_) async => fhirResponse(
          200,
          bundle([
            appointment(
              _appointment,
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
      h.api.on('GET /appointments/$_appointment/actions', (_) async => http.Response('', 500));
      await h.pump(tester);
      await tapOutlined(tester, 'My appointments');

      expect(find.text('At the clinic'), findsOneWidget);
      expect(find.text('Placeholder road 2, 71210'), findsNothing);
    });
  });
}
