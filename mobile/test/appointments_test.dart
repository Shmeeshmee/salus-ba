import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'support.dart';

// Now is Wednesday 4 November 2026, 09:00 UTC (support.dart).
Map<String, Object?> appointments() => bundle([
  appointment('past', '2026-11-02T08:00:00Z', '2026-11-02T08:30:00Z', status: 'fulfilled'),
  appointment('inside', '2026-11-05T08:59:59Z', '2026-11-05T09:29:59Z'), // 24 h - 1 s ahead
  appointment('edge', '2026-11-05T09:00:00Z', '2026-11-05T09:30:00Z'), // exactly 24 h ahead
  appointment('gone', '2026-11-07T08:00:00Z', '2026-11-07T08:30:00Z', status: 'cancelled'),
]);

Future<Harness> atAppointments(WidgetTester tester, {Map<String, Object?>? found}) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.on('GET /fhir/R4/Appointment', (_) async => fhirResponse(200, found ?? appointments()));
  h.api.answer('GET /me/consents', 200, {
    'consents': [consent('c1')],
  });
  await h.pump(tester);
  await tapOutlined(tester, 'My appointments');
  return h;
}

Finder cancelButtons() => find.widgetWithText(OutlinedButton, 'Cancel appointment');

Finder joinButtons() => find.widgetWithText(FilledButton, 'Join consultation');

const joinHint = 'You can join from 10 minutes before the start.';

void main() {
  testWidgets("lists the patient's appointments in the clinic's zone, upcoming first", (tester) async {
    final h = await atAppointments(tester);

    expect(h.api.last('GET /fhir/R4/Appointment').url.queryParameters, {'patient': 'Patient/$patientId'});
    expect(h.api.last('GET /fhir/R4/PractitionerRole').url.queryParameters['organization'], clinicId);
    expect(find.text('Upcoming'), findsOneWidget);
    expect(find.text('Past'), findsOneWidget);
    expect(find.text('09:59 to 10:29 (Europe/Sarajevo)'), findsOneWidget);
    expect(find.text('Thursday 5 November 2026'), findsNWidgets(2));
    expect(find.text('Cy Example, $clinicName'), findsNWidgets(4));
    expect(find.text('Cancelled'), findsOneWidget);
    expect(find.text('Attended'), findsOneWidget);
    expect(find.text('Booked'), findsNWidgets(2));
    final upcoming = tester.getTopLeft(find.text('Upcoming')).dy;
    final past = tester.getTopLeft(find.text('Past')).dy;
    expect(upcoming, lessThan(past));
    expect(tester.getTopLeft(find.text('Attended')).dy, greaterThan(past));
  });

  testWidgets('cancelling is possible until 24 hours before the start, and only for a booked appointment', (
    tester,
  ) async {
    await atAppointments(tester);

    final buttons = tester.widgetList<OutlinedButton>(cancelButtons()).toList();
    expect(buttons, hasLength(2), reason: 'none for the cancelled or past appointment');
    expect(buttons[0].onPressed, isNull, reason: '24 h - 1 s ahead');
    expect(buttons[1].onPressed, isNotNull, reason: 'exactly 24 h ahead');
    expect(find.text('Can be cancelled in the app until 24 hours before the start.'), findsOneWidget);
  });

  testWidgets('cancels after a confirmation and reads the list again', (tester) async {
    final h = await atAppointments(tester);

    await tapFound(tester, cancelButtons().at(1));
    expect(find.text('Cancel this appointment?'), findsOneWidget);
    await tapLink(tester, 'Keep it');
    expect(h.api.calls.where((c) => c.startsWith('POST')), isEmpty);

    h.api.answer('POST /appointments/edge/cancel', 200, {
      ...booked('edge', status: 'cancelled'),
      'cancellationReason': 'pat',
    });
    final searches = h.api.calls.where((c) => c == 'GET /fhir/R4/Appointment').length;
    await tapFound(tester, cancelButtons().at(1));
    await tapButton(tester, 'Cancel appointment');

    expect(h.api.calls.where((c) => c.startsWith('POST')), ['POST /appointments/edge/cancel']);
    expect(find.text('Your appointment is cancelled.'), findsOneWidget);
    expect(h.api.calls.where((c) => c == 'GET /fhir/R4/Appointment').length, searches + 1);
    expectNoCsrfHeader(h.api);
  });

  testWidgets('a cancellation the API refuses as too late says so', (tester) async {
    final h = await atAppointments(tester);
    h.api.answer('POST /appointments/edge/cancel', 409, error('cancellation_window_closed'));
    await tapFound(tester, cancelButtons().at(1));
    await tapButton(tester, 'Cancel appointment');

    expect(
      find.text('This appointment starts in less than 24 hours and can no longer be cancelled in the app.'),
      findsOneWidget,
    );
  });

  testWidgets('an appointment with a clinic no longer listed is shown in UTC', (tester) async {
    await atAppointments(
      tester,
      found: bundle([appointment('elsewhere', '2026-11-10T08:00:00Z', '2026-11-10T08:30:00Z', role: 'unknown-role')]),
    );

    expect(find.text('08:00 to 08:30 (UTC)'), findsOneWidget);
    expect(find.text('A clinic no longer listed'), findsOneWidget);
  });

  testWidgets('without a profile there are no appointments to search', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    h.api.answer('GET /me/patient', 404, error('not_found'));
    await h.pump(tester);
    await tapLink(tester, 'Not now');
    await tapOutlined(tester, 'My appointments');

    expect(find.text('You have no appointments yet.'), findsOneWidget);
    expect(h.api.calls, isNot(contains('GET /fhir/R4/Appointment')));
  });

  testWidgets('Join shows from 10 minutes before the start until 30 minutes after the end, when booked', (
    tester,
  ) async {
    await atAppointments(
      tester,
      found: bundle([
        appointment('closed', '2026-11-04T08:00:00Z', '2026-11-04T08:30:00Z'), // closed at 09:00:00
        appointment('closing', '2026-11-04T08:00:00Z', '2026-11-04T08:30:01Z'), // closes in 1 s
        appointment('opens', '2026-11-04T09:10:00Z', '2026-11-04T09:40:00Z'), // opened now
        appointment('soon', '2026-11-04T09:10:01Z', '2026-11-04T09:40:01Z'), // opens in 1 s
        appointment('gone', '2026-11-04T09:05:00Z', '2026-11-04T09:35:00Z', status: 'cancelled'),
        appointment('done', '2026-11-04T08:40:00Z', '2026-11-04T08:50:00Z', status: 'fulfilled'),
      ]),
    );

    expect(joinButtons(), findsNWidgets(2));
    expect(find.text(joinHint), findsOneWidget, reason: 'only the booked appointment whose window opens later');
  });

  testWidgets('Join appears when the window opens while the list is open', (tester) async {
    final h = await atAppointments(
      tester,
      found: bundle([appointment('soon', '2026-11-04T09:10:10Z', '2026-11-04T09:40:10Z')]),
    );
    expect(joinButtons(), findsNothing);
    expect(find.text(joinHint), findsOneWidget);

    h.now = h.now.add(const Duration(seconds: 15));
    await tester.pump(const Duration(seconds: 15));

    expect(joinButtons(), findsOneWidget);
    expect(find.text(joinHint), findsNothing);
  });
}
