import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'support.dart';

const slotId = 'a6000000-0000-4000-8000-0000000000a1';
const appointmentId = 'e1000000-0000-4000-8000-0000000000a1';

/// Answers the free slots per local day: 14:00 on Wednesday 4, 09:00 on Thursday 5 November (Europe/Sarajevo).
Future<http.Response> slotsByDay(http.Request request) async {
  final from = request.url.queryParametersAll['start']!.first;
  return fhirResponse(200, switch (from) {
    'ge2026-11-03T23:00:00Z' => bundle([slot('wed', '2026-11-04T13:00:00Z', '2026-11-04T13:30:00Z')]),
    'ge2026-11-04T23:00:00Z' => bundle([slot(slotId, '2026-11-05T08:00:00Z', '2026-11-05T08:30:00Z')]),
    _ => bundle([]),
  });
}

const consentResponsible = "$clinicName is responsible for your data. Salus processes it on the clinic's behalf.";
const consentRevoke =
    'You can revoke this consent at any time under My consents. The clinic then no longer sees your profile. '
    'It keeps your name and the details of appointments that have already taken place, as part of its care '
    'records. Your future appointments there are cancelled.';

Future<Harness> atSlots(WidgetTester tester, {DateTime? now}) async {
  final h = Harness(store: FakeTokenStore(token), now: now);
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.on('GET /fhir/R4/Schedule', (_) async => fhirResponse(200, bundle([schedule()])));
  h.api.on('GET /fhir/R4/Slot', slotsByDay);
  await h.pump(tester);
  await tapButton(tester, 'Book an appointment');
  await tapText(tester, clinicName);
  await tapText(tester, 'Cy Example');
  return h;
}

/// Thursday's 09:00, on the screen that books it.
Future<Harness> atBook(WidgetTester tester) async {
  final h = await atSlots(tester);
  await tapLink(tester, 'Next day');
  await tapOutlined(tester, '09:00');
  return h;
}

/// The appointments screen the booking leads to.
void answerAppointments(FakeApi api) {
  api.on(
    'GET /fhir/R4/Appointment',
    (_) async =>
        fhirResponse(200, bundle([appointment(appointmentId, '2026-11-05T08:00:00Z', '2026-11-05T08:30:00Z')])),
  );
  api.answer('GET /me/consents', 200, {
    'consents': [consent('c1')],
  });
}

int count(FakeApi api, String route) => api.calls.where((c) => c == route).length;

void main() {
  testWidgets("lists the listed clinics, their practitioners with specialty and free times in the clinic's zone", (
    tester,
  ) async {
    final h = await atSlots(tester);

    expect(h.api.last('GET /fhir/R4/Organization').url.queryParameters, {'listed': 'true'});
    expect(h.api.last('GET /fhir/R4/Schedule').url.queryParameters, {'actor': 'PractitionerRole/$roleId'});
    expect(find.text('Times are in Europe/Sarajevo.'), findsOneWidget);
    expect(find.text('Wednesday 4 November 2026'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, '14:00'), findsOneWidget);
    expect(h.api.last('GET /fhir/R4/Slot').url.queryParametersAll['start'], [
      'ge2026-11-03T23:00:00Z',
      'lt2026-11-04T23:00:00Z',
    ]);
    expect(tester.widget<TextButton>(find.widgetWithText(TextButton, 'Previous day')).onPressed, isNull);

    await tapLink(tester, 'Next day');
    expect(find.text('Thursday 5 November 2026'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, '09:00'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, '14:00'), findsNothing);

    await tapLink(tester, 'Previous day');
    expect(find.text('Wednesday 4 November 2026'), findsOneWidget);
  });

  testWidgets('shows the practitioners with the specialty the API gives', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    answerDirectory(h.api);
    await h.pump(tester);
    await tapButton(tester, 'Book an appointment');
    await tapText(tester, clinicName);

    expect(find.text('Cy Example'), findsOneWidget);
    expect(find.text('Cardiology'), findsOneWidget);
    expect(h.api.last('GET /fhir/R4/PractitionerRole').url.queryParameters, {
      'organization': clinicId,
      '_include': 'PractitionerRole:practitioner',
    });
  });

  testWidgets('offers the next 28 days, today first', (tester) async {
    await atSlots(tester);
    for (var i = 0; i < 27; i++) {
      await tapLink(tester, 'Next day');
    }

    expect(find.text('Tuesday 1 December 2026'), findsOneWidget);
    expect(tester.widget<TextButton>(find.widgetWithText(TextButton, 'Next day')).onPressed, isNull);
  });

  // Owner, 2026-11-07 (docs/attachments/IMPLEMENTATION_PLAN.md, package 1.7): a slot that starts within 5 minutes
  // is not offered.
  for (final (before, shown) in [
    (const Duration(minutes: 4, seconds: 59), false),
    (const Duration(minutes: 5), true),
  ]) {
    testWidgets('a time ${before.inSeconds} s away is ${shown ? '' : 'not '}offered', (tester) async {
      await atSlots(tester, now: DateTime.utc(2026, 11, 4, 13).subtract(before));

      expect(find.widgetWithText(OutlinedButton, '14:00'), shown ? findsOneWidget : findsNothing);
      expect(find.text('No free times on this day.'), shown ? findsNothing : findsOneWidget);
    });
  }

  testWidgets('a practitioner without a schedule has no open times', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    answerDirectory(h.api);
    h.api.on('GET /fhir/R4/Schedule', (_) async => fhirResponse(200, bundle([])));
    await h.pump(tester);
    await tapButton(tester, 'Book an appointment');
    await tapText(tester, clinicName);
    await tapText(tester, 'Cy Example');

    expect(find.text('This practitioner has no open times yet.'), findsOneWidget);
    expect(count(h.api, 'GET /fhir/R4/Slot'), 0);
  });

  testWidgets('the first booking with a clinic asks for consent, grants it and books', (tester) async {
    final h = await atBook(tester);
    expect(find.text('Thursday 5 November 2026'), findsOneWidget);
    expect(find.text('09:00 to 09:30 (Europe/Sarajevo)'), findsOneWidget);

    var consented = false;
    h.api.on(
      'POST /appointments',
      (_) async => consented ? jsonResponse(201, booked(appointmentId)) : jsonResponse(409, error('consent_required')),
    );
    h.api.on('POST /consents', (_) async {
      consented = true;
      return jsonResponse(201, consent('c1'));
    });
    answerAppointments(h.api);
    await tapButton(tester, 'Book');

    expect(find.text('Share your details with $clinicName'), findsOneWidget);
    for (final heading in ['What is shared', 'Who sees it', 'What for', 'Who is responsible', 'How to revoke']) {
      expect(find.text(heading), findsOneWidget, reason: heading);
    }
    expect(find.text(consentResponsible), findsOneWidget);
    expect(find.text(consentRevoke), findsOneWidget);
    expect(count(h.api, 'POST /consents'), 0);

    await tapButton(tester, 'Agree and book');

    expect(jsonDecode(h.api.last('POST /consents').body), {'organizationId': clinicId, 'kind': 'care'});
    expect(jsonDecode(h.api.last('POST /appointments').body), {'slotId': slotId});
    expect(h.api.calls.where((c) => c.startsWith('POST')).toList(), [
      'POST /appointments',
      'POST /consents',
      'POST /appointments',
    ]);
    expect(find.text('My appointments'), findsOneWidget);
    expect(find.text('Your appointment is booked.'), findsOneWidget);
    expectNoCsrfHeader(h.api);
  });

  testWidgets('a consent granted in the meantime is not asked again: it books', (tester) async {
    final h = await atBook(tester);
    var tries = 0;
    h.api.on(
      'POST /appointments',
      (_) async =>
          tries++ == 0 ? jsonResponse(409, error('consent_required')) : jsonResponse(201, booked(appointmentId)),
    );
    h.api.answer('POST /consents', 409, error('consent_active'));
    answerAppointments(h.api);
    await tapButton(tester, 'Book');
    await tapButton(tester, 'Agree and book');

    expect(find.text('Your appointment is booked.'), findsOneWidget);
    expect(count(h.api, 'POST /appointments'), 2);
  });

  testWidgets('a list that fails to load can be tried again', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    h.api.answer('GET /fhir/R4/Organization', 500, error('internal error'));
    await h.pump(tester);
    await tapButton(tester, 'Book an appointment');

    expect(find.text('Something went wrong. Try again.'), findsOneWidget);
    expect(find.textContaining('internal error'), findsNothing);

    h.api.on('GET /fhir/R4/Organization', (_) async => fhirResponse(200, bundle([])));
    await tapButton(tester, 'Try again');
    expect(find.text('No clinics are listed yet.'), findsOneWidget);
  });

  testWidgets('declining the consent books nothing', (tester) async {
    final h = await atBook(tester);
    h.api.answer('POST /appointments', 409, error('consent_required'));
    await tapButton(tester, 'Book');
    await tapLink(tester, 'Not now');

    expect(find.text('Booking with this clinic needs your consent.'), findsOneWidget);
    expect(count(h.api, 'POST /consents'), 0);
    expect(count(h.api, 'POST /appointments'), 1);
  });

  testWidgets('the consent screen opens the privacy notice and stays open', (tester) async {
    final h = await atBook(tester);
    h.api.answer('POST /appointments', 409, error('consent_required'));
    await tapButton(tester, 'Book');
    await tapLink(tester, 'Read the privacy notice');

    expect(h.opened, [privacyNoticeUrl]);
    expect(find.text('Share your details with $clinicName'), findsOneWidget);
    expect(find.text('The privacy notice could not be opened.'), findsNothing);
    expect(count(h.api, 'POST /consents'), 0);
  });

  testWidgets('a privacy notice that cannot be opened says so', (tester) async {
    final h = await atBook(tester);
    h.openSucceeds = false;
    h.api.answer('POST /appointments', 409, error('consent_required'));
    await tapButton(tester, 'Book');
    await tapLink(tester, 'Read the privacy notice');

    expect(find.text('The privacy notice could not be opened.'), findsOneWidget);
    expect(find.text('Share your details with $clinicName'), findsOneWidget);
  });

  testWidgets('without a profile it asks for it first, then books', (tester) async {
    final h = await atBook(tester);
    var saved = false;
    h.api.on(
      'POST /appointments',
      (_) async => saved ? jsonResponse(201, booked(appointmentId)) : jsonResponse(409, error('profile_required')),
    );
    h.api.on('PUT /me/patient', (_) async {
      saved = true;
      return jsonResponse(201, patient());
    });
    answerAppointments(h.api);
    await tapButton(tester, 'Book');

    expect(find.text('Complete your profile'), findsOneWidget);
    await enter(tester, 'Given name', 'Ada');
    await enter(tester, 'Family name', 'Example');
    await enter(tester, 'Date of birth', '1990-02-03');
    await tapButton(tester, 'Save');

    expect(h.api.calls.where((c) => c.startsWith('P')).toList(), [
      'POST /appointments',
      'PUT /me/patient',
      'POST /appointments',
    ]);
    expect(find.text('Your appointment is booked.'), findsOneWidget);
  });

  testWidgets('a time taken in the meantime says so, and the times are read again', (tester) async {
    final h = await atBook(tester);
    h.api.answer('POST /appointments', 409, error('slot_unavailable'));
    await tapButton(tester, 'Book');

    expect(find.text('This time is no longer free. Choose another time.'), findsOneWidget);
    expect(count(h.api, 'POST /appointments'), 1);
    expect(count(h.api, 'POST /consents'), 0);

    final before = count(h.api, 'GET /fhir/R4/Slot');
    await tapLink(tester, 'Choose another time');
    expect(find.text('Thursday 5 November 2026'), findsOneWidget);
    expect(count(h.api, 'GET /fhir/R4/Slot'), before + 1);
  });

  testWidgets('an unreachable API while booking says so', (tester) async {
    final h = await atBook(tester);
    h.api.on('POST /appointments', (_) async => throw http.ClientException('network'));
    await tapButton(tester, 'Book');

    expect(find.text('Salus could not be reached. Check your connection and try again.'), findsOneWidget);
  });

  testWidgets('every booking screen fits a phone 390 px wide without overflowing', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final h = await atBook(tester);
    var consented = false;
    h.api.on(
      'POST /appointments',
      (_) async => consented ? jsonResponse(201, booked(appointmentId)) : jsonResponse(409, error('consent_required')),
    );
    h.api.on('POST /consents', (_) async {
      consented = true;
      return jsonResponse(201, consent('c1'));
    });
    answerAppointments(h.api);
    await tapButton(tester, 'Book');
    await tapButton(tester, 'Agree and book');

    expect(find.text('Your appointment is booked.'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('prints nothing while browsing and booking', (tester) async {
    final printed = <String?>[];
    final original = debugPrint;
    debugPrint = (String? message, {int? wrapWidth}) => printed.add(message);
    try {
      final h = await atBook(tester);
      var consented = false;
      h.api.on(
        'POST /appointments',
        (_) async =>
            consented ? jsonResponse(201, booked(appointmentId)) : jsonResponse(409, error('consent_required')),
      );
      h.api.on('POST /consents', (_) async {
        consented = true;
        return jsonResponse(201, consent('c1'));
      });
      answerAppointments(h.api);
      await tapButton(tester, 'Book');
      await tapButton(tester, 'Agree and book');

      expect(find.text('Your appointment is booked.'), findsOneWidget);
    } finally {
      debugPrint = original;
    }
    expect(printed, isEmpty);
  });
}
