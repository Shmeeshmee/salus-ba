import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'support.dart';

// Now is Wednesday 4 November 2026, 09:00 UTC (support.dart); the clinic is in Europe/Sarajevo (UTC+1).
const _id = 'tentative';

Map<String, Object?> _actions({
  bool confirm = true,
  bool reschedule = true,
  int left = 2,
  String? until = '2026-11-09T08:00:00Z',
}) => {
  'confirm': confirm,
  'cancel': true,
  'cancel_until': '2026-11-09T08:00:00Z',
  'reschedule': reschedule,
  'reschedule_until': ?(reschedule ? until : null),
  'reschedules_left': left,
};

Future<Harness> _atAppointments(
  WidgetTester tester, {
  String patientStatus = 'tentative',
  Map<String, Object?>? actions,
}) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.on(
    'GET /fhir/R4/Appointment',
    (_) async => fhirResponse(
      200,
      bundle([appointment(_id, '2026-11-10T08:00:00Z', '2026-11-10T08:30:00Z', patientStatus: patientStatus)]),
    ),
  );
  h.api.answer('GET /me/consents', 200, {
    'consents': [consent('c1')],
  });
  h.api.answer('GET /appointments/$_id/actions', 200, actions ?? _actions());
  h.api.on('GET /fhir/R4/Schedule', (_) async => fhirResponse(200, bundle([schedule()])));
  h.api.on(
    'GET /fhir/R4/Slot',
    (_) async => fhirResponse(200, bundle([slot('s1', '2026-11-04T13:00:00Z', '2026-11-04T13:30:00Z')])),
  );
  await h.pump(tester);
  await tapOutlined(tester, 'My appointments');
  return h;
}

void main() {
  testWidgets('an unconfirmed appointment offers Confirm, which confirms and reads the list again', (tester) async {
    final h = await _atAppointments(tester);

    expect(h.api.calls, contains('GET /appointments/$_id/actions'));
    expect(find.text('Confirmed'), findsNothing);
    h.api.answer('POST /appointments/$_id/confirm', 200, booked(_id));
    final searches = h.api.calls.where((c) => c == 'GET /fhir/R4/Appointment').length;

    await tapButton(tester, 'Confirm I will come');

    expect(h.api.calls.where((c) => c.startsWith('POST')), ['POST /appointments/$_id/confirm']);
    expect(find.text('Your appointment is confirmed.'), findsOneWidget);
    expect(h.api.calls.where((c) => c == 'GET /fhir/R4/Appointment').length, searches + 1);
    expectNoCsrfHeader(h.api);
  });

  testWidgets('a confirmed appointment says so and offers no Confirm', (tester) async {
    await _atAppointments(tester, patientStatus: 'accepted', actions: _actions(confirm: false));

    expect(find.text('Confirmed'), findsOneWidget);
    expect(find.widgetWithText(FilledButton, 'Confirm I will come'), findsNothing);
  });

  testWidgets('rescheduling shows until when and how often, in the clinic zone', (tester) async {
    await _atAppointments(tester);

    expect(find.text('The time can be changed until Monday 9 November 2026 at 09:00.'), findsOneWidget);
    expect(find.text('Time changes left: 2'), findsOneWidget);
  });

  testWidgets('without rescheduling left there is no Change the time', (tester) async {
    await _atAppointments(tester, actions: _actions(reschedule: false, left: 0));

    expect(find.widgetWithText(OutlinedButton, 'Change the time'), findsNothing);
  });

  testWidgets('reschedules to a free slot of the same role after a confirmation', (tester) async {
    final h = await _atAppointments(tester);

    await tapOutlined(tester, 'Change the time');
    expect(find.text('Choose a new time'), findsOneWidget);
    expect(h.api.last('GET /fhir/R4/Schedule').url.queryParameters, {'actor': 'PractitionerRole/$roleId'});
    expect(find.text('Booked now: Tuesday 10 November 2026 at 09:00'), findsOneWidget);

    await tapOutlined(tester, '14:00');
    expect(find.text('Move your appointment?'), findsOneWidget);
    await tapLink(tester, 'Keep it');
    expect(h.api.calls.where((c) => c.startsWith('POST')), isEmpty);

    h.api.answer('POST /appointments/$_id/reschedule', 200, booked(_id));
    await tapOutlined(tester, '14:00');
    await tapButton(tester, 'Move appointment');

    expect(h.api.calls.where((c) => c.startsWith('POST')), ['POST /appointments/$_id/reschedule']);
    expect(jsonDecode(h.api.last('POST /appointments/$_id/reschedule').body), {'slotId': 's1'});
    expect(find.text('Your appointment is moved to the new time.'), findsOneWidget);
    expect(find.text('My appointments'), findsOneWidget);
  });

  for (final (code, message) in [
    ('reschedule_window_closed', 'The time of this appointment can no longer be changed in the app. Please contact the clinic.'),
    ('reschedule_limit', "This appointment's time has been changed as often as the clinic allows."),
    ('not_reschedulable', 'The time of this appointment cannot be changed in the app.'),
    ('slot_unavailable', 'This time is no longer free. Choose another time.'),
  ]) {
    testWidgets('a reschedule the API refuses with $code says so', (tester) async {
      final h = await _atAppointments(tester);
      h.api.answer('POST /appointments/$_id/reschedule', 409, error(code));
      await tapOutlined(tester, 'Change the time');
      await tapOutlined(tester, '14:00');
      await tapButton(tester, 'Move appointment');

      expect(find.text('Choose a new time'), findsOneWidget, reason: 'stays on the picker');
      expect(find.text(message), findsOneWidget);
    });
  }

  testWidgets('a confirmation the API refuses as closed says so', (tester) async {
    final h = await _atAppointments(tester);
    h.api.answer('POST /appointments/$_id/confirm', 409, error('appointment_closed'));

    await tapButton(tester, 'Confirm I will come');

    expect(find.text('This appointment is no longer booked.'), findsOneWidget);
    expect(find.text('Your appointment is confirmed.'), findsNothing);
  });

  testWidgets('without the actions route the card still shows, without the new actions', (tester) async {
    final h = await _atAppointments(tester);
    h.api.routes.remove('GET /appointments/$_id/actions');
    await tester.pumpWidget(const SizedBox());
    await h.pump(tester);
    await tapOutlined(tester, 'My appointments');

    expect(find.widgetWithText(FilledButton, 'Confirm I will come'), findsNothing);
    expect(find.widgetWithText(OutlinedButton, 'Cancel appointment'), findsOneWidget);
  });
}
